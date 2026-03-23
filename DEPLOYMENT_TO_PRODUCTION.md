# 🚀 部署到服务器的详细步骤

**目标服务器**: 76.13.218.203:22
**域名**: ai.a889.cloud
**最新代码**: 已推送到 GitHub (https://github.com/XavierMary56/go-server.git)

---

## 📋 前置检查清单

在开始部署前，请确保：

```bash
✅ 已有服务器 76.13.218.203 的 SSH 访问权限
✅ 已知 SSH 用户名和密码/密钥
✅ 服务器已安装 Git (或会自动安装)
✅ 服务器已安装 Docker (推荐) 或 Go 1.21+
✅ 了解 Anthropic API Key 的具体值
✅ 了解需要配置的项目密钥列表
```

---

## 🔑 Step 1: SSH 登录到服务器

```bash
# 使用 SSH 登录 (根据您的认证方式选择)
ssh -p 22 your_username@76.13.218.203

# 或使用密钥文件
ssh -i /path/to/private/key -p 22 your_username@76.13.218.203
```

**提示**: 替换 `your_username` 为实际的用户名。

---

## 📥 Step 2: 下载并执行部署脚本

SSH 登录后，在服务器上执行：

```bash
# 进入临时目录
cd /tmp

# 下载最新的部署脚本
wget -O deploy.sh https://raw.githubusercontent.com/XavierMary56/go-server/main/deploy-to-production.sh

# 或使用 curl
curl -o deploy.sh https://raw.githubusercontent.com/XavierMary56/go-server/main/deploy-to-production.sh

# 给予执行权限
chmod +x deploy.sh

# 执行部署脚本 (需要 sudo 权限)
sudo bash deploy.sh
```

**脚本会自动执行**:
1. ✅ 检查 Git 和 Docker 依赖
2. ✅ 创建部署目录 `/opt/moderation`
3. ✅ 从 GitHub 拉取最新代码
4. ✅ 验证 .env 配置
5. ✅ 启动服务 (Docker 或 systemd)
6. ✅ 验证服务健康

---

## ⚙️ Step 3: 配置环境变量

如果脚本要求您编辑 `.env` 文件：

```bash
# SSH 后进入部署目录
cd /opt/moderation

# 编辑配置
nano .env
```

**必填配置项**:
```bash
# Anthropic API Key (必填)
ANTHROPIC_API_KEY=sk-ant-api03-你的真实密钥

# 启用鉴权
ENABLE_AUTH=true

# 项目密钥列表 (逗号分隔)
ALLOWED_KEYS=proj_forum_xxx,proj_service_yyy

# 启用审计日志
ENABLE_AUDIT=true

# 启用监控
ENABLE_METRICS=true
```

**可选配置项**:
```bash
# 服务监听端口
PORT=8080

# 缓存配置
CACHE_DRIVER=redis         # 使用 Redis (需要预装)
CACHE_TTL=60              # 缓存时间(秒)

# 日志配置
LOG_LEVEL=info            # debug|info|warn|error
LOG_DIR=./logs            # 日志目录

# 管理员令牌
ADMIN_TOKEN=your-secret-token
```

---

## 🐳 Step 4: 选择部署方式

### 方式 A: Docker (推荐 ⭐)

脚本会自动检测 Docker 并使用 Docker 部署：

```bash
# Docker 部署自动执行
docker-compose up -d

# 查看日志
docker-compose logs -f moderation

# 停止服务
docker-compose down
```

### 方式 B: 二进制部署 (无 Docker)

如果服务器没有 Docker，脚本会自动编译二进制：

```bash
# 自动编译
go build -o moderation-server ./cmd/server

# 自动配置 systemd
sudo systemctl start moderation
sudo systemctl status moderation

# 查看日志
sudo journalctl -u moderation -f
```

---

## ✅ Step 5: 验证部署成功

部署完成后，验证服务是否运行正常：

```bash
# 健康检查
curl http://localhost:8080/v1/health

# 应该返回类似
{
  "status": "ok",
  "version": "2.0.0",
  "time": "2026-03-23T10:30:45Z"
}

# 或进入全面检查
cd /opt/moderation
bash monitor.sh status
```

---

## 🌐 Step 6: 配置域名和 HTTPS

### 6.1 DNS 配置

在您的域名供应商处，添加 A 记录：

```
记录类型: A
子域名: ai          (或使用 *)
值: 76.13.218.203
TTL: 3600
```

### 6.2 配置 Nginx 反向代理

```bash
# SSH 进服务器后
cd /opt/moderation

# 查看 Nginx 配置示例
cat deploy/nginx.conf.production

# 复制配置到系统
sudo cp deploy/nginx.conf.production /etc/nginx/conf.d/moderation.conf

# 编辑配置，将 mod.your-company.com 改为 ai.a889.cloud
sudo nano /etc/nginx/conf.d/moderation.conf

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx
```

### 6.3 申请 SSL 证书 (Let's Encrypt)

```bash
# 安装 Certbot (如果未安装)
sudo apt-get install certbot python3-certbot-nginx

# 申请证书
sudo certbot certonly --nginx -d ai.a889.cloud

# 自动续期证书
sudo systemctl enable certbot.timer
```

### 6.4 验证 HTTPS

```bash
# 测试 HTTPS 连接
curl https://ai.a889.cloud/v1/health

# 应该返回成功响应
```

---

## 🔧 Step 7: 日常管理

### 查看服务状态

```bash
cd /opt/moderation

# 查看完整状态
bash monitor.sh status

# 查看最近日志
bash monitor.sh logs -n 100

# 查看项目统计
bash monitor.sh audit --project your_project_name
```

### 密钥管理

```bash
cd /opt/moderation

# 列出所有密钥
bash manage-keys.sh list

# 生成新密钥
bash manage-keys.sh create new_project_name 300

# 测试密钥
bash manage-keys.sh test proj_xxx_yyy

# 禁用密钥
bash manage-keys.sh disable proj_xxx_yyy
```

### 性能监控

```bash
# 查看实时指标
curl http://76.13.218.203:9090/metrics

# 或通过脚本
bash monitor.sh metrics
```

---

## 📊 完整命令速查

```bash
# ===== 服务管理 =====
sudo systemctl start moderation      # 启动服务
sudo systemctl stop moderation       # 停止服务
sudo systemctl restart moderation    # 重启服务
sudo systemctl status moderation     # 查看状态

# ===== Docker 管理 =====
docker-compose up -d                 # 后台启动
docker-compose down                  # 停止关闭
docker-compose logs -f               # 查看日志
docker-compose ps                    # 查看容器

# ===== 脚本命令 =====
bash monitor.sh status              # 检查状态
bash monitor.sh logs -n 50          # 查看日志
bash manage-keys.sh list            # 列出密钥
bash manage-keys.sh create xxx 300  # 生成密钥

# ===== 测试 API =====
curl http://localhost:8080/v1/health              # 健康检查
curl http://ai.a889.cloud/v1/health               # HTTPS 测试
curl http://ai.a889.cloud/v1/models -H "X-Project-Key: xxx"
```

---

## ❌ 常见问题排查

### 问题 1: "Permission denied" (权限错误)

```bash
# 解决方案：使用 sudo
sudo bash deploy.sh

# 或给予权限
chmod +x deploy.sh
sudo bash deploy.sh
```

### 问题 2: "ANTHROPIC_API_KEY not configured"

```bash
# 解决方案：编辑 .env 文件
cd /opt/moderation
nano .env

# 填入有效的 API Key
ANTHROPIC_API_KEY=sk-ant-api03-你的真实密钥
```

### 问题 3: "Docker: command not found"

```bash
# 解决方案：安装 Docker
sudo apt-get update
sudo apt-get install docker.io docker-compose

# 或让脚本自动使用二进制部署 (需要 Go)
go version  # 确保已安装 Go 1.21+
```

### 问题 4: 服务启动失败

```bash
# 查看详细错误日志
sudo journalctl -u moderation -n 50 -f

# 或 Docker 日志
docker-compose logs moderation

# 检查端口是否被占用
sudo netstat -tulpn | grep 8080
```

### 问题 5: 域名无法访问

```bash
# 检查 DNS 解析
nslookup ai.a889.cloud
dig ai.a889.cloud

# 检查 Nginx 配置
sudo nginx -t

# 检查防火墙
sudo ufw status
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

---

## 📈 部署后的建议

### 即刻操作
1. ✅ 测试 `/v1/health` 接口
2. ✅ 配置第一个项目密钥
3. ✅ 测试一个简单的审核请求

### 当天完成
1. ✅ 配置 Nginx 和 HTTPS
2. ✅ 解析域名 ai.a889.cloud
3. ✅ 启用审计日志
4. ✅ 启用性能监控

### 本周完成
1. ✅ 集成第一个客户项目 (参考 CLIENT_INTEGRATION.md)
2. ✅ 配置告警和监控
3. ✅ 备份数据和配置
4. ✅ 制定维护计划

---

## 📞 获取帮助

| 问题类型 | 查看文档 |
|---------|--------|
| 部署问题 | `docs/02-deployment/API_AND_DEPLOYMENT.md` |
| API 使用 | `docs/02-deployment/API_AND_DEPLOYMENT.md` |
| 客户对接 | `docs/03-integration/CLIENT_INTEGRATION.md` |
| 脚本使用 | `docs/04-operations/SCRIPTS_GUIDE.md` |
| 监控管理 | `docs/04-operations/AUTH_AND_MONITORING.md` |

---

## ✨ 部署完成！

一旦所有步骤完成，您的系统将运行在：

```
🌐 https://ai.a889.cloud
📍 76.13.218.203:22
```

Ready to serve content moderation requests! 🎉

---

**最后更新**: 2026-03-23
**版本**: 2.0.0
**状态**: ✅ Production Ready
