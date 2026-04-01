# AI 内容审核服务接口文档 (API Guide)

本文档提供 AI 内容审核服务的完整接口说明，包含面向前端业务的公共 API (V1/V2) 和面向系统管理的 Admin API。

---

## 1. 基础信息

- **本地 URL**: `http://localhost:888`
- **生产 URL**: `https://zyaokkmo.cc`
- **认证方式**:
  - **业务 API**: Header 携带 `X-Project-Key`。
  - **管理 API**: Header 携带 `Authorization: Bearer <ADMIN_TOKEN>`。
- **数据格式**: `application/json`

---

## 2. 公共审核 API

系统优先通过本地 **Hard Rules** 规则库进行秒级拦截。如果未触发硬阻断，则进入 AI 模型队列进行深度审核。

### 2.1 同步审核 (推荐 V2)
- **POST** `/v2/moderations`
- **请求体**:
```json
{
  "content": "qq我",
  "type": "comment",
  "strictness": "standard" // loose | standard | strict
}
```
- **响应 (拦截示例)**:
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "result": {
      "verdict": "rejected",
      "category": "spam",
      "reason": "命中广告导流或联系方式",
      "model_used": "hard-rule",
      "latency_ms": 0
    }
  }
}
```

### 2.2 同步审核 (兼容 V1)
- **POST** `/v1/moderate`
- **说明**: 返回结构为平铺模式，无 `data` 包装。

---

## 3. 管理端接口 (Admin API)

### 3.1 密钥管理
- **GET** `/v1/admin/keys`: 列出所有接入项目。
- **POST** `/v1/admin/keys`: 动态新增项目密钥 (立即生效，无需重启)。

### 3.2 审计日志查询
- **GET** `/v1/admin/projects/logs`: 查询指定项目的审核流水。支持 `project`, `start`, `end` 参数。

---

## 4. 常见问题 (FAQ)

**Q: 为什么某些内容在 V1 通过了，在 V2 被拦截？**
A: 两个接口底层共用一套审核引擎。如果出现不一致，请检查是否是因为 60s 缓存导致的。现在版本已强制统一拦截逻辑。

**Q: 如何新增拦截关键字？**
A: 核心关键字在 `internal/service/dictionary.go` 中维护，修改后执行 `bash deploy.sh` 即可。

---

## 5. 运维指南 (简版)

- **健康检查**: `GET /v1/health`
- **审计日志路径**: `logs/audit/<project_name>/audit_YYYY-MM-DD.log`
- **统计指标**: `GET /v1/stats` (查看实时各模型调用占比)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
