# 📋 API 鉴权和监控配置快速参考

## 🔐 API 鉴权配置

### 启用 API 鉴权

编辑 `.env` 文件：

```env
# 启用鉴权
ENABLE_AUTH=true

# 配置密钥列表（格式：项目ID|密钥|速率限制）
ALLOWED_KEYS=forum_service|sk-proj-forum-a-k3j9x2m1|300,bbs_service|sk-proj-bbs-b-p8q4w7n5|200,community|sk-proj-community-c-r6t2y9|500
```

### 密钥格式说明

```
项目ID|密钥|速率限制(请求/分钟)
├─ 项目ID：项目的唯一标识符（不含特殊字符）
├─ 密钥：至少16个字符的随机字符串（建议使用 sk-proj- 前缀）
└─ 速率限制：每分钟允许的请求数（0=无限制）

示例：
forum_service|sk-proj-forum-a-k3j9x2m1|300    # 论坛服务，300请求/分钟
bbs_service|sk-proj-bbs-b-p8q4w7n5|200        # 社区服务，200请求/分钟
```

### 在请求中使用密钥

所有需要鉴权的 API 都需要在请求头中包含密钥：

```bash
# 方法1：X-Project-Key 头
curl -X POST https://ai.a889.cloud/v1/moderate \
  -H "Content-Type: application/json" \
  -H "X-Project-Key: sk-proj-forum-a-k3j9x2m1" \
  -d '{"content":"内容","type":"comment"}'

# 返回成功（200）
{
  "code": 200,
  "verdict": "pass",
  "category": "normal",
  "confidence": 0.98
}

# 返回密钥无效（401）
{
  "code": 401,
  "error": "无效的项目密钥"
}

# 返回被限流（429）
{
  "code": 429,
  "error": "请求过于频繁: 301/300"
}
```

### 密钥管理命令

```bash
# 1. 列出所有密钥
bash manage-keys.sh list

# 2. 测试密钥是否有效
bash manage-keys.sh test sk-proj-forum-a-k3j9x2m1

# 3. 测试速率限制
bash manage-keys.sh rate-test sk-proj-forum-a-k3j9x2m1

# 4. 生成新密钥
bash manage-keys.sh create new_project 300

# 5. 轮换密钥（旧 -> 新）
bash manage-keys.sh rotate sk-proj-old-xxx sk-proj-new-yyy

# 6. 禁用密钥
bash manage-keys.sh disable sk-proj-old-xxx

# 7. 导出为 JSON
bash manage-keys.sh export-json
```

---

## 📊 监控和日志配置

### 启用审计和监控

编辑 `.env` 文件：

```env
# 启用审计日志（记录所有 API 调用、认证尝试等）
ENABLE_AUDIT=true
AUDIT_LOG_DIR=/var/log/moderation/audit

# 启用详细监控指标
ENABLE_METRICS=true
METRICS_PORT=9090
```

### 日志文件位置

| 日志类型 | 位置 | 说明 |
|--------|------|------|
| 应用日志 | `logs/moderation_YYYY-MM-DD.log` | 服务运行日志（JSON 格式，按天切割） |
| 审计日志 | `logs/audit/audit_YYYY-MM-DD.log` | API 调用、认证、限流事件（JSON 格式） |
| Nginx | `/var/log/nginx/moderation_access.log` | HTTP 请求日志 |

### 查看日志

```bash
# 1. 查看实时应用日志
bash monitor.sh logs -n 100

# 2. 查看审计日志
bash monitor.sh audit

# 3. 按项目查询审计日志
bash monitor.sh audit --project forum_service

# 4. 最后 7 天的审计日志
bash monitor.sh audit --days 7

# 5. 使用 grep 搜索
grep "auth_attempt" logs/audit/*.log | jq '.[] | select(.success == false)'
```

### 审计日志格式

```json
{
  "timestamp": "2026-03-23T10:30:45Z",
  "event_type": "auth_attempt",           // auth_attempt / api_call / rate_limit_exceeded
  "project_id": "forum_service",
  "api_key": "sk-p****4w7n5",            // 隐藏的密钥
  "method": "POST",
  "path": "/v1/moderate",
  "status_code": 200,
  "latency_ms": 1234,
  "ip_address": "203.0.113.42",
  "user_agent": "curl/7.64.1",
  "error_msg": "",
  "metadata": {}
}
```

### 实时监控指标

```bash
# 1. 查看服务状态
bash monitor.sh status

# 输出包括：
# - 容器状态和资源使用
# - 健康检查结果
# - 实时性能指标

# 2. 获取详细指标
bash monitor.sh metrics

# 3. 导出为 Prometheus 格式
bash monitor.sh metrics --export prometheus
```

### 性能指标说明

```bash
# 响应指标
{
  "uptime_seconds": 86400,              // 运行时长（秒）
  "total_requests": 50000,              // 总请求数
  "success_requests": 49500,            // 成功请求数
  "failed_requests": 500,               // 失败请求数
  "success_rate_percent": 99.0,         // 成功率
  "avg_latency_ms": 234.5,              // 平均响应时间（毫秒）
  "cached_requests": 15000,             // 从缓存返回的请求

  # API 调用指标
  "api_calls": 50000,                   // 向 Anthropic API 的调用量
  "api_calls_success": 49500,           // 成功的 API 调用
  "api_calls_failed": 500,              // 失败的 API 调用
  "avg_api_latency_ms": 1234.5,         // 平均 API 响应时间

  # 模型使用情况
  "model_usage": {
    "claude-sonnet-4-20250514": 30000,  // Sonnet 被调用 30000 次
    "claude-haiku-4-5-20251001": 15000,
    "claude-opus-4-20250514": 5000
  },

  # 错误统计
  "error_counts": {
    "rate_limit_exceeded": 100,         // 触发限流 100 次
    "auth_failed": 50,                  // 认证失败 50 次
    "api_timeout": 20                   // API 超时 20 次
  },

  # 认证统计
  "auth_success": 50000,                // 认证成功次数
  "auth_fail": 100                      // 认证失败次数
}
```

---

## ⚠️ 告警和监控规则

### 重要指标阈值

```bash
# 1. 错误率过高 (> 5%)
if (failed_requests / total_requests) > 0.05:
  action: 发送告警，检查日志

# 2. 响应延迟过高 (> 5 秒)
if avg_latency_ms > 5000:
  action: 考虑自动扩容或优化缓存

# 3. 认证失败频繁 (> 10次/分钟)
if auth_fail_count > 10:
  action: 检查是否被恶意探测，考虑封禁 IP

# 4. 限流触发频繁 (> 100次/小时)
if rate_limit_exceeded > 100:
  action: 通知相关项目，考虑提高配额

# 5. 缓存命中率低 (< 30%)
if (cached_requests / total_requests) < 0.30:
  action: 增加缓存大小或提高 TTL
```

### 日志清理计划

```bash
# 自动清理 30 天前的日志
bash monitor.sh clean

# 或配置 crontab（每天凌晨 2 点）
0 2 * * * cd /opt/go-server && bash monitor.sh clean

# 每周日备份审计日志
0 3 * * 0 cd /opt/go-server && bash monitor.sh backup
```

---

## 🔍 常见查询示例

### 查询特定项目的 API 调用

```bash
# 查询论坛服务的所有 API 调用
grep '"project_id":"forum_service"' logs/audit/*.log \
  | jq 'select(.event_type == "api_call")'

# 统计失败次数
grep '"project_id":"forum_service"' logs/audit/*.log \
  | jq 'select(.status_code >= 400)' | wc -l

# 平均响应时间
grep '"project_id":"forum_service"' logs/audit/*.log \
  | jq '.latency_ms' | awk '{sum+=$1} END {print sum/NR}'
```

### 查询特定时间段的认证失败

```bash
# 今天的认证失败
grep '"event_type":"auth_attempt"' logs/audit/*.log \
  | jq 'select(.metadata.success == false and .timestamp | startswith("2026-03-23"))'

# 过去 7 天的认证失败统计
find logs/audit -name "*.log" -mtime -7 -exec grep '"event_type":"auth_attempt"' {} \; \
  | jq 'select(.metadata.success == false)' | wc -l
```

### 监控 IP 地址和请求源

```bash
# 统计请求来源 IP
grep '"event_type":"api_call"' logs/audit/*.log \
  | jq '.ip_address' | sort | uniq -c | sort -rn | head -10

# 查找可疑 IP（请求失败次数多）
grep '"event_type":"api_call"' logs/audit/*.log \
  | jq 'select(.status_code >= 400) | .ip_address' \
  | sort | uniq -c | sort -rn | head -5
```

---

## 🚀 部署后的检查清单

- [ ] ✅ API 鉴权已启用（ENABLE_AUTH=true）
- [ ] ✅ 至少配置了 1 个 API 密钥
- [ ] ✅ 审计日志已启用（ENABLE_AUDIT=true）
- [ ] ✅ 监控指标已启用（ENABLE_METRICS=true）
- [ ] ✅ HTTPS 证书已配置（Let's Encrypt）
- [ ] ✅ Nginx 反向代理已配置
- [ ] ✅ 日志轮转和清理计划已设置
- [ ] ✅ 备份策略已配置
- [ ] ✅ 告警规则已配置
- [ ] ✅ 定期审查日志和指标

