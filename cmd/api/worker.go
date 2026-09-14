package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/lewisdalwin/gatekeeper/internal/data"
	"golang.org/x/image/draw"
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

// processNextImageJob orchestrates claiming, processing, and completing an image job.
func (app *application) processNextImageJob(ctx context.Context) error {
	job, err := app.models.Jobs.ClaimNext(ctx, "image_processing")
	if err != nil {
		return err
	}
	app.logger.Info("image processing job started", "job_id", job.PublicID)

	if job.ImageID == nil {
		return app.models.Jobs.MarkFailed(ctx, job.ID, "job missing associated image_id")
	}

	img, err := app.models.Images.Get(job.ImageID)
	if err != nil {
		return app.models.Jobs.MarkFailed(ctx, job.ID, fmt.Sprintf("failed to load image record: %v", err))
	}

	// 1. Decode original file from disk
	srcImg, format, err := loadAndDecodeImage(img)
	if err != nil {
		return app.models.Jobs.MarkFailed(ctx, job.ID, err.Error())
	}

	// 2. Generate variant images and construct metadata
	variants, err := generateVariants(img, srcImg, format)
	if err != nil {
		return app.models.Jobs.MarkFailed(ctx, job.ID, err.Error())
	}

	// 3. Mark job complete in PostgreSQL
	if err := app.completeJob(ctx, job, variants); err != nil {
		return err
	}

	app.logger.Info("image processing job completed", "job_id", job.PublicID)
	return nil
}

// loadAndDecodeImage opens the original file from the upload directory and decodes its header.
func loadAndDecodeImage(img *data.Image) (image.Image, string, error) {
	origPath := filepath.Join("./uploads", img.ID, img.StoredFilename)

	srcFile, err := os.Open(origPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open original file: %w", err)
	}
	defer srcFile.Close()

	srcImg, format, err := image.Decode(srcFile)
	if err != nil {
		return nil, "", fmt.Errorf("unable to decode image header: %w", err)
	}

	return srcImg, format, nil
}

// generateVariants creates thumbnail, preview, and display variants and saves them to disk.
func generateVariants(img *data.Image, srcImg image.Image, format string) ([]data.Variant, error) {
	variantsDir := filepath.Join("./uploads", img.ID, "variants")
	if err := os.MkdirAll(variantsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create variants directory: %w", err)
	}

	specs := []struct {
		Name   string
		Width  int
		Height int
	}{
		{"thumbnail", 150, 150},
		{"preview", 800, 600},
		{"display", 1200, 900},
	}

	ext := ".jpg"
	if format == "png" {
		ext = ".png"
	}

	var variants []data.Variant
	for _, spec := range specs {
		varFileName := fmt.Sprintf("%s-%s%s", img.ID, spec.Name, ext)
		varPath := filepath.Join(variantsDir, varFileName)

		resized := resizeImage(srcImg, spec.Width, spec.Height)

		outFile, err := os.Create(varPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create variant file %s: %w", spec.Name, err)
		}

		if format == "png" {
			err = png.Encode(outFile, resized)
		} else {
			err = jpeg.Encode(outFile, resized, &jpeg.Options{Quality: 85})
		}
		outFile.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to encode variant %s: %w", spec.Name, err)
		}

		variants = append(variants, data.Variant{
			Name:   spec.Name,
			Width:  spec.Width,
			Height: spec.Height,
			URL:    fmt.Sprintf("/v1/images/%s/variants/%s", img.ID, spec.Name),
		})
	}

	return variants, nil
}

// completeJob encodes variant results into JSON and marks the database record as completed.
func (app *application) completeJob(ctx context.Context, job *data.Job, variants []data.Variant) error {
	resultBytes, err := json.Marshal(struct {
		Variants []data.Variant `json:"variants"`
	}{Variants: variants})
	if err != nil {
		return app.models.Jobs.MarkFailed(ctx, job.ID, err.Error())
	}

	return app.models.Jobs.MarkCompleted(ctx, job.ID, resultBytes)
}

// Bilinear interpolation scaling helper
func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}
