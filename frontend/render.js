import { emitter } from './modules/event-emitter.js';
import { API_BASE_URL } from './modules/data-service.js';

emitter.on('stateChanged', (state) => {
  // 1. Target elements from index.html
  const previewImg = document.getElementById('preview-img');
  const fileName = document.getElementById('file-name');
  const fileMeta = document.getElementById('file-meta');
  const processBtn = document.getElementById('process-btn');
  const uploadError = document.getElementById('upload-error');
  const jobId = document.getElementById('job-id');
  const jobStatus = document.getElementById('job-status');
  const tryAgainContainer = document.getElementById('try-again-container');
  const variantsContainer = document.getElementById('variants-container');

  // Render Immediate Upload Error Banner (VAL-05)
  if (state.uploadError) {
    uploadError.textContent = state.uploadError;
    uploadError.classList.remove('hidden');
  } else {
    uploadError.textContent = '';
    uploadError.classList.add('hidden');
  }

  // 2. Render File Selection & Preview (UI-05)
  if (state.selectedFile) {
    fileName.textContent = state.selectedFile.name;
    fileMeta.textContent = `${(state.selectedFile.size / (1024 * 1024)).toFixed(2)} MB • ${state.selectedFile.type}`;
    
    if (!previewImg.src || previewImg.dataset.file !== state.selectedFile.name) {
      previewImg.src = URL.createObjectURL(state.selectedFile);
      previewImg.dataset.file = state.selectedFile.name;
      previewImg.classList.remove('hidden');
    }
  } else {
    fileName.textContent = 'No file selected';
    fileMeta.textContent = '';
    previewImg.src = '';
    previewImg.classList.add('hidden');
  }

  // 3. Render Button State (UI-02, UI-08)
  processBtn.disabled = !state.selectedFile || state.isSubmitting;
  processBtn.textContent = state.isSubmitting ? 'Uploading...' : 'Process image';

  // 4. Render Active Job & Timeline Badges (UI-10, UI-11, UI-16)
  const stepUpload = document.getElementById('step-upload');
  const stepStored = document.getElementById('step-stored');
  const stepVariants = document.getElementById('step-variants');
  const stepComplete = document.getElementById('step-complete');
  const timeComplete = document.getElementById('time-complete');

  if (state.activeJob) {
    const status = state.activeJob.status;
    jobId.textContent = `#${state.activeJob.job_id || state.activeJob.id}`;
    jobStatus.textContent = status;
    jobStatus.className = `badge ${status}`;

    const fmt = (ts) => ts ? new Date(ts).toLocaleTimeString() : '--';
    document.getElementById('time-upload').textContent = state.metrics?.requestStart ? 'Done' : '--';
    document.getElementById('time-stored').textContent = fmt(state.activeJob.queued_at);
    document.getElementById('time-variants').textContent = fmt(state.activeJob.started_at);

    // Always completed once an active job exists (202 response)
    stepUpload.className = 'step-item flex-container completed';
    stepUpload.querySelector('.step-icon').textContent = '✓';

    stepStored.className = 'step-item flex-container completed';
    stepStored.querySelector('.step-icon').textContent = '✓';

    // Update Generating Variants step state
    if (status === 'completed') {
      stepVariants.className = 'step-item flex-container completed';
      stepVariants.querySelector('.step-icon').textContent = '✓';
    } else if (status === 'processing') {
      stepVariants.className = 'step-item flex-container active';
      stepVariants.querySelector('.step-icon').textContent = '⚙';
    } else {
      stepVariants.className = 'step-item flex-container';
      stepVariants.querySelector('.step-icon').textContent = '○';
    }

    // Update Complete step state & render timeline failure error
    if (status === 'completed') {
      stepComplete.className = 'step-item flex-container completed';
      stepComplete.querySelector('.step-icon').textContent = '✓';
      timeComplete.textContent = fmt(state.activeJob.completed_at);
      timeComplete.style.color = 'var(--text-muted)';
    } else if (status === 'failed') {
      stepComplete.className = 'step-item flex-container failed';
      stepComplete.querySelector('.step-icon').textContent = '✕';
      timeComplete.textContent = state.activeJob.error_message || 'Processing failed';
      timeComplete.style.color = 'var(--status-failed-text)';
    } else {
      stepComplete.className = 'step-item flex-container';
      stepComplete.querySelector('.step-icon').textContent = '○';
      timeComplete.textContent = 'Pending';
      timeComplete.style.color = 'var(--text-muted)';
    }
  } else {
    jobId.textContent = '#--';
    jobStatus.textContent = 'No active job';
    jobStatus.className = 'badge';

    // Reset timestamps when no job is active
    document.getElementById('time-upload').textContent = '--';
    document.getElementById('time-stored').textContent = '--';
    document.getElementById('time-variants').textContent = '--';
    timeComplete.textContent = '--';
    timeComplete.style.color = 'var(--text-muted)';

    // Reset timeline step icons and classes
    [stepUpload, stepStored, stepVariants, stepComplete].forEach(el => {
      el.className = 'step-item flex-container';
    });
    stepUpload.querySelector('.step-icon').textContent = '✓';
    stepStored.querySelector('.step-icon').textContent = '✓';
    stepVariants.querySelector('.step-icon').textContent = '⚙';
    stepComplete.querySelector('.step-icon').textContent = '○';
  }

  // 5. Render Observation Error Prompt (POLL-09)
  tryAgainContainer.style.display = state.observationError ? 'flex' : 'none';

  // 6. Render Dynamically Generated Variant Cards into Row 3 Grid (UI-15, IMG-04)
  if (state.activeJob?.status === 'completed' && state.variants?.length) {
    variantsContainer.innerHTML = ''; // Reset container

    state.variants.forEach(variant => {
      const card = document.createElement('div');
      card.className = 'grid-card variant-card flex-container';

      const title = document.createElement('h4');
      title.textContent = variant.name;

      const meta = document.createElement('span');
      meta.textContent = `${variant.width} × ${variant.height}`;

      const img = document.createElement('img');
      img.src = variant.url.startsWith('http') ? variant.url : `${API_BASE_URL}${variant.url}`;
      img.alt = variant.name;

      card.append(img, title, meta);
      variantsContainer.appendChild(card);
    });
  } else if (!state.activeJob || state.activeJob.status !== 'completed') {
    variantsContainer.innerHTML = '<p class="empty-state">No images generated yet.</p>';
  }
});