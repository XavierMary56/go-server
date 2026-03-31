# go-server 文档

## 数据库

SQLite（`modernc.org/sqlite`，WAL 模式），文件路径：`/data/moderation.db`。

---

## 快速启动

**编译并启动：**
```bash
go build -o moderation-server ./cmd/server
./moderation-server
```

**Docker 启动：**
```bash
cp .env.production .env
# 编辑 .env，填入必要配置
docker compose up -d
```

**验证服务：**
```bash
curl http://localhost:8080/v1/health
```

---

## 环境变量配置

```env
# 服务
PORT=8080
APP_ENV=production

# Anthropic API（必填）
ANTHROPIC_API_KEY=sk-ant-xxx

# OpenAI / Grok（可选）
OPENAI_API_KEY=
GROK_API_KEY=

# 鉴权
ENABLE_AUTH=true
ALLOWED_KEYS=forum_service|sk-proj-xxxx|300,bbs_service|sk-proj-yyyy|200

# 管理员
ENABLE_ADMIN_API=true
ADMIN_TOKEN=your-admin-token

# 审计日志
ENABLE_AUDIT=true
AUDIT_LOG_DIR=logs/audit

# 缓存（可选，默认 memory）
CACHE_DRIVER=redis
REDIS_ADDR=localhost:6379
REDIS_DB=0
```

---

## API 接口

### 健康检查（无需鉴权）

```
GET /v1/health
GET /v2/health
```

---

### V1 内容审核

#### POST /v1/moderate

**请求头**

| 字段 | 必填 | 说明 |
|------|------|------|
| `X-Project-Key` | 是 | 项目接入密钥 |
| `Content-Type` | 是 | `application/json` |

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `content` | string | 是 | 待审核的文本内容 |
| `type` | string | 否 | 内容类型：`comment`（评论）/ `post`（帖子），默认 `comment` |
| `strictness` | string | 否 | 审核严格度：`loose` / `standard` / `strict`，默认 `standard` |
| `model` | string | 否 | 指定模型 ID，留空由服务自动调度 |
| `context` | object | 否 | 业务上下文，键值对，透传给审核模型 |

**请求示例**

```json
{
  "content": "待审核内容",
  "type": "comment",
  "strictness": "standard",
  "model": "",
  "context": {"user_id": "1001", "scene": "forum"}
}
```

**响应体**

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 状态码，200 表示成功 |
| `verdict` | string | 审核结论，见下表 |
| `category` | string | 违规分类，见下表 |

**verdict 枚举值**

| 值 | 含义 | 业务处理建议 |
|----|------|-------------|
| `approved` | 内容正常，无违规信号 | 放行 |
| `flagged` | 可疑但不能确定违规 | 放行（如需人工复核可选择进队列） |
| `rejected` | 明确命中违规规则 | 拒绝，不予展示 |

> **接入建议：只需判断 `verdict === "rejected"` 即为拒绝，其余值均视为通过。**

**category 枚举值**

| 值 | 含义 |
|----|------|
| `none` | 无违规 |
| `spam` | 广告、引流、留联系方式、站外交易、群组邀请 |
| `abuse` | 毒品、管制药品相关 |
| `politics` | 政治敏感内容 |
| `adult` | 色情、成人内容 |
| `fraud` | 诈骗、欺诈相关 |
| `violence` | 暴力、武器、爆炸物、恐怖袭击相关 |
| `confidence` | float | 置信度，0~1 |
| `reason` | string | 审核原因说明 |
| `model_used` | string | 实际使用的模型 ID |
| `latency_ms` | int | 审核耗时（毫秒） |
| `from_cache` | bool | 是否来自缓存 |

**响应示例**

```json
{
  "code": 200,
  "verdict": "approved",
  "category": "none",
  "confidence": 0.98,
  "reason": "内容正常",
  "model_used": "claude-3-5-sonnet-20241022",
  "latency_ms": 1234,
  "from_cache": false
}
```

#### POST /v1/moderate/async

请求体与 `POST /v1/moderate` 完全相同，额外支持：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `webhook_url` | string | 否 | 审核完成后回调的 URL，POST JSON 结构与同步响应一致 |

**响应示例**（立即返回）

```json
{
  "code": 200,
  "task_id": "task_1711900000000000000"
}
```

**查询任务结果**

```
GET /v1/task/{task_id}
```

```json
{
  "code": 200,
  "data": {
    "task_id": "task_1711900000000000000",
    "status": "done",
    "result": {
      "verdict": "approved",
      "category": "none",
      "confidence": 0.98,
      "reason": "内容正常",
      "model_used": "claude-3-5-sonnet-20241022",
      "latency_ms": 1234,
      "from_cache": false
    }
  }
}
```

`status`：`pending`（等待中）/ `done`（已完成）

---

### V2 内容审核

#### POST /v2/moderations

**请求头**

| 字段 | 必填 | 说明 |
|------|------|------|
| `X-Project-Key` | 是 | 项目接入密钥 |
| `Content-Type` | 是 | `application/json` |

**请求体**（与 V1 相同）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `content` | string | 是 | 待审核的文本内容 |
| `type` | string | 否 | 内容类型：`comment` / `post`，默认 `comment` |
| `strictness` | string | 否 | 审核严格度：`loose` / `standard` / `strict`，默认 `standard` |
| `model` | string | 否 | 指定模型 ID，留空自动调度 |
| `context` | object | 否 | 业务上下文键值对 |

**响应体**（V2 在外层多了 `data` 包装和 `id`）

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 200 表示成功 |
| `message` | string | `ok` |
| `data.id` | string | 本次请求 ID，格式 `mod_{timestamp}` |
| `data.status` | string | `completed` |
| `data.result.verdict` | string | 审核结论，同 V1 verdict 枚举值 |
| `data.result.category` | string | 违规分类，同 V1 category 枚举值 |
| `data.result.confidence` | float | 置信度 0~1 |
| `data.result.reason` | string | 原因说明 |
| `data.result.model_used` | string | 实际使用的模型 ID |
| `data.result.latency_ms` | int | 审核耗时（毫秒） |
| `data.result.from_cache` | bool | 是否来自缓存 |

**响应示例**

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": "mod_1711900000000000000",
    "status": "completed",
    "result": {
      "verdict": "approved",
      "category": "none",
      "confidence": 0.98,
      "reason": "内容正常",
      "model_used": "claude-3-5-sonnet-20241022",
      "latency_ms": 1234,
      "from_cache": false
    }
  }
}
```

#### POST /v2/moderations/async

请求体与 `POST /v2/moderations` 完全相同，额外支持 `webhook_url`。

**响应示例**（立即返回）

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "task_id": "task_1711900000000000000",
    "status": "pending",
    "created_at": 1711900000
  }
}
```

**查询任务结果**

```
GET /v2/tasks/{task_id}
```

响应结构与 V1 任务查询相同，外层多 `message: "ok"` 字段。

---

### V1 与 V2 差异对比

| 对比项 | V1 `/v1/moderate` | V2 `/v2/moderations` |
|--------|-------------------|----------------------|
| 响应结构 | 字段平铺在顶层 | 结果包在 `data` 对象内 |
| 响应 ID | 无 | 有 `data.id` |
| `message` 字段 | 无 | 有 `message: "ok"` |
| 任务查询路径 | `/v1/task/{id}` | `/v2/tasks/{id}` |
| 请求参数 | 完全相同 | 完全相同 |

---

### 管理 API（需要 Authorization: Bearer <ADMIN_TOKEN>）

| 接口 | 说明 |
|------|------|
| `GET /v1/admin/keys` | 列出项目密钥 |
| `POST /v1/admin/keys` | 添加密钥 |
| `PUT /v1/admin/keys/:key` | 更新密钥 |
| `DELETE /v1/admin/keys/:key` | 删除密钥 |
| `GET /v1/admin/projects/stats` | 项目统计 |
| `GET /v1/admin/projects/logs?project=xxx` | 审计日志查询 |
| `GET /v1/admin/anthropic-keys` | Anthropic 密钥列表 |
| `GET /v1/admin/models` | 模型配置列表 |

---

## 部署（Linux + systemd）

**1. 编译二进制：**
```bash
go build -o moderation-server ./cmd/server
cp moderation-server /opt/moderation/
cp .env.production /opt/moderation/.env
```

**2. 安装 systemd 服务：**
```bash
cp deploy/moderation.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable moderation
systemctl start moderation
```

**3. Nginx 反向代理：**
```bash
cp deploy/nginx.conf.production /etc/nginx/sites-available/moderation
ln -s /etc/nginx/sites-available/moderation /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

配置示例见 `deploy/` 目录。

---

## 运维脚本

| 脚本 | 说明 |
|------|------|
| `bash deploy.sh` | 一键 Docker 部署 |
| `bash manage-keys.sh list` | 列出所有密钥 |
| `bash manage-keys.sh create <项目> <限流>` | 生成新密钥 |
| `bash manage-keys.sh test <key>` | 测试密钥有效性 |
| `bash monitor.sh status` | 查看服务状态 |
| `bash monitor.sh logs -n 100` | 查看最近日志 |
| `bash monitor.sh clean` | 清理 30 天前的日志 |
| `bash monitor.sh backup` | 备份审计日志 |

**通过管理 API 添加密钥：**
```bash
curl -X POST http://localhost:8080/v1/admin/keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"project_id": "my_project", "key": "sk-proj-xxx", "rate_limit": 300}'
```

---

## 日志位置

```
logs/
├── moderation_YYYY-MM-DD.log       # 应用日志
└── audit/
    └── <project_id>/
        └── audit_YYYY-MM-DD.log    # 各项目审计日志
```

**查看日志：**
```bash
tail -f logs/moderation_$(date +%Y-%m-%d).log
bash monitor.sh audit --project forum_service
```

**定时清理（crontab）：**
```bash
0 2 * * * cd /opt/moderation && bash monitor.sh clean
0 3 * * 0 cd /opt/moderation && bash monitor.sh backup
```

---

## 错误码

| 状态码 | 含义 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 鉴权失败（密钥无效） |
| 429 | 触发速率限制 |
| 500 | 服务内部错误 |
