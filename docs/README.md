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
```

### 内容审核（需要 X-Project-Key）

```
POST /v1/moderate
```
```json
{
  "content": "待审核内容",
  "type": "comment",
  "strictness": "standard"
}
```

返回：
```json
{
  "code": 200,
  "verdict": "approved",
  "category": "none",
  "confidence": 0.98,
  "reason": "内容正常",
  "model_used": "claude-sonnet-4-20250514",
  "latency_ms": 1234
}
```

`verdict`：`approved` / `flagged` / `rejected`
`type`：`comment` / `post`
`strictness`：`loose` / `standard` / `strict`

### 异步审核（需要 X-Project-Key）

```
POST /v1/moderate/async
```
立即返回 `task_id`，结果通过 Webhook 回调。

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
