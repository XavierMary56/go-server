# CLAUDE.md

本文件为 Claude Code（claude.ai/code）在此仓库中工作时提供协作指引。

## 常用命令

### 构建与运行
- 本地构建二进制：`go build -o moderation-server ./cmd/server`
- 交叉编译 Linux 二进制：`GOOS=linux GOARCH=amd64 go build -o moderation-server ./cmd/server`
- 直接从源码运行：`go run ./cmd/server`
- 使用 Docker 启动本地环境：`docker-compose up -d`
- 查看容器日志：`docker-compose logs -f moderation`
- 停止本地环境：`docker-compose down`
- 生产风格启动脚本：`bash deploy.sh`

### 测试
- 运行全部测试：`go test ./...`
- 运行单个包测试：`go test ./internal/service`
- 运行规则相关测试：`go test ./internal/service -run Rules`
- 运行批量回归测试：`go test ./internal/service -run Bulk`
- 运行指定测试：`go test ./internal/service -run TestServiceBehavior`

### 运维脚本
- 查看或管理项目密钥：`bash manage-keys.sh help`
- 检查服务状态与指标：`bash monitor.sh status`
- 查看最近日志：`bash monitor.sh logs -n 50`
- 查看审计日志：`bash monitor.sh audit --project <project_id>`

## 仓库说明
- 这个仓库之前并没有现成的 `CLAUDE.md`。
- 遵循 `.github/copilot-instructions.md`：在代码改动或总结中，优先链接到 `docs/` 下现有文档，而不是重复编写部署或集成说明。
- 当前仓库没有专门的 lint 命令，也没有 Makefile；不要在改动或说明中虚构这些内容。
- 测试通常与被测包放在一起；顶层 `tests/` 目录仅用于测试文档、脚本和夹具。

## 高层架构

### 运行时结构
- `cmd/server/main.go` 是组装入口。它负责加载基于环境变量的配置、初始化结构化日志、按需连接 MariaDB 管理/鉴权数据库、组装审核服务和审计日志器、注册公开与管理端路由，并启动 HTTP 服务。
- 公共 API 实现在 `internal/handler/handler.go`。这一层负责路由、请求校验、鉴权/限流中间逻辑、异步任务状态管理，以及 JSON 响应。
- 核心审核引擎位于 `internal/service/`。`moderation.go` 负责请求默认值、硬阻断规则短路、缓存查询、并发请求去重、模型队列与故障切换、供应商 API 调用，以及内存态统计信息。

### 模型与供应商流程
- 服务采用“模型队列”设计，而不是绑定单一供应商。供应商选择由模型 ID 推导，Anthropic/OpenAI/Grok 的密钥优先从 MariaDB 管理数据中读取，环境变量作为兜底。
- `internal/storage/storage.go` 是运行时状态的持久化层：保存项目密钥、供应商密钥、模型配置以及轻量级管理设置。如果启用了鉴权或管理 API，应默认认为运行时状态来自 MariaDB，而不是静态 `.env`。
- `internal/admin/` 提供 `/v1/admin/*` 管理接口，包括项目密钥管理、供应商密钥检查、模型配置管理、项目日志查询、项目统计以及管理 token 设置。同时该包也负责提供嵌入式管理后台 Web UI。

### 鉴权、审计与监控
- 公共请求鉴权位于 `internal/handler/handler.go`，而不是单独的中间件包。它会优先根据 MariaDB 校验 `X-Project-Key`，只有在数据库鉴权不可用时才回退到配置中的 `ALLOWED_KEYS`。
- 限流按项目密钥维度进行，并在 handler 层以内存方式跟踪。
- `internal/audit/audit.go` 会异步把结构化 JSON 审计事件写入配置的审计日志根目录下、按项目分目录存储。管理端项目日志接口直接读取这些文件，而不是从 MariaDB 读取。
- `internal/service` 维护 `/v1/stats` 所需的进程内审核统计；这是轻量级运行时状态，不是持久化报表存储。

### 配置与部署
- `internal/config/config.go` 会按需加载 `.env`，再叠加环境变量。关键开关包括 `ENABLE_AUTH`、`ENABLE_ADMIN_API`、`ENABLE_AUDIT` 以及各供应商凭据。
- `docker-compose.yml` 是本地部署的主要方式。它会启动 Go 服务、MariaDB 和 Redis，将 `./logs` 挂载为日志目录，MariaDB 数据持久化到 `moderation-data` 卷中。数据库连接通过 `DB_HOST / DB_PORT / DB_USER / DB_PASS / DB_NAME` 配置，默认连接 `mariadb:3306`，库名 `moderation`。
- Docker 镜像由 `Dockerfile` 构建，运行时基于精简 Alpine 环境，仅包含编译后的 `./cmd/server` 二进制文件。

## 文档导航
- 文档入口：`docs/README.md`
- 部署与 API 细节：`docs/02-deployment/API_AND_DEPLOYMENT.md`
- 集成说明：`docs/03-integration/CLIENT_INTEGRATION.md`
- 鉴权、监控与脚本：`docs/04-operations/AUTH_AND_MONITORING.md` 与 `docs/04-operations/SCRIPTS_GUIDE.md`
