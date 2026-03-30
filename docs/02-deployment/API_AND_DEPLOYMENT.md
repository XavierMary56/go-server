# 🚀 部署和功能完整指南

## 📋 快速导航

| 文档 | 内容 | 耗时 |
|-----|------|------|
| [1. 快速开始](#1-快速开始) | 3步启动服务 | 5分钟 |
| [2. 功能清单](#2-功能清单) | 所有可用功能 | 10分钟 |
| [3. API 端点清单](#3-api-端点完整清单) | 所有可调用接口 | 查阅 |
| [4. 配置说明](#4-配置说明) | 环境变量配置 | 查阅 |
| [5. 部署架构](#5-部署架构) | 系统设计 | 查阅 |

---

## 1. 快速开始

### 方式 A：Docker 部署（推荐）

```bash
# Step 1: 配置
cp docs/examples/.env.production.example .env
nano .env  # 填入 ANTHROPIC_API_KEY

# Step 2: 启动
docker-compose up -d

# Step 3: 验证
curl http://localhost:8080/v1/health
```

### 方式 B：二进制部署

```bash
# Step 1: 编译
go build -o moderation-server ./cmd/server

# Step 2: 配置
cp .env.production .env
nano .env

# Step 3: 启动
./moderation-server
```

---

## 2. 功能清单

### ✅ 核心功能

| 功能 | 说明 | 状态 |
|-----|------|-----|
| **内容审核** | 同步调用 Anthropic Claude 进行实时审核 | ✅ |
| **异步审核** | 支持 Webhook 回调，适合高并发 | ✅ |
| **多模型调度** | 自动在 Sonnet/Haiku/Opus 间轮换 | ✅ |
| **故障转移** | 模型失败自动切换下一个 | ✅ |
| **缓存系统** | 支持 Redis 和内存缓存 | ✅ |
| **速率限制** | 按项目密钥限制请求频率 | ✅ |
| **API 鉴权** | 支持密钥验证和管理 | ✅ |

### ✅ 管理功能

| 功能 | 说明 | 状态 |
|-----|------|-----|
| **密钥管理** | 无需重启，动态添加/删除/更新密钥 | ✅ |
| **项目隔离** | 每个项目的日志完全独立存储 | ✅ |
| **审计日志** | 完整记录所有 API 调用和认证尝试 | ✅ |
| **性能监控** | 实时统计请求数、成功率、延迟等 | ✅ |
| **项目统计** | 查看每个项目的使用情况 | ✅ |

### ✅ 可观测性

| 功能 | 说明 | 状态 |
|-----|------|-----|
| **结构化日志** | JSON 格式日志，按天自动切割 | ✅ |
| **项目日志分离** | 日志按项目分目录存储 | ✅ |
| **日志查询 API** | 可通过 API 查询历史日志 | ✅ |
| **实时指标** | 通过 /v1/stats 查看实时数据 | ✅ |
| **管理后台** | 通过 /v1/admin 管理系统 | ✅ |

---

## 3. API 端点完整清单

### 📍 用户 API（不需认证的）

#### GET /v1/health
**功能**：服务健康检查
```bash
curl http://localhost:8080/v1/health

# 返回
{
  "status": "ok",
  "version": "2.0.0",
  "time": "2026-03-23T10:30:45Z"
}
```

---

### 📍 业务 API（需要 API 密钥）

#### POST /v1/moderate
**功能**：同步内容审核
```bash
curl -X POST http://localhost:8080/v1/moderate \
  -H "Content-Type: application/json" \
  -H "X-Project-Key: sk-proj-xxxx" \
  -d '{
    "content": "待审核内容",
    "type": "comment",        # comment | post
    "strictness": "standard"  # loose | standard | strict
  }'

# 返回
{
  "code": 200,
  "verdict": "approved",           # approved | flagged | rejected
  "category": "none",              # none | spam | abuse | politics | adult | fraud | violence
  "confidence": 0.98,
  "reason": "内容正常",
  "model_used": "claude-sonnet-4-20250514",
  "latency_ms": 1234,
  "from_cache": false
}
```

#### POST /v1/moderate/async
**功能**：异步内容审核（立即返回，结果通过 Webhook 回调）
```bash
curl -X POST http://localhost:8080/v1/moderate/async \
  -H "Content-Type: application/json" \
  -H "X-Project-Key: sk-proj-xxxx" \
  -d '{
    "content": "待审核内容",
    "type": "comment",
    "webhook_url": "https://yourapp.com/webhook"
  }'

# 返回（立即）
{
  "code": 202,
  "task_id": "task_1234567890",
  "message": "任务已接受"
}

# Webhook 回调（审核完成后）
POST https://yourapp.com/webhook
{
  "task_id": "task_1234567890",
  "status": "done",
  "verdict": "approved",
  "category": "none",
  "confidence": 0.98,
  "reason": "内容正常",
  "model_used": "claude-sonnet-4-20250514",
  "latency_ms": 1234
}
```

#### GET /v1/task/{id}
**功能**：查询异步任务结果
```bash
curl -H "X-Project-Key: sk-proj-xxxx" \
  http://localhost:8080/v1/task/task_1234567890

# 返回
{
  "code": 200,
  "data": {
    "task_id": "task_1234567890",
    "status": "done",
    "verdict": "approved",
    "category": "none",
    "confidence": 0.98,
    "model_used": "claude-sonnet-4-20250514",
    "latency_ms": 1234
  }
}
```

#### GET /v1/models
**功能**：查看可用模型列表和权重
```bash
curl -H "X-Project-Key: sk-proj-xxxx" \
  http://localhost:8080/v1/models

# 返回
{
  "code": 200,
  "models": [
    {
      "id": "claude-sonnet-4-20250514",
      "name": "Claude Sonnet 4",
      "weight": 60,
      "priority": 1,
      "status": "active"
    },
    {
      "id": "claude-haiku-4-5-20251001",
      "name": "Claude Haiku 4.5",
      "weight": 30,
      "priority": 2,
      "status": "active"
    },
    {
      "id": "claude-opus-4-20250514",
      "name": "Claude Opus 4",
      "weight": 10,
      "priority": 3,
      "status": "active"
    }
  ]
}
```

#### GET /v1/stats
**功能**：查看实时服务统计
```bash
curl -H "X-Project-Key: sk-proj-xxxx" \
  http://localhost:8080/v1/stats

# 返回
{
  "code": 200,
  "data": {
    "uptime_seconds": 86400,
    "total_requests": 50000,
    "success_requests": 49500,
    "failed_requests": 500,
    "success_rate_percent": "99.00",
    "avg_latency_ms": "234.50",
    "cached_requests": 15000,
    "api_calls": 50000,
    "api_calls_success": 49500,
    "api_calls_failed": 500,
    "avg_api_latency_ms": "1234.50",
    "model_usage": {
      "claude-sonnet-4-20250514": 30000,
      "claude-haiku-4-5-20251001": 15000,
      "claude-opus-4-20250514": 5000
    },
    "error_counts": {
      "rate_limit_exceeded": 100,
      "auth_failed": 50,
      "api_timeout": 20
    },
    "auth_success": 50000,
    "auth_fail": 100,
    "start_time": "2026-03-23T00:00:00Z",
    "last_reset_time": "2026-03-23T00:00:00Z"
  }
}
```

---

### 📍 业务 API V2（推荐新接入使用）

V2 保留与现有审核能力一致的核心逻辑，但调整了路由命名和响应结构，便于后续扩展。旧版 `/v1/*` 仍然兼容，建议新项目优先接入 `/v2/*`。

#### GET /v2/health
**功能**：服务健康检查（V2 结构化响应）
```bash
curl http://localhost:8080/v2/health

# 返回
{
  "code": 200,
  "message": "ok",
  "data": {
    "status": "ok",
    "version": "2.0.0",
    "time": "2026-03-30T10:30:45Z"
  }
}
```

#### POST /v2/moderations
**功能**：同步内容审核（推荐）
```bash
curl -X POST http://localhost:8080/v2/moderations \
  -H "Content-Type: application/json" \
  -H "X-Project-Key: sk-proj-xxxx" \
  -d '{
    "content": "待审核内容",
    "type": "comment",
    "strictness": "standard",
    "model": "auto",
    "context": {
      "biz_id": "post_123"
    }
  }'

# 返回
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": "mod_1743290000000000000",
    "status": "completed",
    "result": {
      "verdict": "approved",
      "category": "none",
      "confidence": 0.98,
      "reason": "内容正常",
      "model_used": "claude-sonnet-4-20250514",
      "latency_ms": 1234,
      "from_cache": false
    }
  }
}
```

#### POST /v2/moderations/async
**功能**：异步内容审核
```bash
curl -X POST http://localhost:8080/v2/moderations/async \
  -H "Content-Type: application/json" \
  -H "X-Project-Key: sk-proj-xxxx" \
  -d '{
    "content": "待审核内容",
    "type": "comment",
    "webhook_url": "https://yourapp.com/webhook"
  }'

# 返回
{
  "code": 202,
  "message": "accepted",
  "data": {
    "task_id": "task_1743290000000000000",
    "status": "pending"
  }
}
```

#### GET /v2/tasks/{id}
**功能**：查询异步任务结果
```bash
curl -H "X-Project-Key: sk-proj-xxxx" \
  http://localhost:8080/v2/tasks/task_1743290000000000000

# 返回
{
  "code": 200,
  "message": "ok",
  "data": {
    "task_id": "task_1743290000000000000",
    "status": "done",
    "result": {
      "verdict": "approved",
      "category": "none",
      "confidence": 0.98,
      "reason": "内容正常",
      "model_used": "claude-sonnet-4-20250514",
      "latency_ms": 1234,
      "from_cache": false
    }
  }
}
```

#### GET /v2/models
**功能**：查看可用模型列表（V2 结构化响应）
```bash
curl -H "X-Project-Key: sk-proj-xxxx" \
  http://localhost:8080/v2/models

# 返回
{
  "code": 200,
  "message": "ok",
  "data": {
    "models": [
      {
        "id": "claude-sonnet-4-20250514",
        "name": "Claude Sonnet 4",
        "weight": 60,
        "priority": 1,
        "status": "active"
      }
    ]
  }
}
```

#### V1 与 V2 的主要区别
- V1 使用动作式路径：`/v1/moderate`、`/v1/moderate/async`
- V2 使用资源式路径：`/v2/moderations`、`/v2/moderations/async`
- V2 统一采用 `code + message + data` 响应结构
- V2 将审核结果包装在 `data.result` 中，便于后续扩展状态、ID、元数据

---

### 📍 管理 API（需要管理员令牌）

#### 密钥管理

##### GET /v1/admin/keys
**功能**：列出所有 API 密钥
```bash
curl -H "Authorization: Bearer admin-token-default" \
  http://localhost:8080/v1/admin/keys

# 返回
{
  "code": 200,
  "data": {
    "sk-...-4w7n5": {
      "project_id": "forum_service",
      "key": "sk-proj-forum-a-k3j9x2m1",
      "rate_limit": 300,
      "created_at": "2026-03-23T00:00:00Z",
      "updated_at": "2026-03-23T00:00:00Z",
      "enabled": true
    }
  }
}
```

##### POST /v1/admin/keys
**功能**：添加新密钥（无需重启）
```bash
curl -X POST http://localhost:8080/v1/admin/keys \
  -H "Authorization: Bearer admin-token-default" \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "51dm_service",
    "key": "sk-proj-51dm-a1b2c3d4",
    "rate_limit": 300
  }'

# 返回
{
  "code": 201,
  "message": "密钥已添加",
  "data": {
    "project_id": "51dm_service",
    "key": "sk-proj-51dm-a1b2c3d4",
    "rate_limit": 300,
    "created_at": "2026-03-23T10:30:45Z",
    "updated_at": "2026-03-23T10:30:45Z",
    "enabled": true
  }
}
```

##### GET /v1/admin/keys/{key}
**功能**：查看单个密钥详情
```bash
curl -H "Authorization: Bearer admin-token-default" \
  http://localhost:8080/v1/admin/keys/sk-proj-51dm-a1b2c3d4
```

##### PUT /v1/admin/keys/{key}
**功能**：更新密钥配置
```bash
curl -X PUT http://localhost:8080/v1/admin/keys/sk-proj-51dm-a1b2c3d4 \
  -H "Authorization: Bearer admin-token-default" \
  -H "Content-Type: application/json" \
  -d '{
    "rate_limit": 500,    # 提高限流
    "enabled": true
  }'
```

##### DELETE /v1/admin/keys/{key}
**功能**：删除密钥
```bash
curl -X DELETE http://localhost:8080/v1/admin/keys/sk-proj-51dm-a1b2c3d4 \
  -H "Authorization: Bearer admin-token-default"
```

---

#### 项目和日志管理

##### GET /v1/admin/projects
**功能**：列出所有项目及其统计
```bash
curl -H "Authorization: Bearer admin-token-default" \
  http://localhost:8080/v1/admin/projects

# 返回
{
  "code": 200,
  "data": {
    "total_projects": 3,
    "projects": [
      {
        "project_id": "forum_service",
        "total_size_bytes": 1024000,
        "total_size_mb": "1.00",
        "event_counts": {
          "api_call": 5000,
          "auth_attempt": 100,
          "rate_limit_exceeded": 10,
          "config_change": 2
        }
      }
    ]
  }
}
```

##### GET /v1/admin/projects/logs
**功能**：查询项目日志
```bash
# 查询过去 24 小时的所有日志
curl -H "Authorization: Bearer admin-token-default" \
  "http://localhost:8080/v1/admin/projects/logs?project=forum_service"

# 查询特定时间范围
curl -H "Authorization: Bearer admin-token-default" \
  "http://localhost:8080/v1/admin/projects/logs?project=forum_service&start=2026-03-20&end=2026-03-23"

# 查询特定事件类型
curl -H "Authorization: Bearer admin-token-default" \
  "http://localhost:8080/v1/admin/projects/logs?project=forum_service&type=api_call"

# 返回
{
  "code": 200,
  "data": {
    "project_id": "forum_service",
    "total_count": 100,
    "start_time": "2026-03-22T00:00:00Z",
    "end_time": "2026-03-23T00:00:00Z",
    "event_type": "",
    "logs": [
      {
        "timestamp": "2026-03-23T10:30:45Z",
        "event_type": "api_call",
        "project_id": "forum_service",
        "api_key": "sk-p****w7n5",
        "method": "POST",
        "path": "/v1/moderate",
        "status_code": 200,
        "latency_ms": 1234,
        "error_msg": "",
        "ip_address": "203.0.113.42"
      }
    ]
  }
}
```

##### GET /v1/admin/projects/stats
**功能**：查看项目统计（大小、事件计数等）
```bash
curl -H "Authorization: Bearer admin-token-default" \
  "http://localhost:8080/v1/admin/projects/stats?project=forum_service"

# 返回
{
  "code": 200,
  "data": {
    "project_id": "forum_service",
    "total_size_bytes": 1024000,
    "total_size_mb": "1.00",
    "event_counts": {
      "api_call": 5000,
      "auth_attempt": 100,
      "rate_limit_exceeded": 10,
      "config_change": 2
    }
  }
}
```

##### GET /v1/admin/health
**功能**：管理 API 健康检查（无需令牌）
```bash
curl http://localhost:8080/v1/admin/health

# 返回
{
  "status": "ok",
  "admin_api_available": true
}
```

---

## 4. 配置说明

### 必填配置

```env
# Anthropic API Key（从 console.anthropic.com 获取）
ANTHROPIC_API_KEY=sk-ant-api03-xxxxx

# 启用 API 鉴权
ENABLE_AUTH=true

# 配置项目密钥（格式: 项目ID|密钥|限流数，逗号分隔）
ALLOWED_KEYS=forum_service|sk-proj-forum-a-k3j9x2m1|300,bbs_service|sk-proj-bbs-b-p8q4w7n5|200
```

### 管理员配置

```env
# 启用管理 API
ENABLE_ADMIN_API=true

# 管理员令牌（用于 /v1/admin/* 端点）
ADMIN_TOKEN=admin-token-default

# 可以配置多个令牌（逗号分隔）
ADMIN_TOKEN=token1,token2,token3

# 当数据库中没有模型配置时，是否回退到配置默认模型
ENABLE_MODEL_CONFIG_FALLBACK=true
```

### 审计日志配置

```env
# 启用审计日志记录
ENABLE_AUDIT=true

# 日志存储目录（按项目分目录）
AUDIT_LOG_DIR=/var/log/moderation/audit
```

### 监控配置

```env
# 启用性能监控
ENABLE_METRICS=true

# 监控指标端口
METRICS_PORT=9090
```

### 可选配置

```env
PORT=8080                      # 服务端口
APP_ENV=production             # 环境：production | development
LOG_LEVEL=info                 # 日志级别：debug | info | warn | error
CACHE_DRIVER=redis             # 缓存驱动：memory | redis
CACHE_TTL=3600                 # 缓存有效期（秒）

# Redis 配置（如果使用 Redis 缓存）
REDIS_ADDR=127.0.0.1:6379
REDIS_PASS=password
REDIS_DB=0

# API 配置
API_TIMEOUT=10                 # API 请求超时（秒）
MAX_RETRIES=2                  # 失败重试次数
```

---

## 5. 部署架构

### 系统设计

```
客户端请求
    ↓
┌─────────────────────────────┐
│   Nginx 反向代理            │
│   - SSL/TLS 加密            │
│   - 限流和速率限制          │
│   - 日志记录                │
└──────────────┬──────────────┘
               ↓
┌─────────────────────────────┐
│   API 鉴权中间件            │
│   - 验证 X-Project-Key      │
│   - 检查速率限制            │
│   - 记录审计日志            │
└──────────────┬──────────────┘
               ↓
╔═════════════════════════════════════════════════════╗
║          Go 审核服务                                ║
║  ┌─────────────────────────┐                       ║
║  │  请求处理器             │                       ║
║  │  - /v1/moderate (同步)  │                       ║
║  │  - /v1/moderate/async   │                       ║
║  │  - /v1/models           │                       ║
║  │  - /v1/stats            │                       ║
║  └────────────┬────────────┘                       ║
║               ↓                                     ║
║  ┌─────────────────────────┐                       ║
║  │  多模型调度             │                       ║
║  │  - Claude Sonnet (60%)  │                       ║
║  │  - Claude Haiku (30%)   │                       ║
║  │  - Claude Opus (10%)    │                       ║
║  └────────────┬────────────┘                       ║
║               ↓                                     ║
║  ┌─────────────────────────┐                       ║
║  │  缓存系统               │                       ║
║  │  - 内存缓存             │                       ║
║  │  - Redis 缓存           │                       ║
║  └────────────┬────────────┘                       ║
║               ↓                                     ║
║  ┌─────────────────────────┐                       ║
║  │  审计日志               │                       ║
║  │  - 按项目分别存储       │                       ║
║  │  - JSON 格式            │                       ║
║  └─────────────────────────┘                       ║
╚═════════════════════════════════════════════════════╝
               ↓
┌─────────────────────────────┐
│   外部服务                  │
│  - Anthropic Claude API     │
│  - Redis（可选）            │
│  - Webhook 回调             │
└─────────────────────────────┘

┌─────────────────────────────┐
│   管理 API     (/v1/admin)   │
│  - 密钥管理                  │
│  - 项目管理                  │
│  - 日志查询                  │
│  - 统计数据                  │
└─────────────────────────────┘
```

---

## 6. 错误处理

### 常见错误码

| 代码 | 含义 | 解决方案 |
|-----|------|--------|
| 200 | 成功 | - |
| 201 | 创建成功 | - |
| 202 | 异步任务已接受 | 使用返回的 task_id 查询结果 |
| 400 | 请求参数错误 | 检查请求体和查询参数 |
| 401 | 认证失败 | 检查 API 密钥是否正确 |
| 404 | 资源不存在 | 检查任务 ID 或路径 |
| 429 | 请求过于频繁 | 等待后重试，或增加配额 |
| 500 | 服务器错误 | 查看日志获取详细信息 |

### 错误响应格式

```json
{
  "code": 400,
  "error": "错误描述信息"
}
```

---

## 7. 日志和监控

### 日志位置

```
logs/
├── moderation_2026-03-23.log       # 应用日志
└── audit/
    ├── forum_service/
    │   └── audit_2026-03-23.log    # 论坛项目审计日志
    ├── bbs_service/
    │   └── audit_2026-03-23.log    # 社区项目审计日志
    └── 51dm_service/
        └── audit_2026-03-23.log    # 51dm 项目审计日志
```

### 查看日志

```bash
# 查看应用日志
tail -f logs/moderation_2026-03-23.log

# 查看特定项目的审计日志
tail -f logs/audit/forum_service/audit_2026-03-23.log

# 查询项目统计
curl -H "Authorization: Bearer admin-token-default" \
  "http://localhost:8080/v1/admin/projects/stats?project=forum_service"
```

---

## ✅ 部署检查清单

- [ ] 已配置 ANTHROPIC_API_KEY
- [ ] 已配置 ALLOWED_KEYS（至少一个项目）
- [ ] 已配置 ADMIN_TOKEN
- [ ] 已启用 ENABLE_AUTH=true
- [ ] 已启用 ENABLE_AUDIT=true
- [ ] Nginx 已配置反向代理
- [ ] HTTPS 证书已配置（Let's Encrypt）
- [ ] Redis 已部署（如果使用 Redis 缓存）
- [ ] 日志目录权限正确（755）
- [ ] 防火墙规则已设置

---

**保存日期**：2026-03-23
**文档版本**：3.0.0
