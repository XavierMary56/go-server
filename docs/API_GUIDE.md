# AI 内容审核服务接口文档 (API Guide)

本文档提供 AI 内容审核服务的完整接口说明，包含面向前端业务的公共 API (V1/V2) 和面向系统管理的 Admin API。

---

## 1. 基础信息

- **本地 URL**: `http://localhost:888`
- **生产 URL**: `https://zyaokkmo.cc`
- **认证方式**:
  - **业务 API**: Header 携带 `X-Project-Key`。
  - **管理 API**: Header 携带 `Authorization: Bearer <ADMIN_TOKEN>` 或 `x-api-key: <ADMIN_TOKEN>`。
- **数据格式**: `application/json`

---

## 2. 公共审核 API

系统优先通过本地 **Hard Rules** 规则库进行秒级拦截（延迟 0ms）。如果未触发硬阻断，则进入 AI 模型队列进行深度审核。

### 2.1 同步审核 V2（推荐，优先使用）

- **POST** `/v2/moderations`
- V2 为新版接口，返回结构更规范（`data.result` 嵌套），便于前端统一解析。**新接入项目请统一使用 V2。**

#### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `content` | string | **是** | - | 待审核的文本内容 |
| `type` | string | 否 | `post` | 内容类型，影响模型审核上下文判断 |
| `model` | string | 否 | `auto` | 指定审核模型，`auto` 按权重随机选择 |
| `strictness` | string | 否 | `standard` | 审核严格程度 |
| `webhook_url` | string | 否 | - | 异步回调地址（异步模式时使用） |
| `context` | object | 否 | - | 附加上下文信息，详见下方说明 |

#### `type` 可选值

| 值 | 说明 |
|----|------|
| `post` | 帖子/动态内容（默认） |
| `comment` | 评论内容 |
| `text` | 纯文本内容 |

> `type` 为自由文本字段，以上为推荐值。传入其他值也可正常工作，该值会作为上下文提示传递给 AI 模型。

#### `model` 可选值

| 值 | 供应商 | 说明 |
|----|--------|------|
| `auto` | 自动 | 按权重随机选择可用模型（默认） |
| `claude-haiku-4-5` | Anthropic | Claude Haiku 4.5，速度快、成本低 |
| `claude-sonnet-4-5` | Anthropic | Claude Sonnet 4.5，综合能力强 |
| `gpt-4o` | OpenAI | GPT-4o |
| `gpt-4o-mini` | OpenAI | GPT-4o Mini，速度快、成本低 |
| `o1-*` | OpenAI | OpenAI o1 系列 |
| `o3-*` | OpenAI | OpenAI o3 系列 |
| `o4-*` | OpenAI | OpenAI o4 系列 |
| `grok-*` | xAI | Grok 系列 |

> 实际可用模型取决于管理后台的模型配置和对应供应商密钥是否有效。可通过 `GET /v1/models` 查看当前可用模型列表。

**供应商自动识别规则**：
- `claude-*` 前缀 → Anthropic
- `gpt-*` / `o1-*` / `o3-*` / `o4-*` 前缀 → OpenAI
- `grok-*` 前缀 → xAI (Grok)
- 其他 → 默认走 Anthropic

#### `strictness` 可选值

| 值 | 说明 |
|----|------|
| `standard` | 标准模式（默认）。明确违规拒绝，正常讨论放行，边界内容可标记 |
| `strict` | 严格模式。有明显风险或强烈嫌疑即拒绝，不给可疑内容放行空间 |
| `loose` | 宽松模式。仅拦截明确违规内容，正常讨论一律放行 |

#### `context` 附加上下文（可选）

用于携带更丰富的业务信息，提升模型审核准确度：

```json
{
  "context": {
    "scene": "product_review",
    "payload": {
      "title": "商品标题",
      "content": "商品详细描述"
    }
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `context.scene` | string | 业务场景标识（如 `product_review`、`chat_message`） |
| `context.payload.title` | string | 标题内容（会拼接进审核文本） |
| `context.payload.content` | string | 正文内容（会拼接进审核文本） |

#### 请求示例

```json
{
  "content": "这是一条评论",
  "type": "comment",
  "model": "auto",
  "strictness": "standard"
}
```

#### 响应参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `verdict` | string | 审核结论：`approved`（通过）/ `rejected`（拒绝）/ `flagged`（待人工复审） |
| `category` | string | 违规分类：`none` / `spam` / `adult` / `fraud` / `abuse` / `politics` / `violence` |
| `confidence` | float | 置信度 0-1 |
| `reason` | string | 审核原因说明（中文） |
| `model_used` | string | 使用的模型：`hard-rule`（规则引擎）/ 具体模型 ID / `fallback`（兜底） |
| `latency_ms` | int | 审核耗时（毫秒），规则引擎命中时为 0 |
| `from_cache` | bool | 是否命中缓存 |

#### 响应示例（规则引擎拦截）

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": "mod_1775114787528719570",
    "result": {
      "verdict": "rejected",
      "category": "fraud",
      "confidence": 0.8,
      "reason": "命中诈骗、赌博或黑产内容",
      "model_used": "hard-rule",
      "latency_ms": 0,
      "from_cache": false
    },
    "status": "completed"
  }
}
```

#### 响应示例（AI 模型审核通过）

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": "mod_1775114354814252945",
    "result": {
      "verdict": "approved",
      "category": "none",
      "confidence": 0.92,
      "reason": "内容为正常讨论，未发现违规信息",
      "model_used": "claude-haiku-4-5",
      "latency_ms": 1580,
      "from_cache": false
    },
    "status": "completed"
  }
}
```

### 2.2 同步审核 V1（旧版兼容，不推荐新接入使用）

- **POST** `/v1/moderate`
- **请求参数**: 与 V2 完全一致。
- **说明**: V1 为旧版接口，返回结构为平铺模式，无 `data` 包装，结果字段直接在顶层。已接入的项目可继续使用，新项目请使用 V2。

#### V1 响应示例

```json
{
  "code": 200,
  "verdict": "rejected",
  "category": "spam",
  "confidence": 0.85,
  "reason": "命中广告导流或联系方式",
  "model_used": "hard-rule",
  "latency_ms": 0,
  "from_cache": false
}
```

### 2.3 审核流程说明

```
请求 → Hard Rules 规则引擎（毫秒级）
        ├─ 命中 → 直接返回 rejected（不调用模型）
        └─ 未命中 → 检查缓存
                     ├─ 命中缓存 → 返回缓存结果
                     └─ 未命中 → AI 模型队列（按优先级逐个尝试）
                                  ├─ 成功 → 返回模型结果
                                  └─ 全部失败 → safeFallback 兜底（flagged + 转人工）
```

---

## 3. 管理端接口 (Admin API)

所有管理接口需要在 Header 中携带 `Authorization: Bearer <ADMIN_TOKEN>` 或 `x-api-key: <ADMIN_TOKEN>`。

### 3.1 密钥管理
- **GET** `/v1/admin/keys`: 列出所有接入项目。
- **POST** `/v1/admin/keys`: 动态新增项目密钥（立即生效，无需重启）。

### 3.2 供应商密钥管理
- **GET** `/v1/admin/anthropic-keys`: 列出 Anthropic 供应商密钥。
- **POST** `/v1/admin/anthropic-keys/check`: 检查指定密钥健康状态。
- 同理支持 OpenAI、Grok 密钥管理。

### 3.3 模型配置
- **GET** `/v1/admin/models`: 列出已配置的模型。
- **POST** `/v1/admin/models`: 新增模型配置。

### 3.4 审计日志查询
- **GET** `/v1/admin/projects/logs`: 查询指定项目的审核流水。支持 `project`, `start`, `end` 参数。

### 3.5 统计与监控
- **GET** `/v1/stats`: 查看实时各模型调用占比与统计数据。

---

## 4. 常见问题 (FAQ)

**Q: `type` 参数传什么值？**
A: 推荐使用 `post`（帖子）、`comment`（评论）、`text`（纯文本）。该字段为自由文本，传入其他值也不会报错，值会作为上下文传给 AI 模型辅助判断。不传时默认为 `post`。

**Q: `model` 参数传什么值？**
A: 推荐使用 `auto`（默认，自动选择）。如需指定模型，可通过 `GET /v1/models` 查看当前可用模型列表。指定的模型不存在时会回退到自动选择。

**Q: `strictness` 怎么选？**
A: 大多数业务场景使用 `standard`（默认）即可。对于用户生成内容（UGC）评论区等需要宽松一些的场景可用 `loose`；对于涉及未成年人、金融等敏感场景可用 `strict`。

**Q: 为什么某些内容在 V1 通过了，在 V2 被拦截？**
A: 两个接口底层共用一套审核引擎。如果出现不一致，请检查是否是因为 60s 缓存导致的。现在版本已强制统一拦截逻辑。

**Q: 返回 `model_used: "fallback"` 是什么意思？**
A: 表示所有配置的 AI 模型均调用失败（密钥无效、超时、网络异常等），系统使用兜底策略返回 `flagged`，建议转人工处理。请检查供应商密钥健康状态。

**Q: 如何新增拦截关键字？**
A: 核心关键字在 `internal/service/dictionary.go` 中维护，修改后重新部署即可生效。

**Q: 使用 curl 测试中文内容结果异常？**
A: Windows Git Bash 下 curl 可能以 GBK 编码发送中文，导致服务端收到乱码。解决方案：将 JSON 写入 UTF-8 文件后用 `curl -d @file.json` 发送，或使用 PowerShell / RunAPI / Postman 测试。

---

## 5. 运维指南 (简版)

- **健康检查**: `GET /v1/health`
- **模型列表**: `GET /v1/models`
- **审计日志路径**: `logs/audit/<project_name>/audit_YYYY-MM-DD.log`
- **统计指标**: `GET /v1/stats`（查看实时各模型调用占比）
- **部署更新**: `git pull origin main && docker-compose up -d --build`
