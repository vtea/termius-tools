import { GetStatus, CreateBackup, RestoreBackup, InspectBackup, RevealInFinder } from './wailsjs/go/main/App.js';

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => document.querySelectorAll(sel);

let status = {};

// Tab navigation
$$('.nav-item').forEach((btn) => {
  btn.addEventListener('click', () => {
    $$('.nav-item').forEach((b) => b.classList.remove('active'));
    $$('.panel').forEach((p) => p.classList.remove('active'));
    btn.classList.add('active');
    $(`#panel-${btn.dataset.tab}`).classList.add('active');
  });
});

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(iso) {
  if (!iso) return '—';
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });
}

function showToast(message, type = 'success') {
  const toast = $('#toast');
  toast.textContent = message;
  toast.className = `toast ${type}`;
  toast.hidden = false;
  setTimeout(() => { toast.hidden = true; }, 4000);
}

function confirmDialog(title, message) {
  return new Promise((resolve) => {
    const dialog = $('#confirm-dialog');
    $('#confirm-title').textContent = title;
    $('#confirm-message').textContent = message;
    dialog.showModal();

    const cleanup = (result) => {
      dialog.close();
      $('#confirm-ok').onclick = null;
      $('#confirm-cancel').onclick = null;
      resolve(result);
    };

    $('#confirm-ok').onclick = () => cleanup(true);
    $('#confirm-cancel').onclick = () => cleanup(false);
    dialog.addEventListener('cancel', () => cleanup(false), { once: true });
  });
}

async function refreshStatus() {
  try {
    status = await GetStatus();
    $('#status-os').textContent = status.os;
    $('#app-version').textContent = `v${status.version}`;

    const termiusEl = $('#status-termius');
    if (status.termiusRunning) {
      termiusEl.textContent = '运行中';
      termiusEl.className = 'status-badge warn';
    } else {
      termiusEl.textContent = '已关闭';
      termiusEl.className = 'status-badge ok';
    }

    const dataEl = $('#status-data');
    if (status.dataDirExists) {
      dataEl.textContent = '已找到';
      dataEl.className = 'status-badge ok';
    } else {
      dataEl.textContent = '未找到';
      dataEl.className = 'status-badge err';
    }

    // Update warnings
    $('#backup-warning').hidden = !status.termiusRunning;
    $('#restore-running-warning').hidden = !status.termiusRunning;
    $('#restore-hint').textContent = status.closeHint || '恢复前请先关闭 Termius。';
    $('#btn-restore').disabled = status.termiusRunning;
  } catch (e) {
    console.error('Status refresh failed:', e);
  }
}

// Backup
$('#btn-backup').addEventListener('click', async () => {
  const btn = $('#btn-backup');
  const resultEl = $('#backup-result');
  resultEl.hidden = true;

  if (status.termiusRunning && !$('#backup-force').checked) {
    showToast('请关闭 Termius，或勾选“仍要继续”', 'error');
    return;
  }

  btn.disabled = true;
  btn.textContent = '正在创建备份…';

  try {
    const result = await CreateBackup('', status.termiusRunning && $('#backup-force').checked);
    resultEl.innerHTML = `
      <strong>备份创建成功</strong><br>
      已保存 ${result.fileCount} 个文件至 <code>${result.path}</code>
      <br><button class="btn-link" id="reveal-backup">在访达中显示</button>
    `;
    resultEl.hidden = false;
    $('#reveal-backup').addEventListener('click', () => RevealInFinder(result.path));
    showToast('备份已创建！');
  } catch (e) {
    if (e !== 'cancelled' && !String(e).includes('cancelled')) {
      showToast(String(e), 'error');
    }
  } finally {
    btn.disabled = false;
    btn.innerHTML = `
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
      创建备份…
    `;
    refreshStatus();
  }
});

// Restore
$('#btn-restore').addEventListener('click', async () => {
  if (status.termiusRunning) {
    showToast('恢复前请先关闭 Termius', 'error');
    return;
  }

  const confirmed = await confirmDialog(
    '确认恢复 Termius 备份？',
    '此操作将覆盖当前的 Termius 数据，并从备份中恢复加密密钥。请确保 Termius 已关闭。'
  );
  if (!confirmed) return;

  const btn = $('#btn-restore');
  const resultEl = $('#restore-result');
  resultEl.hidden = true;
  btn.disabled = true;

  try {
    const result = await RestoreBackup('');
    resultEl.innerHTML = `
      <strong>恢复完成</strong><br>
      已恢复 ${result.restored} 个文件，现在可以打开 Termius。
    `;
    resultEl.hidden = false;
    showToast('恢复完成！');
  } catch (e) {
    if (e !== 'cancelled' && !String(e).includes('cancelled')) {
      showToast(String(e), 'error');
    }
  } finally {
    btn.disabled = status.termiusRunning;
    refreshStatus();
  }
});

// Inspect
$('#btn-inspect').addEventListener('click', async () => {
  const resultEl = $('#inspect-result');
  const errorEl = $('#inspect-error');
  resultEl.hidden = true;
  errorEl.hidden = true;

  try {
    const info = await InspectBackup('');
    $('#inspect-version').textContent = info.version;
    $('#inspect-created').textContent = formatDate(info.createdAt);
    $('#inspect-host').textContent = info.hostname;
    $('#inspect-count').textContent = `${info.fileCount} 个文件`;

    const fileList = $('#inspect-files');
    fileList.innerHTML = info.files
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((f) => `<li><span>${f.name}</span><span class="file-size">${formatBytes(f.size)}</span></li>`)
      .join('');

    resultEl.hidden = false;
  } catch (e) {
    if (e !== 'cancelled' && !String(e).includes('cancelled')) {
      errorEl.textContent = String(e);
      errorEl.hidden = false;
    }
  }
});

// Init
refreshStatus();
setInterval(refreshStatus, 5000);
