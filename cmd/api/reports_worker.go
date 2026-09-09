// Filename: cmd/api/worker.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Called in main.go to initialize the report worker.
// The worker is a background goroutine that polls the database for new jobs to process.
// The worker will continue to run until the application is shut down.
func (app *application) startReportWorker(ctx context.Context) {
	// We use a WaitGroup to keep track of the worker goroutine so that
	// the application can wait for it to finish during shutdown.
	app.wg.Add(1)
	go func() {
		// When the worker goroutine exits, app.wg.Done() decrements the
		// counter to indicate that the goroutine has finished.
		// We use defer to ensure that this is called when the goroutine exits,
		// even if the goroutine exits due to an error or panic.
		defer app.wg.Done()
		ticker := time.NewTicker(app.config.workerPollInterval)
		// Stop the ticker when the worker exits because it is no longer needed.
		// Using defer ensures the ticker is stopped regardless of how the worker exits.
		defer ticker.Stop()
		for {
			// The select waits for one of two signals:
			// ctx.Done() means the application is shutting down, so we stop the worker.
			// ticker.C means the polling interval has elapsed, so we check for a new job.
			select {
			case <-ctx.Done():
				app.logger.Info("report worker stopped")
				return
			case <-ticker.C:
				err := app.processNextReportJob(ctx)
				if err != nil && !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, context.Canceled) {
					app.logger.Error("report worker failed", "error", err)
				}
				// Logs provide runtime information when the background worker encounters
				// an unexpected error while processing a job.

			}
		}
	}()
}

// We call processNextReport to continue the job lifecycle by processing the next job in the queue
func (app *application) processNextReportJob(ctx context.Context) error {
	job, err := app.models.ReportsJobs.ClaimNextReportJob(ctx)
	if err != nil {
		return err
	}
	// To simulate a long-running job, we add an artificial delay before generating the report.
	app.logger.Info("report job started", "job_id", job.PublicID,
		"artificial_delay", app.config.jobDelay)

	// The exact same simulated work now belongs to the worker, not the POST.
	if app.config.jobDelay > 0 {
		timer := time.NewTimer(app.config.jobDelay)
		defer timer.Stop()
		select {
		// ctx.Done() sends a signal when the context is canceled, allowing us to return early
		// and avoid generating the report if the application is shutting down.
		// Using this case means we do not have to wait for the timer to expire in order to
		// return early and avoid generating the report if the application is shutting down.
		case <-ctx.Done():
			return ctx.Err()
		// timer.C sends a signal when the timer expires, allowing us to proceed with generating
		// the report.
		case <-timer.C:
		}
	}

	report, err := app.models.Reports.GenerateReport(job.ConsumerID, job.Payload.From, job.Payload.To)
	if err != nil {
		return app.models.ReportsJobs.MarkFailedReportJob(ctx, job.ID, err.Error())
	}
	// We Marshal the report struct into JSON so we can store it in the database with MarkCompleted.
	// This allows us to return the report to the client when they check the job status later.
	result, err := json.Marshal(report)
	// MarkCompleted sets status to completed, stores the JSON result in result, and sets
	// completed_at to the current time. MarkFailed sets status to failed, stores the
	// error message in error_message, and also sets completed_at to the current time.
	if err != nil {
		return app.models.ReportsJobs.MarkFailedReportJob(ctx, job.ID, err.Error())
	}

	if err := app.models.ReportsJobs.MarkCompletedReportJob(ctx, job.ID, result); err != nil {
		return err
	}
	app.logger.Info("report job completed", "job_id", job.PublicID)
	return nil
}
