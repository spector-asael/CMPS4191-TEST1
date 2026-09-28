import { getState, setState } from "./state.js";
import { DataService } from "./modules/data-service.js";

// Elements
const fileInput = document.getElementById("file-input");
const chooseImageBtn = document.getElementById("choose-image-btn");
const processBtn = document.getElementById("process-btn");
const tryAgainBtn = document.getElementById("try-again-btn");

// 1. Cancel observation if the browser tab or page is closed/refreshed (POLL-06)
window.addEventListener("beforeunload", () => {
  stopPolling();
});

// Preview Image Selection (UI-05, UI-06)
fileInput.addEventListener("change", (e) => {
  if (getState().isSubmitting) return;

  const file = e.target.files[0];
  if (!file) return;

  const maxImageSize = 10 * 1024 * 1024;
  const allowedTypes = ["image/jpeg", "image/png"];

  let error = null;

  if (file.size === 0) {
    error = "This file is empty. Choose a JPEG or PNG image.";
  } else if (!allowedTypes.includes(file.type)) {
    error = "Choose a JPEG or PNG image.";
  } else if (file.size > maxImageSize) {
    error = "This image exceeds the 10 MB limit. Choose a smaller image.";
  }

  if (error) {
    fileInput.value = "";

    setState({
      selectedFile: null,
      uploadError: error,
    });

    // A rejected selection must not interrupt an existing job.
    return;
  }

  // A valid new selection replaces the previous observation.
  stopPolling();

  setState({
    selectedFile: file,
    uploadError: null,
    activeJob: null,
    variants: [],
    observationError: false,
    metrics: {
      requestStart: null,
      ackLatency: null,
    },
  });
});

// Primary Upload Submission Handler (SUB-01, SUB-02)
processBtn.addEventListener("click", async () => {
  const { isSubmitting, selectedFile, activeJob } = getState();
  const isJobActive =
    activeJob &&
    (activeJob.status === "queued" || activeJob.status === "processing");
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

chooseImageBtn.addEventListener("click", () => {
  if (getState().isSubmitting) return;

  // Allow selecting the same file again.
  fileInput.value = "";
  fileInput.click();
});

// Polling Control Loop (POLL-02 to POLL-10)
function startPolling(statusUrl) {
  stopPolling();

  const controller = new AbortController();

  const timer = setTimeout(() => {
    poll(statusUrl, controller);
  }, 1000);

  setState({
    pollingTimer: timer,
    abortController: controller,
  });
}

function stopPolling() {
  const { pollingTimer, abortController } = getState();

  if (pollingTimer !== null) {
    clearTimeout(pollingTimer);
  }

  if (abortController) {
    abortController.abort();
  }

  setState({
    pollingTimer: null,
    abortController: null,
  });
}

async function poll(statusUrl, controller) {
  // Does this request still belong to the currrent observation?
  const isCurrent = () =>
    getState().abortController === controller && !controller.signal.aborted;

  if (!isCurrent()) return;

  try {
    const jobData = await DataService.fetchJobStatus(
      statusUrl,
      controller.signal,
    );

    // The user might have switched images while we waited.
    if (!isCurrent()) return;

    const activeJob = getState().activeJob;
    const expectedJobId = activeJob?.job_id ?? activeJob?.id;

    if (jobData.id !== expectedJobId) {
      throw new Error("Status response belongs to a different job");
    }

    const mergedJob = {
      ...jobData,
      status_url: statusUrl,
    };

    if (jobData.status === "completed" || jobData.status === "failed") {
      stopPolling();

      setState({
        activeJob: mergedJob,
        variants: jobData.variants || [],
      });

      return;
    }

    // The job is still running. Update its display status.
    setState({ activeJob: mergedJob });

    // Only now schedule the next check.
    const timer = setTimeout(() => {
      poll(statusUrl, controller);
    }, 1000);

    setState({ pollingTimer: timer });
  } catch (err) {
    // An old or deliberately cancelled request should do nothing.
    if (!isCurrent() || err.name === "AbortError") return;

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
