package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewisdalwin/gatekeeper/internal/data"
)

// Opt in with IMAGELAB_INTEGRATION_TESTS=1 go test ./cmd/api -run TestUploadAcceptanceIntegration -v.
// This starts its own PostgreSQL instance; it never uses .envrc or an existing database.
func TestUploadAcceptanceIntegration(t *testing.T) {
	if os.Getenv("IMAGELAB_INTEGRATION_TESTS") != "1" {
		t.Skip("set IMAGELAB_INTEGRATION_TESTS=1 to run isolated PostgreSQL tests")
	}
	db := startUploadTestDatabase(t)

	for _, tc := range []struct {
		name       string
		format     string
		failure    string
		exactLimit bool
	}{
		{name: "PNG accepted", format: "png"},
		{name: "JPEG accepted", format: "jpeg"},
		{name: "exact file limit allows form overhead", format: "png", exactLimit: true},
		{name: "image insert failure creates no files", format: "png", failure: "image"},
		{name: "storage failure rolls back image", format: "png", failure: "storage"},
		{name: "job insert failure rolls back and removes original", format: "png", failure: "job"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := db.Exec("TRUNCATE jobs, images CASCADE"); err != nil {
				t.Fatal(err)
			}
			// Every case has its own filesystem. Do not run these subtests in parallel.
			t.Chdir(t.TempDir())
			switch tc.failure {
			case "image", "job":
				table, condition := "images", "media_type <> 'image/png'"
				if tc.failure == "job" {
					table, condition = "jobs", "job_type <> 'image_processing'"
				}
				if _, err := db.Exec("ALTER TABLE " + table + " ADD CONSTRAINT test_reject_upload CHECK (" + condition + ")"); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if _, err := db.Exec("ALTER TABLE " + table + " DROP CONSTRAINT test_reject_upload"); err != nil {
						t.Error(err)
					}
				})
			case "storage":
				// A regular file blocks creation of the uploads directory, even when run as its owner.
				if err := os.WriteFile("uploads", []byte("keep this existing file"), 0600); err != nil {
					t.Fatal(err)
				}
			}

			var original bytes.Buffer
			picture := image.NewRGBA(image.Rect(0, 0, 4, 3))
			var err error
			if tc.format == "jpeg" {
				err = jpeg.Encode(&original, picture, nil)
			} else {
				err = png.Encode(&original, picture)
			}
			if err != nil {
				t.Fatal(err)
			}
			input := original.Bytes()
			if tc.exactLimit {
				// PNG permits a decoder to finish at IEND; padding exercises the file-byte boundary.
				padded := make([]byte, 10<<20)
				copy(padded, input)
				input = padded
			}
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			part, err := form.CreateFormFile("image", "example."+tc.format)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(input); err != nil {
				t.Fatal(err)
			}
			if err := form.Close(); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/v1/images", &body)
			request.Header.Set("Content-Type", form.FormDataContentType())
			response := httptest.NewRecorder()
			app := &application{models: data.NewModels(db), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			app.uploadImageHandler(response, request)

			var imageCount, jobCount int
			if err := db.QueryRow("SELECT count(*) FROM images").Scan(&imageCount); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow("SELECT count(*) FROM jobs").Scan(&jobCount); err != nil {
				t.Fatal(err)
			}
			if tc.failure != "" {
				if response.Code != http.StatusInternalServerError {
					t.Fatalf("want 500, got %d: %s", response.Code, response.Body.String())
				}
				if imageCount != 0 || jobCount != 0 {
					t.Fatalf("rejected upload left %d images and %d jobs", imageCount, jobCount)
				}
				if response.Header().Get("Location") != "" {
					t.Error("rejection advertised an accepted job")
				}
				var result struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Error != "the server encountered a problem and could not process your request" {
					t.Errorf("unexpected public error: %q", result.Error)
				}
				if tc.failure == "storage" {
					contents, err := os.ReadFile("uploads")
					if err != nil || string(contents) != "keep this existing file" {
						t.Fatalf("pre-existing file was changed: %v", err)
					}
				} else {
					entries, err := os.ReadDir("uploads")
					if err != nil && !os.IsNotExist(err) {
						t.Fatal(err)
					}
					if len(entries) != 0 {
						t.Fatal("rejected upload left files behind")
					}
				}
				return
			}

			if response.Code != http.StatusAccepted {
				t.Fatalf("want 202, got %d: %s", response.Code, response.Body.String())
			}
			if imageCount != 1 || jobCount != 1 {
				t.Fatalf("want one image and job, got %d and %d", imageCount, jobCount)
			}
			var accepted struct {
				ImageID   string `json:"image_id"`
				JobID     string `json:"job_id"`
				Status    string `json:"status"`
				StatusURL string `json:"status_url"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &accepted); err != nil {
				t.Fatal(err)
			}
			if accepted.JobID == "" || accepted.Status != "queued" || accepted.StatusURL != "/v1/jobs/"+accepted.JobID || response.Header().Get("Location") != accepted.StatusURL {
				t.Fatalf("incorrect acceptance response: %s", response.Body.String())
			}
			var storedName, status string
			var storedSize int64
			if err := db.QueryRow(`SELECT i.stored_filename, i.size, j.status FROM images i JOIN jobs j ON j.image_id = i.id WHERE i.id = $1 AND j.public_id = $2`, accepted.ImageID, accepted.JobID).Scan(&storedName, &storedSize, &status); err != nil {
				t.Fatal(err)
			}
			stored, err := os.ReadFile(filepath.Join("uploads", accepted.ImageID, storedName))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(stored, input) || storedSize != int64(len(input)) {
				t.Error("original bytes or recorded size differ from upload")
			}
			if storedName == "example."+tc.format || status != "queued" {
				t.Error("expected server filename and queued job")
			}
			entries, err := os.ReadDir(filepath.Join("uploads", accepted.ImageID))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Error("handler should store only the original, not variants")
			}
		})
	}
}

func startUploadTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	for _, name := range []string{"initdb", "pg_ctl"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Fatalf("integration tests require PostgreSQL 18 server tool %s: %v", name, err)
		}
	}
	// Keep the Unix socket path short. All files belong to this test instance.
	root, err := os.MkdirTemp("/tmp", "imagelab-pg-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	cluster := filepath.Join(root, "data")
	run := func(name string, args ...string) {
		t.Helper()
		if output, err := exec.Command(name, args...).CombinedOutput(); err != nil {
			serverLog, _ := os.ReadFile(filepath.Join(root, "server.log"))
			t.Fatalf("%s: %v\n%s\n%s", name, err, output, serverLog)
		}
	}
	run("initdb", "-D", cluster, "-U", "imagelab_test", "-A", "trust", "--no-locale", "--encoding=UTF8")
	run("pg_ctl", "-D", cluster, "-l", filepath.Join(root, "server.log"), "-o", "-F -c logging_collector=off -c listen_addresses='' -k "+root, "-w", "start")
	t.Cleanup(func() {
		if output, err := exec.Command("pg_ctl", "-D", cluster, "-m", "immediate", "-w", "stop").CombinedOutput(); err != nil {
			t.Errorf("stop temporary database: %v\n%s", err, output)
		}
	})
	db, err := sql.Open("postgres", "host="+root+" user=imagelab_test dbname=postgres sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil || len(paths) == 0 {
		t.Fatalf("find migrations: %v", err)
	}
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(contents)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	return db
}
