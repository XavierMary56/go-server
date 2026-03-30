// Admin settings for the admin web UI.
function openChangePwdModal() {
  document.getElementById('admin-token-new').value = '';
  document.getElementById('admin-token-confirm').value = '';
  document.getElementById('change-pwd-modal').classList.add('show');
  loadAdminTokenSettings();
}

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
    // 优先展示后端详细错误
    toast((json && (json.error || json.message)) || '管理员密码更新失败', 'error');
    return;
  }

  sessionStorage.setItem('adminToken', newToken);
  document.getElementById('admin-token-new').value = '';
  document.getElementById('admin-token-confirm').value = '';
  closeModal('change-pwd-modal');
  // 先刷新设置区域内容，再弹出提示
  await loadAdminTokenSettings();
  toast(json.message || '管理员密码已更新');
}
