-- 1. Store original upload metadata only (Section 11)
CREATE TABLE IF NOT EXISTS images (
    id BIGSERIAL PRIMARY KEY,
    original_filename TEXT NOT NULL,
    stored_filename TEXT NOT NULL UNIQUE,
    media_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 2. Store execution progress and variant metadata inside result JSONB
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS image_id BIGINT REFERENCES images(id) ON DELETE CASCADE;

-- Enforce the four required job states (JOB-02, DATA-04)
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check 
    CHECK (status IN ('queued', 'processing', 'completed', 'failed'));