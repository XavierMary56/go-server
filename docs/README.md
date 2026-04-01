# go-server AI 内容审核服务

本项目是一个高性能、多模型调度的 AI 内容审核服务，支持同步/异步审核，提供完整的管理后台和审计日志功能。

## 1. 快速启动

### 本地测试环境 (Docker)
```bash
# 1. 准备环境
cp .env.production .env
# 2. 自动化部署
bash deploy.sh
```
- **本地 API 地址**: `http://localhost:888` (由 Docker 映射)
- **健康检查**: `curl http://localhost:888/v1/health`

### 生产环境部署
- **生产 API 地址**: `https://zyaokkmo.cc`
- **管理后台**: [zyaokkmo.cc/admin/](https://zyaokkmo.cc/admin/)

---

## 2. 核心文档

- **[API 接口说明 (V1/V2 & Admin)](docs/API_GUIDE.md)**: 业务对接与管理接口完整说明。
- **[运维与监控指南](docs/API_GUIDE.md#5-运维指南-简版)**: 日志、监控与脚本说明。

---

## 3. 审核规则说明 (Hard Rules)

系统内置了严苛的本地硬阻断规则，以下内容将直接被拦截 (Rejected)，不经过 AI 模型：
- **URL/域名**: 包含 `.cc`、`.xyz`、`http://` 等地址 (如 `zyaokkmo.cc`)。
- **博彩/诈骗**: 包含 `BC`、`博彩`、`上分`、`盘口`、`庄家` 等。
- **引流/联系方式**: 包含 `QQ`、`Telegram`、`TG`、`加群`、`微信`、`加v` 及其变体。
- **其他**: 涉政、暴恐、毒品、严重色情等关键字。

---

## 4. 技术架构

- **核心**: Go (Gin-like routing, MariaDB, Redis)
- **模型队列**: 支持 Anthropic Claude (3.5 Sonnet/Haiku/Opus), OpenAI, Grok。
- **持久化**: MariaDB (存储项目密钥、模型权重)，文件系统 (按项目隔离的审计日志)。

---

## 5. 运维常用命令

| 命令 | 说明 |
|------|------|
| `bash monitor.sh status` | 查看实时 QPS、拦截率等指标 |
| `bash deploy.sh` | 自动化构建并重启服务 |
| `docker-compose logs -f` | 查看容器实时日志 |

🤖 Generated with [Claude Code](https://claude.com/claude-code)
