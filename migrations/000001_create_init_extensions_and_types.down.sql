-- Filename: 000001_create_init_extensions_and_types.down.sql

BEGIN;

DROP TYPE IF EXISTS report_job_status;
DROP TYPE IF EXISTS key_status;
DROP TYPE IF EXISTS consumer_status;
DROP EXTENSION IF EXISTS citext;


COMMIT;
