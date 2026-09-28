export const API_BASE_URL = "http://localhost:4000";

export const DataService = {
  async uploadImage(file) {
    const formData = new FormData();
    formData.append("image", file);

    const response = await fetch(`${API_BASE_URL}/v1/images`, {
      method: "POST",
      body: formData,
    });

    if (response.status !== 202) {
      const errData = await response.json().catch(() => ({}));
      throw new Error(
        errData.error || `Upload failed with status ${response.status}`,
      );
    }

    return await response.json();
  },

  async fetchJobStatus(statusUrl, signal, timeoutMs = 5000) {
    const url = statusUrl.startsWith("http")
      ? statusUrl
      : `${API_BASE_URL}${statusUrl}`;

    // This controller belongs to this individual request.
    const requestController = new AbortController();
    let timedOut = false;

    // Forward deliberate cancellation from the polling loop.
    const cancelRequest = () => {
      requestController.abort();
    };

    if (signal?.aborted) {
      cancelRequest();
    } else {
      signal?.addEventListener("abort", cancelRequest, { once: true });
    }

    const timeoutId = setTimeout(() => {
      timedOut = true;
      requestController.abort();
    }, timeoutMs);

    try {
      const response = await fetch(url, {
        signal: requestController.signal,
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        throw new Error(
          errData.error || `Failed to observe job: ${response.status}`,
        );
      }

      const data = await response.json();

      const expectedStates = ["queued", "processing", "completed", "failed"];

      if (!data || typeof data.id !== "string" || data.id.trim() === "") {
        throw new Error("Status response is missing a usable job ID");
      }

      if (!expectedStates.includes(data.status)) {
        throw new Error("Status response contains an unexpected status");
      }

      return data;
    } catch (err) {
      // Switching jobs or leaving the page is deliberate cancellation.
      if (signal?.aborted) {
        throw err;
      }

      // A timeout must reach the polling loop as an observation error.
      if (timedOut) {
        const timeoutError = new Error("Status request timed out");
        timeoutError.name = "TimeoutError";
        throw timeoutError;
      }

      throw err;
    } finally {
      clearTimeout(timeoutId);
      signal?.removeEventListener("abort", cancelRequest);
    }
  },
};
