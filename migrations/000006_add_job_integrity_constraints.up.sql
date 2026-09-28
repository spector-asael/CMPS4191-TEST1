BEGIN;

ALTER TABLE jobs
ADD CONSTRAINT jobs_public_id_unique
UNIQUE (public_id);

ALTER TABLE jobs
ADD CONSTRAINT image_processing_job_requires_image
CHECK (
    job_type <> 'image_processing'
    OR image_id IS NOT NULL
);

COMMIT;