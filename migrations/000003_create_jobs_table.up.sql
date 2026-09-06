BEGIN;

CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id UUID NOT NULL DEFAULT gen_random_uuid(),
    image_id BIGINT REFERENCES images(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL DEFAULT 'image_processing',
    status TEXT NOT NULL DEFAULT 'queued',
    payload JSONB DEFAULT '{}',
    result JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT jobs_status_check CHECK (status IN ('queued', 'processing', 'completed', 'failed'))
);

COMMIT;