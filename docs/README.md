# 📚 AI 内容审核服务 - 文档中心

完整的部署、集成、对接文档库。

🔗 **文档位置**：所有文档都已按分类整理到子目录

---

## 🚀 快速开始（按需求选择）

| 🎯 我要... | 📖 读这个文档 | ⏱️ 耗时 |
|-----------|-------------|-------|
| **🌟 从零开始了解** | [01-gettingstarted/00-START-HERE.md](./01-gettingstarted/00-START-HERE.md) | 5分钟 |
| **👨‍💼 部署这套系统** | [02-deployment/API_AND_DEPLOYMENT.md](./02-deployment/API_AND_DEPLOYMENT.md) | 20分钟 |
| **👥 让新项目对接** | [03-integration/CLIENT_INTEGRATION.md](./03-integration/CLIENT_INTEGRATION.md) | 30分钟 |
| **⚙️ 配置和管理** | [04-operations/AUTH_AND_MONITORING.md](./04-operations/AUTH_AND_MONITORING.md) | 15分钟 |
| **🔨 运行脚本命令** | [04-operations/SCRIPTS_GUIDE.md](./04-operations/SCRIPTS_GUIDE.md) | 10分钟 |
| **🔧 集成代码** | [03-integration/INTEGRATION_GUIDE.md](./03-integration/INTEGRATION_GUIDE.md) | 20分钟 |
| **📋 部署检查清单** | [01-gettingstarted/DEPLOYMENT_CHECKLIST.md](./01-gettingstarted/DEPLOYMENT_CHECKLIST.md) | 10分钟 |

---

## 📁 文档分类结构

### 📂 1️⃣ 01-gettingstarted（快速开始）
快速上手文档，适合第一次接触。

- **[00-START-HERE.md](./01-gettingstarted/00-START-HERE.md)** - 5 分钟快速导览
- **[DEPLOYMENT_CHECKLIST.md](./01-gettingstarted/DEPLOYMENT_CHECKLIST.md)** - 部署前检查清单

### 📂 2️⃣ 02-deployment（部署指南）
完整的部署方案和配置说明。

- **[API_AND_DEPLOYMENT.md](./02-deployment/API_AND_DEPLOYMENT.md)** - 部署 + 完整 API 文档 ⭐ 必读
  - 包含 V1 与推荐的 V2 公开接口说明
- **[DEPLOYMENT.md](./02-deployment/DEPLOYMENT.md)** - 详细的生产部署指南
- **examples/** - 配置文件示例
  - `.env.production.example` - 环境配置模板
  - `nginx.conf.production` - Nginx 反向代理配置
  - `moderation.service` - systemd 服务配置
  - `README.md` - 配置文件使用说明

### 📂 3️⃣ 03-integration（集成和对接）
代码集成和新项目对接文档。

- **[CLIENT_INTEGRATION.md](./03-integration/CLIENT_INTEGRATION.md)** - 客户对接指南 + 完整 Demo ⭐ 必读
  - 3 步对接流程
  - PHP SDK 完整代码
  - Node.js SDK 完整代码
  - Webhook 回调处理
  - cURL 命令示例
- **[INTEGRATION_GUIDE.md](./03-integration/INTEGRATION_GUIDE.md)** - 代码集成深度指南
  - handler.go 和 main.go 修改步骤
  - 鉴权模块集成
  - 监控模块集成

### 📂 4️⃣ 04-operations（运维管理）
日常运维、监控、密钥管理文档。

- **[AUTH_AND_MONITORING.md](./04-operations/AUTH_AND_MONITORING.md)** - 鉴权 + 监控 + 日志
  - API 密钥管理
  - 监控指标说明
  - 日志查看和查询
  - 告警规则设置
- **[SCRIPTS_GUIDE.md](./04-operations/SCRIPTS_GUIDE.md)** - 脚本使用指南
  - deploy.sh - 部署脚本
  - manage-keys.sh - 密钥管理脚本
  - monitor.sh - 监控脚本
  - push-go-to-github.sh - GitHub 推送脚本
  - 日常运维流程
  - 最佳实践

---

## 👥 按角色快速查找

### 👨‍💻 开发者
1. [01-gettingstarted/00-START-HERE.md](./01-gettingstarted/00-START-HERE.md) - 快速了解整体
2. [03-integration/CLIENT_INTEGRATION.md](./03-integration/CLIENT_INTEGRATION.md) - 学习如何对接
3. [02-deployment/API_AND_DEPLOYMENT.md](./02-deployment/API_AND_DEPLOYMENT.md) - 查看 API 文档

### 👨‍🔧 运维 / DBA
1. [01-gettingstarted/DEPLOYMENT_CHECKLIST.md](./01-gettingstarted/DEPLOYMENT_CHECKLIST.md) - 了解部署要求
2. [02-deployment/API_AND_DEPLOYMENT.md](./02-deployment/API_AND_DEPLOYMENT.md) - 部署方式
3. [04-operations/AUTH_AND_MONITORING.md](./04-operations/AUTH_AND_MONITORING.md) - 日常管理
4. [04-operations/SCRIPTS_GUIDE.md](./04-operations/SCRIPTS_GUIDE.md) - 脚本命令

### 🏗️ 架构师
1. [02-deployment/API_AND_DEPLOYMENT.md](./02-deployment/API_AND_DEPLOYMENT.md) - 系统架构
2. [04-operations/AUTH_AND_MONITORING.md](./04-operations/AUTH_AND_MONITORING.md) - 整体设计

### 👨‍💼 PM / 产品经理
1. [01-gettingstarted/00-START-HERE.md](./01-gettingstarted/00-START-HERE.md) - 功能概览
2. [02-deployment/API_AND_DEPLOYMENT.md](./02-deployment/API_AND_DEPLOYMENT.md) - 功能清单
3. [03-integration/CLIENT_INTEGRATION.md](./03-integration/CLIENT_INTEGRATION.md) - 对接流程

---

## 📋 按任务快速查找

| 任务 | 文档 | 位置 |
|-----|------|------|
| 第一次部署 | API_AND_DEPLOYMENT.md | 02-deployment/ |
| 生成 API 密钥 | SCRIPTS_GUIDE.md | 04-operations/ |
| 对接新项目 | CLIENT_INTEGRATION.md | 03-integration/ |
| 查看日志 | AUTH_AND_MONITORING.md | 04-operations/ |
| 监控服务 | SCRIPTS_GUIDE.md | 04-operations/ |
| 配置鉴权 | AUTH_AND_MONITORING.md | 04-operations/ |
| 运行脚本 | SCRIPTS_GUIDE.md | 04-operations/ |
| 查看 API 文档 | API_AND_DEPLOYMENT.md | 02-deployment/ |
| 获取 SDK 代码 | CLIENT_INTEGRATION.md | 03-integration/ |

---

## 🎯 快速命令速查

### 部署相关
```bash
# 一键部署
bash deploy.sh

# 查看完整部署指南
cat docs/02-deployment/API_AND_DEPLOYMENT.md | grep -A 20 "快速开始"
```

### 密钥管理
```bash
# 列出所有密钥
bash manage-keys.sh list

# 生成新密钥
bash manage-keys.sh create 51dm_service 300

# 测试密钥
bash manage-keys.sh test sk-proj-xxxx
```

### 监控日志
```bash
# 查看服务状态
bash monitor.sh status

# 查看实时日志
bash monitor.sh logs -n 100

# 查看项目审计日志
bash monitor.sh audit --project 51dm_service
```

---

## 📊 文档统计

| 分类 | 文件数 | 说明 |
|-----|-------|------|
| 01-gettingstarted | 2 | 快速上手指南 |
| 02-deployment | 2 + examples | 部署方案和配置 |
| 03-integration | 2 | 对接和集成 |
| 04-operations | 2 | 运维管理和脚本 |
| **总计** | **10 + examples** | 全覆盖，均为中文 |

---

## 🔔 重要提醒

- 📌 **第一次使用？** → 从 [01-gettingstarted/00-START-HERE.md](./01-gettingstarted/00-START-HERE.md) 开始
- 🔑 **要添加新项目？** → [03-integration/CLIENT_INTEGRATION.md](./03-integration/CLIENT_INTEGRATION.md)
- ⚙️ **要管理系统？** → [04-operations/SCRIPTS_GUIDE.md](./04-operations/SCRIPTS_GUIDE.md)
- 💡 **遇到问题？** → 查看对应文档中的"常见问题"部分
- 📝 **需要配置示例？** → [02-deployment/examples/](./02-deployment/examples/)

---

## 📂 完整目录树

```
docs/
├── README.md                           ← 你在这里（导航中心）
│
├── 01-gettingstarted/                 【快速开始】
│   ├── 00-START-HERE.md               5分钟快速导览
│   └── DEPLOYMENT_CHECKLIST.md        部署前检查
│
├── 02-deployment/                     【部署指南】
│   ├── API_AND_DEPLOYMENT.md          部署 + API 文档 ⭐ 必读
│   ├── DEPLOYMENT.md                  详细部署指南
│   └── examples/
│       ├── .env.production.example
│       ├── nginx.conf.production
│       ├── moderation.service
│       └── README.md
│
├── 03-integration/                    【集成对接】
│   ├── CLIENT_INTEGRATION.md          客户对接指南 ⭐ 必读
│   └── INTEGRATION_GUIDE.md           代码集成步骤
│
└── 04-operations/                     【运维管理】
    ├── AUTH_AND_MONITORING.md         鉴权 + 监控
    └── SCRIPTS_GUIDE.md               脚本使用
```

---

**最后更新**：2026-03-23
**文档版本**：3.1.0（分类组织版）
**状态**：✅ 生产级
