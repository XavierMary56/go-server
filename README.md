# go-server — AI 内容审核服务

Go 实现的高性能内容审核服务，支持多模型自动调度、同步/异步审核、项目级 API 密钥管理和嵌入式管理后台。

---

## 1. 核心目录结构

```
go-server/
├── cmd/server/main.go          # 程序入口，组装服务并启动 HTTP
├── internal/
│   ├── api/v1/                 # 公共 API V1（/v1/moderate）
│   ├── api/v2/                 # 公共 API V2（/v2/moderations，推荐）
│   ├── admin/                  # 管理 API + 嵌入式管理后台 UI
│   ├── service/                # 核心审核引擎、模型调度、Hard Rules 规则库
│   ├── storage/                # MariaDB 持久化层，含自动迁移
│   ├── config/                 # 配置加载（.env + 环境变量）
│   ├── audit/                  # 审计日志写入（按项目分目录）
│   └── logger/                 # 结构化日志
├── docs/
│   ├── API_GUIDE.md            # 完整 API 接口说明 (V1/V2/Admin)
│   └── AGENT_TOOL_SPEC.md     # AI Agent Tool 定义与集成指南
├── deploy.sh                   # 自动化生产部署脚本
├── docker-compose.yml          # 本地/生产容器编排
└── Dockerfile                  # 多阶段构建镜像
```

---

## 2. 快速启动

### 本地环境（Docker）

```bash
# 1. 复制配置文件
cp .env.production .env

# 2. 启动服务（含 MariaDB + Redis）
docker-compose up -d --build
```

### 生产部署

```bash
git pull origin main
docker-compose up -d --build
```

### 访问地址

| 环境 | API 地址 | 管理后台 |
|------|---------|---------|
| 本地 | `http://localhost:888` | `http://localhost:888/admin/` |
| 生产 | `https://zyaokkmo.cc` | `https://zyaokkmo.cc/admin/` |

### 健康检查

```bash
curl http://localhost:888/v1/health
```

---

## 3. 审核流程

系统采用 **"本地规则优先 + AI 模型补位"** 的策略：

```
请求 → Hard Rules 规则引擎（0ms）
        ├─ 命中 → 直接 rejected（不消耗 AI 额度）
        └─ 未命中 → 缓存检查
                     ├─ 命中缓存 → 返回缓存结果
                     └─ 未命中 → AI 模型队列（按优先级调度）
                                  ├─ 成功 → 返回模型结果
                                  └─ 全部失败 → flagged（转人工队列）
```

### Hard Rules（本地拦截）

在 `internal/service/dictionary.go` 中定义，以下内容直接拒绝，不经过 AI 模型：

| 类别 | 示例关键词 |
|------|-----------|
| 广告导流 | QQ、微信、Telegram、TG、加群、加v 及变体 |
| 博彩诈骗 | 赌博、博彩、彩票、上分、盘口、庄家、刷单 |
| 色情成人 | 约炮、裸聊、援交 |
| 毒品违禁 | 吸毒、大麻、K粉 |
| 政治敏感 | 涉政敏感词 |
| 暴力恐怖 | 杀人、恐怖袭击 |

### 模型队列

支持多供应商自动调度与故障切换：

| 供应商 | 支持模型 |
|--------|---------|
| Anthropic | claude-haiku-4-5、claude-sonnet-4-5 |
| OpenAI | gpt-4o-mini（推荐）、gpt-4o、o1/o3/o4 系列 |
| xAI | grok 系列 |

- `model=auto` 时按权重随机选择，优先级高的模型优先
- 供应商密钥全部 unhealthy 时，该供应商模型自动从队列中移除

---

## 4. 技术架构

- **语言**: Go
- **存储**: MariaDB（项目密钥、供应商密钥、模型配置）
- **缓存**: Redis（审核结果缓存，默认 60s TTL）
- **日志**: 文件系统（按项目隔离的审计日志）
- **部署**: Docker Compose（Go 服务 + MariaDB + Redis）

---

## 5. 文档导航

| 文档 | 说明 |
|------|------|
| [API 接口文档](docs/API_GUIDE.md) | 完整的 V1/V2/Admin API 说明、请求参数、响应格式 |
| [Agent Tool 定义](docs/AGENT_TOOL_SPEC.md) | AI Agent SDK 集成指南、Tool Schema、PHP/Python 示例 |

---

## 6. 运维常用命令

| 命令 | 说明 |
|------|------|
| `docker-compose up -d --build` | 构建并启动服务 |
| `docker-compose logs -f moderation` | 查看服务实时日志 |
| `bash deploy.sh` | 自动化构建并重启 |
| `bash monitor.sh status` | 查看实时 QPS、拦截率 |
| `bash manage-keys.sh help` | 管理项目密钥 |
| `curl localhost:888/v1/health` | 健康检查 |
| `curl localhost:888/v1/stats` | 查看统计指标 |
| `curl localhost:888/v1/models` | 查看可用模型 |
