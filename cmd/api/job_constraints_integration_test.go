package main

import (
	"errors"
	"os"
	"testing"

	"github.com/lewisdalwin/gatekeeper/internal/data"
	"github.com/lib/pq"
)

func TestJobConstraintsIntegration(t *testing.T) {
	if os.Getenv("IMAGELAB_INTEGRATION_TESTS") != "1" {
		t.Skip("set IMAGELAB_INTEGRATION_TESTS=1 to run PostgreSQL tests")
	}

	db := startUploadTestDatabase(t)
	models := data.NewModels(db)

	// Create one valid image and job as our starting point.
	img := &data.Image{
		OriginalFilename: "example.png",
		StoredFilename:   "constraint-test.png",
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

	t.Run("duplicate public ID rejected", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO jobs (public_id, image_id, job_type)
			VALUES ($1, $2, 'image_processing')`,
			job.PublicID,
			img.ID,
		)

		var pgErr *pq.Error
		if !errors.As(err, &pgErr) {
			t.Fatalf("expected PostgreSQL constraint error; got %v", err)
		}
		if pgErr.Code != "23505" ||
			pgErr.Constraint != "jobs_public_id_unique" {
			t.Fatalf("unexpected database error: %v", pgErr)
		}
	})

	t.Run("missing image rejected", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO jobs (job_type)
			VALUES ('image_processing')
		`)

		var pgErr *pq.Error
		if !errors.As(err, &pgErr) {
			t.Fatalf("expected PostgreSQL constraint error; got %v", err)
		}
		if pgErr.Code != "23514" ||
			pgErr.Constraint != "image_processing_job_requires_image" {
			t.Fatalf("unexpected database error: %v", pgErr)
		}
	})

	// Neither rejected insert should have added another job.
	var count int
	if err := db.QueryRow("SELECT count(*) FROM jobs").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected only the original valid job; got %d", count)
	}
}
