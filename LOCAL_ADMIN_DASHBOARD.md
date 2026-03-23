# 🚀 本地管理后台完整指南

> **生成时间**: 2026-03-23 13:16:01
> **状态**: ✅ 已启动并经过验证
> **端口**: 8088

---

## 📍 管理后台访问地址

### 1️⃣ Web UI 管理界面（推荐 ⭐）

```
🌐 http://localhost:8088/admin/
```

**特点**:
- ✅ 可视化管理面板
- ✅ 无需认证（直接访问）
- ✅ 实时密钥管理
- ✅ 项目监控统计

**访问方式**:
```bash
# 浏览器直接访问
http://localhost:8088/admin/

# 或使用 curl
curl http://localhost:8088/admin/
```

---

### 2️⃣ REST API 管理接口

```
🔌 Base URL: http://localhost:8088/v1/admin/
🔐 认证方式: Bearer Token
🔑 默认令牌: admin-token-default
```

---

## 📚 API 端点完整清单

### ✅ 无需认证的端点

#### GET /v1/admin/health
**功能**: 管理接口健康检查

```bash
curl http://localhost:8088/v1/admin/health
```

**返回示例**:
```json
{
  "admin_api_available": true,
  "status": "ok"
}
```

---

### 🔐 需要认证的端点

**认证方式**:
```bash
curl -H "Authorization: Bearer admin-token-default" \
  http://localhost:8088/v1/admin/...
```

---

#### GET /v1/admin/keys
**功能**: 列出所有 API 密钥

```bash
curl -H "Authorization: Bearer admin-token-default" \
  http://localhost:8088/v1/admin/keys
```

**返回示例**:
```json
{
  "code": 200,
  "data": {
    "proj_forum_a_k3j9x2m1": {
      "project_id": "forum",
      "key": "proj_forum_a_k3j9x2m1",
      "rate_limit": 500,
      "created_at": "2026-03-23T12:00:00Z",
      "enabled": true
    }
  }
}
```

---

#### POST /v1/admin/keys
**功能**: 添加新 API 密钥（无需重启服务）

```bash
curl -X POST http://localhost:8088/v1/admin/keys \
  -H "Authorization: Bearer admin-token-default" \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "new_project",
    "key": "proj_new_xyz789abc",
    "rate_limit": 1000
  }'
```

**请求参数**:
| 参数 | 类型 | 说明 | 必填 |
|-----|------|------|-----|
| project_id | string | 项目ID（唯一） | ✅ |
| key | string | API 密钥 | ✅ |
| rate_limit | int | 每分钟请求限制 | ✅ |

**返回示例**:
```json
{
  "code": 200,
  "message": "密钥添加成功",
  "data": {
    "project_id": "new_project",
    "key": "proj_new_xyz789abc",
    "rate_limit": 1000,
    "created_at": "2026-03-23T13:16:05Z",
    "enabled": true
  }
}
```

---

#### GET /v1/admin/keys/{key}
**功能**: 获取指定密钥的详细信息

```bash
curl -H "Authorization: Bearer admin-token-default" \
  http://localhost:8088/v1/admin/keys/proj_forum_a_k3j9x2m1
```

---

#### PUT /v1/admin/keys/{key}
**功能**: 更新密钥配置

```bash
curl -X PUT http://localhost:8088/v1/admin/keys/proj_forum_a_k3j9x2m1 \
  -H "Authorization: Bearer admin-token-default" \
  -H "Content-Type: application/json" \
  -d '{
    "rate_limit": 2000,
    "enabled": true
  }'
```

---

#### DELETE /v1/admin/keys/{key}
**功能**: 删除 API 密钥

```bash
curl -X DELETE http://localhost:8088/v1/admin/keys/proj_forum_a_k3j9x2m1 \
  -H "Authorization: Bearer admin-token-default"
```

---

#### GET /v1/admin/projects
**功能**: 列出所有项目

```bash
curl -H "Authorization: Bearer admin-token-default" \
  http://localhost:8088/v1/admin/projects
```

**返回示例**:
```json
{
  "code": 200,
  "data": {
    "projects": [
      {
        "project_id": "forum",
        "api_key": "proj_forum_a_k3j9x2m1",
        "created_at": "2026-03-20T10:00:00Z"
      }
    ],
    "total_projects": 1
  }
}
```

---

#### GET /v1/admin/projects/logs
**功能**: 查询项目的审计日志

```bash
# 查询指定项目的所有日志
curl -H "Authorization: Bearer admin-token-default" \
  'http://localhost:8088/v1/admin/projects/logs?project=forum'

# 按时间范围查询
curl -H "Authorization: Bearer admin-token-default" \
  'http://localhost:8088/v1/admin/projects/logs?project=forum&start=2026-03-20&end=2026-03-23'

# 按事件类型过滤
curl -H "Authorization: Bearer admin-token-default" \
  'http://localhost:8088/v1/admin/projects/logs?project=forum&type=api_call'
```

**查询参数**:
| 参数 | 说明 | 示例 |
|-----|------|------|
| project | 项目ID（必填） | forum |
| start | 起始日期 | 2026-03-20 |
| end | 结束日期 | 2026-03-23 |
| type | 事件类型 | api_call, auth_success, auth_failed |

---

#### GET /v1/admin/projects/stats
**功能**: 查看项目的统计信息

```bash
curl -H "Authorization: Bearer admin-token-default" \
  'http://localhost:8088/v1/admin/projects/stats?project=forum'
```

**返回示例**:
```json
{
  "code": 200,
  "data": {
    "project_id": "forum",
    "total_requests": 1250,
    "success_requests": 1200,
    "failed_requests": 50,
    "success_rate": 0.96,
    "average_latency_ms": 450,
    "created_at": "2026-03-20T10:00:00Z",
    "statistics": {
      "by_date": {
        "2026-03-23": {
          "requests": 250,
          "success": 240,
          "failed": 10
        }
      }
    }
  }
}
```

---

## 🔑 认证配置

### 默认令牌

```
admin-token-default
```

### 自定义令牌

在 `.env` 文件中配置:

```bash
ADMIN_TOKEN=your-custom-token-here
```

支持多个令牌（逗号分隔）:

```bash
ADMIN_TOKEN=token1,token2,token3
```

### 更新令牌

修改 `.env` 后需要重启服务:

```bash
# 1. 编辑配置
nano .env
# ADMIN_TOKEN=new-token-here

# 2. 重启服务
pkill moderation-server
./moderation-server &
```

---

## 💡 常见使用场景

### 场景 1: 添加新客户项目

```bash
# 1. 生成新密钥
NEW_KEY="proj_customer_$(date +%s | md5sum | cut -c1-10)"

# 2. 通过 API 添加
curl -X POST http://localhost:8088/v1/admin/keys \
  -H "Authorization: Bearer admin-token-default" \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "customer_name",
    "key": "'$NEW_KEY'",
    "rate_limit": 500
  }'

# 3. Web UI 中验证
# 访问 http://localhost:8088/admin/ 查看新密钥
```

---

### 场景 2: 监控项目使用情况

```bash
# 获取项目统计
curl -H "Authorization: Bearer admin-token-default" \
  'http://localhost:8088/v1/admin/projects/stats?project=forum'

# 查看最近的日志
curl -H "Authorization: Bearer admin-token-default" \
  'http://localhost:8088/v1/admin/projects/logs?project=forum&end=2026-03-23'
```

---

### 场景 3: 禁用/启用密钥

```bash
# 禁用密钥（不删除）
curl -X PUT http://localhost:8088/v1/admin/keys/proj_forum_a_k3j9x2m1 \
  -H "Authorization: Bearer admin-token-default" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": false
  }'

# 启用密钥
curl -X PUT http://localhost:8088/v1/admin/keys/proj_forum_a_k3j9x2m1 \
  -H "Authorization: Bearer admin-token-default" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true
  }'
```

---

### 场景 4: 调整项目速率限制

```bash
# 增加限制
curl -X PUT http://localhost:8088/v1/admin/keys/proj_forum_a_k3j9x2m1 \
  -H "Authorization: Bearer admin-token-default" \
  -H "Content-Type: application/json" \
  -d '{
    "rate_limit": 5000
  }'
```

---

## 🛠️ 故障排查

### 问题 1: 访问 `/admin/` 返回 404

**原因**: 服务未启动或端口错误

**解决**:
```bash
# 检查服务是否运行
ps aux | grep moderation-server

# 检查日志
tail -50 server.log

# 核实端口配置
grep PORT .env
```

---

### 问题 2: API 返回 "无效的管理员令牌"

**原因**: 令牌不匹配或未配置

**解决**:
```bash
# 1. 检查 .env 配置
grep ADMIN_TOKEN .env

# 2. 确保使用正确的令牌
curl -H "Authorization: Bearer admin-token-default" \
  http://localhost:8088/v1/admin/health

# 3. 如果仍不工作，尝试默认令牌
# ADMIN_TOKEN=admin-token-default
```

---

### 问题 3: Web UI 打开但无法加载内容

**原因**: 静态文件加载失败

**解决**:
```bash
# 检查 static 目录是否存在
ls -la internal/admin/static/

# 检查服务日志
tail -100 server.log | grep -i static

# 重新编译
go build -o moderation-server ./cmd/server
```

---

## 📊 集成到您的应用

### JavaScript/Node.js

```javascript
const adminToken = 'admin-token-default';
const adminUrl = 'http://localhost:8088/v1/admin';

// 获取所有项目
async function getProjects() {
  const response = await fetch(`${adminUrl}/projects`, {
    headers: {
      'Authorization': `Bearer ${adminToken}`
    }
  });
  return response.json();
}

// 添加新密钥
async function addKey(projectId, key, rateLimit) {
  const response = await fetch(`${adminUrl}/keys`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${adminToken}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      project_id: projectId,
      key: key,
      rate_limit: rateLimit
    })
  });
  return response.json();
}
```

### Python

```python
import requests

admin_token = 'admin-token-default'
admin_url = 'http://localhost:8088/v1/admin'

headers = {
    'Authorization': f'Bearer {admin_token}'
}

# 获取所有项目
response = requests.get(f'{admin_url}/projects', headers=headers)
projects = response.json()

# 添加新密钥
new_key = {
    'project_id': 'test',
    'key': 'proj_test_123',
    'rate_limit': 1000
}
response = requests.post(
    f'{admin_url}/keys',
    json=new_key,
    headers=headers
)
```

---

## 🎯 下一步

1. 📱 **尝试 Web UI**: http://localhost:8088/admin/
2. 🔌 **测试 API**: 使用上面的命令
3. 🔑 **添加密钥**: 通过 API 或 Web UI
4. 📊 **监控项目**: 查看实时统计和日志
5. 🚀 **部署到生产**: 使用 deploy-to-production.sh

---

**状态**: ✅ 管理后台已完全就绪！
**最后更新**: 2026-03-23 13:16:01
