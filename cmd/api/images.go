package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/lewisdalwin/gatekeeper/internal/data"
)

func (app *application) uploadImageHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Enforce 10 MB limit on request body (VAL-01)
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("file size exceeds 10 MB limit"))
		return
	}

	// 2. Extract image file from form field
	file, header, err := r.FormFile("image")
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing 'image' form field"))
		return
	}
	defer file.Close()

	// 3. Validate MIME type by reading header bytes
	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Reset file reader pointer after sniffing content type
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	mimeType := http.DetectContentType(buf)
	switch mimeType {
	case "image/jpeg", "image/png":
	default:
		app.badRequestResponse(w, r, fmt.Errorf("unsupported file type: %s (only JPEG and PNG are allowed)", mimeType))
		return
	}

	// 4. Save file to disk in local uploads directory
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	storedFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(header.Filename))
	dstPath := filepath.Join(uploadDir, storedFilename)

	dst, err := os.Create(dstPath)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// 5. Persist image record in database
	image := &data.Image{
		OriginalFilename: header.Filename,
		StoredFilename:   storedFilename,
		MediaType:        mimeType,
		Size:             header.Size,
	}
	if err := app.models.Images.Insert(image); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// 6. Queue processing job in database
	job := &data.ImageJob{
		ImageID: &image.ID,
		JobType: "image_processing",
	}
	if err := app.models.ImageJobs.InsertImageJob(job); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// 7. Send 202 Accepted response with polling location
	statusURL := fmt.Sprintf("/v1/image-jobs/%s", job.PublicID)
	headers := make(http.Header)
	headers.Set("Location", statusURL)

	response := envelope{
		"image_id":   image.ID,
		"job_id":     job.PublicID,
		"status":     job.Status,
		"status_url": statusURL,
	}

	if err := app.writeJSON(w, http.StatusAccepted, response, headers); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getImageJobHandler(w http.ResponseWriter, r *http.Request) {
	job, err := app.models.ImageJobs.GetImageJobByPublicID(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	response := envelope{
		"id":            job.PublicID,
		"status":        job.Status,
		"variants":      job.Variants,
		"queued_at":     job.QueuedAt,
		"started_at":    job.StartedAt,
		"completed_at":  job.CompletedAt,
		"error_message": job.ErrorMessage,
	}

	if err := app.writeJSON(w, http.StatusOK, response, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getVariantHandler(w http.ResponseWriter, r *http.Request) {
	// Extract path parameters (Go 1.22+ routing)
	imageIDStr := r.PathValue("id")
	variantName := r.PathValue("name")

	// Validate variant name per spec constraints (IMG-01)
	switch variantName {
	case "thumbnail", "preview", "display":
	default:
		app.notFoundResponse(w, r)
		return
	}

	imageID, err := strconv.ParseInt(imageIDStr, 10, 64)
	if err != nil || imageID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch image record to find the stored filename
	image, err := app.models.Images.Get(imageID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	// Serve the stored original file from ./uploads/ with 200 OK
	filePath := filepath.Join("./uploads", image.StoredFilename)
	w.Header().Set("Content-Type", image.MediaType)
	http.ServeFile(w, r, filePath)
}
