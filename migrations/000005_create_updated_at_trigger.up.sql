-- Filename: 000005_create_updated_at_trigger.up.sql

BEGIN;

CREATE OR REPLACE FUNCTION set_updated_at() 
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER consumers_updated_at
    BEFORE UPDATE ON consumers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
