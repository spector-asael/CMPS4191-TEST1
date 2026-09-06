import { emitter } from './modules/event-emitter.js';

emitter.on('stateChanged', (state) => {
  // 1. Target elements from index.html
  const previewImg = document.getElementById('preview-img');
  const fileName = document.getElementById('file-name');
  const fileMeta = document.getElementById('file-meta');
  const processBtn = document.getElementById('process-btn');
  const jobId = document.getElementById('job-id');
  const jobStatus = document.getElementById('job-status');
  const tryAgainContainer = document.getElementById('try-again-container');
  const variantsContainer = document.getElementById('variants-container');

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

  // 4. Render Active Job & Timeline Badges (UI-10, UI-11)
  if (state.activeJob) {
    jobId.textContent = `#${state.activeJob.job_id || state.activeJob.id}`;
    jobStatus.textContent = state.activeJob.status;
    
    const fmt = (ts) => ts ? new Date(ts).toLocaleTimeString() : '--';
    document.getElementById('time-upload').textContent = state.metrics?.requestStart ? 'Done' : '--';
    document.getElementById('time-stored').textContent = fmt(state.activeJob.queued_at);
    document.getElementById('time-variants').textContent = fmt(state.activeJob.started_at);
    document.getElementById('time-complete').textContent = state.activeJob.completed_at 
      ? fmt(state.activeJob.completed_at) 
      : (state.activeJob.status === 'failed' ? 'Failed' : 'Pending');
  } else {
    jobId.textContent = '#--';
    jobStatus.textContent = 'No active job';
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
      title.textContent = variant.name; // Safe text node manipulation

      const meta = document.createElement('span');
      meta.textContent = `${variant.width} × ${variant.height}`;

      const img = document.createElement('img');
      img.src = variant.url;
      img.alt = variant.name;

      card.append(img, title, meta);
      variantsContainer.appendChild(card);
    });
  } else if (!state.activeJob || state.activeJob.status !== 'completed') {
    variantsContainer.innerHTML = '<p class="empty-state">No images generated yet.</p>';
  }
});