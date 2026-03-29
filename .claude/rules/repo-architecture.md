---
description: 当前 Go 仓库的架构速查与改动落点指引
---

# 当前 Go 仓库架构速查

本规则用于帮助快速判断功能、问题和改动应该落在哪一层。

## 一、核心结构
- `cmd/server/main.go`：程序组装入口，负责加载配置、初始化日志、组装服务并启动 HTTP 服务。
- `internal/handler/handler.go`：公共 API 层，负责路由、请求校验、鉴权、限流、异步任务状态与 JSON 响应。
- `internal/service/`：核心审核逻辑，处理规则、缓存、请求去重、模型调度、供应商调用与统计。
- `internal/admin/`：管理接口与嵌入式管理后台 UI。
- `internal/storage/storage.go`：运行时管理数据持久化层，保存项目密钥、供应商密钥、模型配置与管理设置。
- `internal/audit/audit.go`：将审计事件写入文件系统中的项目日志目录。
- `internal/config/config.go`：加载 `.env` 与环境变量。

## 二、关键运行事实
- 公共请求鉴权在 `internal/handler/handler.go`，不是单独的 middleware 包。
- `X-Project-Key` 优先走 SQLite 管理数据校验；只有数据库鉴权不可用时才回退到配置项。
- 管理端项目日志接口读取的是审计日志文件，不是 SQLite 中的日志表。
- 运行时状态（如项目密钥、供应商配置、模型配置）默认优先来自 SQLite，而不是静态 `.env`。

## 三、改动落点映射
- 改公共 API 行为：优先看 `internal/handler/` 与 `internal/service/`
- 改审核流程、模型调度、规则判断：优先看 `internal/service/`
- 改管理接口或管理后台页面：优先看 `internal/admin/`
- 改运行时配置来源：优先看 `internal/config/` 与 `internal/storage/`
- 改审计日志写入与读取：优先看 `internal/audit/` 与管理端日志查询逻辑

## 四、常见误区
- 不要把数据库管理态误判为仅靠 `.env` 驱动。
- 不要把审计日志查询误判为数据库查询。
- 不要在不确认架构前，把鉴权逻辑拆到仓库中并不存在的独立中间件层。
