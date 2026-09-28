package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/lewisdalwin/gatekeeper/internal/data"
)

func TestJobTimestampsIntegration(t *testing.T) {
	if os.Getenv("IMAGELAB_INTEGRATION_TESTS") != "1" {
		t.Skip("set IMAGELAB_INTEGRATION_TESTS=1 to run PostgreSQL tests")
	}

	// Reuse the existing disposable-database helper.
	db := startUploadTestDatabase(t)
	models := data.NewModels(db)

	app := &application{
		models: models,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	for _, outcome := range []string{"completed", "failed"} {
		t.Run(outcome, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(
				context.Background(), 10*time.Second,
			)
			defer cancel()

			// Prepare database records for this timestamp test.
			img := &data.Image{
				OriginalFilename: "example.png",
				StoredFilename:   outcome + ".png",
				MediaType:        "image/png",
				Size:             100,
			}
			if err := models.Images.Insert(img); err != nil {
				t.Fatal(err)
			}

			job := &data.Job{
				ImageID: &img.ID,
				JobType: "image_processing",
			}
			if err := models.Jobs.Insert(job); err != nil {
				t.Fatal(err)
			}

			// Move queued → processing using the worker's claim method.
			claimed, err := models.Jobs.ClaimNext(ctx, "image_processing")
			if err != nil {
				t.Fatal(err)
			}
			if claimed.ID != job.ID {
				t.Fatal("claimed a different job")
			}

			// Exercise each final timestamp-writing method.
			if outcome == "failed" {
				err = models.Jobs.MarkFailed(
					ctx, job.ID, "Test processing failure",
				)
			} else {
				// These example records test timestamps, not image generation.
				variants := []data.Variant{
					{
						ImageID:        img.ID,
						Name:           "thumbnail",
						StoredFilename: "thumbnail.png",
						Width:          150,
						Height:         150,
						Size:           100,
						URL:            "/v1/images/" + img.ID + "/variants/thumbnail",
					},
					{
						ImageID:        img.ID,
						Name:           "preview",
						StoredFilename: "preview.png",
						Width:          800,
						Height:         600,
						Size:           200,
						URL:            "/v1/images/" + img.ID + "/variants/preview",
					},
					{
						ImageID:        img.ID,
						Name:           "display",
						StoredFilename: "display.png",
						Width:          1200,
						Height:         900,
						Size:           300,
						URL:            "/v1/images/" + img.ID + "/variants/display",
					},
				}

				err = models.Jobs.CompleteWithVariants(ctx, job.ID, variants)
			}
			if err != nil {
				t.Fatal(err)
			}

			// Ask the real status handler to retrieve the saved job.
			request := httptest.NewRequest(
				http.MethodGet, "/v1/jobs/"+job.PublicID, nil,
			)
			request.SetPathValue("id", job.PublicID)
			response := httptest.NewRecorder()

			app.getJobHandler(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf(
					"expected 200; got %d: %s",
					response.Code, response.Body.String(),
				)
			}

			var result struct {
				ID           string     `json:"id"`
				Status       string     `json:"status"`
				QueuedAt     time.Time  `json:"queued_at"`
				StartedAt    *time.Time `json:"started_at"`
				CompletedAt  *time.Time `json:"completed_at"`
				FailedAt     *time.Time `json:"failed_at"`
				ErrorMessage *string    `json:"error_message"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}

			if result.ID != job.PublicID || result.Status != outcome {
				t.Fatalf("unexpected job or status: %+v", result)
			}
			if result.QueuedAt.IsZero() || result.StartedAt == nil {
				t.Fatal("missing queue or start time")
			}
			if result.StartedAt.Before(result.QueuedAt) {
				t.Fatal("start time precedes queue time")
			}

			if outcome == "failed" {
				if result.FailedAt == nil || result.CompletedAt != nil {
					t.Fatal("failed job needs failed_at, not completed_at")
				}
				if result.FailedAt.Before(*result.StartedAt) {
					t.Fatal("failure time precedes start time")
				}
				if result.ErrorMessage == nil ||
					*result.ErrorMessage != "Test processing failure" {
					t.Fatal("failure message was not preserved")
				}
			} else {
				if result.CompletedAt == nil || result.FailedAt != nil {
					t.Fatal("completed job needs completed_at, not failed_at")
				}
				if result.CompletedAt.Before(*result.StartedAt) {
					t.Fatal("completion time precedes start time")
				}
				if result.ErrorMessage != nil {
					t.Fatal("successful job unexpectedly has an error")
				}
			}
		})
	}
}
