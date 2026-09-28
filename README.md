# ImageLab: Asynchronous Image Processing API

ImageLab is an asynchronous image-processing web application built with Go, PostgreSQL, and a vanilla JavaScript frontend.

Work is accepted durably via `202 Accepted`, processed independently by an in-process background worker, and observed by the browser through automatic short polling.

---

## Core Architecture

1. **Browser (Client)**: Selects, previews locally, submits the image once (with double-click protection), and polls for job status.
2. **Go HTTP API**: Validates request and MIME types, stores original image files using server-controlled names, creates durable job records in PostgreSQL, and immediately acknowledges receipt with `202 Accepted`.
3. **PostgreSQL**: Authoritative store for images, job lifecycles (`queued` → `processing` → `completed` / `failed`), timestamps, and variant metadata.
4. **Background Worker**: Independently claims queued jobs using `FOR UPDATE SKIP LOCKED`, generates image variants, persists variant metadata, and updates job completion.
5. **Local Filesystem**: Stores original uploaded images under `./uploads/` and generated variants.

---

## Prerequisites & Setup

### Requirements

- **Go** (1.26 or compatible newer tooling)
- **PostgreSQL** (18)
- **golang-migrate CLI**
- **Node.js** (for `npx http-server` frontend hosting)

### 1. Environment Configuration

Copy the example environment file and set your PostgreSQL DSN:

```bash
cp .envrc.example .envrc
# Edit .envrc with your database credentials:
# export GATEKEEPER_DB_DSN='postgres://gatekeeper:YOUR_PASSWORD@localhost:5432/imagelab?sslmode=disable'
source .envrc
```

### 2. Database Setup & Migrations

Create once, using the PostgreSQL administrator account.
Assumes the gatekeeper database login already exists.

```
sudo -u postgres psql -c "CREATE DATABASE imagelab OWNER gatekeeper;"
```

Verify the connection settings loaded from .envrc.

```
psql "$GATEKEEPER_DB_DSN" -c "SELECT current_database(), current_user;"
```

Create the application tables. Enter y when prompted.

```
make db/migrations/up
```

The connection check should show database `imagelab` and user `gatekeeper`.

---

## Running the Application

### 1. Start the API Server

Start the Go backend server (runs on `http://localhost:4000` by default):

```bash
make run/api
# Or manually:
go run ./cmd/api -db-dsn="$GATEKEEPER_DB_DSN"
```

### 2. Start the Frontend

In a separate terminal, launch the static frontend dev server (runs on `http://localhost:5500`):

```bash
make run/frontend
```

Then open `http://localhost:5500` in your web browser.

---

## API Endpoints & Contract

### 1. Image Submission (`POST /v1/images`)

Receives multipart form data with a single `image` field.

- **Constraints**: Exactly one decodable JPEG or PNG image, up to 10,485,760 bytes (10 MiB, labelled 10 MB in the interface). The server enforces presence, size, format, and decodability.
- **Response**: `202 Accepted` with `Location` header and JSON status body:

```http
HTTP/1.1 202 Accepted
Location: /v1/jobs/01917f90-1234-7000-8000-000000000001
Content-Type: application/json
```

```json
{
  "image_id": "01a0e95a-3207-705b-a826-8448e74b75ea",
  "job_id": "01917f90-1234-7000-8000-000000000001",
  "status": "queued",
  "status_url": "/v1/jobs/01917f90-1234-7000-8000-000000000001"
}
```

### 2. Job Status Query (`GET /v1/jobs/{job_id}`)

Returns the current job state, timestamps, and completed variant metadata:

```json
{
  "id": "01917f90-1234-7000-8000-000000000001",
  "image_id": "01a0e95a-3207-705b-a826-8448e74b75ea",
  "status": "completed",
  "queued_at": "2026-09-08T13:00:00Z",
  "started_at": "2026-09-08T13:00:01Z",
  "completed_at": "2026-09-08T13:00:03Z",
  "variants": [
    {
      "name": "thumbnail",
      "width": 150,
      "height": 150,
      "url": "/v1/images/01a0e95a-3207-705b-a826-8448e74b75ea/variants/thumbnail"
    },
    {
      "name": "preview",
      "width": 800,
      "height": 600,
      "url": "/v1/images/01a0e95a-3207-705b-a826-8448e74b75ea/variants/preview"
    },
    {
      "name": "display",
      "width": 1200,
      "height": 900,
      "url": "/v1/images/01a0e95a-3207-705b-a826-8448e74b75ea/variants/display"
    }
  ]
}
```

---

## CLI Testing with curl

### Successful Upload

```bash
curl -i -X POST http://localhost:4000/v1/images \
  -F "image=@/path/to/valid_image.png"
```

### Unsupported File Rejection (e.g., text, PDF, GIF)

```bash
curl -i -X POST http://localhost:4000/v1/images \
  -F "image=@README.md"
# Expect: 400 Bad Request ("unsupported file type")
```

### Oversized File Rejection (> 10 MB)

```bash
# Expect: 400 Bad Request ("file size exceeds 10 MB limit")
```

### Check Job Status

```bash
curl -i http://localhost:4000/v1/jobs/<JOB_PUBLIC_ID>
```

## Tests

Run commands from the project root.

### Go tests and checks

```bash
go test ./...
go vet ./...
```

### PostgreSQL integration tests

These tests require PostgreSQL 18 tools `initdb` and `pg_ctl` on PATH.
Run as a normal user, not root.

```bash
IMAGELAB_INTEGRATION_TESTS=1 go test ./... -count=1
```

The tests create disposable PostgreSQL instances and temporary upload
folders. They do not use the application's existing database or `.envrc`.

They cover upload acceptance, job timestamps, variant persistence,
database constraints, and variant downloads.

See [upload testing](docs/upload-testing.md) for acceptance-test details.

### JavaScript tests

Tested with Node.js 22.22.2.

```bash
node --experimental-default-type=module --test frontend/modules/*.test.mjs
```

These tests cover status-request timeouts, cancellation, response
validation, and measurement calculations. Browser behavior is checked
separately using the acceptance evidence.

## Worker and measurement settings

By default, one worker checks for queued jobs every 250 ms and applies
an intentional seven-second delay before processing the original.

The delay makes background processing observable during demonstrations.
Recorded processing durations include it.

To run with a different delay:

```bash
go run ./cmd/api -db-dsn="$GATEKEEPER_DB_DSN" -job-delay=7s
```

Browser polling waits approximately one second after each successful
active-state response before making the next request. Polling stops on
completion, processing failure, or a retrieval error.

After a retrieval error, Try again observes the same job without
resubmitting the image.

## Assessment documentation

- [Acceptance checklist](docs/acceptance-checklist.md)
- [Measurement procedure](docs/measurement-testing.md)
- [Recorded measurements](docs/measurements.csv)
- [Technical reflection](docs/reflection.md)
- [Demonstration script](docs/demo-script.md)
- [Week 4 check-in](CHECKIN.md)

## Version 1 boundaries

Selecting another image cancels browser observation, not server work.
Frontend submission protection does not prevent duplicate requests
across different tabs or clients.

The application uses local file storage, one in-process worker, and
short polling. Server push and automatic retries are not implemented.

Keep `.envrc` and uploaded/generated image files out of version control.
