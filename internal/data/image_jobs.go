package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"
)

// Variant matches the ImageLab API variant result structure (Section 10.2)
type Variant struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

type ImageJob struct {
	ID           string          `json:"-"`
	PublicID     string          `json:"id"`
	ImageID      *int64          `json:"image_id,omitempty"`
	JobType      string          `json:"job_type,omitempty"`
	Status       string          `json:"status"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	Variants     []Variant       `json:"variants,omitempty"` // Automatically populated on completion
	ErrorMessage *string         `json:"error_message,omitempty"`
	QueuedAt     time.Time       `json:"queued_at"` // Spec requires queued_at
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
}

type ImageJobModel struct {
	DB *sql.DB
}

// Insert creates a durable job record in PostgreSQL (JOB-01)
func (m ImageJobModel) InsertImageJob(image_job *ImageJob) error {
	query := `
		INSERT INTO image_jobs (image_id, job_type, payload, status)
		VALUES ($1, $2, COALESCE($3, '{}'::jsonb), 'queued')
		RETURNING id, public_id, status, created_at`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, image_job.ImageID, image_job.JobType, image_job.Payload).Scan(
		&image_job.ID, &image_job.PublicID, &image_job.Status, &image_job.QueuedAt,
	)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrRecordNotFound
		}
		return err
	}
	return nil
}

// ClaimNext claims the next queued job for a specific jobType using FOR UPDATE SKIP LOCKED (WRK-02)
func (m ImageJobModel) ClaimNextImageJob(ctx context.Context, jobType string) (*ImageJob, error) {
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, public_id, image_id, job_type, COALESCE(payload, '{}'::jsonb)
		FROM image_jobs
		WHERE status = 'queued' AND job_type = $1
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1`

	var job ImageJob
	if err := tx.QueryRowContext(ctx, query, jobType).Scan(
		&job.ID, &job.PublicID, &job.ImageID, &job.JobType, &job.Payload,
	); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE image_jobs SET status = 'processing', started_at = now() WHERE id = $1`, job.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	job.Status = "processing"
	return &job, nil
}

// GetByPublicID returns current status and unmarshals variant metadata if completed (Section 10)
func (m ImageJobModel) GetImageJobByPublicID(publicID string) (*ImageJob, error) {
	query := `
		SELECT id, public_id, image_id, job_type, status,
		       COALESCE(payload, 'null'::jsonb),
		       COALESCE(result, 'null'::jsonb),
		       error_message, started_at, completed_at, created_at
		FROM image_jobs WHERE public_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var job ImageJob
	err := m.DB.QueryRowContext(ctx, query, publicID).Scan(
		&job.ID, &job.PublicID, &job.ImageID, &job.JobType, &job.Status,
		&job.Payload, &job.Result, &job.ErrorMessage, &job.StartedAt, &job.CompletedAt, &job.QueuedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	// Extract variants array from result JSONB when job status is completed
	if len(job.Result) > 0 && string(job.Result) != "null" {
		var res struct {
			Variants []Variant `json:"variants"`
		}
		if err := json.Unmarshal(job.Result, &res); err == nil {
			job.Variants = res.Variants
		}
	}

	return &job, nil
}

func (m ImageJobModel) MarkImageJobCompleted(ctx context.Context, id string, result []byte) error {
	_, err := m.DB.ExecContext(ctx,
		`UPDATE image_jobs SET status = 'completed', result = $2, completed_at = now() WHERE id = $1`,
		id, result)
	return err
}

func (m ImageJobModel) MarkImageJobFailed(ctx context.Context, id, message string) error {
	_, err := m.DB.ExecContext(ctx,
		`UPDATE image_jobs SET status = 'failed', error_message = $2, completed_at = now() WHERE id = $1`,
		id, message)
	return err
}
