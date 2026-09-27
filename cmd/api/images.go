package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/lewisdalwin/gatekeeper/internal/data"
)

func (app *application) uploadImageHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Limit the image to 10 MiB, with extra room for form packaging.
	const maxImageSize = 10 << 20
	const maxRequestSize = maxImageSize + (1 << 20)

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		var sizeError *http.MaxBytesError

		if errors.As(err, &sizeError) {
			app.badRequestResponse(w, r,
				fmt.Errorf("upload request is too large"))
		} else {
			app.badRequestResponse(w, r,
				fmt.Errorf("invalid multipart upload"))
		}
		return
	}
	defer r.MultipartForm.RemoveAll()

	// 2. Require exactly one uploaded file, in the "image" field.
	files := r.MultipartForm.File["image"]
	fileCount := 0
	for _, uploadedFiles := range r.MultipartForm.File {
		fileCount += len(uploadedFiles)
	}

	if len(files) != 1 || fileCount != 1 {
		app.badRequestResponse(w, r,
			fmt.Errorf("provide exactly one file in the 'image' field"))
		return
	}

	header := files[0]

	if header.Size == 0 {
		app.badRequestResponse(w, r,
			fmt.Errorf("image file must not be empty"))
		return
	}

	if header.Size > maxImageSize {
		app.badRequestResponse(w, r,
			fmt.Errorf("file size exceeds 10 MB limit"))
		return
	}

	file, err := header.Open()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	defer file.Close()

	// 3. Decode the contents before accepting the image.
	_, format, err := image.Decode(file)
	if err != nil {
		app.badRequestResponse(w, r,
			fmt.Errorf("upload must be a readable JPEG or PNG image"))
		return
	}

	var mimeType string
	switch format {
	case "jpeg":
		mimeType = "image/jpeg"
	case "png":
		mimeType = "image/png"
	default:
		app.badRequestResponse(w, r,
			fmt.Errorf("only JPEG and PNG images are allowed"))
		return
	}

	// Return to the beginning so the original can be copied in full.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// 4. Persist image record in database to obtain generated image.ID
	extension := ".jpg"
	if mimeType == "image/png" {
		extension = ".png"
	}

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	storedFilename := fmt.Sprintf("%x%s", randomBytes, extension)
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

	// 5. Create dedicated directory on disk: ./uploads/{image_id}
	imgDir := filepath.Join("./uploads", image.ID)
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// 6. Save original file inside ./uploads/{image_id}/{storedFilename}
	dstPath := filepath.Join(imgDir, storedFilename)
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

	// 7. Queue processing job in database
	job := &data.Job{
		ImageID: &image.ID,
		JobType: "image_processing",
	}
	if err := app.models.Jobs.Insert(job); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// 8. Send 202 Accepted response with polling location header
	statusURL := fmt.Sprintf("/v1/jobs/%s", job.PublicID)
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

func (app *application) getJobHandler(w http.ResponseWriter, r *http.Request) {
	job, err := app.models.Jobs.GetByPublicID(r.PathValue("id"))
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
		"image_id":      job.ImageID,
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
	imageIDStr := r.PathValue("id")
	variantName := r.PathValue("name")

	switch variantName {
	case "thumbnail", "preview", "display":
	default:
		app.notFoundResponse(w, r)
		return
	}

	image, err := app.models.Images.Get(&imageIDStr)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	ext := ".jpg"
	if image.MediaType == "image/png" {
		ext = ".png"
	}

	// Match the new file naming pattern: {image_id}-{variant_name}.{ext}
	variantFileName := fmt.Sprintf("%s-%s%s", image.ID, variantName, ext)
	variantPath := filepath.Join("./uploads", image.ID, "variants", variantFileName)

	if _, err := os.Stat(variantPath); os.IsNotExist(err) {
		app.notFoundResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", image.MediaType)
	http.ServeFile(w, r, variantPath)
}
