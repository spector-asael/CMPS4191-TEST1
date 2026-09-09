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
- **Go** (1.23+ / 1.25)
- **PostgreSQL** (14+)
- **golang-migrate CLI**
- **Node.js** (for `npx http-server` frontend hosting)

### 1. Environment Configuration
Copy the example environment file and set your PostgreSQL DSN:
```bash
cp .envrc.example .envrc
# Edit .envrc with your database credentials:
# export GATEKEEPER_DB_DSN='postgres://user:password@localhost:5432/imagelab?sslmode=disable'
source .envrc
```

### 2. Database Setup & Migrations
Create your database and run migrations:
```bash
createdb imagelab
make db/migrations/up
```

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
- **Constraints**: JPEG or PNG format only; max 10 MB (**VAL-01**, **VAL-02**).
- **Response**: `202 Accepted` with `Location` header and JSON status body:
```http
HTTP/1.1 202 Accepted
Location: /v1/jobs/01917f90-1234-7000-8000-000000000001
Content-Type: application/json

{
  "image_id": 1,
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
  "status": "completed",
  "queued_at": "2026-09-08T13:00:00Z",
  "started_at": "2026-09-08T13:00:01Z",
  "completed_at": "2026-09-08T13:00:03Z",
  "variants": [
    {"name": "thumbnail", "width": 150, "height": 150, "url": "/v1/images/1/variants/thumbnail"},
    {"name": "preview", "width": 800, "height": 600, "url": "/v1/images/1/variants/preview"},
    {"name": "display", "width": 1200, "height": 900, "url": "/v1/images/1/variants/display"}
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
