package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrVariantCommitUncertain = errors.New(
	"variant commit outcome is uncertain",
)

func (m JobModel) CompleteWithVariants(
	parent context.Context,
	jobID string,
	variants []Variant,
) error {
	// Require exactly one of each expected variant.
	if len(variants) != 3 {
		return errors.New("completion requires exactly three variants")
	}

	seen := make(map[string]bool)
	for _, variant := range variants {
		switch variant.Name {
		case "thumbnail", "preview", "display":
		default:
			return fmt.Errorf("unknown variant name %q", variant.Name)
		}

		if seen[variant.Name] {
			return fmt.Errorf("duplicate variant name %q", variant.Name)
		}
		seen[variant.Name] = true
	}

	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Lock the job while preparing its completion.
	var imageID string
	var status string

	err = tx.QueryRowContext(ctx, `
		SELECT image_id, status
		FROM jobs
		WHERE id = $1
		FOR UPDATE`,
		jobID,
	).Scan(&imageID, &status)
	if err != nil {
		return err
	}
	if status != "processing" {
		return errors.New("only a processing job can be completed")
	}

	// Work on a copy so a rollback does not leave generated IDs
	// in the caller's variant list.
	saved := append([]Variant(nil), variants...)

	for i := range saved {
		variant := &saved[i]

		if variant.ImageID != imageID {
			return errors.New("variant belongs to a different image")
		}

		err = tx.QueryRowContext(ctx, `
			INSERT INTO image_variants (
				image_id, name, stored_filename, width, height, size
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at`,
			variant.ImageID,
			variant.Name,
			variant.StoredFilename,
			variant.Width,
			variant.Height,
			variant.Size,
		).Scan(&variant.ID, &variant.CreatedAt)
		if err != nil {
			return err
		}
	}

	// Preserve the existing job-response format, including the
	// IDs and creation times returned by PostgreSQL.
	result, err := json.Marshal(struct {
		Variants []Variant `json:"variants"`
	}{
		Variants: saved,
	})
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE jobs
		SET status = 'completed',
		    result = $2,
		    completed_at = now(),
		    failed_at = NULL,
		    error_message = NULL
		WHERE id = $1`,
		jobID,
		result,
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", ErrVariantCommitUncertain, err)
	}

	return nil
}
