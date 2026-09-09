package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/healthcheck", app.healthcheckHandler)

	mux.HandleFunc("POST /v1/consumers", app.createConsumersHandler)
	mux.HandleFunc("POST /v1/reports", app.createReportHandler)
	mux.HandleFunc("GET /v1/report-jobs/{id}", app.getReportsJobHandler)

	mux.HandleFunc("GET /v1/image-jobs/{id}", app.getImageJobHandler)
	mux.HandleFunc("POST /v1/images", app.uploadImageHandler)
	mux.HandleFunc("GET /v1/images/{id}/variants/{name}", app.getVariantHandler)

	loggingMiddleware := app.loggingMiddleware(mux)
	enableCorse := app.enableCORS(loggingMiddleware)
	recoverPanic := app.recoverPanic(enableCorse)
	return recoverPanic
}
