BEGIN;

ALTER TABLE jobs
ADD COLUMN failed_at TIMESTAMPTZ;

-- Move failure times recorded by the previous implementation.
UPDATE jobs
SET failed_at = completed_at,
    completed_at = NULL
WHERE status = 'failed';

COMMIT;