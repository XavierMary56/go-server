// Provider key and model management for the admin web UI.
async function loadAnthropicKeys() {
  document.getElementById('ak-tbody').innerHTML = '<tr class="empty-row"><td colspan="7"><span class="spinner"></span> 加载中...</td></tr>';
  const resp = await api('GET', '/v1/admin/anthropic-keys');
  if (!resp) return;
  const json = await resp.json();
  akData = json.data || [];
  renderAnthropicKeys();
}

function statusBadge(status) {
  const map = {
    healthy: 'badge-valid',
    unhealthy: 'badge-invalid',
    unknown: 'badge-checking'
  };
  const label = {
    healthy: '✓ 正常',
    unhealthy: '✗ 不可用',
    unknown: '? 未检测'
  };
  return `<span class="badge ${map[status] || 'badge-checking'}">${label[status] || status}</span>`;
}

function renderAnthropicKeys() {
  const tbody = document.getElementById('ak-tbody');
  if (!akData.length) {
    tbody.innerHTML = '<tr class="empty-row"><td colspan="7">暂无密钥，点击右上角添加</td></tr>';
    return;
  }

  tbody.innerHTML = akData.map(function (k) {
    return `
    <tr id="ak-row-${k.id}">
      <td><strong>${k.name}</strong></td>
      <td><span class="key-cell">${k.key_masked}</span></td>
      <td><span class="usage-num">${(k.usage_count || 0).toLocaleString()}</span></td>
      <td>${formatDate(k.last_used_at)}</td>
      <td id="ak-health-${k.id}">${statusBadge(k.status || 'unknown')}<br><small style="color:#94a3b8">${k.checked_at ? formatDate(k.checked_at) : ''}</small></td>
      <td><span class="badge ${k.enabled ? 'badge-active' : 'badge-inactive'}">${k.enabled ? '已启用' : '已停用'}</span></td>
      <td><div class="actions">
        <button class="btn btn-sm btn-info" onclick="checkAK(${k.id})">检测</button>
        <button class="btn btn-sm btn-primary" onclick="editProviderKeyModal('anthropic', ${k.id}, ${JSON.stringify(k.name)})">编辑</button>
        <button class="btn btn-sm ${k.enabled ? 'btn-warning' : 'btn-success'}" onclick="toggleAK(${k.id}, ${!k.enabled})">${k.enabled ? '停用' : '启用'}</button>
        <button class="btn btn-sm btn-danger" onclick="confirmDelete('/v1/admin/anthropic-keys/${k.id}', '${k.name} 的密钥', loadAnthropicKeys)">删除</button>
      </div></td>
    </tr>`;
  }).join('');
}

async function addAnthropicKey() {
  const name = document.getElementById('new-ak-name').value.trim();
  const key = document.getElementById('new-ak-key').value.trim();
  if (!name || !key) {
    toast('名称和密钥不能为空', 'error');
    return;
  }

  const resp = await api('POST', '/v1/admin/anthropic-keys', { name, key });
  if (!resp) return;

  if (resp.status === 201) {
    toast('密钥已添加');
    ['new-ak-name', 'new-ak-key'].forEach(function (id) {
      document.getElementById(id).value = '';
    });
    document.getElementById('add-ak-form').classList.remove('show');
    loadAnthropicKeys();
    return;
  }

  const json = await resp.json();
  toast(json.error || '添加失败', 'error');
}

async function checkAK(id) {
  const row = akData.find(function (k) { return k.id === id; });
  const cell = document.getElementById('ak-health-' + id);
  cell.innerHTML = '<span class="badge badge-checking"><span class="spinner"></span> 检测中...</span>';

  const resp = await api('POST', '/v1/admin/anthropic-keys/check', { id });
  if (!resp) return;

  const json = await resp.json();
  const result = json.data || {};
  cell.innerHTML = statusBadge(result.status || 'unknown');
  toast(
    result.status === 'healthy' ? `"${row.name}" 检测正常` : `"${row.name}" 不可用：${result.error || ''}`,
    result.status === 'healthy' ? 'success' : 'error'
  );
  loadAnthropicKeys();
}

async function checkAllKeys() {
  toast('正在检测所有密钥，请稍候...');
  const resp = await api('POST', '/v1/admin/keys/check-all', {});
  if (!resp) return;

  const json = await resp.json();
  const results = json.data || [];
  const healthyCount = results.filter(function (result) {
    return result.status === 'healthy';
  }).length;
  toast(`检测完成：${healthyCount}/${results.length} 个密钥正常`, healthyCount === results.length ? 'success' : 'error');
  loadAnthropicKeys();
  if (document.getElementById('tab-openai').classList.contains('active')) loadProviderKeys('openai');
  if (document.getElementById('tab-grok').classList.contains('active')) loadProviderKeys('grok');
}

async function toggleAK(id, enable) {
  const resp = await api('PUT', '/v1/admin/anthropic-keys/' + id, { enabled: enable });
  if (resp && resp.ok) {
    toast(enable ? '已启用' : '已停用');
    loadAnthropicKeys();
    return;
  }
  toast('操作失败', 'error');
}

var editProviderKeyState = { provider: '', id: 0 };

function editProviderKeyModal(provider, id, name) {
  editProviderKeyState = { provider, id };
  document.getElementById('edit-provider-key-name').value = name || '';
  const title = { anthropic: 'Anthropic', openai: 'OpenAI', grok: 'Grok' }[provider] || provider;
  document.getElementById('edit-provider-key-title').textContent = '编辑 ' + title + ' 密钥备注';
  document.getElementById('edit-provider-key-modal').classList.add('show');
}

async function saveProviderKeyName() {
  const { provider, id } = editProviderKeyState;
  const name = document.getElementById('edit-provider-key-name').value.trim();
  if (!name) { toast('备注名称不能为空', 'error'); return; }
  const url = provider === 'anthropic'
    ? '/v1/admin/anthropic-keys/' + id
    : '/v1/admin/provider-keys/' + id;
  const resp = await api('PUT', url, { name });
  if (resp && resp.ok) {
    toast('备注已更新');
    closeModal('edit-provider-key-modal');
    if (provider === 'anthropic') loadAnthropicKeys();
    else loadProviderKeys(provider);
    return;
  }
  toast('更新失败', 'error');
}

async function loadProviderKeys(provider) {
  const tbodyId = provider === 'openai' ? 'oai-tbody' : 'grok-tbody';
  document.getElementById(tbodyId).innerHTML = '<tr class="empty-row"><td colspan="7"><span class="spinner"></span> 加载中...</td></tr>';
  const resp = await api('GET', '/v1/admin/provider-keys?provider=' + provider);
  if (!resp) return;
  const json = await resp.json();
  providerKeyData[provider] = json.data || [];
  renderProviderKeys(provider);
}

function renderProviderKeys(provider) {
  const tbodyId = provider === 'openai' ? 'oai-tbody' : 'grok-tbody';
  const tbody = document.getElementById(tbodyId);
  const data = providerKeyData[provider] || [];
  if (!data.length) {
    tbody.innerHTML = '<tr class="empty-row"><td colspan="7">暂无密钥，点击右上角添加</td></tr>';
    return;
  }

  tbody.innerHTML = data.map(function (k) {
    return `
    <tr>
      <td><strong>${k.name}</strong></td>
      <td><span class="key-cell">${k.key_masked}</span></td>
      <td><span class="usage-num">${(k.usage_count || 0).toLocaleString()}</span></td>
      <td>${formatDate(k.last_used_at)}</td>
      <td id="pk-health-${k.id}">${statusBadge(k.status || 'unknown')}<br><small style="color:#94a3b8">${k.checked_at ? formatDate(k.checked_at) : ''}</small></td>
      <td><span class="badge ${k.enabled ? 'badge-active' : 'badge-inactive'}">${k.enabled ? '已启用' : '已停用'}</span></td>
      <td><div class="actions">
        <button class="btn btn-sm btn-info" onclick="checkPK('${provider}', ${k.id})">检测</button>
        <button class="btn btn-sm btn-primary" onclick="editProviderKeyModal('${provider}', ${k.id}, ${JSON.stringify(k.name)})">编辑</button>
        <button class="btn btn-sm ${k.enabled ? 'btn-warning' : 'btn-success'}" onclick="toggleProviderKey('${provider}', ${k.id}, ${!k.enabled})">${k.enabled ? '停用' : '启用'}</button>
        <button class="btn btn-sm btn-danger" onclick="confirmDelete('/v1/admin/provider-keys/${k.id}', '${k.name} 的密钥', () => loadProviderKeys('${provider}'))">删除</button>
      </div></td>
    </tr>`;
  }).join('');
}

function formatProviderName(provider) {
  return {
    anthropic: 'Anthropic（anthropic）',
    openai: 'OpenAI（openai）',
    grok: 'Grok（grok）'
  }[provider] || provider || '-';
}

async function checkPK(provider, id) {
  const data = providerKeyData[provider] || [];
  const row = data.find(function (k) { return k.id === id; });
  const cell = document.getElementById('pk-health-' + id);
  if (cell) {
    cell.innerHTML = '<span class="badge badge-checking"><span class="spinner"></span> 检测中...</span>';
  }

  const resp = await api('POST', '/v1/admin/provider-keys/check', { id });
  if (!resp) return;

  const json = await resp.json();
  const result = json.data || {};
  if (cell) {
    cell.innerHTML = statusBadge(result.status || 'unknown');
  }
  toast(
    result.status === 'healthy' ? `"${row && row.name || id}" 检测正常` : `"${row && row.name || id}" 不可用：${result.error || ''}`,
    result.status === 'healthy' ? 'success' : 'error'
  );
  loadProviderKeys(provider);
}

async function addProviderKey(provider) {
  const nameId = provider === 'openai' ? 'new-oai-name' : 'new-grok-name';
  const keyId = provider === 'openai' ? 'new-oai-key' : 'new-grok-key';
  const formId = provider === 'openai' ? 'add-oai-form' : 'add-grok-form';
  const name = document.getElementById(nameId).value.trim();
  const key = document.getElementById(keyId).value.trim();
  if (!name || !key) {
    toast('名称和密钥不能为空', 'error');
    return;
  }

  const resp = await api('POST', '/v1/admin/provider-keys', { provider, name, key });
  if (!resp) return;

  if (resp.status === 201) {
    toast('密钥已添加');
    document.getElementById(nameId).value = '';
    document.getElementById(keyId).value = '';
    document.getElementById(formId).classList.remove('show');
    loadProviderKeys(provider);
    return;
  }

  const json = await resp.json();
  toast(json.error || '添加失败', 'error');
}

async function toggleProviderKey(provider, id, enable) {
  const resp = await api('PUT', '/v1/admin/provider-keys/' + id, { enabled: enable });
  if (resp && resp.ok) {
    toast(enable ? '已启用' : '已停用');
    loadProviderKeys(provider);
    return;
  }
  toast('操作失败', 'error');
}

async function loadModels() {
  document.getElementById('models-tbody').innerHTML = '<tr class="empty-row"><td colspan="7"><span class="spinner"></span> 加载中...</td></tr>';
  const resp = await api('GET', '/v1/admin/models');
  if (!resp) return;
  const json = await resp.json();
  const models = json.data || [];
  const tbody = document.getElementById('models-tbody');
  if (!models.length) {
    tbody.innerHTML = '<tr class="empty-row"><td colspan="7">暂无模型配置，请点击右上角添加</td></tr>';
    return;
  }

  const maxWeight = Math.max.apply(null, models.map(function (m) { return m.weight || 1; }).concat([1]));
  tbody.innerHTML = models.map(function (m) {
    const provider = m.provider || inferProvider(m.model_id);
    const providerBadge = {
      anthropic: 'badge-active',
      openai: 'badge-checking',
      grok: 'badge-valid'
    }[provider] || 'badge-inactive';
    const isFallback = m.source === 'config-fallback';
    const sourceHint = isFallback ? '<br><small style="color:#94a3b8">配置回退</small>' : '';
    const actions = isFallback
      ? '<span style="color:#94a3b8;font-size:12px;">请先在后台保存为正式模型</span>'
      : `<div class="actions">
        <button class="btn btn-sm ${m.enabled ? 'btn-warning' : 'btn-success'}" onclick="updateModel(${m.id}, { enabled: ${!m.enabled} }).then(() => loadModels())">${m.enabled ? '停用' : '启用'}</button>
        <button class="btn btn-sm btn-danger" onclick="confirmDelete('/v1/admin/models/${m.id}', '模型 ${m.name}', loadModels)">删除</button>
      </div>`;
    return `
    <tr>
      <td><span class="model-id-cell">${m.model_id}</span>${sourceHint}</td>
      <td>${m.name}</td>
      <td><span class="badge ${providerBadge}">${formatProviderName(provider)}</span></td>
      <td>
        <div class="weight-bar">
          <input class="inline-num" type="number" value="${m.weight}" min="1" max="100" ${isFallback ? 'disabled' : ''} onchange="updateModel(${m.id}, { weight: parseInt(this.value) })" />
          <div class="weight-bar-inner"><div class="weight-bar-fill" style="width:${Math.round(m.weight / maxWeight * 100)}%"></div></div>
          <span style="font-size:12px;color:#94a3b8">${m.weight}</span>
        </div>
      </td>
      <td>
        <input class="inline-num" type="number" value="${m.priority}" min="1" ${isFallback ? 'disabled' : ''} onchange="updateModel(${m.id}, { priority: parseInt(this.value) })" />
      </td>
      <td><span class="badge ${m.enabled ? 'badge-active' : 'badge-inactive'}">${m.enabled ? '已启用' : '已停用'}</span></td>
      <td>${actions}</td>
    </tr>`;
  }).join('');
}

function inferProvider(modelId) {
  if (!modelId) return 'anthropic';
  if (/^(gpt-|o1-|o3-|o4-)/.test(modelId)) return 'openai';
  if (/^grok-/.test(modelId)) return 'grok';
  return 'anthropic';
}

function inferModelProvider() {
  const id = document.getElementById('new-model-id').value.trim();
  document.getElementById('new-model-provider').value = inferProvider(id);
}

async function addModel() {
  const modelId = document.getElementById('new-model-id').value.trim();
  const name = document.getElementById('new-model-name').value.trim();
  const provider = document.getElementById('new-model-provider').value;
  const weight = parseInt(document.getElementById('new-model-weight').value) || 50;
  const priority = parseInt(document.getElementById('new-model-priority').value) || 1;
  if (!modelId || !name) {
    toast('模型 ID 和名称不能为空', 'error');
    return;
  }
  if (modelId.length < 3 || /\s/.test(modelId)) {
    toast('模型 ID 格式无效，请填写真实模型名', 'error');
    return;
  }
  if (provider === 'openai' && !/^(gpt-|o1-|o3-|o4-)/.test(modelId)) {
    toast('OpenAI 模型 ID 格式无效', 'error');
    return;
  }
  if (provider === 'grok' && !/^grok-/.test(modelId)) {
    toast('Grok 模型 ID 格式无效', 'error');
    return;
  }
  if (provider === 'anthropic' && !/^claude-/.test(modelId)) {
    toast('Anthropic 模型 ID 格式无效', 'error');
    return;
  }

  const resp = await api('POST', '/v1/admin/models', {
    model_id: modelId,
    name,
    provider,
    weight,
    priority,
    enabled: true
  });
  if (!resp) return;

  if (resp.status === 201) {
    toast('模型已添加');
    ['new-model-id', 'new-model-name'].forEach(function (id) {
      document.getElementById(id).value = '';
    });
    document.getElementById('add-model-form').classList.remove('show');
    loadModels();
    return;
  }

  const json = await resp.json();
  toast(json.error || '添加失败', 'error');
}

async function updateModel(id, data) {
  const resp = await api('PUT', '/v1/admin/models/' + id, data);
  if (resp && resp.ok) {
    toast('模型已更新');
    loadModels();
    return;
  }
  toast('模型更新失败', 'error');
}
