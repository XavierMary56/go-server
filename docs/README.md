# go-server AI 内容审核服务

本项目是一个高性能、多模型调度的 AI 内容审核服务，支持同步/异步审核，提供完整的管理后台和审计日志功能。

## 1. 快速启动

### Docker 启动
```bash
cp .env.production .env
# 编辑 .env，填入 ANTHROPIC_API_KEY
docker compose up -d
```

### 二进制启动
```bash
go build -o moderation-server ./cmd/server
./moderation-server
```

---

## 2. 核心文档

- **[API 接口文档 (V1/V2 & Admin)](docs/API_GUIDE.md)**: 业务对接与管理接口完整说明。
- **[管理后台](http://localhost:8080/admin/)**: 动态管理项目密钥、模型权重和查看统计。

---

## 3. 技术栈

- **后端**: Go (Gin-like routing, MariaDB, Redis)
- **模型**: Anthropic Claude, OpenAI, Grok
- **存储**: MariaDB (配置/密钥), 文件系统 (审计日志), Redis (缓存)

---

## 4. 运维命令

| 脚本 | 说明 |
|------|------|
| `bash manage-keys.sh list` | 列出所有接入密钥 |
| `bash monitor.sh status` | 查看服务运行状态与指标 |
| `bash monitor.sh audit` | 查看实时审计日志 |
| `bash deploy.sh` | 自动化部署脚本 |

---

## 5. 日志说明

- **应用日志**: `logs/moderation_YYYY-MM-DD.log`
- **审计日志**: `logs/audit/<project_id>/audit_YYYY-MM-DD.log` (按项目隔离存储)

🤖 Generated with [Claude Code](https://claude.com/claude-code)