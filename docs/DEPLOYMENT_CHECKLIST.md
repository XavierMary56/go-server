# 📦 部署物料清单

> **最后更新**: 2026-03-23
> **部署目标**: 76.13.218.203:22 (ai.a889.cloud)
> **完成度**: ✅ 100% - 所有物料已准备就绪

---

## 📋 物料清单总览

### ✅ 代码与配置文件
| 文件/目录 | 用途 | 位置 | 状态 |
|---------|------|------|-----|
| `cmd/server/main.go` | Go 应用入口 | 项目根目录 | ✅ |
| `internal/` | 核心模块集合 | 项目根目录 | ✅ |
| `internal/config/config.go` | 配置加载 | internal/ | ✅ |
| `internal/handler/handler.go` | HTTP 路由和处理 | internal/ | ✅ |
| `internal/service/` | 业务逻辑（审核、缓存） | internal/ | ✅ |
| `internal/logger/logger.go` | 日志管理 | internal/ | ✅ |
| `internal/admin/` | 管理 API 和日志查询 | internal/ | ✅ |
| `internal/auth/auth.go` | API 密钥认证 | internal/ | ✅ |
| `internal/audit/audit.go` | 审计日志（项目隔离） | internal/ | ✅ |
| `internal/monitor/metrics.go` | 性能监控 | internal/ | ✅ |
| `go.mod` | Go 依赖管理 | 项目根目录 | ✅ |
| `.env.production` | 生产环境配置 | 项目根目录 | ✅ |

### ✅ 部署配置文件
| 文件 | 用途 | 位置 | 状态 |
|-----|------|------|-----|
| `Dockerfile` | Docker 构建配置 | 项目根目录 | ✅ |
| `docker-compose.yml` | 容器编排 | 项目根目录 | ✅ |
| `deploy/nginx.conf.production` | Nginx 反向代理 | deploy/ | ✅ |
| `deploy/moderation.service` | systemd 服务配置 | deploy/ | ✅ |
| `.env.production.example` | 配置模板 | docs/examples/ | ✅ |

### ✅ 部署脚本
| 脚本 | 用途 | 位置 | 状态 |
|-----|------|------|-----|
| `deploy.sh` | 自动化部署脚本 | 项目根目录 | ✅ |
| `manage-keys.sh` | API 密钥生命周期管理 | 项目根目录 | ✅ |
| `monitor.sh` | 服务监控与维护 | 项目根目录 | ✅ |
| 可选: `push-go-to-github.sh` | 推送代码到 GitHub | 项目根目录 | ✅ |

### ✅ 文档（完整覆盖）
| 文档 | 用途 | 位置 | 状态 |
|-----|------|------|-----|
| `docs/README.md` | 📍 文档导航中心 | docs/ | ✅ |
| `docs/00-START-HERE.md` | 5分钟快速导览 | docs/01-gettingstarted/ | ✅ |
| `docs/DEPLOYMENT_CHECKLIST.md` | 部署前检查 | docs/01-gettingstarted/ | ✅ |
| `docs/API_AND_DEPLOYMENT.md` | ⭐ API文档 + 部署指南 | docs/02-deployment/ | ✅ |
| `docs/DEPLOYMENT.md` | 详细部署指南 | docs/02-deployment/ | ✅ |
| `docs/CLIENT_INTEGRATION.md` | 客户对接指南 + SDK | docs/03-integration/ | ✅ |
| `docs/INTEGRATION_GUIDE.md` | 代码集成步骤 | docs/03-integration/ | ✅ |
| `docs/AUTH_AND_MONITORING.md` | 鉴权 + 监控 + 日志 | docs/04-operations/ | ✅ |
| `docs/SCRIPTS_GUIDE.md` | 脚本使用手册 | docs/04-operations/ | ✅ |

### ✅ 示例与模板
| 文件 | 用途 | 位置 | 状态 |
|-----|------|------|-----|
| `docs/examples/.env.production.example` | 环境变量模板 | docs/examples/ | ✅ |
| `docs/examples/nginx.conf.production` | Nginx 配置示例 | docs/examples/ | ✅ |
| `docs/examples/moderation.service` | systemd 服务示例 | docs/examples/ | ✅ |
| `docs/examples/README.md` | 示例文件说明 | docs/examples/ | ✅ |

---

## 🔧 依赖环境要求

### 开发/编译环境
```bash
✅ Go        >= 1.21
✅ Git       (版本控制)
✅ bash      (脚本执行)
```

### 生产运行环境
```bash
✅ Docker    >= 20.10 (方式A: Docker 部署)
   ├─ docker-compose >= 1.29
   └─ Linux kernel >= 5.4

或

✅ 二进制部署 (方式B)
   ├─ Linux 操作系统
   ├─ systemd (服务管理)
   └─ Nginx (反向代理)

可选:
⭐ Redis    (缓存加速) - 默认使用内存缓存
```

### 外部服务
```bash
✅ Anthropic API
   └─ API Key (必填)
   └─ 网络连接 (生产 API)
```

---

## 📦 部署方式选择矩阵

### 方式 A：Docker Compose（推荐 ⭐）
**最简单、最快速、最完整**

所需物料:
```
✅ docker-compose.yml
✅ Dockerfile
✅ .env.production (配置)
✅ 所有 internal/ 源代码
✅ go.mod 和 go.sum
```

启动时间: **2-3 分钟**
学习成本: **低**
适用场景: 快速部署、测试、小型生产环境

### 方式 B：二进制 + systemd
**更轻量、便于管理、适合 Linux 生产环境**

所需物料:
```
✅ 预编译二进制 (或 go.mod 编译)
✅ .env 配置文件
✅ deploy/moderation.service
✅ systemd 配置
✅ deploy/nginx.conf.production
```

启动时间: **编译 2-3 分钟 + 启动 1 分钟**
学习成本: **中**
适用场景: 生产环境、需要定制的环境

### 方式 C：Kubernetes
**企业级、高可用、容器编排**

所需物料:
```
✅ Dockerfile (基础镜像)
✅ 构建镜像
✅ K8S YAML 清单
✅ ConfigMap/Secret (配置)
```

启动时间: **变长**
学习成本: **高**
适用场景: 互联网公司、需要自动扩展

---

## 📂 完整文件清单（按需下载）

### 核心文件（必须）
```
go-server/
├── cmd/server/main.go                         ✅ 必须
├── internal/
│   ├── config/config.go                       ✅ 必须
│   ├── handler/handler.go                     ✅ 必须
│   ├── service/                               ✅ 必须
│   │   ├── moderation.go
│   │   └── cache.go
│   ├── logger/logger.go                       ✅ 必须
│   ├── admin/                                 ✅ 推荐
│   │   ├── admin.go
│   │   └── logs.go
│   ├── auth/auth.go                           ✅ 推荐
│   ├── audit/audit.go                         ✅ 推荐
│   └── monitor/metrics.go                     ✅ 推荐
├── go.mod                                      ✅ 必须
├── go.sum                                      ✅ 依赖
├── Dockerfile                                  ✅ 必须
└── docker-compose.yml                          ✅ 必须
```

### 配置与脚本（必须）
```
go-server/
├── .env.production                             ✅ 必须
├── deploy.sh                                   ✅ 必须
├── deploy/
│   ├── nginx.conf.production                  ✅ 推荐
│   └── moderation.service                     ✅ 推荐
├── manage-keys.sh                             ✅ 推荐
└── monitor.sh                                 ✅ 推荐
```

### 文档（参考）
```
go-server/docs/
├── README.md                                  📖 导航
├── 01-gettingstarted/
│   ├── 00-START-HERE.md                       📖 快速了解
│   └── DEPLOYMENT_CHECKLIST.md                📖 部署检查
├── 02-deployment/
│   ├── API_AND_DEPLOYMENT.md                  📖 ⭐ 部署指南
│   ├── DEPLOYMENT.md                          📖 详细部署
│   └── examples/                              📖 示例配置
├── 03-integration/
│   ├── CLIENT_INTEGRATION.md                  📖 ⭐ 对接指南
│   └── INTEGRATION_GUIDE.md                   📖 代码集成
└── 04-operations/
    ├── AUTH_AND_MONITORING.md                 📖 运维管理
    └── SCRIPTS_GUIDE.md                       📖 脚本使用
```

---

## ⚙️ 配置物料清单

### 必填配置项
```bash
# .env.production 中必须配置
✅ ANTHROPIC_API_KEY=sk-ant-...              # Anthropic 密钥（必填）
✅ PORT=8080                                  # 监听端口

# 推荐配置
✅ APP_ENV=production
✅ ENABLE_AUTH=true                          # 启用鉴权
✅ ALLOWED_KEYS=proj_demo_xxx,proj_forum_yyy # 项目密钥列表
✅ ENABLE_ADMIN_API=true                     # 启用管理 API
✅ ADMIN_TOKEN=your-secret-token             # 管理员令牌
✅ ENABLE_AUDIT=true                         # 启用审计日志
✅ ENABLE_METRICS=true                       # 启用性能监控
```

### 可选配置项
```bash
# 缓存
CACHE_DRIVER=redis                    # memory | redis
CACHE_TTL=60                          # 缓存时间(秒)
REDIS_ADDR=127.0.0.1:6379           # Redis 地址

# API 限制
API_TIMEOUT=10                        # 超时时间(秒)
MAX_RETRIES=2                         # 重试次数

# 日志
LOG_LEVEL=info                        # debug|info|warn|error
LOG_DIR=./logs                        # 日志目录

# 监控
METRICS_PORT=9090                     # Prometheus 指标端口
```

---

## 🚀 部署步骤总结

### 第 1 步：准备阶段（5 分钟）
```bash
✅ 获取所有源代码和配置文件
✅ 准备 .env.production 配置
✅ 填入 ANTHROPIC_API_KEY
✅ 生成或准备 API 项目密钥
```

### 第 2 步：选择部署方式（1 分钟）
```bash
✅ 方式 A：Docker (推荐)
✅ 方式 B：二进制 + systemd
✅ 方式 C：其他自定义方式
```

### 第 3 步：部署执行（2-5 分钟）

**如果选择 Docker:**
```bash
✅ docker-compose up -d
✅ docker-compose logs -f
```

**如果选择二进制:**
```bash
✅ go build -o server ./cmd/server
✅ systemctl enable moderation
✅ systemctl start moderation
```

### 第 4 步：验证部署（2 分钟）
```bash
✅ curl http://localhost:8080/v1/health
✅ 验证返回 {"status":"ok",...}
✅ 测试 API: curl -X POST http://localhost:8080/v1/moderate ...
```

### 第 5 步：配置域名与 HTTPS（10 分钟）
```bash
✅ 配置 Nginx 反向代理
✅ 申请 SSL 证书
✅ 配置 ai.a889.cloud 指向 76.13.218.203
✅ 验证 https://ai.a889.cloud/v1/health
```

### 第 6 步：监控与维护（5 分钟）
```bash
✅ 启用审计日志
✅ 启用性能监控
✅ 配置日志轮转
✅ 设置告警规则
```

---

## 📊 物料完整度检查表

### 源代码
- [x] main.go 入口
- [x] config 配置模块
- [x] handler HTTP 处理
- [x] service 业务逻辑
- [x] logger 日志
- [x] admin 管理 API
- [x] auth 认证
- [x] audit 审计日志
- [x] monitor 监控

### 配置与部署
- [x] Dockerfile
- [x] docker-compose.yml
- [x] .env.production 模板
- [x] .env.production 配置值
- [x] nginx.conf.production
- [x] moderation.service

### 脚本
- [x] deploy.sh
- [x] manage-keys.sh
- [x] monitor.sh

### 文档
- [x] 部署指南
- [x] API 文档
- [x] 集成指南
- [x] 运维手册
- [x] 脚本使用说明

### 依赖
- [x] go.mod
- [x] go.sum

---

## 🎯 快速开始（按方式选择）

### ⚡ 最快方式：Docker（推荐）
```bash
# 1. 复制配置
cp docs/examples/.env.production.example .env
nano .env  # 填入 ANTHROPIC_API_KEY

# 2. 启动
docker-compose up -d

# 3. 验证
curl http://localhost:8080/v1/health
```
**耗时**: 3 分钟
**难度**: ⭐️ 超简单

### 📦 生产方式：二进制 + systemd
```bash
# 1. 编译
go build -o moderation-server ./cmd/server

# 2. 配置
cp .env.production .env
vim .env

# 3. 安装
sudo cp moderation-server /opt/moderation/
sudo cp deploy/moderation.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable moderation
sudo systemctl start moderation

# 4. 验证
sudo systemctl status moderation
curl http://localhost:8080/v1/health
```
**耗时**: 8 分钟
**难度**: ⭐️⭐️ 简单

### 🌐 生产完整方式：带 Nginx + HTTPS
```bash
# 前置：完成上面的二进制部署

# 1. 配置 Nginx
sudo cp deploy/nginx.conf.production /etc/nginx/conf.d/moderation.conf
sudo vim /etc/nginx/conf.d/moderation.conf  # 修改域名

# 2. SSL 证书
sudo certbot certonly --standalone -d ai.a889.cloud

# 3. 重载 Nginx
sudo nginx -t && sudo systemctl reload nginx

# 4. 验证
curl https://ai.a889.cloud/v1/health
```
**耗时**: 15 分钟
**难度**: ⭐️⭐️⭐️ 中等

---

## ❌ 常见遗漏物料

### ❌ 遗漏：API 密钥
- 问题：部署后无法调用 API
- 原因：忘记配置 ANTHROPIC_API_KEY
- 解决：`nano .env` → 填入密钥 → 重启服务

### ❌ 遗漏：项目密钥
- 问题：请求被拒绝 (401 Unauthorized)
- 原因：ALLOWED_KEYS 为空或密钥不匹配
- 解决：使用 manage-keys.sh 生成密钥，或在 .env 中配置

### ❌ 遗漏：配置文件
- 问题：服务无法启动
- 原因：.env 文件不存在
- 解决：`cp .env.production .env`

### ❌ 遗漏：Dockerfile
- 问题：Docker 部署失败
- 原因：Dockerfile 不在项目根目录
- 解决：确保 Dockerfile 存在且在正确位置

### ❌ 遗漏：脚本执行权限
- 问题：bash: ./deploy.sh: Permission denied
- 原因：脚本没有执行权限
- 解决：`chmod +x deploy.sh manage-keys.sh monitor.sh`

---

## 📞 物料清单检查清单

在部署前，使用这个清单逐一检查：

```bash
# 源代码
[ ] cmd/server/main.go 存在
[ ] internal/ 目录完整
[ ] go.mod 存在
[ ] go.sum 存在

# 配置
[ ] .env.production 存在
[ ] ANTHROPIC_API_KEY 已填入
[ ] ALLOWED_KEYS 已配置

# 部署文件
[ ] Dockerfile 存在
[ ] docker-compose.yml 存在
[ ] deploy/ 目录存在

# 脚本
[ ] deploy.sh 存在且可执行
[ ] manage-keys.sh 存在且可执行
[ ] monitor.sh 存在且可执行

# 文档
[ ] docs/ 目录存在
[ ] 02-deployment/API_AND_DEPLOYMENT.md 存在
[ ] 03-integration/CLIENT_INTEGRATION.md 存在

# 环境
[ ] Go >= 1.21 (go version)
[ ] Docker >= 20.10 (docker --version)
[ ] docker-compose >= 1.29 (docker-compose --version)
[ ] bash 可用

# 网络
[ ] 服务器网络可访问
[ ] 76.13.218.203:22 可 SSH 登录
[ ] 可访问 Anthropic API
```

---

## 📞 技术支持

| 问题 | 文档位置 |
|-----|--------|
| 如何部署? | docs/02-deployment/API_AND_DEPLOYMENT.md |
| API 怎么用? | docs/02-deployment/API_AND_DEPLOYMENT.md |
| 怎么对接? | docs/03-integration/CLIENT_INTEGRATION.md |
| 密钥怎么管理? | docs/04-operations/SCRIPTS_GUIDE.md |
| 日志怎么查? | docs/04-operations/AUTH_AND_MONITORING.md |
| 怎么监控? | docs/04-operations/SCRIPTS_GUIDE.md |

---

**Status**: ✅ 所有物料已准备完毕，可随时部署！
