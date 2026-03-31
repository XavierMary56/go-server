// Project stats and audit log management for the admin web UI.
var logsPageSize = 20;
var currentLogsPage = 1;

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
    var details = formatLogDetails(event.details);
    var time = escapeHtml(formatDate(event.ts));
    var projectIdText = escapeHtml(event.project_name || '-');
    var eventTypeText = escapeHtml(event.event_type || '-');
    var clientIpText = escapeHtml(event.client_ip || '-');
    var resultText = escapeHtml(formatLogResult(event));
    var detailsText = escapeHtml(details);
    return '<tr>' +
      '<td>' + time + '</td>' +
      '<td>' + projectIdText + '</td>' +
      '<td>' + eventTypeText + '</td>' +
      '<td>' + clientIpText + '</td>' +
      '<td>' + resultText + '</td>' +
      '<td title="' + detailsText + '">' + detailsText + '</td>' +
      '<td><button class="btn btn-sm btn-ghost" onclick="openLogDetailModal(' + index + ')">查看详情</button></td>' +
      '</tr>';
  }).join('');

  var pager = document.getElementById('logs-pagination');
  if (totalPages <= 1) { pager.innerHTML = ''; return; }
  var html = '<div class="pagination">';
  html += '<span class="page-info">共 ' + total + ' 条，第 ' + currentLogsPage + ' / ' + totalPages + ' 页</span>';
  html += '<button class="btn btn-sm btn-ghost" onclick="logsGoPage(1)" ' + (currentLogsPage === 1 ? 'disabled' : '') + '>首页</button>';
  html += '<button class="btn btn-sm btn-ghost" onclick="logsGoPage(' + (currentLogsPage - 1) + ')" ' + (currentLogsPage === 1 ? 'disabled' : '') + '>上一页</button>';
  var from = Math.max(1, currentLogsPage - 2);
  var to = Math.min(totalPages, currentLogsPage + 2);
  for (var p = from; p <= to; p++) {
    html += '<button class="btn btn-sm ' + (p === currentLogsPage ? 'btn-primary' : 'btn-ghost') + '" onclick="logsGoPage(' + p + ')">' + p + '</button>';
  }
  html += '<button class="btn btn-sm btn-ghost" onclick="logsGoPage(' + (currentLogsPage + 1) + ')" ' + (currentLogsPage === totalPages ? 'disabled' : '') + '>下一页</button>';
  html += '<button class="btn btn-sm btn-ghost" onclick="logsGoPage(' + totalPages + ')" ' + (currentLogsPage === totalPages ? 'disabled' : '') + '>末页</button>';
  html += '</div>';
  pager.innerHTML = html;
}

function logsGoPage(p) {
  currentLogsPage = p;
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

  const projectCount = projectsData.total_projects || (projectsData.projects || []).length || 0;
  let totalCalls = 0;
  let totalAuth = 0;
  let totalRateLimited = 0;
  Object.values(stats).forEach(function (item) {
    totalCalls += item.api_calls || 0;
    totalAuth += item.auth_attempts || 0;
    totalRateLimited += item.rate_limited || 0;
  });

  const statsSummary = document.getElementById('stats-summary');
  if (statsSummary) statsSummary.innerHTML = `
    <div class="stat-card"><div class="label">项目总数</div><div class="value">${projectCount}</div><div class="sub">当前活跃项目</div></div>
    <div class="stat-card"><div class="label">API 调用</div><div class="value">${totalCalls.toLocaleString()}</div><div class="sub">累计请求次数</div></div>
    <div class="stat-card"><div class="label">认证次数</div><div class="value">${totalAuth.toLocaleString()}</div><div class="sub">鉴权请求总量</div></div>
    <div class="stat-card"><div class="label">限流次数</div><div class="value">${totalRateLimited.toLocaleString()}</div><div class="sub">触发速率限制</div></div>`;

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

  if (tbody) tbody.innerHTML = entries.map(function (entry) {
    const pid = entry[0];
    const statsItem = entry[1];
    return `
    <tr>
      <td><strong>${pid}</strong></td>
      <td>${(statsItem.api_calls || 0).toLocaleString()}</td>
      <td>${(statsItem.auth_attempts || 0).toLocaleString()}</td>
      <td>${(statsItem.rate_limited || 0).toLocaleString()}</td>
      <td>${(statsItem.errors || 0).toLocaleString()}</td>
      <td>${formatBytes(statsItem.log_size || 0)}</td>
      <td><button class="btn btn-sm btn-info" onclick="focusProjectLogs('${pid}')">查看日志</button></td>
    </tr>`;
  }).join('');

  resetProjectLogFilters();
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
  if (typeEl) typeEl.value = '';
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
  const select = document.getElementById('log-project');
  if (select) {
    select.value = projectId;
  }
  loadProjectLogs();
}

function escapeHtml(value) {
  return String(value == null ? '' : value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function formatLogResult(event) {
  const details = event.details || {};
  if (Object.prototype.hasOwnProperty.call(details, 'ok')) return details.ok ? '成功' : '失败';
  if (event.event_type === 'rate_limit_exceeded') return '已限流';
  if (event.event_type === 'config_change') return '已记录';
  return '-';
}

function formatLogDetails(details) {
  if (!details || typeof details !== 'object') return '-';

  const parts = [];
  if (details.path) parts.push(`路径: ${details.path}`);
  if (details.method) parts.push(`方法: ${details.method}`);
  if (details.status_code) parts.push(`状态码: ${details.status_code}`);
  if (details.key_name) parts.push(`键名: ${details.key_name}`);
  if (details.config_type) parts.push(`配置类型: ${details.config_type}`);
  if (details.change_type) parts.push(`变更: ${details.change_type}`);
  if (details.reason) parts.push(`原因: ${details.reason}`);
  if (parts.length) return parts.join(' | ');

  try {
    return JSON.stringify(details);
  } catch {
    return '-';
  }
}

function openLogDetailModal(index) {
  const event = currentProjectLogs[index];
  if (!event) {
    toast('日志详情不存在', 'error');
    return;
  }

  const details = event.details || {};
  document.getElementById('log-detail-project').value = event.project_name || '-';
  document.getElementById('log-detail-type').value = event.event_type || '-';
  document.getElementById('log-detail-time').value = formatDate(event.ts);
  document.getElementById('log-detail-ip').value = event.client_ip || '-';
  document.getElementById('log-detail-summary').value = formatLogDetails(details);
  document.getElementById('log-detail-json').textContent = JSON.stringify(event, null, 2);
  document.getElementById('log-detail-modal').classList.add('show');
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
