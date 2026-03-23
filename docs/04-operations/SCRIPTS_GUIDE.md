# 🔨 脚本使用指南

项目中包含的所有运维脚本。按功能分类，快速查找。

---

## 📋 脚本总览

| 脚本 | 功能 | 用途 | 文件大小 |
|-----|------|------|--------|
| **deploy.sh** | 一键部署服务 | 首次部署或更新部署 | 4.2KB |
| **manage-keys.sh** | API 密钥管理 | 生成、测试、轮换密钥 | 8.8KB |
| **monitor.sh** | 监控和日志 | 查看状态、日志、指标 | 8.1KB |
| **push-go-to-github.sh** | 推送到 GitHub | 代码版本管理 | 2.4KB |

---

## 🚀 按场景快速开始

### 场景1️⃣：第一次部署服务

```bash
# 1. 配置环境
cp .env.production .env
nano .env  # 填入 ANTHROPIC_API_KEY 和密钥

# 2. 一键部署
bash deploy.sh

# 3. 验证部署
bash monitor.sh status
```

### 场景2️⃣：对接新项目（生成密钥）

```bash
# 1. 生成新密钥
bash manage-keys.sh create 51dm_service 300

# 2. 添加到系统（通过管理 API）
curl -X POST https://ai.a889.cloud/v1/admin/keys \
  -H "Authorization: Bearer admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "51dm_service",
    "key": "sk-proj-51dm-xxxx",
    "rate_limit": 300
  }'

# 3. 测试密钥
bash manage-keys.sh test sk-proj-51dm-xxxx
```

### 场景3️⃣：日常监控和维护

```bash
# 查看服务状态
bash monitor.sh status

# 查看实时日志
bash monitor.sh logs -n 100

# 查看某项目的审计日志
bash monitor.sh audit --project 51dm_service

# 查看性能指标
bash monitor.sh metrics

# 清理旧日志
bash monitor.sh clean

# 备份审计日志
bash monitor.sh backup
```

---

## 📖 脚本详细说明

### 1. deploy.sh - 部署脚本

**位置**：`./deploy.sh`

**功能**：一键部署服务到 Docker Compose

**使用方式**：

```bash
bash deploy.sh
```

**执行步骤**：
1. ✅ 环境检查（Docker、Docker Compose）
2. ✅ 配置文件验证
3. ✅ 应用构建
4. ✅ 目录权限设置
5. ✅ 启动容器
6. ✅ 健康检查

**输出**：
```
[1/6] 环境检查...
[2/6] 配置文件准备...
[3/6] 应用构建...
[4/6] 目录权限设置...
[5/6] 启动服务...
[6/6] 服务健康检查...

✓ 部署完成！
```

**常见问题**：

| 错误 | 原因 | 解决方案 |
|-----|------|--------|
| Docker 未安装 | 环境缺少 Docker | 安装 Docker |
| .env 不存在 | 配置文件缺失 | 复制 .env.production 为 .env |
| 端口被占用 | 8080 端口被其他服务占用 | 改为其他端口或停止占用端口的服务 |

---

### 2. manage-keys.sh - 密钥管理脚本

**位置**：`./manage-keys.sh`

**功能**：管理 API 密钥（生成、测试、轮换等）

**所有命令**：

#### 列出所有密钥
```bash
bash manage-keys.sh list

# 输出：
# 密钥（隐藏）     项目 ID          速率限制
# sk-....2m1       forum_service    300
# sk-....4w7       bbs_service      200
# sk-....g7h       51dm_service     300
```

#### 生成新密钥
```bash
bash manage-keys.sh create <项目ID> <限流数>

# 示例：生成 51dm 项目的密钥，限流 300/分钟
bash manage-keys.sh create 51dm_service 300

# 输出：
# 项目 ID：51dm_service
# 速率限制：300 请求/分钟
# 新密钥：sk-proj-51dm-a1b2c3d4...
```

#### 测试密钥
```bash
bash manage-keys.sh test <密钥>

# 示例：
bash manage-keys.sh test sk-proj-51dm-a1b2c3d4

# 输出：
# ✓ 密钥有效
# {
#   "status": "ok",
#   "version": "2.0.0",
#   "time": "2026-03-23T10:30:45Z"
# }
```

#### 测试速率限制
```bash
bash manage-keys.sh rate-test <密钥> [请求数]

# 示例：测试密钥的限流设置（发送 310 个请求）
bash manage-keys.sh rate-test sk-proj-51dm-a1b2c3d4 310

# 输出：
# 已发送 50 个请求... (成功: 50, 限流: 0, 出错: 0)
# 已发送 100 个请求... (成功: 100, 限流: 0, 出错: 0)
# ...
# 测试结果：
# 成功：300
# 限流：10  ← 触发了限流
```

#### 轮换密钥
```bash
bash manage-keys.sh rotate <旧密钥> <新密钥>

# 示例：
bash manage-keys.sh rotate sk-proj-old-xxx sk-proj-new-yyy
```

#### 禁用密钥
```bash
bash manage-keys.sh disable <密钥>

# 示例：
bash manage-keys.sh disable sk-proj-old-xxx
```

#### 导出为 JSON
```bash
bash manage-keys.sh export-json

# 输出：所有密钥的 JSON 格式
```

---

### 3. monitor.sh - 监控脚本

**位置**：`./monitor.sh`

**功能**：监控服务状态、查看日志、导出指标

**所有命令**：

#### 查看服务状态
```bash
bash monitor.sh status

# 输出：
# 容器状态：运行中
# CONTAINER   CPU %   MEM USAGE / LIMIT   MEM %
# moderation  2.5%    256MB / 2GB         12.8%
#
# 健康检查：✓ OK
# API 健康：正常
#
# 性能指标：
# 总请求数：50000
# 成功率：99%
# 平均延迟：234.5ms
```

#### 查看实时日志
```bash
# 查看最近 100 条日志
bash monitor.sh logs -n 100

# 实时跟踪日志
bash monitor.sh logs -f

# 输出：JSON 格式的日志
```

#### 查看审计日志
```bash
# 查看所有审计日志
bash monitor.sh audit

# 查看特定项目的日志
bash monitor.sh audit --project forum_service

# 查看最近 7 天的日志
bash monitor.sh audit --project forum_service --days 7

# 输出：
# 审计日志查询
# 项目：forum_service
# 时间范围：最近 7 天
#
# 认证尝试：
#   forum_service	100
#
# API 调用统计（按项目）：
#   forum_service	5000
#
# 速率限制触发：0
```

#### 查看性能指标
```bash
# 获取实时指标
bash monitor.sh metrics

# 导出为 Prometheus 格式
bash monitor.sh metrics --export prometheus

# 输出：
# 总请求数：50000
# 成功率：99.0%
# 平均延迟：234.50ms
# API 调用：50000
# 模型使用：
#   claude-sonnet-4-20250514：30000
#   claude-haiku-4-5-20251001：15000
```

#### 健康检查
```bash
bash monitor.sh health

# 输出：API 健康检查结果
```

#### 清理旧日志
```bash
# 清理 30 天前的日志
bash monitor.sh clean

# 输出：
# ✓ 已删除：logs/moderation_2026-02-20.log
# ✓ 已删除：logs/audit/forum_service/audit_2026-02-20.log
# ...
# ✓ 日志清理完成
```

#### 备份日志
```bash
bash monitor.sh backup

# 输出：
# ✓ 审计日志已备份：backups/audit_backup_20260323_103045.tar.gz
```

#### 告警配置
```bash
bash monitor.sh alert

# 输出：告警配置指南和 crontab 示例
```

---

### 4. push-go-to-github.sh - GitHub 推送脚本

**位置**：`./push-go-to-github.sh`

**功能**：快速推送代码变更到 GitHub

**使用方式**：

```bash
bash push-go-to-github.sh
```

**功能**：
- ✅ 检查 git 状态
- ✅ 添加所有变更
- ✅ 创建提交
- ✅ 推送到 GitHub

---

## 🔄 日常运维流程

### 每天

```bash
# 早上检查
bash monitor.sh status

# 查看错误日志
bash monitor.sh logs -n 50 | grep ERROR

# 查看关键指标
bash monitor.sh metrics
```

### 每周

```bash
# 周一查看一周的日志
bash monitor.sh logs --days 7

# 备份重要数据
bash monitor.sh backup

# 查看所有项目统计
curl -H "Authorization: Bearer admin-token" \
  https://ai.a889.cloud/v1/admin/projects
```

### 每月

```bash
# 清理旧日志（保留最近 30 天）
bash monitor.sh clean

# 生成月度报告
bash monitor.sh metrics --export json > report_$(date +%Y-%m).json
```

---

## 💡 脚本最佳实践

### 1. 自动化定时任务

编辑 crontab：
```bash
crontab -e
```

添加以下任务：

```bash
# 每天凌晨 2 点清理日志
0 2 * * * cd /opt/moderation && bash monitor.sh clean

# 每周日凌晨 3 点备份日志
0 3 * * 0 cd /opt/moderation && bash monitor.sh backup

# 每 5 分钟检查一次服务状态
*/5 * * * * cd /opt/moderation && bash monitor.sh status >> /var/log/moderation-cron.log 2>&1
```

### 2. 监控告警集成

```bash
# 获取监控脚本的建议
bash monitor.sh alert
```

### 3. 脚本日志记录

```bash
# 将脚本输出重定向到日志文件
bash deploy.sh >> deploy_$(date +%Y-%m-%d_%H-%M-%S).log 2>&1

# 查看脚本执行记录
tail -f deploy_2026-03-23_10-30-45.log
```

---

## 🆘 常见问题

### Q: 脚本权限不足的错误？
A: 添加执行权限
```bash
chmod +x deploy.sh manage-keys.sh monitor.sh push-go-to-github.sh
```

### Q: 脚本找不到 docker-compose？
A: 检查 Docker 安装
```bash
docker-compose --version
# 如果提示找不到，确保已安装 Docker Desktop
```

### Q: 如何在远程服务器上运行脚本？
A: 使用 SSH
```bash
ssh user@server "cd /opt/moderation && bash monitor.sh status"
```

### Q: 脚本输出太长，怎么保存？
A: 重定向到文件
```bash
bash monitor.sh status > status_report.txt
bash monitor.sh logs -n 1000 > recent_logs.txt
```

---

## 📝 脚本清单

- [x] deploy.sh - 部署脚本
- [x] manage-keys.sh - 密钥管理
- [x] monitor.sh - 监控和日志
- [x] push-go-to-github.sh - GitHub 推送
- [ ] backup.sh - 定期备份（可选）
- [ ] alert.sh - 告警通知（可选）

---

**脚本文档版本**：1.0.0
**最后更新**：2026-03-23
**状态**：生产级
