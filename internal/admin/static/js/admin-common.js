// Shared helpers for the admin web UI.
function getToken() {
  return sessionStorage.getItem('adminToken');
}

async function api(method, path, body) {
  const opts = {
    method,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + getToken()
    }
  };
  if (body !== undefined) {
    opts.body = JSON.stringify(body);
  }

  const resp = await fetch(path, opts);
  if (resp.status === 401) {
    logout();
    return null;
  }
  return resp;
}

function toast(msg, type = 'success') {
  const el = document.createElement('div');
  el.className = `toast toast-${type}`;
  el.textContent = msg;
  document.getElementById('toast-wrap').appendChild(el);
  // 增加成功提示的显示时间为 4 秒，错误提示为 5 秒
  setTimeout(() => el.remove(), type === 'success' ? 4000 : 5000);
}

function formatDate(iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  return d.toLocaleDateString('zh-CN') + ' ' + d.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit'
  });
}

function formatBytes(b) {
  if (!b) return '0 B';
  if (b < 1024) return b + ' B';
  if (b < 1048576) return (b / 1024).toFixed(1) + ' KB';
  return (b / 1048576).toFixed(1) + ' MB';
}

function toggleAddForm(id) {
  const form = document.getElementById(id);
  form.classList.toggle('show');
}

function closeModal(id) {
  document.getElementById(id).classList.remove('show');
}

function login() {
  const token = document.getElementById('token-input').value.trim();
  if (!token) {
    showLoginError('请输入管理员令牌');
    return;
  }

  const btn = document.getElementById('login-btn');
  btn.disabled = true;
  btn.textContent = '验证中...';

  fetch('/v1/admin/keys', {
    headers: { 'Authorization': 'Bearer ' + token }
  }).then(resp => {
    if (resp.ok) {
      sessionStorage.setItem('adminToken', token);
      showDashboard();
      return;
    }
    showLoginError('令牌无效，请检查配置');
  }).catch(() => {
    showLoginError('连接失败，请检查服务状态');
  }).finally(() => {
    btn.disabled = false;
    btn.textContent = '登录';
  });
}

function showLoginError(msg) {
  const el = document.getElementById('login-error');
  el.textContent = msg;
  el.style.display = 'block';
}

function logout() {
  sessionStorage.removeItem('adminToken');
  document.getElementById('dashboard').style.display = 'none';
  document.getElementById('login-page').style.display = 'flex';
  document.getElementById('token-input').value = '';
  document.getElementById('login-error').style.display = 'none';
}

function showDashboard() {
  document.getElementById('login-page').style.display = 'none';
  document.getElementById('dashboard').style.display = 'block';
  document.getElementById('topbar-host').textContent = window.location.host;
  loadProjectKeys();
  loadStats();
  loadAdminTokenSettings();
}

function switchTab(name, btn) {
  document.querySelectorAll('.tab-panel').forEach(panel => panel.classList.remove('active'));
  document.querySelectorAll('.tab-btn').forEach(tabBtn => tabBtn.classList.remove('active'));
  document.getElementById('tab-' + name).classList.add('active');
  btn.classList.add('active');

  if (name === 'keys') {
    loadProjectKeys();
    loadStats();
    resetProjectLogFilters();
  }
  if (name === 'anthropic') loadAnthropicKeys();
  if (name === 'openai') loadProviderKeys('openai');
  if (name === 'grok') loadProviderKeys('grok');
  if (name === 'models') loadModels();
}

function confirmDelete(apiPath, label, onSuccess) {
  document.getElementById('delete-modal-msg').textContent = '确定要删除“' + label + '”吗？此操作不可恢复。';
  document.getElementById('delete-modal').classList.add('show');
  document.getElementById('delete-confirm-btn').onclick = async function () {
    closeModal('delete-modal');
    const resp = await api('DELETE', apiPath);
    if (resp && resp.ok) {
      toast('已删除');
      onSuccess();
      return;
    }
    toast('删除失败', 'error');
  };
}

var projectKeysData = [];
var activeProjectKey = null;
var statsProjectIds = [];
var currentProjectLogs = [];
var akData = [];
var providerKeyData = { openai: [], grok: [] };

document.addEventListener('DOMContentLoaded', function () {
  document.getElementById('token-input').addEventListener('keydown', function (e) {
    if (e.key === 'Enter') login();
  });

  const token = getToken();
  if (token) {
    showDashboard();
  }
});
