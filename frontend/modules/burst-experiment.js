import { DataService } from "./data-service.js";
import { buildMeasurement } from "./measurements.js";

const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export async function runBurst(file) {
  if (!(file instanceof File)) {
    throw new Error("Choose a valid image before starting the experiment.");
  }

  async function runOne(run) {
    const requestStart = performance.now();
    const accepted = await DataService.uploadImage(file);

    const metrics = {
      ackLatency: performance.now() - requestStart,
      pollingCount: 0,
      completionObservedAt: null,
    };

    console.log(`Burst ${run} accepted: ${accepted.job_id}`);

    // Limit the experiment to 120 status requests per job.
    for (let attempt = 0; attempt < 120; attempt++) {
      await wait(1000);
      metrics.pollingCount++;

      let job;
      try {
        job = await DataService.fetchJobStatus(accepted.status_url);
      } catch (error) {
        throw new Error(
          `Run ${run}, job ${accepted.job_id}: observation stopped. ${error.message}`,
        );
      }

      const observedAt = Date.now();

      if (job.id !== accepted.job_id) {
        throw new Error(`Run ${run}: received a different job's response.`);
      }

      if (job.status === "completed" || job.status === "failed") {
        metrics.completionObservedAt =
          job.status === "completed" ? observedAt : null;

        const row = {
          scenario: "burst",
          run,
          ...buildMeasurement(job, metrics),
        };

        console.table([row]);
        return row;
      }
    }

    throw new Error(
      `Run ${run}, job ${accepted.job_id}: reached the observation limit.`,
    );
  }

  const outcomes = await Promise.allSettled(
    [1, 2, 3, 4, 5].map((run) => runOne(run)),
  );

  const rows = outcomes
    .filter((outcome) => outcome.status === "fulfilled")
    .map((outcome) => outcome.value);

  const errors = outcomes
    .filter((outcome) => outcome.status === "rejected")
    .map((outcome) => outcome.reason.message);

  const csv = rows
    .map((row) =>
      [
        row.scenario,
        row.run,
        row.job_id,
        row.status,
        row.acknowledgement_ms,
        row.queue_wait_ms,
        row.processing_ms,
        row.job_duration_ms,
        row.polling_count,
        row.detection_delay_ms,
      ].join(","),
    )
    .join("\n");

  console.table(rows);
  console.log("CSV rows:\n" + csv);

  if (errors.length > 0) {
    console.error("Experiment errors:", errors);
  }

  return { rows, csv, errors };
}
