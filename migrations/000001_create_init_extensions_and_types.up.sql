-- Filename: 000001_create_init_extensions_and_types.up.sql

BEGIN;

-- job_status tracks the lifecycle of an asynchronous image processing job.
-- JOB-02 & DATA-04: Use exactly queued, processing, completed, and failed for Version 1.
CREATE TYPE job_status AS ENUM ('queued', 'processing', 'completed', 'failed');

COMMIT;
