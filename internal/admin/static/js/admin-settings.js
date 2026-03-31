// Admin settings for the admin web UI.
function openChangePwdModal() {
  document.getElementById('admin-token-new').value = '';
  document.getElementById('admin-token-confirm').value = '';
  document.getElementById('change-pwd-modal').classList.add('show');
  loadAdminTokenSettings();
}

function openSettingsModal() {
  document.getElementById('settings-modal').classList.add('show');
  loadStaticVersionSettings();
}

async function loadAdminTokenSettings() {
  const resp = await api('GET', '/v1/admin/settings/admin-token');
  if (!resp) return;

  const json = await resp.json();
  const data = json.data || {};
  const elSource = document.getElementById('admin-token-source');
  const elConfigured = document.getElementById('admin-token-configured');
  const elUpdatedAt = document.getElementById('admin-token-updated-at');
  if (elSource) elSource.value = data.source === 'database' ? '数据库' : '环境变量';
  if (elConfigured) elConfigured.value = data.configured ? '已配置' : '未配置';
  if (elUpdatedAt) elUpdatedAt.value = formatDate(data.updated_at);
}

async function updateAdminTokenSettings() {
  const newToken = document.getElementById('admin-token-new').value.trim();
  const confirmToken = document.getElementById('admin-token-confirm').value.trim();
  if (!newToken || !confirmToken) {
    toast('新密码和确认密码不能为空', 'error');
    return;
  }
  if (newToken !== confirmToken) {
    toast('两次输入的密码不一致', 'error');
    return;
  }

  const resp = await api('PUT', '/v1/admin/settings/admin-token', {
    new_token: newToken,
    confirm_token: confirmToken
  });
  if (!resp) {
    toast('保存失败，请联系管理员检查数据库配置', 'error');
    return;
  }

  const json = await resp.json();
  if (!resp.ok) {
    toast((json && (json.error || json.message)) || '管理员密码更新失败', 'error');
    return;
  }

  sessionStorage.setItem('adminToken', newToken);
  document.getElementById('admin-token-new').value = '';
  document.getElementById('admin-token-confirm').value = '';
  toast(json.message || '管理员密码已更新');
}

async function loadStaticVersionSettings() {
  const resp = await api('GET', '/v1/admin/settings/static-version');
  if (!resp) return;

  const json = await resp.json();
  const data = json.data || {};
  const elCurrent = document.getElementById('static-version-current');
  const elUpdatedAt = document.getElementById('static-version-updated-at');
  if (elCurrent) elCurrent.value = data.version || window.STATIC_VERSION || '-';
  if (elUpdatedAt) elUpdatedAt.value = formatDate(data.updated_at);
}

async function updateStaticVersionSettings() {
  var newVersion = document.getElementById('static-version-new').value.trim();
  if (!newVersion) {
    // 默认使用当前时间戳 yyyyMMddHHmm
    var now = new Date();
    var pad = function(n) { return n < 10 ? '0' + n : '' + n; };
    newVersion = '' + now.getFullYear() + pad(now.getMonth() + 1) + pad(now.getDate()) + pad(now.getHours()) + pad(now.getMinutes());
  }

  const resp = await api('PUT', '/v1/admin/settings/static-version', { version: newVersion });
  if (!resp) {
    toast('保存失败', 'error');
    return;
  }

  const json = await resp.json();
  if (!resp.ok) {
    toast((json && (json.error || json.message)) || '版本号更新失败', 'error');
    return;
  }

  window.STATIC_VERSION = newVersion;
  document.getElementById('static-version-new').value = '';
  await loadStaticVersionSettings();
  toast((json.message || '版本号已更新') + '，刷新页面后生效');
}
