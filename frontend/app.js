import { getState, setState } from "./state.js";
import { DataService } from "./modules/data-service.js";

// Elements
const fileInput = document.getElementById("file-input");
const processBtn = document.getElementById("process-btn");
const tryAgainBtn = document.getElementById("try-again-btn");

// Preview Image Selection (UI-05, UI-06)
fileInput.addEventListener("change", (e) => {
  const file = e.target.files[0];
  if (!file) return;

  // Reset existing polling if picking a new file (POLL-06)
  stopPolling();

  setState({
    selectedFile: file,
    uploadError: null, // Clear past upload errors on new selection
    activeJob: null,
    variants: [],
    observationError: false,
  });
});

// Primary Upload Submission Handler (SUB-01, SUB-02)
processBtn.addEventListener("click", async () => {
  const { isSubmitting, selectedFile, activeJob } = getState();
  const isJobActive = activeJob && (activeJob.status === "queued" || activeJob.status === "processing");
  if (isSubmitting || !selectedFile || isJobActive) return;

  const requestStart = performance.now();
  setState({ isSubmitting: true, uploadError: null, observationError: false });

  try {
    const data = await DataService.uploadImage(selectedFile);
    const ackLatency = performance.now() - requestStart; // Measure Ack Latency

    setState({
      activeJob: data,
      metrics: { requestStart, ackLatency },
    });

    startPolling(data.status_url); // POLL-01
  } catch (err) {
    // Store rejection message into state instead of browser alert
    setState({ uploadError: err.message || "Submission error" });
  } finally {
    setState({ isSubmitting: false });
  }
});

// Polling Control Loop (POLL-02 to POLL-10)
function startPolling(statusUrl) {
  stopPolling();

  const controller = new AbortController();
  const timer = setInterval(() => poll(statusUrl, controller.signal), 1000); // ~1 sec interval

  setState({ pollingTimer: timer, abortController: controller });
}

function stopPolling() {
  const { pollingTimer, abortController } = getState();
  if (pollingTimer) clearInterval(pollingTimer);
  if (abortController) abortController.abort(); // Cancel active GETs (POLL-06)
  setState({ pollingTimer: null, abortController: null });
}

async function poll(statusUrl, signal) {
  try {
    const jobData = await DataService.fetchJobStatus(statusUrl, signal);

    // Server processed job (completed or failed)
    if (jobData.status === "completed" || jobData.status === "failed") {
      stopPolling(); // POLL-05
      setState({ activeJob: jobData, variants: jobData.variants || [] });
    } else {
      setState({ activeJob: jobData }); // queued or processing
    }
  } catch (err) {
    if (err.name === "AbortError") return;

    // Retrieval Error Policy: Stop polling, preserve job, show "Try again" (POLL-07 to POLL-10)
    stopPolling();
    setState({ observationError: true });
  }
}

// Try Again Handler (POLL-09, POLL-10)
tryAgainBtn.addEventListener("click", () => {
  const { activeJob } = getState();
  if (activeJob?.status_url) {
    setState({ observationError: false });
    startPolling(activeJob.status_url);
  }
});
