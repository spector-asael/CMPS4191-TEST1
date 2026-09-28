# Upload acceptance tests

## Ordinary tests and code checks

Run these from the project root:

```bash
go test ./...
go vet ./...
```

The ordinary tests include invalid-upload rejection and image sizing/cropping.
The PostgreSQL integration test is skipped unless explicitly enabled.

## Isolated PostgreSQL integration test

Prerequisites: Go matching `go.mod`, PostgreSQL 18 server tools `initdb` and
`pg_ctl` available on PATH, and permission to create a local Unix socket.
Run as your normal user, not root. No running application server is needed.

```bash
IMAGELAB_INTEGRATION_TESTS=1 go test ./cmd/api -run TestUploadAcceptanceIntegration -v -count=1
```

This test starts a disposable PostgreSQL instance in a private temporary folder,
applies the project's migrations, and uses temporary upload folders. It does not
read `.envrc` or connect to the existing `imagelab` or `gatekeeper` databases.
It stops the temporary server and removes its files when finished.

Cases checked:

- PNG and JPEG return 202 with a matching Location/status URL, a queued job,
  related image record, server-controlled filename, and unchanged original bytes.
- A readable PNG padded to exactly 10,485,760 bytes is accepted despite the extra
  multipart request packaging. This checks the existing binary interpretation of
  the application's 10 MB limit.
- A database constraint rejecting the image insert leaves no records or files.
- A file blocking the uploads directory causes rejection and rolls back the image
  record while preserving the pre-existing blocking file.
- A database constraint rejecting the job insert rolls back both records and
  removes the original image folder.
- Rejections return a generic client-safe error and no job Location header.

The database constraints used to simulate failures exist only in the disposable
test database. The worker is not started: these tests check acceptance, not
background processing or browser behavior. They do not simulate a lost commit
confirmation, disk-full errors, file-close failures, or a process crash.

## Verification recorded for this change

- All six integration cases passed.
- `go test ./...` passed.
- `go vet ./...` passed.

The initial sandboxed integration attempt could not create a Unix socket; the
same test passed when run with permission to start the temporary local server.
Re-run the commands after changing upload acceptance code.
