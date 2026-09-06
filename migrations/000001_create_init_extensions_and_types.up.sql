-- Filename: 000001_create_init_extensions_and_types.up.sql

BEGIN;

CREATE TYPE consumer_status AS ENUM ('active', 'suspended', 'terminated');
CREATE TYPE key_status     AS ENUM ('active', 'rotating', 'revoked');
CREATE TYPE job_status     AS ENUM ('queued', 'processing', 'completed', 'failed', 'cancelled');

COMMIT;
