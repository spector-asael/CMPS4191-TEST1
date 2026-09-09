package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lewisdalwin/gatekeeper/internal/data"
)

func (app *application) startImageWorker(ctx context.Context) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		ticker := time.NewTicker(app.config.workerPollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				app.logger.Info("image worker stopped")
				return
			case <-ticker.C:
				err := app.processNextImageJob(ctx)
				if err != nil && !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, context.Canceled) {
					app.logger.Error("image worker failed", "error", err)
				}
			}
		}
	}()
}

func (app *application) processNextImageJob(ctx context.Context) error {
	// 1. Claim next queued image processing job (WRK-02)
	job, err := app.models.Jobs.ClaimNext(ctx, "image_processing")
	if err != nil {
		return err
	}
	app.logger.Info("image processing job started", "job_id", job.PublicID,
		"artificial_delay", app.config.jobDelay)

	if job.ImageID == nil {
		return app.models.Jobs.MarkFailed(ctx, job.ID, "job missing associated image_id")
	}

	if app.config.jobDelay > 0 {
		timer := time.NewTimer(app.config.jobDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	// 2. Fetch original image record from PostgreSQL
	img, err := app.models.Images.Get(*job.ImageID)
	if err != nil {
		return app.models.Jobs.MarkFailed(ctx, job.ID, fmt.Sprintf("failed to load image: %v", err))
	}

	// 3. Construct generated variant metadata (Section 10.2 spec)
	variants := []data.Variant{
		{
			Name:   "thumbnail",
			Width:  150,
			Height: 150,
			URL:    fmt.Sprintf("/v1/images/%d/variants/thumbnail", img.ID),
		},
		{
			Name:   "preview",
			Width:  800,
			Height: 600,
			URL:    fmt.Sprintf("/v1/images/%d/variants/preview", img.ID),
		},
		{
			Name:   "display",
			Width:  1200,
			Height: 900,
			URL:    fmt.Sprintf("/v1/images/%d/variants/display", img.ID),
		},
	}

	resultPayload := struct {
		Variants []data.Variant `json:"variants"`
	}{
		Variants: variants,
	}

	resultBytes, err := json.Marshal(resultPayload)
	if err != nil {
		return app.models.Jobs.MarkFailed(ctx, job.ID, err.Error())
	}

	// 4. Update job to completed and attach JSONB variants
	if err := app.models.Jobs.MarkCompleted(ctx, job.ID, resultBytes); err != nil {
		return err
	}

	app.logger.Info("image processing job completed", "job_id", job.PublicID)
	return nil
}
