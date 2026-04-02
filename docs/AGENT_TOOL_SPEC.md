# AI Agent Tool 定义 — 内容审核服务

本文档为 AI Agent（如 Claude Agent SDK、OpenAI Function Calling、LangChain 等）提供标准化的 tool 定义，使 agent 可直接调用内容审核服务。

---

## 1. Tool 定义（JSON Schema）

### 1.1 content_moderation — 内容审核

```json
{
  "name": "content_moderation",
  "description": "审核用户生成的文本内容（评论、帖子、消息等），判断是否包含违规信息（色情、暴力、赌博、政治敏感、广告导流等）。返回审核结论、违规分类、置信度和原因说明。",
  "input_schema": {
    "type": "object",
    "properties": {
      "content": {
        "type": "string",
        "description": "待审核的文本内容"
      },
      "type": {
        "type": "string",
        "enum": ["post", "comment", "text"],
        "default": "post",
        "description": "内容类型。post=帖子/动态，comment=评论，text=纯文本"
      },
      "strictness": {
        "type": "string",
        "enum": ["standard", "strict", "loose"],
        "default": "standard",
        "description": "审核严格程度。standard=标准，strict=严格（有嫌疑即拒绝），loose=宽松（仅拦截明确违规）"
      },
      "scene": {
        "type": "string",
        "description": "业务场景标识，如 product_review、chat_message、forum_post"
      },
      "title": {
        "type": "string",
        "description": "内容标题（可选，帖子/文章类型时提供）"
      }
    },
    "required": ["content"]
  }
}
```

### 1.2 moderation_health_check — 审核服务健康检查

```json
{
  "name": "moderation_health_check",
  "description": "检查内容审核服务的运行状态，包括服务是否可用、模型队列状态等。",
  "input_schema": {
    "type": "object",
    "properties": {},
    "required": []
  }
}
```

---

## 2. Agent 调用实现示例

### 2.1 Claude Agent SDK (Python)

```python
import anthropic
import httpx

MODERATION_API_URL = "https://zyaokkmo.cc/v2/moderations"
MODERATION_API_KEY = "sk-proj-xxx"  # 你的项目密钥


def call_moderation(content: str, content_type: str = "post", strictness: str = "standard", scene: str = "", title: str = "") -> dict:
    """调用内容审核服务"""
    payload = {
        "content": content,
        "type": content_type,
        "model": "auto",
        "strictness": strictness,
    }

    # 附加上下文
    if scene or title:
        context = {}
        if scene:
            context["scene"] = scene
        if title:
            context["payload"] = {"title": title}
        payload["context"] = context

    resp = httpx.post(
        MODERATION_API_URL,
        json=payload,
        headers={
            "X-Project-Key": MODERATION_API_KEY,
            "Content-Type": "application/json",
        },
        timeout=15.0,
    )
    resp.raise_for_status()
    return resp.json()


# 定义 tool 供 Claude Agent 使用
content_moderation_tool = {
    "name": "content_moderation",
    "description": "审核用户生成的文本内容，判断是否包含违规信息。返回审核结论（approved/rejected/flagged）、违规分类和原因。",
    "input_schema": {
        "type": "object",
        "properties": {
            "content": {
                "type": "string",
                "description": "待审核的文本内容",
            },
            "type": {
                "type": "string",
                "enum": ["post", "comment", "text"],
                "default": "post",
                "description": "内容类型",
            },
            "strictness": {
                "type": "string",
                "enum": ["standard", "strict", "loose"],
                "default": "standard",
                "description": "审核严格程度",
            },
        },
        "required": ["content"],
    },
}

# 使用示例：创建带审核工具的 agent
client = anthropic.Anthropic()

response = client.messages.create(
    model="claude-sonnet-4-5-20250514",
    max_tokens=1024,
    tools=[content_moderation_tool],
    messages=[
        {"role": "user", "content": "请帮我审核这条评论：'加微信领红包 test123'"}
    ],
)

# 处理 tool_use
for block in response.content:
    if block.type == "tool_use" and block.name == "content_moderation":
        result = call_moderation(**block.input)
        print(result)
```

### 2.2 OpenAI Function Calling

```python
import openai
import httpx

MODERATION_API_URL = "https://zyaokkmo.cc/v2/moderations"
MODERATION_API_KEY = "sk-proj-xxx"

# OpenAI function 定义
functions = [
    {
        "name": "content_moderation",
        "description": "审核用户文本内容，判断是否违规",
        "parameters": {
            "type": "object",
            "properties": {
                "content": {"type": "string", "description": "待审核文本"},
                "type": {"type": "string", "enum": ["post", "comment", "text"]},
                "strictness": {"type": "string", "enum": ["standard", "strict", "loose"]},
            },
            "required": ["content"],
        },
    }
]

# agent 调用后处理 function_call
def handle_moderation_call(args: dict) -> dict:
    resp = httpx.post(
        MODERATION_API_URL,
        json={
            "content": args["content"],
            "type": args.get("type", "post"),
            "model": "auto",
            "strictness": args.get("strictness", "standard"),
        },
        headers={
            "X-Project-Key": MODERATION_API_KEY,
            "Content-Type": "application/json",
        },
        timeout=15.0,
    )
    return resp.json()
```

### 2.3 PHP 业务系统集成

```php
<?php
/**
 * 内容审核服务客户端
 * 适用于 PHP 7.3+
 */
class ModerationClient
{
    private $api_url;
    private $project_key;

    public function __construct($api_url, $project_key)
    {
        $this->api_url = rtrim($api_url, '/');
        $this->project_key = $project_key;
    }

    /**
     * 审核内容
     *
     * @param string $content    待审核文本
     * @param string $type       内容类型: post|comment|text
     * @param string $strictness 严格程度: standard|strict|loose
     * @return array
     */
    public function moderate($content, $type = 'post', $strictness = 'standard')
    {
        $payload = json_encode([
            'content'    => $content,
            'type'       => $type,
            'model'      => 'auto',
            'strictness' => $strictness,
        ], JSON_UNESCAPED_UNICODE);

        $ch = curl_init($this->api_url . '/v2/moderations');
        curl_setopt_array($ch, [
            CURLOPT_POST           => true,
            CURLOPT_POSTFIELDS     => $payload,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT        => 15,
            CURLOPT_CONNECTTIMEOUT => 5,
            CURLOPT_HTTPHEADER     => [
                'Content-Type: application/json; charset=utf-8',
                'X-Project-Key: ' . $this->project_key,
            ],
        ]);

        $response = curl_exec($ch);
        $http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $error = curl_error($ch);
        curl_close($ch);

        if ($error) {
            return ['code' => 500, 'error' => $error];
        }

        return json_decode($response, true);
    }

    /**
     * 判断内容是否通过审核
     *
     * @param string $content
     * @param string $type
     * @param string $strictness
     * @return bool
     */
    public function is_approved($content, $type = 'post', $strictness = 'standard')
    {
        $result = $this->moderate($content, $type, $strictness);
        $verdict = $result['data']['result']['verdict'] ?? 'flagged';
        return $verdict === 'approved';
    }
}

// 使用示例
$client = new ModerationClient('https://zyaokkmo.cc', 'sk-proj-xxx');

// 审核评论
$result = $client->moderate('加微信领红包 test123', 'comment');
// $result['data']['result']['verdict'] => "rejected"

// 快速判断
if ($client->is_approved($user_comment, 'comment')) {
    // 通过，正常展示
} else {
    // 拒绝或待审，进入人工队列
}
```

---

## 3. 响应结果处理指南

### 3.1 verdict 判定逻辑

| verdict | 含义 | Agent 处理建议 |
|---------|------|---------------|
| `approved` | 内容通过审核 | 允许发布/展示 |
| `rejected` | 内容违规被拒 | 阻止发布，提示用户修改 |
| `flagged` | 内容存疑待人工复审 | 暂时隐藏，推入人工队列 |

### 3.2 category 违规分类

| category | 含义 | 示例 |
|----------|------|------|
| `none` | 无违规 | 正常评论 |
| `spam` | 广告/导流/联系方式 | "加微信 xxx"、"QQ群 123456" |
| `adult` | 色情/成人内容 | 约炮、裸聊 |
| `fraud` | 诈骗/赌博/黑产 | 赌博、彩票、刷单 |
| `abuse` | 毒品/违禁品 | 吸毒、大麻 |
| `politics` | 政治敏感 | 涉政敏感词 |
| `violence` | 暴力/恐怖 | 杀人、恐怖袭击 |

### 3.3 model_used 审核来源

| model_used | 含义 | 延迟 |
|------------|------|------|
| `hard-rule` | 本地规则引擎命中 | 0ms |
| `claude-*` / `gpt-*` | AI 模型审核 | 1000-5000ms |
| `fallback` | 所有模型失败，兜底处理 | 不定 |

### 3.4 Agent 推荐处理流程

```
收到审核结果
  ├─ verdict == "approved"
  │    └─ 直接放行
  ├─ verdict == "rejected"
  │    ├─ confidence >= 0.8 → 直接拒绝，告知用户违规原因
  │    └─ confidence < 0.8  → 拒绝但建议人工复核
  └─ verdict == "flagged"
       ├─ model_used == "fallback" → 服务异常，建议稍后重试或转人工
       └─ 其他 → 推入人工审核队列
```

---

## 4. 错误处理

### 4.1 HTTP 状态码

| 状态码 | 含义 | Agent 处理 |
|--------|------|-----------|
| 200 | 成功 | 正常解析 result |
| 400 | 请求参数错误 | 检查 content 是否为空 |
| 401 | 认证失败 | 检查 X-Project-Key |
| 429 | 请求过于频繁 | 退避重试（建议 2-5 秒） |
| 500 | 服务内部错误 | 重试或降级处理 |

### 4.2 超时与重试建议

```
超时设置: 15 秒（硬规则 0ms，模型审核通常 1-5 秒）
重试策略: 最多重试 2 次，间隔 2 秒
降级方案: 审核服务不可用时，将内容标记为待人工审核
```

---

## 5. 接入清单

接入前确认以下事项：

- [ ] 获取项目密钥（`X-Project-Key`），通过管理后台创建
- [ ] 确认使用 V2 接口（`/v2/moderations`）
- [ ] 设置合理的超时时间（建议 15 秒）
- [ ] 实现降级策略（服务不可用时的兜底逻辑）
- [ ] 请求体使用 **UTF-8 编码**
- [ ] 根据业务场景选择合适的 `strictness` 级别
