BEGIN;

-- Restore the previous representation before removing the column.
UPDATE jobs
SET completed_at = failed_at
WHERE status = 'failed';

ALTER TABLE jobs
DROP COLUMN failed_at;

COMMIT;