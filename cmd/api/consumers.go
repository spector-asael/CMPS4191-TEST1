package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lewisdalwin/gatekeeper/internal/data"
	"github.com/lewisdalwin/gatekeeper/internal/validator"
)

func (app *application) createConsumersHandler(w http.ResponseWriter, r *http.Request) {
	// sandbox for JSON parsing and validation
	var input struct {
		Name string `json:"name"`
		Email string `json:"email"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	consumer := &data.Consumer{
		Name: input.Name,
		Email: input.Email,
	}

	v := validator.New()
	// validate the consumer struct and return a response if any checks fail
	if data.ValidateConsumer(v, consumer); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// insert the consumer into the database and handle any errors
	err = app.models.Consumers.Insert(consumer)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			app.badRequestResponse(w, r, err)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/consumers/%s", consumer.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"consumer": consumer}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}