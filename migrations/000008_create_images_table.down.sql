-- 1. Remove state constraint and image_id foreign key from jobs table
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_status_check;
ALTER TABLE jobs DROP COLUMN IF EXISTS image_id;

-- 2. Drop images table
DROP TABLE IF EXISTS images CASCADE;