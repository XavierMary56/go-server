// Admin settings for the admin web UI.
async function loadAdminTokenSettings() {
  const resp = await api('GET', '/v1/admin/settings/admin-token');
  if (!resp) return;

  const json = await resp.json();
  const data = json.data || {};
  document.getElementById('admin-token-source').value = data.source === 'database' ? '数据库' : '环境变量';
  document.getElementById('admin-token-configured').value = data.configured ? '已配置' : '未配置';
  document.getElementById('admin-token-updated-at').value = formatDate(data.updated_at);
}

async function updateAdminTokenSettings() {
  const newToken = document.getElementById('admin-token-new').value.trim();
  const confirmToken = document.getElementById('admin-token-confirm').value.trim();
  if (!newToken || !confirmToken) {
    toast('新令牌和确认令牌不能为空', 'error');
    return;
  }
  if (newToken !== confirmToken) {
    toast('两次输入的令牌不一致', 'error');
    return;
  }

  const resp = await api('PUT', '/v1/admin/settings/admin-token', {
    new_token: newToken,
    confirm_token: confirmToken
  });
  if (!resp) return;

  const json = await resp.json();
  if (!resp.ok) {
    toast(json.error || '管理员令牌更新失败', 'error');
    return;
  }

  sessionStorage.setItem('adminToken', newToken);
  document.getElementById('admin-token-new').value = '';
  document.getElementById('admin-token-confirm').value = '';
  toast(json.message || '管理员令牌已更新');
  loadAdminTokenSettings();
}
