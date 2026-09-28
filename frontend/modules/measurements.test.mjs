import test from "node:test";
import assert from "node:assert/strict";
import { buildMeasurement } from "./measurements.js";

test("calculates all six measurements for a completed job", () => {
  const job = {
    id: "test-job",
    status: "completed",
    queued_at: "2026-09-28T12:00:00.000Z",
    started_at: "2026-09-28T12:00:00.250Z",
    completed_at: "2026-09-28T12:00:02.250Z",
  };

  const metrics = {
    ackLatency: 40,
    pollingCount: 3,
    completionObservedAt: Date.parse("2026-09-28T12:00:02.500Z"),
  };

  assert.deepEqual(buildMeasurement(job, metrics), {
    job_id: "test-job",
    status: "completed",
    acknowledgement_ms: 40,
    queue_wait_ms: 250,
    processing_ms: 2000,
    job_duration_ms: 2250,
    polling_count: 3,
    detection_delay_ms: 250,
  });
});

test("uses the failure timestamp without inventing completion measurements", () => {
  const job = {
    id: "failed-job",
    status: "failed",
    queued_at: "2026-09-28T12:00:00.000Z",
    started_at: "2026-09-28T12:00:01.000Z",
    failed_at: "2026-09-28T12:00:03.000Z",
  };

  const result = buildMeasurement(job, {
    ackLatency: 30,
    pollingCount: 4,
    completionObservedAt: null,
  });

  assert.equal(result.queue_wait_ms, 1000);
  assert.equal(result.processing_ms, 2000);
  assert.equal(result.job_duration_ms, null);
  assert.equal(result.detection_delay_ms, null);
});

test("missing or unreadable timestamps produce unavailable measurements", () => {
  const result = buildMeasurement(
    {
      id: "incomplete-job",
      status: "completed",
      queued_at: null,
      started_at: "not a timestamp",
      completed_at: null,
    },
    {
      ackLatency: 25,
      pollingCount: 2,
      completionObservedAt: null,
    },
  );

  assert.equal(result.queue_wait_ms, null);
  assert.equal(result.processing_ms, null);
  assert.equal(result.job_duration_ms, null);
  assert.equal(result.detection_delay_ms, null);
});

test("preserves negative detection delay so clock problems remain visible", () => {
  const result = buildMeasurement(
    {
      id: "clock-check",
      status: "completed",
      queued_at: "2026-09-28T12:00:00.000Z",
      started_at: "2026-09-28T12:00:01.000Z",
      completed_at: "2026-09-28T12:00:03.000Z",
    },
    {
      ackLatency: 25,
      pollingCount: 3,
      completionObservedAt: Date.parse("2026-09-28T12:00:02.900Z"),
    },
  );

  assert.equal(result.detection_delay_ms, -100);
});
