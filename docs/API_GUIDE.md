# AI 内容审核服务接口文档 (API Guide)

本文档提供 AI 内容审核服务的完整接口说明，包含面向前端业务的公共 API (V1/V2) 和面向系统管理的 Admin API。

---

## 1. 基础信息

- **基础 URL**: `https://ai.a889.cloud` (生产) 或 `http://localhost:8080` (本地)
- **认证方式**:
  - **公共 API**: 在请求头中使用 `X-Project-Key`。
  - **Admin API**: 在请求头中使用 `Authorization: Bearer <ADMIN_TOKEN>`。
- **数据格式**: `application/json`

---

## 2. 公共 API (业务对接)

### 2.1 健康检查
- **GET** `/v1/health` 或 `/v2/health`
- **说明**: 检查服务是否存活，无需鉴权。

### 2.2 内容审核 (同步)
- **POST** `/v1/moderate` (旧版响应结构)
- **POST** `/v2/moderations` (推荐，带 `data` 包装)
- **请求头**:
  - `X-Project-Key`: 必须，项目接入密钥
- **请求体**:
```json
{
  "content": "待审核的文本内容",
  "type": "comment",        // comment (默认) | post
  "strictness": "standard", // loose | standard (默认) | strict
  "model": "",              // 可选，指定模型 ID
  "context": {"user_id": "1001"} // 可选，业务上下文
}
```
- **响应示例 (V2)**:
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": "mod_1711900000000000000",
    "status": "completed",
    "result": {
      "verdict": "approved", // approved | flagged | rejected
      "category": "none",     // none | spam | abuse | politics | adult | fraud | violence
      "reason": "内容正常",
      "confidence": 0.98,
      "model_used": "claude-3-5-sonnet-20241022",
      "latency_ms": 1234,
      "from_cache": false
    }
  }
}
```

### 2.3 内容审核 (异步)
- **POST** `/v1/moderate/async`
- **POST** `/v2/moderations/async`
- **参数**: 同同步接口，额外支持 `webhook_url`。
- **查询结果**: `GET /v1/task/{task_id}` 或 `GET /v2/tasks/{task_id}`。

---

## 3. 管理 API (Admin API)

所有管理接口均需 `Authorization: Bearer <ADMIN_TOKEN>` 鉴权。

### 3.1 项目密钥管理
- **GET** `/v1/admin/keys`: 列出所有项目密钥
- **POST** `/v1/admin/keys`: 添加密钥
  - 请求体: `{"project_id": "name", "key": "sk-xxx", "rate_limit": 300}`
- **PUT** `/v1/admin/keys/:id`: 更新密钥 (包含名称、速率限制、开关状态)
- **DELETE** `/v1/admin/keys/:id`: 删除密钥

### 3.2 供应商与模型管理
- **GET** `/v1/admin/anthropic-keys`: Anthropic 密钥列表
- **GET** `/v1/admin/provider-keys?provider=openai|grok`: 第三方密钥列表
- **POST** `/v1/admin/provider-keys/check`: 检测密钥可用性 (参数 `{"id": 123}`)
- **GET** `/v1/admin/models`: 获取所有模型配置
- **PUT** `/v1/admin/models/:id`: 更新模型权重或优先级

### 3.3 统计与日志
- **GET** `/v1/admin/projects/stats`: 获取各项目的调用统计 (请求数、限流数、错误数等)
- **GET** `/v1/admin/projects/logs?project=xxx&start=YYYY-MM-DD&end=YYYY-MM-DD`: 查询特定项目的审计日志

---

## 4. 常见错误码

| 状态码 | 含义 | 处理建议 |
|--------|------|----------|
| 200    | 成功 | -	|
| 400    | 参数错误 | 检查请求体格式或必填项 |
| 401    | 鉴权失败 | 检查 X-Project-Key 或 Admin Token |
| 429    | 触发限流 | 降低请求频率或联系管理员调优配额 |
| 500    | 服务错误 | 检查服务日志 |

---

## 5. 运维指南 (简版)

### 5.1 日志位置
- 应用日志: `logs/moderation_YYYY-MM-DD.log`
- 审计日志: `logs/audit/<project_id>/audit_YYYY-MM-DD.log` (按项目隔离)

### 5.2 常用脚本
- `bash deploy.sh`: 自动化部署脚本
- `bash monitor.sh status`: 查看服务健康状态

🤖 Generated with [Claude Code](https://claude.com/claude-code)