package data

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// A connection error during commit can leave its outcome uncertain.
// Preserve the original file in that case instead of risking deleting
// input belonging to a job that PostgreSQL actually committed.
var ErrUploadCommitUncertain = errors.New("upload commit outcome is uncertain")

func (m ImageModel) AcceptUpload(
	parent context.Context,
	img *Image,
	saveOriginal func() error,
) (*Job, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Prepare the image record, but do not commit it yet.
	err = tx.QueryRowContext(ctx, `
		INSERT INTO images (
			original_filename, stored_filename, media_type, size
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		img.OriginalFilename,
		img.StoredFilename,
		img.MediaType,
		img.Size,
	).Scan(&img.ID, &img.CreatedAt)
	if err != nil {
		return nil, err
	}

	// The database-generated image ID is now available for the folder name.
	if err := saveOriginal(); err != nil {
		return nil, err
	}

	job := &Job{
		ImageID: &img.ID,
		JobType: "image_processing",
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO jobs (image_id, job_type, status)
		VALUES ($1, $2, 'queued')
		RETURNING id, public_id, status, created_at`,
		job.ImageID,
		job.JobType,
	).Scan(&job.ID, &job.PublicID, &job.Status, &job.QueuedAt)
	if err != nil {
		return nil, err
	}

	// Only now can another database connection see the queued job.
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUploadCommitUncertain, err)
	}

	return job, nil
}
