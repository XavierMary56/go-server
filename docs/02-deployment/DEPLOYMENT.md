# 🚀 AI 内容审核服务 - 生产部署完整指南

## 📋 目录
1. [系统架构](#系统架构)
2. [前置要求](#前置要求)
3. [部署步骤](#部署步骤)
4. [API 鉴权配置](#api-鉴权配置)
5. [监控和日志](#监控和日志)
6. [故障排查](#故障排查)

---

## 系统架构

```
┌─────────────────────────────────────────────────────┐
│                 互联网 (HTTPS)                       │
│           ai.a889.cloud:443 (Let's Encrypt)        │
└────────────────┬────────────────────────────────────┘
                 │
┌─────────────────▼────────────────────────────────────┐
│              Nginx 反向代理                          │
│  - SSL/TLS 加密                                     │
│  - 限流 & 速率限制                                   │
│  - 日志记录                                         │
│  - 负载均衡                                         │
└────────────────┬────────────────────────────────────┘
                 │
┌─────────────────▼────────────────────────────────────┐
│           Go 审核服务（Docker）                      │
│  端口：8080                                         │
│  - API 鉴权中间件                                   │
│  - 内容审核引擎                                     │
│  - 缓存管理（Redis）                                │
│  - 监控指标收集                                     │
└─────────────────────────────────────────────────────┘
        │                    │
   ┌────▼────┐        ┌─────▼──────┐
   │ Anthropic│        │ Redis      │
   │  Claude  │        │ 缓存       │
   │   API    │        │            │
   └──────────┘        └────────────┘

┌─────────────────────────────────────────────────────┐
│              监控和日志                              │
│  - 应用日志：logs/moderation_YYYY-MM-DD.log        │
│  - 审计日志：logs/audit/audit_YYYY-MM-DD.log       │
│  - 指标：/v1/stats                                 │
│  - 监控面板：Grafana + Prometheus                   │
└─────────────────────────────────────────────────────┘
```

---

## 前置要求

### 服务器配置
- **IP**：76.13.218.203:22
- **OS**：Ubuntu 20.04 LTS 或更新版本
- **CPU**：2+ cores
- **内存**：4GB+
- **存储**：20GB+ (用于日志)

### 软件依赖
```bash
# 检查 Docker
docker --version      # 20.10+
docker-compose --version  # 1.29+

# 检查域名 DNS
nslookup ai.a889.cloud
# 应该返回 76.13.218.203
```

---

## 部署步骤

### Step 1：准备服务器

```bash
# 登录服务器
ssh root@76.13.218.203 -p 22

# 更新系统
apt update && apt upgrade -y

# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 安装 Nginx
apt install -y nginx

# 安装 Certbot（HTTPS 证书）
apt install -y certbot python3-certbot-nginx
```

### Step 2：克隆项目并配置

```bash
# 进入部署目录
cd /opt && git clone <你的仓库地址> go-server
cd go-server

# 复制并编辑生产配置
cp .env.production .env

# 编辑 .env，填入真实的 API Key
nano .env
```

**必须配置的关键项**：
```env
ANTHROPIC_API_KEY=sk-ant-api03-xxxxx
ENABLE_AUTH=true
ALLOWED_KEYS=forum_service|sk-proj-forum-a-k3j9x2m1|300
CACHE_DRIVER=redis
```

### Step 3：配置 Nginx

```bash
# 复制 Nginx 配置
sudo cp deploy/nginx.conf.production /etc/nginx/conf.d/moderation.conf

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx
```

### Step 4：申请 HTTPS 证书

```bash
# 使用 Let's Encrypt 申请免费证书
sudo certbot --nginx -d ai.a889.cloud

# 自动续期配置
sudo systemctl enable certbot.timer
sudo systemctl start certbot.timer
```

### Step 5：启动服务

```bash
# 赋予脚本执行权限
chmod +x deploy.sh monitor.sh

# 运行部署脚本
bash deploy.sh

# 验证服务
curl https://ai.a889.cloud/v1/health
```

---

## API 鉴权配置

### 鉴权方式

#### 方式1：API 密钥（推荐）

```bash
# 请求示例
curl -X POST https://ai.a889.cloud/v1/moderate \
  -H "Content-Type: application/json" \
  -H "X-Project-Key: sk-proj-forum-a-k3j9x2m1" \
  -d '{
    "content": "需要审核的内容",
    "type": "comment"
  }'

# 响应示例
{
  "code": 200,
  "verdict": "pass",
  "category": "normal",
  "confidence": 0.98,
  "reason": "内容无违规",
  "model_used": "claude-sonnet-4-20250514",
  "latency_ms": 1234
}
```

#### 方式2：HMAC-SHA256 签名（可选）

用于更高安全需求的场景：

```bash
# 签名生成示例（PHP）
$timestamp = time();
$method = "POST";
$path = "/v1/moderate";
$secret = "sk-proj-forum-a-k3j9x2m1";

$message = "$timestamp|$method|$path";
$signature = hash_hmac('sha256', $message, $secret);

// 发送请求
curl -X POST https://ai.a889.cloud/v1/moderate \
  -H "X-Timestamp: $timestamp" \
  -H "X-Signature: $signature" \
  -d '{"content":"..."}'
```

### 密钥管理

#### 生成新的项目密钥

在 `.env` 中添加：
```env
ALLOWED_KEYS=项目1|密钥1|限流数,项目2|密钥2|限流数

# 示例
ALLOWED_KEYS=forum_service|sk-proj-forum-a-k3j9x2m1|300,bbs_service|sk-proj-bbs-b-p8q4w7n5|200
```

#### 密钥轮换

```bash
# 编辑 .env 文件
nano .env

# 更新密钥列表
ALLOWED_KEYS=forum_service|sk-proj-forum-new-xyz|300

# 重启服务
docker-compose restart moderation

# 验证
curl -H "X-Project-Key: sk-proj-forum-new-xyz" https://ai.a889.cloud/v1/health
```

#### 速率限制

在密钥配置中设置每分钟请求限制：
```env
ALLOWED_KEYS=forum_service|sk-proj-xxx|300  # 300请求/分钟
```

超限返回 429 Conflict：
```json
{
  "code": 429,
  "error": "请求过于频繁: 301/300"
}
```

---

## 监控和日志

### 查看服务状态

```bash
bash monitor.sh status

# 输出示例
容器状态：运行中
CONTAINER   CPU %   MEM USAGE / LIMIT   MEM %   NET I/O
moderation  2.5%    256MB / 2GB         12.8%   1.2MB / 890KB

健康检查：✓ OK
```

### 实时日志

```bash
# 查看最近 100 条日志
bash monitor.sh logs -n 100

# 实时跟踪日志
bash monitor.sh logs -f
```

### 审计日志查询

```bash
# 查看所有审计日志
bash monitor.sh audit

# 按项目查询
bash monitor.sh audit --project forum_service

# 查看最近 7 天
bash monitor.sh audit --project forum_service --days 7
```

### 性能指标

```bash
# 获取实时指标
bash monitor.sh metrics

# 导出为 Prometheus 格式
bash monitor.sh metrics --export prometheus

# API 接口直接查询
curl https://ai.a889.cloud/v1/stats -H "X-Project-Key: xxx" | jq '.data'

# 返回示例
{
  "uptime_seconds": 86400,
  "total_requests": 50000,
  "success_requests": 49500,
  "failed_requests": 500,
  "success_rate_percent": 99.0,
  "avg_latency_ms": 234.5,
  "api_calls": 50000,
  "model_usage": {
    "claude-sonnet-4-20250514": 30000,
    "claude-haiku-4-5-20251001": 15000,
    "claude-opus-4-20250514": 5000
  },
  "error_counts": {
    "rate_limit_exceeded": 100,
    "auth_failed": 50,
    "api_timeout": 20
  }
}
```

### 日志清理

```bash
# 清理 30 天前的日志
bash monitor.sh clean

# 备份审计日志
bash monitor.sh backup
```

---

## 日志文件位置

| 日志类型 | 位置 | 说明 |
|--------|------|------|
| 应用日志 | `logs/moderation_YYYY-MM-DD.log` | 服务运行日志，按天切割 |
| 审计日志 | `logs/audit/audit_YYYY-MM-DD.log` | API 调用和认证日志 |
| Nginx 访问日志 | `/var/log/nginx/moderation_access.log` | HTTP 请求日志 |
| Nginx 错误日志 | `/var/log/nginx/moderation_error.log` | Nginx 错误日志 |
| 系统日志 | `journalctl -u moderation -f` | systemd 日志 |

---

## 故障排查

### 问题1：服务无法启动

```bash
# 检查 Docker 容器状态
docker-compose ps

# 查看错误日志
docker-compose logs moderation

# 检查端口占用
netstat -tlnp | grep 8080

# 重启服务
docker-compose restart moderation
```

### 问题2：API 返回 401 Unauthorized

```bash
# 检查 .env 文件配置
grep ENABLE_AUTH .env
grep ALLOWED_KEYS .env

# 确保密钥包含在请求中
curl -H "X-Project-Key: sk-proj-xxxx" https://ai.a889.cloud/v1/health

# 查看审计日志中的认证失败
grep '"event_type":"auth_attempt"' logs/audit/*.log
```

### 问题3：性能下降

```bash
# 查看缓存状态
redis-cli INFO memory

# 检查 CPU 和内存使用
docker stats moderation

# 查看错误率
bash monitor.sh metrics | grep success_rate

# 调整并发配置
# 在 .env 中增加 Go 运行时参数：
# GOMAXPROCS=4
```

### 问题4：HTTPS 证书过期

```bash
# 检查证书有效期
certbot certificates

# 手动更新证书
sudo certbot renew --dry-run

# 强制更新
sudo certbot renew --force-renewal
```

### 问题5：速率限制频繁触发

```bash
# 检查触发频率
grep "rate_limit_exceeded" logs/audit/*.log | wc -l

# 提高限制阈值
# 编辑 .env：
ALLOWED_KEYS=forum_service|sk-proj-xxx|500  # 从 300 提升到 500

# 重启服务
docker-compose restart moderation
```

---

## 最佳实践

### 安全性
- ✅ 启用 API 鉴权（ENABLE_AUTH=true）
- ✅ 使用 HTTPS（Let's Encrypt）
- ✅ 定期轮换 API 密钥
- ✅ 启用审计日志
- ✅ 限制 Nginx access_log 的保留时间

### 性能
- ✅ 使用 Redis 缓存而非内存
- ✅ 设置合理的 API 超时时间
- ✅ 启用 Nginx 缓存和压缩
- ✅ 使用 CDN 分发（可选）

### 可靠性
- ✅ 定期备份审计日志
- ✅ 配置自动日志清理
- ✅ 设置告警规则
- ✅ 定期更新依赖和补丁

### 运维
- ✅ 每天查看日志和指标
- ✅ 每周审查审计日志
- ✅ 每月检查证书过期时间
- ✅ 每季度进行一次灾备演练

---

## 联系方式

有问题？查看：
- 📖 应用日志：`logs/moderation_*.log`
- 🔐 审计日志：`logs/audit/audit_*.log`
- 📊 性能指标：`curl https://ai.a889.cloud/v1/stats`
- 🐛 运行错误：`docker-compose logs moderation`
