package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadRejectsInvalidFiles(t *testing.T) {
	tests := []struct {
		name      string
		content   []byte
		fileCount int
		wantError string
	}{
		{
			name:      "missing image",
			fileCount: 0,
			wantError: "provide exactly one file in the 'image' field",
		},
		{
			name:      "empty file",
			content:   []byte{},
			fileCount: 1,
			wantError: "image file must not be empty",
		},
		{
			name:      "text disguised as PNG",
			content:   []byte("This is text, not an image."),
			fileCount: 1,
			wantError: "upload must be a readable JPEG or PNG image",
		},
		{
			name:      "damaged PNG",
			content:   []byte("\x89PNG\r\n\x1a\nbroken image contents"),
			fileCount: 1,
			wantError: "upload must be a readable JPEG or PNG image",
		},
		{
			name:      "file exceeds size limit",
			content:   make([]byte, (10<<20)+1),
			fileCount: 1,
			wantError: "file size exceeds 10 MB limit",
		},
		{
			name:      "multiple files",
			content:   []byte("test"),
			fileCount: 2,
			wantError: "provide exactly one file in the 'image' field",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			for i := 0; i < tc.fileCount; i++ {
				part, err := writer.CreateFormFile("image", "test.png")
				if err != nil {
					t.Fatal(err)
				}

				if _, err := part.Write(tc.content); err != nil {
					t.Fatal(err)
				}
			}

			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(
				http.MethodPost, "/v1/images", &body,
			)
			request.Header.Set(
				"Content-Type", writer.FormDataContentType(),
			)
			response := httptest.NewRecorder()

			// Invalid uploads must stop before accessing the database.
			app := &application{}
			app.uploadImageHandler(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status 400; got %d: %s",
					response.Code, response.Body.String(),
				)
			}

			var result struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}

			if result.Error != tc.wantError {
				t.Errorf(
					"expected error %q; got %q",
					tc.wantError, result.Error,
				)
			}

			if response.Header().Get("Location") != "" {
				t.Error("rejected upload must not provide a job location")
			}
		})
	}
}
