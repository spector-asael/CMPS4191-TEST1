# GateKeeper: asynchronous report API

This complete starter repository accompanies the `202 Accepted + Job + Worker`
empirical lab. It preserves the synchronous lab's request and report, while
changing the contract and where the simulated work occurs.

**Reasoning model:** Requirement → Contract → Implementation → Measurement → Evaluation.

## Requirement and contract

Accept valid work promptly, continue processing independently, and expose the
eventual outcome through an observable job.

```http
POST /v1/reports
HTTP/1.1 202 Accepted
Location: /v1/jobs/{public-id}

{"job_id":"...","status":"queued","status_url":"/v1/jobs/..."}
```

Later, `GET /v1/jobs/{public-id}` returns the job's current state. A completed
job includes the same consumer-activity report returned by the synchronous API.

`202 Accepted` acknowledges accepted work; it does not promise that work has
finished or will succeed. This remains an asynchronously executed command API.

## Prerequisites and setup

Use the Go version in `go.mod`, PostgreSQL, `psql`, and the `migrate` CLI. The
original migrations require `citext`, `uuidv7()`, and `uuidv4()`.

```bash
cp .envrc.example .envrc
source .envrc
psql "$GATEKEEPER_DB_DSN" -c 'CREATE EXTENSION IF NOT EXISTS citext;'
psql "$GATEKEEPER_DB_DSN" -c 'SELECT uuidv7(), uuidv4();'
make db/migrations/up
```

Edit the example credentials before connecting. The original Makefile requires
`.envrc` to exist.

## Start, submit, and observe

```bash
go run ./cmd/api -db-dsn="$GATEKEEPER_DB_DSN" -report-delay=7s
```

Submit the same request body used in the synchronous version:

```bash
curl --include --silent --show-error \
  --write-out '\nacknowledgement_time=%{time_total}s\n' \
  --header 'Content-Type: application/json' \
  --data @request.json http://localhost:4000/v1/reports
```

Copy the returned job URL and inspect it:

```bash
curl --silent http://localhost:4000/v1/jobs/REPLACE_WITH_JOB_ID
```

Expected transitions: `queued → processing → completed`, or `failed`.

## Measurement experiment

Restart the server with `-report-delay=0s`, `3s`, `7s`, and `12s`. Measure two
separate clocks for each setting:

1. **Acknowledgement latency:** POST started → `202 Accepted` received.
2. **Completion latency:** POST started → job reaches `completed` or `failed`.

The artificial delay belongs to `cmd/api/worker.go`, never the POST handler.
Consequently acknowledgement should stay relatively stable while completion
increases with work duration. A 12-second worker delay can complete even though
the original server still has a 10-second HTTP `WriteTimeout`: no single HTTP
request must remain open throughout the background work.

The worker checks for queued jobs every 250 ms by default. Adjust this with
`-worker-poll-interval=1s` if your instructor wants queueing delay to be more
visible. This internal queue check is not the later client-polling lesson.

## Important files

- `cmd/api/reports.go`: create queued work, return `202`, read job status.
- `cmd/api/worker.go`: independent background execution and controlled delay.
- `internal/data/jobs.go`: durable jobs, public IDs, and `SKIP LOCKED` claims.
- `internal/data/reports.go`: unchanged consumer-activity report query.
- `migrations/000007_seed_sample_data.up.sql`: the same reproducible seed data.

Graceful shutdown cancels the worker before waiting for background tasks.
Production-grade retries, crash recovery, delivery guarantees, and push-based
notifications are deliberately outside this starter's scope.
