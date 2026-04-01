# go-server — AI 内容审核服务

Go 实现的高性能内容审核服务，支持多模型自动调度、同步/异步审核、项目级 API 密钥管理和嵌入式管理后台。

---

## 1. 核心目录结构

```
go-server/
├── cmd/server/main.go          # 程序入口，组装服务并启动 HTTP
├── internal/
│   ├── api/v1/                 # 公共 API V1（/v1/moderate）
│   ├── api/v2/                 # 公共 API V2（/v2/moderations）
│   ├── admin/                  # 管理 API + 嵌入式管理后台 UI
│   ├── service/                # 核心审核引擎、模型调度、Hard Rules 规则库
│   ├── storage/                # MariaDB 持久化层，含自动迁移
│   └── audit/                  # 审计日志写入（按项目分目录）
├── docs/
│   ├── README.md               # 项目总览与快速上手文档
│   └── API_GUIDE.md            # 完整 API 接口说明 (V1/V2/Admin)
├── deploy.sh                   # 自动化生产部署脚本
└── docker-compose.yml          # 本地/生产容器编排
```

---

## 2. 快速启动

```bash
# 1. 复制配置文件
cp .env.production .env
# 2. 自动化部署 (包含构建、权限设置与启动)
bash deploy.sh
```

- **本地 API 地址**: `http://localhost:888` (由 Docker 映射)
- **生产 API 地址**: `https://zyaokkmo.cc`
- **管理后台**: `http://localhost:888/admin/` 或 `https://zyaokkmo.cc/admin/`

---

## 3. 核心功能与拦截逻辑

系统采用 **“本地规则优先 + AI 模型补位”** 的策略：
- **Hard Rules (本地拦截)**: 在 `internal/service/dictionary.go` 中定义。命中 QQ、微信、Telegram、BC (博彩)、指定 URL (.cc) 等关键字时直接拒绝，不消耗 AI 额度。
- **AI 审核**: 规则未命中时，自动调度 Claude (3.5 Sonnet/Haiku/Opus)、OpenAI 或 Grok 模型进行深度判定。

---

## 4. 详细文档

- **[快速上手与项目总览](docs/README.md)**
- **[API 接入与管理指南](docs/API_GUIDE.md)**

🤖 Generated with [Claude Code](https://claude.com/claude-code)
