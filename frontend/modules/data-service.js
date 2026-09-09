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

    return await response.json();
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