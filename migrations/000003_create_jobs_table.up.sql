BEGIN;

CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    public_id UUID NOT NULL DEFAULT uuidv4(),
    image_id UUID REFERENCES images(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL DEFAULT 'image_processing',
    status job_status NOT NULL DEFAULT 'queued',
    payload JSONB DEFAULT '{}',
    result JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;