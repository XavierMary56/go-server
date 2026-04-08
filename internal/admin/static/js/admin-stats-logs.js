// Project stats and audit log management for the admin web UI.
var logsPageSize = 20;
var currentLogsPage = 1;
var statsPageSize = 10;
var currentStatsPage = 1;
var statsEntries = [];

function renderProjectLogs() {
  var tbody = document.getElementById('project-logs-tbody');
  if (!currentProjectLogs.length) {
    tbody.innerHTML = '<tr class="empty-row"><td colspan="7">当前筛选条件下暂无日志</td></tr>';
    document.getElementById('logs-pagination').innerHTML = '';
    return;
  }
  var total = currentProjectLogs.length;
  var totalPages = Math.ceil(total / logsPageSize);
  if (currentLogsPage > totalPages) currentLogsPage = totalPages;
  if (currentLogsPage < 1) currentLogsPage = 1;
  var start = (currentLogsPage - 1) * logsPageSize;
  var pageItems = currentProjectLogs.slice(start, start + logsPageSize);

  tbody.innerHTML = pageItems.map(function (event, i) {
    var index = start + i;
    var summary = formatLogSummary(event);
    var time = escapeHtml(formatDate(event.timestamp || event.ts));
    var projectIdText = escapeHtml(event.project_name || '-');
    var eventTypeText = escapeHtml(event.event_type || '-');
    var clientIpText = escapeHtml(event.ip_address || event.client_ip || '-');
    var resultText = escapeHtml(formatLogResult(event));
    var summaryText = escapeHtml(summary);
    return '<tr>' +
      '<td>' + time + '</td>' +
      '<td>' + projectIdText + '</td>' +
      '<td>' + eventTypeText + '</td>' +
      '<td>' + clientIpText + '</td>' +
      '<td>' + resultText + '</td>' +
      '<td title="' + summaryText + '" style="max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">' + summaryText + '</td>' +
      '<td><button class="btn btn-sm btn-ghost" onclick="openLogDetailModal(' + index + ')">查看详情</button></td>' +
      '</tr>';
  }).join('');

  renderPagination('logs-pagination', total, totalPages, currentLogsPage, 'logsGoPage', 'logsChangePageSize', logsPageSize);
}

function logsGoPage(p) {
  currentLogsPage = p;
  renderProjectLogs();
}

function logsChangePageSize(size) {
  logsPageSize = size;
  currentLogsPage = 1;
  renderProjectLogs();
}

async function loadStats() {
  const statsTbody = document.getElementById('stats-tbody');
  if (statsTbody) statsTbody.innerHTML = '<tr class="empty-row"><td colspan="7"><span class="spinner"></span> 加载中...</td></tr>';
  const responses = await Promise.all([
    api('GET', '/v1/admin/projects/stats'),
    api('GET', '/v1/admin/projects')
  ]);
  const statsResp = responses[0];
  const projectsResp = responses[1];
  if (!statsResp || !projectsResp) return;

  const stats = (await statsResp.json()).data || {};
  const projectsData = (await projectsResp.json()).data || {};
  statsProjectIds = (projectsData.projects || []).map(function (project) {
    return project.project_name;
  }).filter(Boolean);
  syncProjectLogProjectOptions();

  // 区分有密钥和仅日志的项目
  var keyProjectNames = {};
  projectKeysData.forEach(function (k) { if (k.project_name) keyProjectNames[k.project_name] = true; });
  var allProjectIds = Object.keys(stats);
  var keyedCount = 0;
  var logOnlyCount = 0;
  allProjectIds.forEach(function (pid) {
    if (keyProjectNames[pid]) { keyedCount++; } else { logOnlyCount++; }
  });

  var totalCalls = 0;
  var totalAuth = 0;
  var totalRateLimited = 0;
  Object.values(stats).forEach(function (item) {
    totalCalls += item.api_calls || 0;
    totalAuth += item.auth_attempts || 0;
    totalRateLimited += item.rate_limited || 0;
  });

  var statsSummary = document.getElementById('stats-summary');
  if (statsSummary) statsSummary.innerHTML =
    '<div class="stat-card"><div class="label">有密钥项目</div><div class="value">' + keyedCount + '</div><div class="sub">已配置密钥</div></div>' +
    '<div class="stat-card"><div class="label">仅日志项目</div><div class="value">' + logOnlyCount + '</div><div class="sub">仅有审计日志（历史/已移除）</div></div>' +
    '<div class="stat-card"><div class="label">API 调用</div><div class="value">' + totalCalls.toLocaleString() + '</div><div class="sub">累计请求次数</div></div>' +
    '<div class="stat-card"><div class="label">认证次数</div><div class="value">' + totalAuth.toLocaleString() + '</div><div class="sub">鉴权请求总量</div></div>' +
    '<div class="stat-card"><div class="label">限流次数</div><div class="value">' + totalRateLimited.toLocaleString() + '</div><div class="sub">触发速率限制</div></div>';

  const tbody = document.getElementById('stats-tbody');
  const entries = Object.entries(stats).sort(function (a, b) {
    return (b[1].api_calls || 0) - (a[1].api_calls || 0) ||
      (b[1].auth_attempts || 0) - (a[1].auth_attempts || 0) ||
      a[0].localeCompare(b[0]);
  });

  if (!entries.length) {
    if (tbody) tbody.innerHTML = '<tr class="empty-row"><td colspan="7">暂无统计数据</td></tr>';
    return;
  }

  statsEntries = entries;
  currentStatsPage = 1;
  renderStatsPage();
  resetProjectLogFilters();
}

function renderStatsPage() {
  var tbody = document.getElementById('stats-tbody');
  if (!tbody || !statsEntries.length) return;
  var total = statsEntries.length;
  var totalPages = Math.ceil(total / statsPageSize);
  if (currentStatsPage > totalPages) currentStatsPage = totalPages;
  if (currentStatsPage < 1) currentStatsPage = 1;
  var start = (currentStatsPage - 1) * statsPageSize;
  var pageItems = statsEntries.slice(start, start + statsPageSize);

  // 构建密钥项目名集合用于标识
  var keyNames = {};
  projectKeysData.forEach(function (k) { if (k.project_name) keyNames[k.project_name] = true; });

  tbody.innerHTML = pageItems.map(function (entry) {
    var pid = entry[0];
    var s = entry[1];
    var hasKey = keyNames[pid];
    var badge = hasKey
      ? '<span class="badge badge-active" style="font-size:11px;margin-left:6px;">有密钥</span>'
      : '<span class="badge badge-inactive" style="font-size:11px;margin-left:6px;">仅日志</span>';
    return '<tr>' +
      '<td><strong>' + escapeHtml(pid) + '</strong>' + badge + '</td>' +
      '<td>' + (s.api_calls || 0).toLocaleString() + '</td>' +
      '<td>' + (s.auth_attempts || 0).toLocaleString() + '</td>' +
      '<td>' + (s.rate_limited || 0).toLocaleString() + '</td>' +
      '<td>' + (s.errors || 0).toLocaleString() + '</td>' +
      '<td>' + formatBytes(s.log_size || 0) + '</td>' +
      '<td><button class="btn btn-sm btn-info" onclick="focusProjectLogs(\'' + escapeHtml(pid) + '\')">查看日志</button></td>' +
      '</tr>';
  }).join('');

  renderPagination('stats-pagination', total, totalPages, currentStatsPage, 'statsGoPage', 'statsChangePageSize', statsPageSize);
}

function statsGoPage(p) {
  currentStatsPage = p;
  renderStatsPage();
}

function statsChangePageSize(size) {
  statsPageSize = size;
  currentStatsPage = 1;
  renderStatsPage();
}

function resetProjectLogFilters() {
  const today = new Date();
  const start = new Date(today);
  start.setDate(today.getDate() - 7);
  const formatDay = function (d) {
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  };

  const startEl = document.getElementById('log-start');
  const endEl = document.getElementById('log-end');
  const typeEl = document.getElementById('log-type');
  if (startEl) startEl.value = formatDay(start);
  if (endEl) endEl.value = formatDay(today);
  if (typeEl) typeEl.value = 'moderation_request';
  syncProjectLogProjectOptions();
}

function syncProjectLogProjectOptions() {
  const select = document.getElementById('log-project');
  if (!select) return;

  const current = select.value;
  const options = ['<option value="">请选择项目</option>'].concat(
    statsProjectIds.map(function (pid) {
      return `<option value="${pid}">${pid}</option>`;
    })
  );
  select.innerHTML = options.join('');
  if (statsProjectIds.includes(current)) {
    select.value = current;
  } else if (!select.value && statsProjectIds.length === 1) {
    select.value = statsProjectIds[0];
  }
}

function focusProjectLogs(projectId) {
  // 切换到项目日志标签
  document.querySelectorAll('.tab-panel').forEach(function (p) { p.classList.remove('active'); });
  document.querySelectorAll('.tab-btn').forEach(function (b) { b.classList.remove('active'); });
  document.getElementById('tab-logs').classList.add('active');
  var btns = document.querySelectorAll('.tab-btn');
  btns.forEach(function (b) { if (b.textContent.trim() === '项目日志') b.classList.add('active'); });

  // 确保下拉框有选项
  syncProjectLogProjectOptions();
  resetProjectLogFilters();

  var select = document.getElementById('log-project');
  if (select) select.value = projectId;
  loadProjectLogs();
}

function formatLogResult(event) {
  // 优先从 metadata 中读取审核结果
  var meta = event.metadata || {};
  if (meta.verdict) {
    if (meta.verdict === 'approved') return '✅ 通过';
    if (meta.verdict === 'rejected' || meta.verdict === 'flagged') return '❌ 拒绝';
    return meta.verdict;
  }
  // 兼容旧格式
  var details = event.details || {};
  if (Object.prototype.hasOwnProperty.call(details, 'ok')) return details.ok ? '成功' : '失败';
  if (event.event_type === 'rate_limit_exceeded') return '已限流';
  if (event.event_type === 'config_change') return '已记录';
  if (event.event_type === 'admin_auth_failed') return '❌ 失败';
  if (event.status_code) {
    return event.status_code >= 200 && event.status_code < 300 ? '成功' : '失败(' + event.status_code + ')';
  }
  return '-';
}

function formatLogSummary(event) {
  var parts = [];
  var meta = event.metadata || {};
  var reqBody = event.request_body || {};

  // 审核请求：显示内容摘要 + 结果
  if (event.event_type === 'moderation_request') {
    if (reqBody.content) {
      var content = String(reqBody.content);
      parts.push('内容: ' + (content.length > 40 ? content.substring(0, 40) + '...' : content));
    }
    if (meta.verdict) parts.push('结果: ' + meta.verdict);
    if (meta.category && meta.category !== 'clean') parts.push('分类: ' + meta.category);
    if (meta.model_used) parts.push('模型: ' + meta.model_used);
    if (meta.from_cache) parts.push('(缓存)');
    if (parts.length) return parts.join(' | ');
  }

  // 通用：显示路径和方法
  if (event.path) parts.push(event.method + ' ' + event.path);
  if (event.status_code) parts.push('状态: ' + event.status_code);
  if (event.error_msg) parts.push('错误: ' + event.error_msg);
  if (parts.length) return parts.join(' | ');

  // 兼容旧 details 格式
  return formatLogDetailsLegacy(event.details);
}

function formatLogDetailsLegacy(details) {
  if (!details || typeof details !== 'object') return '-';

  const parts = [];
  if (details.path) parts.push('路径: ' + details.path);
  if (details.method) parts.push('方法: ' + details.method);
  if (details.status_code) parts.push('状态码: ' + details.status_code);
  if (details.key_name) parts.push('键名: ' + details.key_name);
  if (details.config_type) parts.push('配置类型: ' + details.config_type);
  if (details.change_type) parts.push('变更: ' + details.change_type);
  if (details.reason) parts.push('原因: ' + details.reason);
  if (parts.length) return parts.join(' | ');

  try {
    return JSON.stringify(details);
  } catch (e) {
    return '-';
  }
}

function openLogDetailModal(index) {
  var event = currentProjectLogs[index];
  if (!event) {
    toast('日志详情不存在', 'error');
    return;
  }

  var meta = event.metadata || {};
  var reqBody = event.request_body || {};
  var eventType = event.event_type || '';

  // 事件类型中文映射
  var typeLabels = {
    'moderation_request': '审核请求',
    'api_call': 'API 调用',
    'auth_attempt': '认证请求',
    'rate_limit_exceeded': '限流触发',
    'config_change': '配置变更',
    'admin_auth_failed': '管理认证失败'
  };
  var subtitleLabels = {
    'moderation_request': '审核请求的完整信息',
    'api_call': 'API 调用记录',
    'auth_attempt': '认证请求记录',
    'rate_limit_exceeded': '限流触发记录',
    'config_change': '配置变更记录'
  };

  // 更新标题和副标题
  var titleEl = document.getElementById('log-detail-title');
  var subtitleEl = titleEl ? titleEl.nextElementSibling : null;
  if (titleEl) titleEl.textContent = typeLabels[eventType] || '日志详情';
  if (subtitleEl) subtitleEl.textContent = subtitleLabels[eventType] || '事件详细信息';

  // 基础信息
  document.getElementById('log-detail-project').value = event.project_name || '-';
  document.getElementById('log-detail-type').value = typeLabels[eventType] || eventType || '-';
  document.getElementById('log-detail-time').value = formatDate(event.timestamp || event.ts);
  document.getElementById('log-detail-ip').value = event.ip_address || event.client_ip || '-';
  document.getElementById('log-detail-path').value = (event.method || '') + ' ' + (event.path || '-');
  document.getElementById('log-detail-latency').value = event.latency_ms ? (event.latency_ms + 'ms') : (meta.latency_ms ? (meta.latency_ms + 'ms') : '-');

  // 隐藏所有动态区块
  var sections = ['log-section-request', 'log-section-result', 'log-section-reply', 'log-section-apicall', 'log-section-auth'];
  sections.forEach(function(id) {
    var el = document.getElementById(id);
    if (el) el.style.display = 'none';
  });

  // 根据事件类型显示对应区块
  if (eventType === 'moderation_request') {
    // 请求内容区块
    document.getElementById('log-section-request').style.display = '';
    var contentEl = document.getElementById('log-detail-content');
    if (contentEl) contentEl.textContent = reqBody.content || '-';
    var reqParamsEl = document.getElementById('log-detail-req-params');
    if (reqParamsEl) {
      var params = [];
      if (reqBody.type) params.push('type: ' + reqBody.type);
      if (reqBody.model) params.push('model: ' + reqBody.model);
      if (reqBody.strictness) params.push('strictness: ' + reqBody.strictness);
      reqParamsEl.value = params.length ? params.join(', ') : '';
    }
    // 请求参数为空时隐藏
    var reqParamsRow = document.getElementById('log-row-req-params');
    if (reqParamsRow) reqParamsRow.style.display = reqParamsEl && reqParamsEl.value ? '' : 'none';

    // 审核结果区块
    document.getElementById('log-section-result').style.display = '';
    document.getElementById('log-detail-verdict').value = meta.verdict ? (meta.verdict + (meta.category && meta.category !== 'none' && meta.category !== 'clean' ? ' (' + meta.category + ')' : '')) : '-';
    document.getElementById('log-detail-confidence').value = meta.confidence != null ? (Math.round(meta.confidence * 100) + '%') : '-';
    document.getElementById('log-detail-model').value = meta.model_used || '-';
    document.getElementById('log-detail-cache').value = meta.from_cache ? '是' : '否';
    document.getElementById('log-detail-reason').value = meta.reason || '';
    // 有原因就显示
    var reasonRow = document.getElementById('log-row-reason');
    if (reasonRow) reasonRow.style.display = meta.reason ? '' : 'none';

    // 自动回复区块
    var replySection = document.getElementById('log-section-reply');
    var replyEl = document.getElementById('log-detail-reply');
    if (meta.reply_content) {
      replySection.style.display = '';
      replyEl.textContent = meta.reply_content;
    }

  } else if (eventType === 'api_call') {
    document.getElementById('log-section-apicall').style.display = '';
    document.getElementById('log-detail-status').value = event.status_code || '-';
    document.getElementById('log-detail-apikey').value = event.api_key || '-';
    var errorRow = document.getElementById('log-row-error');
    var errorEl = document.getElementById('log-detail-error');
    if (errorEl) errorEl.value = event.error_msg || '';
    if (errorRow) errorRow.style.display = event.error_msg ? '' : 'none';

  } else if (eventType === 'auth_attempt' || eventType === 'rate_limit_exceeded' || eventType === 'admin_auth_failed') {
    document.getElementById('log-section-auth').style.display = '';
    document.getElementById('log-detail-auth-key').value = event.api_key || '-';
    var authResult = '-';
    if (eventType === 'auth_attempt') {
      authResult = (meta.success === true) ? '✅ 认证成功' : '❌ 认证失败';
    } else if (eventType === 'rate_limit_exceeded') {
      authResult = '⚠️ 已限流';
    } else {
      authResult = '❌ 管理认证失败';
    }
    document.getElementById('log-detail-auth-result').value = authResult;
  }

  // 完整 JSON
  document.getElementById('log-detail-json').textContent = JSON.stringify(event, null, 2);
  document.getElementById('log-detail-modal').classList.add('show');
}

function copyLogJson() {
  var el = document.getElementById('log-detail-json');
  if (!el) return;
  var text = el.textContent || '';
  if (navigator.clipboard) {
    navigator.clipboard.writeText(text).then(function() {
      toast('JSON 已复制到剪贴板');
    }).catch(function() {
      fallbackCopyText(text);
    });
  } else {
    fallbackCopyText(text);
  }
}

function fallbackCopyText(text) {
  var ta = document.createElement('textarea');
  ta.value = text;
  ta.style.position = 'fixed';
  ta.style.opacity = '0';
  document.body.appendChild(ta);
  ta.select();
  try {
    document.execCommand('copy');
    toast('JSON 已复制到剪贴板');
  } catch (e) {
    toast('复制失败，请手动选择复制', 'error');
  }
  document.body.removeChild(ta);
}

async function loadProjectLogs() {
  const tbody = document.getElementById('project-logs-tbody');
  const projectId = document.getElementById('log-project') && document.getElementById('log-project').value || '';
  if (!projectId) {
    currentProjectLogs = [];
    tbody.innerHTML = '<tr class="empty-row"><td colspan="7">请选择项目后查看日志明细</td></tr>';
    return;
  }

  const params = new URLSearchParams({ project: projectId });
  const start = document.getElementById('log-start') && document.getElementById('log-start').value;
  const end = document.getElementById('log-end') && document.getElementById('log-end').value;
  const type = document.getElementById('log-type') && document.getElementById('log-type').value;
  if (start) params.set('start', start);
  if (end) params.set('end', end);
  if (type) params.set('type', type);

  tbody.innerHTML = '<tr class="empty-row"><td colspan="7"><span class="spinner"></span> 加载中...</td></tr>';
  const resp = await api('GET', '/v1/admin/projects/logs?' + params.toString());
  if (!resp) return;

  const body = await resp.json();
  if (!resp.ok) {
    currentProjectLogs = [];
    tbody.innerHTML = '<tr class="empty-row"><td colspan="7">日志读取失败</td></tr>';
    toast(body.error || '日志读取失败', 'error');
    return;
  }

  const logs = body.data && body.data.logs || [];
  currentProjectLogs = logs;
  currentLogsPage = 1;
  renderProjectLogs();
}

function exportYesterdayComments(format) {
  var yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  var dateStr = yesterday.getFullYear() + '-' + String(yesterday.getMonth() + 1).padStart(2, '0') + '-' + String(yesterday.getDate()).padStart(2, '0');

  var params = new URLSearchParams();
  params.set('format', format || 'csv');
  params.set('start', dateStr);
  params.set('end', dateStr);

  var project = document.getElementById('log-project') && document.getElementById('log-project').value;
  if (project) params.set('project', project);

  var url = '/v1/admin/export/training-data?' + params.toString() + '&token=' + encodeURIComponent(getToken());
  var a = document.createElement('a');
  a.href = url;
  a.download = '';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  toast('正在导出昨天（' + dateStr + '）的评论数据...');
}

function exportTrainingData(format) {
  var params = new URLSearchParams();
  params.set('format', format || 'jsonl');

  var project = document.getElementById('log-project') && document.getElementById('log-project').value;
  var start = document.getElementById('log-start') && document.getElementById('log-start').value;
  var end = document.getElementById('log-end') && document.getElementById('log-end').value;
  if (project) params.set('project', project);
  if (start) params.set('start', start);
  if (end) params.set('end', end);

  // 通过隐藏 iframe 触发下载，带上认证 token
  var url = '/v1/admin/export/training-data?' + params.toString() + '&token=' + encodeURIComponent(getToken());
  var a = document.createElement('a');
  a.href = url;
  a.download = '';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  toast('正在导出训练数据...');
}
