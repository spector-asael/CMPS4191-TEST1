import test from "node:test";
import assert from "node:assert/strict";
import { DataService } from "./data-service.js";

// Simulate a request that never finishes unless its signal is aborted.
function waitingFetch(_url, { signal }) {
  return new Promise((resolve, reject) => {
    const rejectOnAbort = () => {
      reject(new DOMException("Request cancelled", "AbortError"));
    };

    if (signal.aborted) {
      rejectOnAbort();
      return;
    }

    signal.addEventListener("abort", rejectOnAbort, { once: true });
  });
}

test("a stalled status request becomes a TimeoutError", async (t) => {
  t.mock.method(globalThis, "fetch", waitingFetch);

  const controller = new AbortController();

  await assert.rejects(
    DataService.fetchJobStatus("/v1/jobs/test-job", controller.signal, 20),
    { name: "TimeoutError" },
  );

  // Timing out one request must not abort the observation controller.
  assert.equal(controller.signal.aborted, false);
});

test("deliberate cancellation remains an AbortError", async (t) => {
  t.mock.method(globalThis, "fetch", waitingFetch);

  const controller = new AbortController();

  const request = DataService.fetchJobStatus(
    "/v1/jobs/test-job",
    controller.signal,
    1000,
  );

  controller.abort();

  await assert.rejects(request, { name: "AbortError" });
});

test("a successful status response is returned normally", async (t) => {
  const expected = {
    id: "test-job",
    status: "processing",
  };

  t.mock.method(globalThis, "fetch", async () => ({
    ok: true,
    json: async () => expected,
  }));

  const controller = new AbortController();

  const result = await DataService.fetchJobStatus(
    "/v1/jobs/test-job",
    controller.signal,
    1000,
  );

  assert.deepEqual(result, expected);
});
