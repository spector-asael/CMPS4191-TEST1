// Filename: cmd/api/reports.go
package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lewisdalwin/gatekeeper/internal/data"
	"github.com/lewisdalwin/gatekeeper/internal/validator"
)

// When a client asks to generate a report using the /v1/reports endpoint,
// we will create a new job in the database, and return a 202 Accepted response to the client.
// The 202 accepted response promises that the report will being generated in the background,
// and the client can check back later for the result.
// The initial response will contain a publicID that the client can use to locate the job later.
// Here we handle the acceptance of work.
func (app *application) createReportHandler(w http.ResponseWriter, r *http.Request) {
	// First, we define a struct to hold the EXPECTED input from the request body.
	var input struct {
		ConsumerID string    `json:"consumer_id"`
		From       time.Time `json:"from"`
		To         time.Time `json:"to"`
	}
	// Next, we call our readJSON helper method to read the request body.
	// If the request body is valid JSON and matches the expected structure, it will populate the input struct with the data.
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err) // Otherwise, we return a bad request response to the client.
		return
	}

	// Now that we have verified that we've received valid JSON, we can perform some additional validation on the input data.
	// We create a new validator instance and use it to check that the required fields are present and valid.
	v := validator.New()
	v.Check(input.ConsumerID != "", "consumer_id", "must be provided")
	v.Check(!input.From.IsZero(), "from", "must be provided")
	v.Check(!input.To.IsZero(), "to", "must be provided")
	v.Check(input.From.Before(input.To), "from", "must be earlier than to")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Now that we have our data, we will be creating a new job in the database.
	// The purpose of this job is to generate a report for the specified consumer within the given date range.
	// We do not generate the report immediately,
	// as this could take a long time and we want to avoid blocking the request.
	// If the requirement is to generate the report immediately, we would need to implement a different approach,
	// However, for this implementation, the requirement is to be able to let the client
	// know that the report is being generated and that they can check back later for the result.
	job := &data.ReportJob{
		ConsumerID: input.ConsumerID,
		JobType:    "consumer_activity_report",
		Payload:    data.ReportPayload{From: input.From, To: input.To},
	}
	if err := app.models.ReportsJobs.InsertReportJob(job); err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}
	// If the job was successfully inserted into the database, with no errors
	// We can proceed with the response to the client.
	// We create a status URL for the job using its public ID.
	// This URL can be used by the client to check the status of the job later.
	statusURL := fmt.Sprintf("/v1/report-jobs/%s", job.PublicID)
	// We create a new HTTP header and set the Location header to the status URL.
	headers := make(http.Header)
	headers.Set("Location", statusURL)
	// We then create a response envelope containing the job ID, status, and status URL.
	// Finally, we call our writeJSON helper method to send the response back to the client with a 202 Accepted status code.
	response := envelope{"job_id": job.PublicID, "status": job.Status, "status_url": statusURL}
	if err := app.writeJSON(w, http.StatusAccepted, response, headers); err != nil {
		app.serverErrorResponse(w, r, err) // If there's an error writing the JSON response, we return a server error response to the client.
	}
}

func (app *application) getReportsJobHandler(w http.ResponseWriter, r *http.Request) {
	job, err := app.models.ReportsJobs.GetReportJobByPublicID(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}
	if err := app.writeJSON(w, http.StatusOK, envelope{"job": job}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
