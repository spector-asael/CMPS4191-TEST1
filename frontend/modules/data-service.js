export const API_BASE_URL = 'http://localhost:4000';

export const DataService = {
  async uploadImage(file) {
    const formData = new FormData();
    formData.append('image', file);

    const response = await fetch(`${API_BASE_URL}/v1/images`, {
      method: 'POST',
      body: formData,
    });

    if (response.status !== 202) {
      // Attempt to extract the error message returned in the JSON payload
      const errData = await response.json().catch(() => ({}));
      throw new Error(errData.error || `Upload failed with status ${response.status}`);
    }

    const data = await response.json();

    const expectedStates = ["queued", "processing", "completed", "failed"];

    // Check that the answer contains a nonempty job ID.
    if (!data || typeof data.id !== "string" || data.id.trim() === "") {
      throw new Error("Status response is missing a usable job ID");
    }

    // Check that the status is one our app understands.
    if (!expectedStates.includes(data.status)) {
      throw new Error("Status response contains an unexpected status");
    }

    return data;
  },

  async fetchJobStatus(statusUrl, signal) {
    // Prefix API_BASE_URL if statusUrl is a relative path like "/v1/jobs/42"
    const url = statusUrl.startsWith('http') 
      ? statusUrl 
      : `${API_BASE_URL}${statusUrl}`;

    const response = await fetch(url, { signal });

    if (!response.ok) {
      const errData = await response.json().catch(() => ({}));
      throw new Error(errData.error || `Failed to observe job: ${response.status}`);
    }

    return await response.json();
  }
};