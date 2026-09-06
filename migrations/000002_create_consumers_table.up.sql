-- Filename: 000002_create_consumers_table.up.sql

BEGIN;

CREATE TABLE IF NOT EXISTS consumers (
    id          uuid             PRIMARY KEY DEFAULT uuidv7(),
    name        text             NOT NULL,
    email       citext           NOT NULL UNIQUE,
    status      consumer_status  NOT NULL DEFAULT 'active',
    version     integer          NOT NULL DEFAULT 1,
    created_at  timestamptz      NOT NULL DEFAULT now(),
    updated_at  timestamptz      NOT NULL DEFAULT now()
);

COMMIT;