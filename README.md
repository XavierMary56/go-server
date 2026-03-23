# 🤖 AI 内容审核系统 — Go 服务端

> 使用 Go 实现的高性能内容审核服务，多模型自动轮换，支持同步/异步两种模式，PHP YAF 一键对接。

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![PHP](https://img.shields.io/badge/PHP-7.3+-blue?logo=php)](https://php.net)
[![Docker](https://img.shields.io/badge/Docker-支持-2496ED?logo=docker)](https://docker.com)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

---

## 📚 文档导航

> 快速查找部署、配置、集成相关的文档（所有文档已按分类整理至 docs/ 目录）

| 类型 | 文档 | 用途 |
|-----|------|------|
| 🌟 **快速了解** | [docs/01-gettingstarted/00-START-HERE.md](docs/01-gettingstarted/00-START-HERE.md) | 5分钟快速导览 |
| 🚀 **快速开始** | [docs/01-gettingstarted/DEPLOYMENT_CHECKLIST.md](docs/01-gettingstarted/DEPLOYMENT_CHECKLIST.md) | 部署前检查清单 |
| 📖 **部署指南** | [docs/02-deployment/API_AND_DEPLOYMENT.md](docs/02-deployment/API_AND_DEPLOYMENT.md) | ⭐ 完整API文档和部署方案 |
| 📋 **详细部署** | [docs/02-deployment/DEPLOYMENT.md](docs/02-deployment/DEPLOYMENT.md) | 生产部署详细指南 |
| 👥 **项目对接** | [docs/03-integration/CLIENT_INTEGRATION.md](docs/03-integration/CLIENT_INTEGRATION.md) | ⭐ 新项目对接指南 + Demo |
| 🔧 **代码集成** | [docs/03-integration/INTEGRATION_GUIDE.md](docs/03-integration/INTEGRATION_GUIDE.md) | 代码集成和开发 |
| 🔐 **运维管理** | [docs/04-operations/AUTH_AND_MONITORING.md](docs/04-operations/AUTH_AND_MONITORING.md) | 鉴权、监控、日志管理 |
| 🔨 **脚本命令** | [docs/04-operations/SCRIPTS_GUIDE.md](docs/04-operations/SCRIPTS_GUIDE.md) | 部署和管理脚本使用 |
| 📝 **配置示例** | [docs/02-deployment/examples/](docs/02-deployment/examples/) | 所有配置文件示例 |

### 🎯 按需求快速导航

- 🌟 **再也理解整体？** → [docs/01-gettingstarted/00-START-HERE.md](docs/01-gettingstarted/00-START-HERE.md)
- 🚀 **要部署服务？** → [docs/02-deployment/API_AND_DEPLOYMENT.md](docs/02-deployment/API_AND_DEPLOYMENT.md)
- 👥 **要对接新项目？** → [docs/03-integration/CLIENT_INTEGRATION.md](docs/03-integration/CLIENT_INTEGRATION.md)
- 🔐 **要配置鉴权和监控？** → [docs/04-operations/AUTH_AND_MONITORING.md](docs/04-operations/AUTH_AND_MONITORING.md)
- 🔨 **怎么运行脚本？** → [docs/04-operations/SCRIPTS_GUIDE.md](docs/04-operations/SCRIPTS_GUIDE.md)
- 📚 **查看全部文档？** → [docs/README.md](docs/README.md)

---

## 📁 目录结构

```
go-server/
├── cmd/server/main.go                    # 程序入口
├── internal/
│   ├── config/config.go                  # 配置加载（环境变量 / .env）
│   ├── handler/handler.go                # HTTP 路由和请求处理
│   ├── service/
│   │   ├── moderation.go                 # 核心：多模型调度 & 审核逻辑
│   │   └── cache.go                      # 内存缓存实现
│   └── logger/logger.go                  # 结构化日志（JSON，按天分割）
├── pkg/client/client.go                  # Go 客户端 SDK（供其他 Go 服务使用）
├── examples/
│   └── php-yaf-client/
│       └── ContentModerationService.php  # PHP YAF 对接客户端
├── deploy/
│   ├── nginx.conf                        # Nginx 反向代理配置
│   └── moderation.service                # systemd 服务配置
├── Dockerfile                            # 多阶段构建，最终镜像 < 10MB
├── docker-compose.yml                    # 一键启动（含 Redis）
├── .env.example                          # 配置模板
└── go.mod
```

---

## 🚀 部署方式

### 方式一：Docker Compose（推荐，最简单）

**第 1 步：克隆并配置**
```bash
git clone https://github.com/XavierMary56/automatic_review.git
cd automatic_review/go-server

# 复制配置模板
cp .env.example .env

# 编辑 .env，填入你的 Anthropic API Key
vim .env
# 修改这一行：
# ANTHROPIC_API_KEY=sk-ant-api03-你的真实密钥
```

**第 2 步：一键启动**
```bash
docker-compose up -d

# 查看启动日志
docker-compose logs -f moderation
```

**第 3 步：验证服务**
```bash
curl http://localhost:8080/v1/health
# 返回：{"status":"ok","version":"2.0.0","time":"..."}
```

**测试一次审核：**
```bash
curl -X POST http://localhost:8080/v1/moderate \
  -H "Content-Type: application/json" \
  -d '{"content":"加微信领红包，限时特惠！","type":"comment"}'
```

---

### 方式二：二进制部署（不用 Docker）

**第 1 步：编译**
```bash
# 本地编译
go build -o moderation-server ./cmd/server

# 或交叉编译 Linux 二进制（在 Mac/Windows 上编译）
GOOS=linux GOARCH=amd64 go build -o moderation-server ./cmd/server
```

**第 2 步：上传并运行**
```bash
# 上传二进制和配置
scp moderation-server user@your-server:/opt/moderation/
scp .env.example user@your-server:/opt/moderation/.env

# SSH 进服务器，配置并启动
ssh user@your-server
cd /opt/moderation
vim .env  # 填入 ANTHROPIC_API_KEY
./moderation-server
```

**第 3 步：配置 systemd（后台常驻）**
```bash
# 复制 systemd 服务文件
sudo cp deploy/moderation.service /etc/systemd/system/

# 启动并设置开机自启
sudo systemctl daemon-reload
sudo systemctl enable moderation
sudo systemctl start moderation

# 查看状态
sudo systemctl status moderation
sudo journalctl -u moderation -f
```

---

### 方式三：配置 Nginx 反向代理（生产环境）

```bash
# 复制 Nginx 配置
sudo cp deploy/nginx.conf /etc/nginx/conf.d/moderation.conf

# 修改域名
sudo vim /etc/nginx/conf.d/moderation.conf
# 将 mod.your-company.com 改为你的实际域名

# 重载 Nginx
sudo nginx -t && sudo systemctl reload nginx
```

---

## 🔌 PHP YAF 项目对接

### 第 1 步：复制客户端文件

```bash
cp examples/php-yaf-client/ContentModerationService.php \
   /your-yaf-project/application/services/ContentModerationService.php
```

### 第 2 步：修改配置（conf/application.ini）

```ini
; 指向 Go 服务地址（将原来的 PHP 服务地址替换为此）
moderation.endpoint   = "http://mod.your-company.com"  ; Go 服务地址
moderation.api_key    = "proj_forum_a_k3j9x2m1"        ; 项目密钥
moderation.strictness = "standard"                      ; standard | strict | loose
moderation.timeout    = 5
moderation.async      = false
project.name          = "project-forum-a"
```

> ✅ **无需修改 Plugin 和 Controller**，`ContentModerationService` 接口与原 PHP 版本完全一致。

### 第 3 步：验证连通性（可选）

在 YAF Controller 中调用：
```php
$service = new ContentModerationService();
if (!$service->ping()) {
    // 服务不可达，记录告警
}
```

---

## 📡 API 接口文档

### POST /v1/moderate — 同步审核

**请求体：**
```json
{
  "content":    "待审核的内容文本",
  "type":       "post",       // post | comment
  "model":      "auto",       // auto | claude-sonnet-4-20250514 | claude-haiku-4-5-20251001
  "strictness": "standard",   // standard | strict | loose
  "context": {
    "user_id":  "12345",
    "project":  "forum-a"
  }
}
```

**成功响应：**
```json
{
  "code":       200,
  "verdict":    "approved",      // approved | flagged | rejected
  "category":   "none",          // none | spam | abuse | politics | adult | fraud | violence
  "confidence": 0.97,
  "reason":     "内容正常",
  "model_used": "claude-sonnet-4-20250514",
  "latency_ms": 823,
  "from_cache": false
}
```

---

### POST /v1/moderate/async — 异步审核

立即返回 `task_id`，审核结果通过 Webhook 回调：

```json
// 请求（新增 webhook_url 字段）
{
  "content":     "待审核内容",
  "type":        "post",
  "webhook_url": "https://your-project.com/moderation/callback"
}

// 立即响应
{"code": 202, "task_id": "task_1234567890", "message": "任务已接受"}

// Webhook 回调（审核完成后 POST 到 webhook_url）
{
  "task_id":    "task_1234567890",
  "verdict":    "approved",
  "category":   "none",
  "confidence": 0.95,
  "reason":     "内容正常",
  "model_used": "claude-haiku-4-5-20251001",
  "latency_ms": 412
}
```

---

### 其他接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/v1/task/{id}` | 查询异步任务结果 |
| GET  | `/v1/models`    | 查看模型列表及权重 |
| GET  | `/v1/stats`     | 运行时统计数据 |
| GET  | `/v1/health`    | 健康检查 |

---

## ⚙️ 配置说明

所有配置通过环境变量或 `.env` 文件设置：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `ANTHROPIC_API_KEY` | — | **必填**，Anthropic API 密钥 |
| `PORT` | `8080` | 服务监听端口 |
| `APP_ENV` | `production` | 运行环境 |
| `API_TIMEOUT` | `10` | 单次 AI 请求超时（秒） |
| `MAX_RETRIES` | `2` | 故障转移重试次数 |
| `CACHE_DRIVER` | `memory` | 缓存驱动：`memory` \| `redis` |
| `CACHE_TTL` | `60` | 缓存时长（秒），`0` 禁用 |
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis 地址 |
| `ENABLE_AUTH` | `false` | 是否启用 API 鉴权 |
| `ALLOWED_KEYS` | — | 项目密钥列表（逗号分隔） |
| `LOG_LEVEL` | `info` | 日志级别 |
| `LOG_DIR` | `./logs` | 日志目录 |

---

## 🤖 模型调度策略

| 模型 | 权重 | 特点 |
|------|------|------|
| claude-sonnet-4-20250514 | 60% | 主力：精度与速度均衡 |
| claude-haiku-4-5-20251001 | 30% | 快速：低延迟高并发 |
| claude-opus-4-20250514 | 10% | 精准：复杂内容兜底 |

**故障转移流程：**
```
请求进入 → 加权随机选主力模型
              ↓ 失败（超时/报错）
           自动切换下一优先级模型
              ↓ 全部失败
           返回 verdict=flagged，转人工队列
```

---

## 📊 审核结果说明

| verdict | 含义 | PHP 插件行为 |
|---------|------|-------------|
| `approved` | 内容正常 | 直接放行，正常发布 |
| `flagged` | 存在疑虑 | 设 `status=pending`，待人工复核 |
| `rejected` | 明确违规 | 直接返回 403，内容不入库 |

---

## 🔧 常见问题

**Q: Go 版本和 PHP 版本有什么区别？**
> Go 版本性能更高，内存占用极低（约 15MB），适合高并发场景。两者 API 接口完全兼容，PHP YAF 项目切换时只需修改配置中的服务地址即可。

**Q: 审核服务挂了怎么办？**
> 客户端 `ContentModerationService` 所有异常情况均安全降级，返回 `verdict=flagged`，内容进入人工审核队列，不会误拦用户。

**Q: 相同内容重复提交？**
> 默认启用 60 秒内存缓存，相同内容直接返回上次结果，节省 API 调用费用。

**Q: 如何升级模型？**
> 修改 `.env` 中的 `MODELS_CONFIG` 或直接在 `config.go` 的 `defaultModels()` 中调整，重启服务即生效。

---

## 📄 License

MIT License
