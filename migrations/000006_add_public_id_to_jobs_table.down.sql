-- Filename: 000006_add_public_id_to_jobs_table.down.sql
BEGIN;

ALTER TABLE jobs
DROP COLUMN IF EXISTS public_id;

COMMIT;