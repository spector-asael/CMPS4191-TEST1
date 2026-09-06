-- Filename: 000004_create_jobs_table.up.sql

BEGIN;

CREATE TABLE IF NOT EXISTS jobs (
    id             uuid        PRIMARY KEY DEFAULT uuidv7(),
    consumer_id    uuid        NOT NULL REFERENCES consumers(id),
    job_type       text        NOT NULL,
    status         job_status  NOT NULL DEFAULT 'queued',
    payload        jsonb       NOT NULL DEFAULT '{}',
    result         jsonb,
    error_message  text,
    started_at     timestamptz,
    completed_at   timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);

COMMIT;