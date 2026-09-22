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

  async fetchJobStatus(statusUrl, signal) {
    const url = statusUrl.startsWith("http")
      ? statusUrl
      : `${API_BASE_URL}${statusUrl}`;

    const response = await fetch(url, { signal });

    if (!response.ok) {
      const errData = await response.json().catch(() => ({}));
      throw new Error(
        errData.error || `Failed to observe job: ${response.status}`,
      );
    }

    // Validate the status answer here, after a GET.
    const data = await response.json();

    const expectedStates = ["queued", "processing", "completed", "failed"];

    if (!data || typeof data.id !== "string" || data.id.trim() === "") {
      throw new Error("Status response is missing a usable job ID");
    }

    if (!expectedStates.includes(data.status)) {
      throw new Error("Status response contains an unexpected status");
    }

    return data;
  },
};
