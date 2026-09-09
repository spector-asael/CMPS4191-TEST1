BEGIN;

CREATE TABLE IF NOT EXISTS images (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    original_filename TEXT NOT NULL,
    stored_filename TEXT NOT NULL UNIQUE,
    media_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;