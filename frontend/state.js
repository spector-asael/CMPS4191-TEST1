import { emitter } from './modules/event-emitter.js';

const state = {
  selectedFile: null,
  isSubmitting: false, // Double-submit guard
  activeJob: null,      // Stores status, job_id, image_id, timestamps
  variants: [],
  pollingTimer: null,
  abortController: null, // For cancelling fetch requests on reset
  observationError: false, // Distinguishes GET failure from job failure[cite: 1]
  metrics: { requestStart: null, ackLatency: null } // Latency measurements[cite: 1]
};

export function getState() {
  return state;
}

export function setState(updates) {
  Object.assign(state, updates);
  emitter.emit('stateChanged', state);
}