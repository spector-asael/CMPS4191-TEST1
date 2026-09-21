import { getState, setState } from "./state.js";
import { DataService } from "./modules/data-service.js";

// Elements
const fileInput = document.getElementById("file-input");
const processBtn = document.getElementById("process-btn");
const tryAgainBtn = document.getElementById("try-again-btn");

// 1. Cancel observation if the browser tab or page is closed/refreshed (POLL-06)
window.addEventListener("beforeunload", () => {
  stopPolling();
});

// Preview Image Selection (UI-05, UI-06)
fileInput.addEventListener("change", (e) => {
  const file = e.target.files[0];
  if (!file) return;

  // Cancel active observation if switching/selecting a new file (POLL-06)
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

  // Clean up any stale polling before initiating a new job
  stopPolling();

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
  if (abortController) {
    abortController.abort(); // Instantly cancels active GET fetch in DevTools (POLL-06)
  }
  setState({ pollingTimer: null, abortController: null });
}

async function poll(statusUrl, signal) {
  try {
    const jobData = await DataService.fetchJobStatus(statusUrl, signal);

    const mergedJob = { ...jobData, status_url: statusUrl };

    // Server processed job (completed or failed)
    if (jobData.status === "completed" || jobData.status === "failed") {
      stopPolling(); // Stop polling when terminal state is reached (POLL-05)
      setState({ activeJob: mergedJob, variants: jobData.variants || [] });
    } else {
      setState({ activeJob: mergedJob }); // queued or processing
    }
  } catch (err) {
    // Ignore aborted fetch errors caused by stopPolling()
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