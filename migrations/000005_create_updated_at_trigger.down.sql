-- Filename: 000005_create_updated_at_trigger.down.sql

BEGIN;

DROP TRIGGER IF EXISTS consumers_updated_at ON consumers;
DROP FUNCTION IF EXISTS set_updated_at();

COMMIT;