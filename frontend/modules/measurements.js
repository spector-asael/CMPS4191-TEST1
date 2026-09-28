function timestamp(value) {
  return typeof value === "string" ? Date.parse(value) : NaN;
}

function difference(end, start) {
  if (!Number.isFinite(end) || !Number.isFinite(start)) {
    return null;
  }

  return end - start;
}

export function buildMeasurement(job, metrics) {
  const queued = timestamp(job.queued_at);
  const started = timestamp(job.started_at);
  const completed = timestamp(job.completed_at);
  const finished =
    job.status === "failed" ? timestamp(job.failed_at) : completed;

  return {
    job_id: job.id,
    status: job.status,
    acknowledgement_ms: metrics.ackLatency,
    queue_wait_ms: difference(started, queued),
    processing_ms: difference(finished, started),
    job_duration_ms:
      job.status === "completed" ? difference(completed, queued) : null,
    polling_count: metrics.pollingCount,
    detection_delay_ms:
      job.status === "completed"
        ? difference(metrics.completionObservedAt, completed)
        : null,
  };
}