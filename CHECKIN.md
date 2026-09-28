# ImageLab — Week 4 check-in

## Completed since the previous check-in

- Strengthened upload validation and durable acceptance.
- Added job timestamps, variant metadata, and database integrity checks.
- Tested successful processing, rollback behavior, and variant downloads.
- Improved the frontend layout, image preview, and processing activity.
- Added status-request timeouts and response validation.
- Demonstrated retrieval failure, same-job recovery, and processing failure.
- Demonstrated submission protection and observation reset.
- Added measurement calculations, tests, and a visible measurements panel.
- Recorded one single-image run and a five-job burst.

## Still incomplete

Final repository review and commit/push of the documentation checkpoint.

## Evidence prepared

- Browser screenshots in docs/evidence/.
- Saved database and filesystem output.
- Automated Go and JavaScript tests.
- Six measurement rows in docs/measurements.csv.
- Experiment instructions and interpretation.

## Current blocker

No known blocker. Final documentation and verification remain.

## Next action

Review the final changes and commit and push the documentation on week-4.

## Concept explanation

The upload request validates and stores the original and creates a saved
job before returning 202 Accepted. One worker performs the image processing.
The browser checks the saved job approximately every second.

A failed status request means the browser cannot observe current progress;
it does not mean processing failed. Try again observes the existing job.
A job response with status failed confirms a processing failure.

The five-job experiment showed that quick acceptance does not increase
worker capacity: later jobs still waited for the single worker.

## Question for the check-in

How would changing the polling interval affect request count and the delay
before the browser notices completion?