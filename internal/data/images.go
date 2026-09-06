package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Image represents the metadata for an uploaded raw image file (Section 11)[cite: 1].
type Image struct {
	ID               int64     `json:"id"`
	OriginalFilename string    `json:"original_filename"`
	StoredFilename   string    `json:"stored_filename"`
	MediaType        string    `json:"media_type"`
	Size             int64     `json:"size"`
	CreatedAt        time.Time `json:"created_at"`
}

type ImageModel struct {
	DB *sql.DB
}

// Insert creates a new record in the images table and scans back generated database fields[cite: 1].
func (m ImageModel) Insert(image *Image) error {
	query := `
		INSERT INTO images (original_filename, stored_filename, media_type, size)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	args := []any{
		image.OriginalFilename,
		image.StoredFilename,
		image.MediaType,
		image.Size,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&image.ID, &image.CreatedAt)
}

// Get retrieves an image record by its primary key ID[cite: 1].
func (m ImageModel) Get(id int64) (*Image, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, original_filename, stored_filename, media_type, size, created_at
		FROM images
		WHERE id = $1`

	var image Image

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&image.ID,
		&image.OriginalFilename,
		&image.StoredFilename,
		&image.MediaType,
		&image.Size,
		&image.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &image, nil
}
