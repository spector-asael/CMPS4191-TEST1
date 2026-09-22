import { emitter } from "./modules/event-emitter.js";
import { API_BASE_URL } from "./modules/data-service.js";

emitter.on("stateChanged", (state) => {
  // 1. Target elements from index.html
  const previewImg = document.getElementById("preview-img");
  const fileName = document.getElementById("file-name");
  const fileMeta = document.getElementById("file-meta");
  const processBtn = document.getElementById("process-btn");
  const uploadError = document.getElementById("upload-error");
  const jobId = document.getElementById("job-id");
  const jobStatus = document.getElementById("job-status");
  const tryAgainContainer = document.getElementById("try-again-container");
  const variantsContainer = document.getElementById("variants-container");
  const labelComplete = document.getElementById("label-complete");
  const jobError = document.getElementById("job-error");
  const pollingIndicator = document.getElementById("polling-indicator");
  const jobWorkText = document.getElementById("job-work-text");

  // Render Immediate Upload Error Banner (VAL-05)
  if (state.uploadError) {
    uploadError.textContent = state.uploadError;
    uploadError.classList.remove("hidden");
  } else {
    uploadError.textContent = "";
    uploadError.classList.add("hidden");
  }

  // 2. Render File Selection & Preview (UI-05)
  if (state.selectedFile) {
    fileName.textContent = state.selectedFile.name;
    fileMeta.textContent = `${(state.selectedFile.size / (1024 * 1024)).toFixed(2)} MB • ${state.selectedFile.type}`;

    if (
      !previewImg.src ||
      previewImg.dataset.file !== state.selectedFile.name
    ) {
      previewImg.src = URL.createObjectURL(state.selectedFile);
      previewImg.dataset.file = state.selectedFile.name;
      previewImg.classList.remove("hidden");
    }
  } else {
    fileName.textContent = "No file selected";
    fileMeta.textContent = "";
    previewImg.src = "";
    previewImg.classList.add("hidden");
  }

  // 3. Render Button State (UI-02, UI-08)
  const isJobActive =
    state.activeJob &&
    (state.activeJob.status === "queued" ||
      state.activeJob.status === "processing");

  // Show this only while the app is observing an unfinished job.
  const isPolling =
    isJobActive && state.abortController !== null && !state.observationError;

  pollingIndicator.classList.toggle("hidden", !isPolling);

  // Explain what is happening to the current job.
  if (state.observationError) {
    jobWorkText.textContent =
      "Status checks paused. The job may still be running.";
  } else if (state.activeJob?.status === "queued") {
    jobWorkText.textContent = "Waiting for processing to start.";
  } else if (state.activeJob?.status === "processing") {
    jobWorkText.textContent = "Generating image variants.";
  } else if (state.activeJob?.status === "completed") {
    jobWorkText.textContent = "Your images are ready below.";
  } else if (state.activeJob?.status === "failed") {
    jobWorkText.textContent = "Image processing failed. See the error below.";
  } else {
    jobWorkText.textContent = "";
  }

  processBtn.disabled =
    !state.selectedFile || state.isSubmitting || isJobActive;
  if (state.isSubmitting) {
    processBtn.textContent = "Uploading...";
  } else if (isJobActive) {
    processBtn.textContent = "Processing...";
  } else {
    processBtn.textContent = "Process image";
  }

  // 4. Render Active Job & Timeline Badges (UI-10, UI-11, UI-16)
  const stepUpload = document.getElementById("step-upload");
  const stepStored = document.getElementById("step-stored");
  const stepVariants = document.getElementById("step-variants");
  const stepComplete = document.getElementById("step-complete");
  const timeComplete = document.getElementById("time-complete");

  if (state.activeJob) {
    const status = state.activeJob.status;
    jobId.textContent = `#${state.activeJob.job_id || state.activeJob.id}`;
    jobStatus.textContent = status;
    jobStatus.className = `badge ${status}`;

    const fmt = (ts) => (ts ? new Date(ts).toLocaleTimeString() : "--");
    document.getElementById("time-upload").textContent = state.metrics
      ?.requestStart
      ? "Done"
      : "--";
    document.getElementById("time-stored").textContent = fmt(
      state.activeJob.queued_at,
    );
    document.getElementById("time-variants").textContent = fmt(
      state.activeJob.started_at,
    );

    stepUpload.className = "step-item flex-container completed";
    stepUpload.querySelector(".step-icon").textContent = "✓";

    stepStored.className = "step-item flex-container completed";
    stepStored.querySelector(".step-icon").textContent = "✓";

    // Update Generating Variants step state
    if (status === "completed") {
      stepVariants.className = "step-item flex-container completed";
      stepVariants.querySelector(".step-icon").textContent = "✓";
    } else if (status === "processing") {
      stepVariants.className = "step-item flex-container active";
      stepVariants.querySelector(".step-icon").textContent = "⚙";
    } else if (status === "failed") {
      stepVariants.className = "step-item flex-container failed";
      stepVariants.querySelector(".step-icon").textContent = "✕";
    } else {
      stepVariants.className = "step-item flex-container";
      stepVariants.querySelector(".step-icon").textContent = "○";
    }

    // Update Final Timeline Step State & Error Display
    if (status === "completed") {
      stepComplete.className = "step-item flex-container completed";
      stepComplete.querySelector(".step-icon").textContent = "✓";
      if (labelComplete) labelComplete.textContent = "Complete";
      timeComplete.textContent = fmt(state.activeJob.completed_at);
      timeComplete.style.color = "var(--text-muted)";
      if (jobError) jobError.classList.add("hidden");
    } else if (status === "failed") {
      stepComplete.className = "step-item flex-container failed";
      stepComplete.querySelector(".step-icon").textContent = "✕";
      if (labelComplete) labelComplete.textContent = "Failed";
      timeComplete.textContent = fmt(
        state.activeJob.completed_at || state.activeJob.started_at,
      );
      timeComplete.style.color = "var(--status-failed-text)";

      // Display clear, prominent processing error banner
      if (jobError) {
        jobError.textContent = `Processing Error: ${state.activeJob.error_message || "Processing failed"}`;
        jobError.classList.remove("hidden");
      }
    } else {
      stepComplete.className = "step-item flex-container";
      stepComplete.querySelector(".step-icon").textContent = "○";
      if (labelComplete) labelComplete.textContent = "Complete";
      timeComplete.textContent = "Pending";
      timeComplete.style.color = "var(--text-muted)";
      if (jobError) jobError.classList.add("hidden");
    }
  } else {
    jobId.textContent = "#--";
    jobStatus.textContent = "No active job";
    jobStatus.className = "badge";

    // Reset timestamps when no job is active
    document.getElementById("time-upload").textContent = "--";
    document.getElementById("time-stored").textContent = "--";
    document.getElementById("time-variants").textContent = "--";
    timeComplete.textContent = "--";
    timeComplete.style.color = "var(--text-muted)";

    if (labelComplete) labelComplete.textContent = "Complete";
    if (jobError) jobError.classList.add("hidden");

    // Reset timeline step icons and classes
    [stepUpload, stepStored, stepVariants, stepComplete].forEach((el) => {
      el.className = "step-item flex-container";
    });
    stepUpload.querySelector(".step-icon").textContent = "✓";
    stepStored.querySelector(".step-icon").textContent = "✓";
    stepVariants.querySelector(".step-icon").textContent = "⚙";
    stepComplete.querySelector(".step-icon").textContent = "○";
  }

  // 5. Render Observation Error Prompt (POLL-09)
  tryAgainContainer.style.display = state.observationError ? "flex" : "none";

  // 6. Render Dynamically Generated Variant Cards
  if (state.activeJob?.status === "completed" && state.variants?.length) {
    variantsContainer.innerHTML = "";

    state.variants.forEach((variant) => {
      const card = document.createElement("div");
      card.className = "grid-card variant-card flex-container";

      const title = document.createElement("h4");
      title.textContent = variant.name;

      const meta = document.createElement("span");
      meta.textContent = `${variant.width} × ${variant.height}`;

      const img = document.createElement("img");
      img.src = variant.url.startsWith("http")
        ? variant.url
        : `${API_BASE_URL}${variant.url}`;
      img.alt = variant.name;

      const link = document.createElement("a");
      link.href = img.src;
      link.textContent = "View image";
      link.target = "_blank";
      link.rel = "noopener";

      const downloadBtn = document.createElement("button");
      downloadBtn.type = "button";
      downloadBtn.className = "btn-secondary";
      downloadBtn.textContent = "Download";

      downloadBtn.addEventListener("click", async () => {
        downloadBtn.disabled = true;
        downloadBtn.textContent = "Downloading…";

        try {
          const response = await fetch(img.src);

          if (!response.ok) {
            throw new Error("Download failed");
          }

          const blob = await response.blob();
          const downloadUrl = URL.createObjectURL(blob);
          const extension = blob.type === "image/png" ? "png" : "jpg";

          const downloadLink = document.createElement("a");
          downloadLink.href = downloadUrl;
          downloadLink.download = `${variant.name}.${extension}`;

          document.body.appendChild(downloadLink);
          downloadLink.click();
          downloadLink.remove();

          setTimeout(() => URL.revokeObjectURL(downloadUrl), 60000);

          downloadBtn.textContent = "Download";
        } catch (err) {
          downloadBtn.textContent = "Download failed — try again";
        } finally {
          downloadBtn.disabled = false;
        }
      });
      card.append(img, title, meta, link, downloadBtn);
      variantsContainer.appendChild(card);
    });
  } else if (isJobActive) {
    variantsContainer.innerHTML = `
    <p class="empty-state">
      Images are being generated and will appear here when processing completes.
    </p>
  `;
  } else {
    variantsContainer.innerHTML = `
    <p class="empty-state">No images generated yet.</p>
  `;
  }
});
