package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewisdalwin/gatekeeper/internal/data"
)

func TestVariantDownloadIntegration(t *testing.T) {
	if os.Getenv("IMAGELAB_INTEGRATION_TESTS") != "1" {
		t.Skip("set IMAGELAB_INTEGRATION_TESTS=1 to run PostgreSQL tests")
	}

	db := startUploadTestDatabase(t)
	t.Chdir(t.TempDir())

	models := data.NewModels(db)
	app := &application{
		models: models,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	ctx, cancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	defer cancel()

	// Prepare one real image and let the worker finish it.
	var original bytes.Buffer
	if err := png.Encode(
		&original, image.NewRGBA(image.Rect(0, 0, 300, 200)),
	); err != nil {
		t.Fatal(err)
	}

	img := &data.Image{
		OriginalFilename: "example.png",
		StoredFilename:   "original.png",
		MediaType:        "image/png",
		Size:             int64(original.Len()),
	}

	job, err := models.Images.AcceptUpload(ctx, img, func() error {
		dir := filepath.Join("uploads", img.ID)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return os.WriteFile(
			filepath.Join(dir, img.StoredFilename),
			original.Bytes(),
			0600,
		)
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := app.processNextImageJob(ctx); err != nil {
		t.Fatal(err)
	}

	// Call the download handler using a simulated browser request.
	download := func(imageID, name string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodGet,
			"/v1/images/"+imageID+"/variants/"+name,
			nil,
		)
		request.SetPathValue("id", imageID)
		request.SetPathValue("name", name)

		response := httptest.NewRecorder()
		app.getVariantHandler(response, request)
		return response
	}

	for _, name := range []string{"thumbnail", "preview", "display"} {
		t.Run("completed "+name, func(t *testing.T) {
			response := download(img.ID, name)

			if response.Code != http.StatusOK {
				t.Fatalf("expected 200; got %d", response.Code)
			}
			if response.Header().Get("Content-Type") != "image/png" {
				t.Fatal("expected PNG content type")
			}

			var filename string
			err := db.QueryRow(`
				SELECT stored_filename
				FROM image_variants
				WHERE image_id = $1 AND name = $2`,
				img.ID, name,
			).Scan(&filename)
			if err != nil {
				t.Fatal(err)
			}

			stored, err := os.ReadFile(
				filepath.Join("uploads", img.ID, "variants", filename),
			)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(response.Body.Bytes(), stored) {
				t.Fatal("download differs from the generated file")
			}
		})
	}

	for _, status := range []string{"queued", "processing", "failed"} {
		t.Run(status+" blocks existing file", func(t *testing.T) {
			// Artificial test setup only: preserve the files and metadata
			// while changing the status to exercise the download gate.
			if _, err := db.Exec(
				"UPDATE jobs SET status = $1 WHERE id = $2",
				status, job.ID,
			); err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if _, err := db.Exec(
					"UPDATE jobs SET status = 'completed' WHERE id = $1",
					job.ID,
				); err != nil {
					t.Error(err)
				}
			})

			response := download(img.ID, "thumbnail")
			if response.Code != http.StatusNotFound {
				t.Fatalf("expected 404; got %d", response.Code)
			}
		})
	}

	for _, tc := range []struct {
		name    string
		imageID string
		variant string
	}{
		{"unknown variant", img.ID, "poster"},
		{
			"unknown image",
			"00000000-0000-0000-0000-000000000000",
			"thumbnail",
		},
		{"malformed image ID", "not-a-uuid", "thumbnail"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := download(tc.imageID, tc.variant)
			if response.Code != http.StatusNotFound {
				t.Fatalf("expected 404; got %d", response.Code)
			}
		})
	}

	t.Run("missing file", func(t *testing.T) {
		var filename string
		err := db.QueryRow(`
			SELECT stored_filename
			FROM image_variants
			WHERE image_id = $1 AND name = 'thumbnail'`,
			img.ID,
		).Scan(&filename)
		if err != nil {
			t.Fatal(err)
		}

		// Delete only this disposable test's generated thumbnail.
		path := filepath.Join("uploads", img.ID, "variants", filename)
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}

		response := download(img.ID, "thumbnail")
		if response.Code != http.StatusNotFound {
			t.Fatalf("expected 404; got %d", response.Code)
		}
	})
}
