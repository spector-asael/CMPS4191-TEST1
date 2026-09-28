BEGIN;

ALTER TABLE jobs
DROP CONSTRAINT image_processing_job_requires_image;

ALTER TABLE jobs
DROP CONSTRAINT jobs_public_id_unique;

COMMIT;