# 📝 配置和脚本示例

本目录包含部署所需的配置文件和脚本示例。

## 📋 文件说明

### `.env.production.example`
**用途**：完整的生产环境配置模板

**使用方式**：
```bash
# 1. 在项目根目录复制文件
cp docs/examples/.env.production.example .env

# 2. 编辑配置
nano .env

# 3. 填入真实的配置项：
#    - ANTHROPIC_API_KEY：你的 Anthropic API 密钥
#    - ALLOWED_KEYS：兼容兜底配置（正常维护走后台）
#    - REDIS_PASS：Redis 密码（如果有的话）
```

**关键配置项**：
- `ANTHROPIC_API_KEY` - **必填**，从 console.anthropic.com 获取
- `ALLOWED_KEYS` - 兼容兜底配置；正常情况下请在后台 Project Keys 中维护
- `ENABLE_AUTH` - 是否启用鉴权（生产环境推荐 true）
- `ENABLE_AUDIT` - 是否启用审计日志记录
- `CACHE_DRIVER` - 缓存驱动（production 推荐 redis）

### `nginx.conf.production`
**用途**：Nginx 反向代理配置（生产级）

**使用方式**：
```bash
# 1. 复制到 Nginx 配置目录
sudo cp docs/examples/nginx.conf.production /etc/nginx/conf.d/moderation.conf

# 2. 修改域名（如果不是 ai.a889.cloud）
sudo sed -i 's/ai.a889.cloud/your-domain.com/g' /etc/nginx/conf.d/moderation.conf

# 3. 验证配置
sudo nginx -t

# 4. 重载 Nginx
sudo systemctl reload nginx
```

**功能**：
- 🔒 HTTPS/SSL 配置
- ⚡ 请求限流（速率限制）
- 🔍 详细日志记录
- 🛡️ 安全头配置
- 📊 Prometheus 指标公开

### `moderation.service`
**用途**：systemd 服务配置（用于 Linux 系统管理）

**使用方式**：
```bash
# 1. 复制到 systemd 目录
sudo cp docs/examples/moderation.service /etc/systemd/system/

# 2. 重新加载 systemd 配置
sudo systemctl daemon-reload

# 3. 启用服务（开机自启）
sudo systemctl enable moderation

# 4. 启动服务
sudo systemctl start moderation

# 5. 查看状态
sudo systemctl status moderation
```

**服务管理命令**：
```bash
sudo systemctl start moderation      # 启动
sudo systemctl stop moderation       # 停止
sudo systemctl restart moderation    # 重启
sudo systemctl status moderation     # 查看状态
journalctl -u moderation -f         # 查看日志
```

---

## 🚀 快速开始（Docker 方式）

如果使用 Docker Compose：

```bash
# 1. 准备配置
cp docs/examples/.env.production.example .env
nano .env  # 编辑配置

# 2. 启动服务
docker-compose up -d

# 3. 验证
curl http://localhost:8080/v1/health
```

---

## 🚀 快速开始（systemd 方式）

如果在 Linux 服务器上运行：

```bash
# 1. 准备配置
cp docs/examples/.env.production.example .env
nano .env

# 2. 编译程序
go build -o moderation-server ./cmd/server

# 3. 安装服务
sudo cp moderation-server /opt/moderation/
sudo cp docs/examples/moderation.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable moderation
sudo systemctl start moderation

# 4. 验证
curl http://localhost:8080/v1/health
```

---

## 📊 配置对应关系

```
docs/examples/
├── .env.production.example      → 项目根目录 .env
├── nginx.conf.production        → /etc/nginx/conf.d/moderation.conf
└── moderation.service           → /etc/systemd/system/moderation.service
```

---

## ⚙️ 配置 API 密钥示例

编辑 `.env` 文件中的 `ALLOWED_KEYS`：

```env
# 单个项目
ALLOWED_KEYS=forum_service|sk-proj-forum-a-xxxx|300

# 多个项目
ALLOWED_KEYS=forum_service|sk-proj-forum-a-xxxx|300,bbs_service|sk-proj-bbs-b-yyyy|200,51dm_service|sk-proj-51dm-zzzz|500
```

**格式详解**：
```
项目ID         |  密钥（建议 sk-proj- 开头）  |  每分钟限流数
forum_service  |  sk-proj-forum-a-k3j9x2m1    |  300
bbs_service    |  sk-proj-bbs-b-p8q4w7n5      |  200
51dm_service   |  sk-proj-51dm-a1b2c3d4e5f6   |  500
```

---

## 🔐 安全启用 HTTPS

使用 Let's Encrypt 免费证书：

```bash
# 安装 certbot
sudo apt install -y certbot python3-certbot-nginx

# 申请证书
sudo certbot --nginx -d ai.a889.cloud

# 自动续期
sudo systemctl enable certbot.timer
sudo systemctl start certbot.timer
```

---

## 📝 常见配置修改

### 修改 API 端口
```env
PORT=9000  # 改为 9000
```

### 启用审计日志
```env
ENABLE_AUDIT=true
AUDIT_LOG_DIR=/var/log/moderation/audit
```

### 使用 Redis 缓存
```env
CACHE_DRIVER=redis
REDIS_ADDR=127.0.0.1:6379
REDIS_PASS=your_password  # 如果 Redis 有密码
```

### 调整日志级别
```env
LOG_LEVEL=debug  # debug | info | warn | error
```

---

## ❓ 常见问题

**Q：为什么文件名是 `.env.production.example`？**
A：防止不小心覆盖真实配置文件。使用时应该：
```bash
cp .env.production.example .env
# 然后编辑 .env 文件
```

**Q：如何更换 ANTHROPIC_API_KEY？**
A：编辑 `.env` 文件，修改这一行，然后重启服务：
```bash
nano .env
docker-compose restart moderation
```

**Q：如何添加新的项目？**
A：编辑 `.env` 文件，在 `ALLOWED_KEYS` 后面追加：
```env
ALLOWED_KEYS=...existing...,new_project|sk-proj-xxx|300
```
然后重启服务。

**Q：Nginx 配置文件在哪里？**
A：取决于你的操作系统：
- Ubuntu/Debian：`/etc/nginx/conf.d/`
- CentOS：`/etc/nginx/conf.d/`

---

## 🔗 相关文档

- 📖 [完整部署指南](../DEPLOYMENT.md)
- 🔐 [鉴权和监控](../AUTH_AND_MONITORING.md)
- 💡 [代码集成指南](../INTEGRATION_GUIDE.md)
- 📋 [部署检查清单](../DEPLOYMENT_CHECKLIST.md)
