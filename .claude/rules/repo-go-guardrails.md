---
description: 当前 Go 仓库的通用硬约束与命令护栏
---

# 当前 Go 仓库通用护栏

本规则用于约束在当前仓库中的默认行为，避免建议偏离仓库真实结构与命令体系。

## 一、只使用仓库真实存在的命令
优先使用 `CLAUDE.md` 中已经列出的命令与脚本，例如：

- `go build -o moderation-server ./cmd/server`
- `go run ./cmd/server`
- `go test ./...`
- `go test ./internal/service`
- `bash deploy.sh`
- `bash manage-keys.sh help`
- `bash monitor.sh status`

不要虚构以下内容，除非仓库后续明确新增并写入 `CLAUDE.md` 或代码库：

- `Makefile` / `make test` / `make lint`
- 不存在的 lint 命令
- 不存在的 coverage 脚本
- 不存在的 CI/CD pipeline 命令

## 二、文档约束
- 优先链接 `docs/` 目录下现有文档。
- 不要重复撰写部署或集成说明，除非用户明确要求新增文档。
- 如果仓库已有对应文档入口，优先引用而不是重写。

## 三、测试与验证约束
- 先从最小相关范围验证，再扩大范围。
- 优先使用仓库中已有的 `go test` 路径与模式。
- 不要默认引入新测试框架或新工具链。

## 四、改动前的默认认知
- 这是一个 Go 服务仓库，不是多语言模板仓库。
- 代码建议、测试建议、排查建议应优先贴合 Go 工作流。
- 如果仓库事实与通用经验冲突，以当前仓库中的 `CLAUDE.md` 和实际代码为准。
