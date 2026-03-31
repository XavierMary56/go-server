# go-server — AI 内容审核服务

Go 实现的高性能内容审核服务，支持多模型自动调度、同步/异步审核、项目级 API 密钥管理和嵌入式管理后台。

---

## 目录结构

```
go-server/
├── cmd/server/main.go          # 程序入口，组装服务并启动 HTTP
├── internal/
│   ├── api/v1/                 # 公共 API V1（/v1/moderate）
│   ├── api/v2/                 # 公共 API V2（/v2/moderations）
│   ├── admin/                  # 管理 API + 嵌入式管理后台 UI
│   │   └── static/             # 管理后台静态文件（CSS/JS/HTML）
│   ├── handler/                # HTTP 路由、鉴权、限流中间件
│   ├── service/                # 核心审核引擎、模型调度、缓存
│   ├── storage/                # MariaDB 持久化层，含启动自动迁移
│   ├── audit/                  # 审计日志写入（按项目分目录）
│   ├── config/                 # 环境变量 / .env 配置加载
│   └── logger/                 # 结构化日志
├── deploy/
│   ├── init.sql                # 数据库全量初始化脚本（新环境用）
│   ├── migrations/             # 数据库迁移说明（迁移由服务启动自动执行）
│   ├── nginx.conf              # Nginx 反向代理配置
│   └── moderation.service      # systemd 服务配置
├── docs/README.md              # 完整 API 文档（V1/V2 接口参数、枚举值）
├── examples/php-yaf-client/    # PHP YAF 客户端示例
├── pkg/client/                 # Go 客户端 SDK
├── tests/                      # 测试文档和脚本
├── docker-compose.yml          # 本地/生产容器编排
├── Dockerfile                  # 镜像构建
├── deploy.sh                   # 生产部署脚本
├── .env.example                # 配置项说明
└── .env.production             # 生产配置模板（复制为 .env 使用）
```

---

## 快速启动

```bash
# 复制配置文件
cp .env.production .env
# 编辑 .env，填入 ANTHROPIC_API_KEY、ADMIN_TOKEN 等

# 启动服务（MariaDB + Redis + 审核服务）
docker-compose up -d

# 验证服务
curl http://localhost:888/v1/health
```

管理后台：http://localhost:888/admin/

---

## 常用命令

```bash
# 构建
go build -o moderation-server ./cmd/server

# 运行测试
go test ./...

# 查看日志
docker-compose logs -f moderation

# 重启服务
docker-compose restart moderation

# 重新构建镜像（修改 Go 代码后）
docker-compose build moderation && docker-compose up -d moderation
```

---

## 核心接口

> 完整参数说明见 [docs/README.md](docs/README.md)

| 接口 | 说明 | 鉴权 |
|------|------|------|
| `POST /v1/moderate` | 同步内容审核 | `X-Project-Key` |
| `POST /v1/moderate/async` | 异步内容审核 | `X-Project-Key` |
| `GET /v1/task/{task_id}` | 查询异步任务结果 | `X-Project-Key` |
| `POST /v2/moderations` | 同步审核（V2 响应结构） | `X-Project-Key` |
| `POST /v2/moderations/async` | 异步审核（V2） | `X-Project-Key` |
| `GET /v1/health` | 健康检查 | 无 |
| `GET /v1/admin/*` | 管理 API | `Bearer ADMIN_TOKEN` |

**接入只需判断一个字段：`verdict === "rejected"` 为拒绝，其余均为通过。**

---

## 主要配置项

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 服务监听端口 |
| `ANTHROPIC_API_KEY` | — | Anthropic API Key（必填） |
| `ENABLE_AUTH` | `false` | 是否开启项目密钥鉴权 |
| `ENABLE_ADMIN_API` | `true` | 是否开启管理 API |
| `ADMIN_TOKEN` | — | 管理后台访问令牌 |
| `ENABLE_AUDIT` | `true` | 是否开启审计日志 |
| `DB_HOST` | `mariadb` | MariaDB 地址 |
| `REDIS_ADDR` | `redis:6379` | Redis 地址 |

---

## 开发说明

- 修改 `internal/admin/static/` 下的静态文件后**直接刷新浏览器**即可（volume 挂载热更新）
- 更新静态资源版本号只需修改 `internal/admin/web.go` 中的 `staticVersion` 常量
- 数据库结构变更在 `internal/storage/storage.go` 的 `migrate()` 函数中追加，服务启动时自动执行
