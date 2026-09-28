# Measurement experiments

## Setup and conditions

Start the API and frontend using the README instructions.

For the recorded experiment:
- The browser and API ran on the same computer.
- One worker processed jobs.
- The worker used the default seven-second artificial processing delay.
- The worker checked for queued work every 250 ms.
- Browser status polling waited approximately one second between requests.
- Network throttling was disabled.
- The same image, `100_1749.JPG`, was used for the single run and burst.
- Existing jobs were allowed to finish before each experiment.
- The experiment tab remained visible.

The processing duration includes the artificial delay. These results are
not measurements of image transformation alone.

## Measurement definitions

All recorded durations are in milliseconds.

| CSV column | Definition |
|---|---|
| acknowledgement_ms | Upload start to receipt and validation of the acceptance response |
| queue_wait_ms | started_at minus queued_at |
| processing_ms | completed_at or failed_at minus started_at |
| job_duration_ms | completed_at minus queued_at; unavailable for failed jobs |
| polling_count | Number of attempted status requests for this job |
| detection_delay_ms | Browser observation of completion minus server completed_at |

Acknowledgement uses the browser's performance clock. Server durations
use job timestamps. Detection delay compares browser and server clocks,
so those clocks must agree.

JavaScript reads server timestamps at millisecond precision.
The UI rounds values for display; the CSV preserves calculated values.

## Single-image run

1. Reload the page and select a valid JPEG or PNG.
2. Click Process image once.
3. Keep the tab visible until completion.
4. Save a screenshot of the measurements panel.
5. In the browser developer console, run:

```js
var singleRun = (await import("./modules/measurements.js")).buildMeasurement(
  (await import("./state.js")).getState().activeJob,
  (await import("./state.js")).getState().metrics,
);

console.log([
  "single",
  1,
  singleRun.job_id,
  singleRun.status,
  singleRun.acknowledgement_ms,
  singleRun.queue_wait_ms,
  singleRun.processing_ms,
  singleRun.job_duration_ms,
  singleRun.polling_count,
  singleRun.detection_delay_ms,
].join(","));
```

Copy the printed CSV line into the results file below its header.

## Five-job burst

This experiment creates five real uploads. It is a separate test helper;
the normal page retains its protection against overlapping submissions.

1. Let existing jobs finish.
2. Reload the page and select the same image.
3. Wait for its preview to appear. Do not click Process image.
4. Open the browser developer console.
5. Run this command once:

```js
var burstResult = await (
  await import("./modules/burst-experiment.js")
).runBurst(
  (await import("./state.js")).getState().selectedFile
);
```

6. Keep the tab visible until all five observations finish.
7. Check that five rows report completed and no experiment errors appear.
8. Print the CSV rows:

```js
console.log(burstResult.csv);
```

Copy the five lines beneath the single-image row without adding another
header. Keep earlier results separately if repeating the experiment.

Run numbers identify submission attempts, not guaranteed queue order.
The helper observes each job independently and does not update the page's
processing card.

Each observer stops on completion, processing failure, retrieval error,
or after 120 status requests. It does not automatically retry uploads.
Stopping observation does not cancel server processing.

## Calculation tests

From the project root:

```bash
node --experimental-default-type=module --test frontend/modules/*.test.mjs
```

## Recorded results

- Data: [measurements.csv](measurements.csv)
- Single-run screenshot: [Measurements](evidence/10-single-image-measurements.png)
- Interpretation: [reflection.md](reflection.md)

The recorded data contains one single-image run and one five-job burst.
It is a small local experiment, not a general performance benchmark.