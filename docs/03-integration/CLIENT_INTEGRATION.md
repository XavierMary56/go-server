# 📱 客户端接入指南 + Demo

## 🎯 说明

本文档用于指导**新项目** 如何对接我们的 AI 内容审核服务。包含完整的 demo 代码和 API 调用示例。

---

## 📋 快速导航

| 章节 | 内容 | 耗时 |
|-----|------|------|
| [1. 对接流程](#1-对接流程) | 3步快速对接 | 5分钟 |
| [2. API 调用示例](#2-api-调用示例) | 完整的 Demo 代码 | 查阅 |
| [3. 错误处理](#3-错误处理) | 常见错误和解决方案 | 查阅 |
| [4. 最佳实践](#4-最佳实践) | 常见问题和建议 | 查阅 |
| [5. 技术支持](#5-技术支持) | 联系方式 | 查阅 |

---

## 1. 对接流程

### Step 1️⃣：通知管理员添加你的项目

联系管理员，提供：
- **项目名称**：例如 `51dm_service`
- **所需限流数**：例如 `300` (每分钟请求数)

管理员会返回给你：
- **API Key**：`sk-proj-51dm-a1b2c3d4e5f6g7h8`
- **API 地址**：`https://ai.a889.cloud`
- **API 版本**：默认推荐 `v2`（兼容保留 `v1`）

### Step 2️⃣：在你的代码中配置

在你的配置文件、環境变量或配置中心中添加：

```env
# 审核服务配置
MODERATION_API_URL=https://ai.a889.cloud
MODERATION_API_KEY=sk-proj-51dm-a1b2c3d4e5f6g7h8
MODERATION_API_VERSION=v2
MODERATION_API_TIMEOUT=10000  # 毫秒
```

### Step 3️⃣：集成 SDK 或 HTTP 调用

选择以下方式之一集成。

---

## 2. API 调用示例

### 2.0 V1 / V2 版本选择

- **推荐新接入**：使用 V2
  - `POST /v2/moderations`
  - `POST /v2/moderations/async`
  - `GET /v2/tasks/{id}`
- **兼容旧项目**：继续使用 V1
  - `POST /v1/moderate`
  - `POST /v1/moderate/async`
  - `GET /v1/task/{id}`

主要区别：
- V1 是动作式路径
- V2 是资源式路径
- V2 统一使用 `code + message + data`
- V2 同步审核结果位于 `data.result`

### 2.1 PHP 集成（推荐）

#### 安装 SDK

```php
// 推荐：使用官方 SDK（如果有）
// composer require moderation/php-sdk

// 或者：直接使用 cURL
```

#### 同步审核示例

```php
<?php

class ContentModerationClient {
    private $apiUrl;
    private $apiKey;
    private $timeout;

    public function __construct($apiUrl, $apiKey, $timeout = 10, $apiVersion = 'v2') {
        $this->apiUrl = $apiUrl;
        $this->apiKey = $apiKey;
        $this->timeout = $timeout;
        $this->apiVersion = strtolower($apiVersion) === 'v1' ? 'v1' : 'v2';
    }

    /**
     * 同步审核内容
     */
    public function moderate($content, $type = 'comment') {
        $endpoint = $this->apiVersion === 'v1'
            ? $this->apiUrl . '/v1/moderate'
            : $this->apiUrl . '/v2/moderations';

        $payload = [
            'content' => $content,
            'type' => $type,
            'strictness' => 'standard'
        ];

        $response = $this->request('POST', $endpoint, $payload);
        if ($this->apiVersion === 'v2' && isset($response['data']['result'])) {
            return $response['data']['result'];
        }
        return $response;
    }

    /**
     * 异步审核内容
     */
    public function moderateAsync($content, $webhookUrl, $type = 'comment') {
        $endpoint = $this->apiVersion === 'v1'
            ? $this->apiUrl . '/v1/moderate/async'
            : $this->apiUrl . '/v2/moderations/async';

        $payload = [
            'content' => $content,
            'type' => $type,
            'webhook_url' => $webhookUrl
        ];

        return $this->request('POST', $endpoint, $payload);
    }

    /**
     * 查询任务结果
     */
    public function getTaskResult($taskId) {
        $endpoint = $this->apiVersion === 'v1'
            ? $this->apiUrl . '/v1/task/' . $taskId
            : $this->apiUrl . '/v2/tasks/' . $taskId;

        $response = $this->request('GET', $endpoint, null);
        return $response['data'] ?? $response;
    }

    /**
     * 查看可用模型
     */
    public function getModels() {
        $endpoint = $this->apiUrl . '/v1/models';
        return $this->request('GET', $endpoint, null);
    }

    /**
     * 查看服务统计
     */
    public function getStats() {
        $endpoint = $this->apiUrl . '/v1/stats';
        return $this->request('GET', $endpoint, null);
    }

    /**
     * 内部方法：发送 HTTP 请求
     */
    private function request($method, $url, $data = null) {
        $ch = curl_init();

        $headers = [
            'Content-Type: application/json',
            'X-Project-Key: ' . $this->apiKey
        ];

        curl_setopt_array($ch, [
            CURLOPT_URL => $url,
            CURLOPT_CUSTOMREQUEST => $method,
            CURLOPT_HTTPHEADER => $headers,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => $this->timeout,
            CURLOPT_SSL_VERIFYPEER => true,
            CURLOPT_SSL_VERIFYHOST => 2,
        ]);

        if ($data !== null && in_array($method, ['POST', 'PUT'])) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
        }

        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $error = curl_error($ch);
        curl_close($ch);

        if ($error) {
            return ['code' => 500, 'error' => 'Network error: ' . $error];
        }

        $result = json_decode($response, true);
        return $result ?? ['code' => 500, 'error' => 'Invalid response'];
    }
}

// ====== 使用示例 ======

$moderator = new ContentModerationClient(
    'https://ai.a889.cloud',
    'sk-proj-51dm-a1b2c3d4e5f6g7h8',
    10,
    'v2' // 新项目推荐 v2
);

// 同步审核
$result = $moderator->moderate('这是一条用户评论');

if (($result['verdict'] ?? '') === 'approved') {
    echo "审核通过\n";
} else {
    echo "审核结果: " . ($result['verdict'] ?? 'unknown') . "\n";
}

// 异步审核（接口保留，当前接入默认不优先使用）
$task = $moderator->moderateAsync(
    '这是一条用户帖子',
    'https://yourdomain.com/api/moderation/callback'
);

if ($task['success']) {
    echo "任务 ID: " . $task['task_id'] . "\n";
}

?>
```

#### Webhook 回调处理

```php
<?php

// 接收审核结果回调
// Route: POST /api/moderation/callback

$jsonInput = file_get_contents('php://input');
$event = json_decode($jsonInput, true);

if (!$event) {
    http_response_code(400);
    exit('Invalid JSON');
}

// 处理回调
$taskId = $event['task_id'];
$verdict = $event['verdict'];  // approved | flagged | rejected
$confidence = $event['confidence'];

// 根据审核结果处理
switch ($verdict) {
    case 'approved':
        // 内容正常，允许发布
        echo "Content approved\n";
        break;

    case 'flagged':
        // 内容可疑，待人工复核
        echo "Content flagged for review\n";
        // 可以这里保存到人工审核队列
        break;

    case 'rejected':
        // 内容违规，直接拒绝
        echo "Content rejected\n";
        break;
}

// 返回 200 表示已接收回调
http_response_code(200);
echo json_encode(['success' => true]);

?>
```

---

### 2.2 Node.js 集成

```javascript
// npm install axios

const axios = require('axios');

class ContentModerator {
    constructor(apiUrl, apiKey, timeout = 10000) {
        this.apiUrl = apiUrl;
        this.apiKey = apiKey;
        this.timeout = timeout;
        this.client = axios.create({
            baseURL: apiUrl,
            timeout: timeout,
            headers: {
                'X-Project-Key': apiKey,
                'Content-Type': 'application/json'
            }
        });
    }

    /**
     * 同步审核
     */
    async moderate(content, type = 'comment') {
        try {
            const response = await this.client.post('/v1/moderate', {
                content,
                type,
                strictness: 'standard'
            });
            return response.data;
        } catch (error) {
            return {
                code: error.response?.status || 500,
                error: error.message
            };
        }
    }

    /**
     * 异步审核（接口保留，当前接入默认不优先使用）
     */
    async moderateAsync(content, webhookUrl, type = 'comment') {
        try {
            const response = await this.client.post('/v1/moderate/async', {
                content,
                type,
                webhook_url: webhookUrl
            });
            return response.data;
        } catch (error) {
            return {
                code: error.response?.status || 500,
                error: error.message
            };
        }
    }

    /**
     * 查询任务结果
     */
    async getTaskResult(taskId) {
        try {
            const response = await this.client.get(`/v1/task/${taskId}`);
            return response.data;
        } catch (error) {
            return {
                code: error.response?.status || 500,
                error: error.message
            };
        }
    }
}

// ====== 使用示例 ======

const moderator = new ContentModerator(
    'https://ai.a889.cloud',
    'sk-proj-51dm-a1b2c3d4e5f6g7h8'
);

// 同步审核
async function testModerate() {
    const result = await moderator.moderate('这是一条用户评论');
    console.log('审核结果:', result);
}

// 异步审核
async function testModerateAsync() {
    const task = await moderator.moderateAsync(
        '这是一条用户帖子',
        'https://yourdomain.com/api/moderation/callback'
    );
    console.log('任务已提交:', task.task_id);
}

testModerate();
```

---

### 2.3 cURL 命令示例

#### 同步审核

```bash
curl -X POST https://ai.a889.cloud/v2/moderations \
  -H "Content-Type: application/json" \
  -H "X-Project-Key: sk-proj-51dm-a1b2c3d4e5f6g7h8" \
  -d '{
    "content": "这是需要审核的内容",
    "type": "comment",
    "strictness": "standard"
  }'
```

> 如需兼容旧项目，可继续使用 `/v1/moderate`。

#### 异步审核（保留能力，当前接入暂不优先对接）

```bash
curl -X POST https://ai.a889.cloud/v2/moderations/async \
  -H "Content-Type: application/json" \
  -H "X-Project-Key: sk-proj-51dm-a1b2c3d4e5f6g7h8" \
  -d '{
    "content": "这是需要审核的内容",
    "type": "comment",
    "webhook_url": "https://yourdomain.com/webhook"
  }'
```

#### 查询任务

```bash
curl -H "X-Project-Key: sk-proj-51dm-a1b2c3d4e5f6g7h8" \
  https://ai.a889.cloud/v2/tasks/task_1234567890
```

#### 查看模型

```bash
curl -H "X-Project-Key: sk-proj-51dm-a1b2c3d4e5f6g7h8" \
  https://ai.a889.cloud/v1/models
```

#### 查看统计

```bash
curl -H "X-Project-Key: sk-proj-51dm-a1b2c3d4e5f6g7h8" \
  https://ai.a889.cloud/v1/stats
```

---

## 3. 错误处理

### 3.1 常见错误和处理

| 错误 | 原因 | 解决方案 |
|-----|------|--------|
| **401 Unauthorized** | API Key 无效或不存在 | 检查密钥是否正确，确保 `X-Project-Key` 头已添加 |
| **400 Bad Request** | 请求参数错误 | 检查 content 是否为空，type 是否有效 |
| **429 Too Many Requests** | 超过速率限制 | 等待后重试，或联系管理员提高限额（最多 300/分钟） |
| **500 Internal Server Error** | 服务器错误 | 检查服务状态，查看日志 |
| **Timeout** | 请求超时 | 增加超时时间；当前默认优先接入同步 API，必要时再评估异步 |

### 3.2 重试策略

```php
<?php

function callWithRetry($moderator, $content, $maxRetries = 3) {
    for ($i = 0; $i < $maxRetries; $i++) {
        try {
            $result = $moderator->moderate($content);

            if ($result['code'] === 200) {
                return $result;
            }

            // 如果是限流错误，等待后重试
            if ($result['code'] === 429) {
                sleep(2 ** $i);  // 指数退避
                continue;
            }

            // 其他错误直接返回
            return $result;

        } catch (Exception $e) {
            if ($i === $maxRetries - 1) {
                throw $e;
            }
            sleep(1);
        }
    }
}

?>
```

---

## 4. 最佳实践

### 4.1 当前接入建议

| 场景 | 当前建议 | 理由 |
|-----|------|------|
| 用户实时发表评论 | 同步 | 当前文档默认按 `/v1/moderate` 对接，直接拿到 verdict |
| 对重要内容的实时审核 | 同步 | 需要立即知道审核结果 |
| 用户实时发帖 | 同步 | 便于在发布前直接做放行/拦截判断 |
| 后台批量审核 | 视业务再评估 | 异步接口仍保留，但当前不是默认对接方案 |
| 长耗时内容审核 | 视业务再评估 | 如后续确实需要再接入 `/v1/moderate/async` |

### 4.2 缓存策略

```php
<?php

// 对相同内容的重复审核进行缓存
// 默认缓存 1 小时

class CachedModerator {
    private $moderator;
    private $cache;  // Redis 或其他缓存
    private $cacheTTL = 3600;

    public function moderate($content, $type = 'comment') {
        // 生成缓存 key
        $cacheKey = 'moderation:' . md5($content . $type);

        // 检查缓存
        $cached = $this->cache->get($cacheKey);
        if ($cached) {
            return json_decode($cached, true);
        }

        // 调用 API
        $result = $this->moderator->moderate($content, $type);

        // 缓存结果
        if ($result['code'] === 200) {
            $this->cache->set($cacheKey, json_encode($result), 'EX', $this->cacheTTL);
        }

        return $result;
    }
}

?>
```

### 4.3 速率限制处理

```php
<?php

// 避免超过速率限制的最佳方法

class RateLimitedModerator {
    private $moderator;
    private $requestsPerMinute = 300;
    private $lastRequestTime = 0;
    private $requestCount = 0;

    public function moderate($content, $type = 'comment') {
        // 重置计数器（每分钟）
        $now = time();
        if ($now - $this->lastRequestTime >= 60) {
            $this->requestCount = 0;
            $this->lastRequestTime = $now;
        }

        // 检查限制
        if ($this->requestCount >= $this->requestsPerMinute) {
            // 等待直到下一个窗口
            $sleepTime = 60 - ($now - $this->lastRequestTime);
            if ($sleepTime > 0) {
                sleep($sleepTime);
            }
            $this->requestCount = 0;
            $this->lastRequestTime = time();
        }

        // 发送请求
        $this->requestCount++;
        return $this->moderator->moderate($content, $type);
    }
}

?>
```

---

## 5. 技术支持

### 常见问题

**Q: 我应该使用同步还是异步 API？**
A: 当前默认建议使用同步 `/v1/moderate`。只有在后续明确需要长耗时、批量化、Webhook 回调链路时，再评估接入异步 `/v1/moderate/async`。

**Q: 审核结果的置信度是什么？**
A: 置信度（0-1）表示 AI 对结果的确定程度。置信度越高，结果越可靠。

**Q: 如果审核服务不可用怎么办？**
A:
1. 检查 API 密钥是否正确
2. 检查网络连接
3. 实施重试逻辑
4. 考虑降级策略（如临时允许所有内容进入人工队列）

**Q: 如何处理审核超时？**
A:
- 同步请求：优先配置重试策略和合理超时
- 异步请求：当前不是默认接入路径，如后续启用可使用 `task_id` 定期查询结果

**Q: 速率限制是怎样的？**
A: 每个 API Key 有独立的速率限制（默认 300 请求/分钟）。超过限制会返回 429 错误。如需提高，联系管理员。

### 获取帮助

1. **查看日志**：检查服务日志了解具体错误
   ```bash
   curl -H "Authorization: Bearer admin-token" \
     "https://ai.a889.cloud/v1/admin/projects/logs?project=51dm_service"
   ```

2. **查看指标**：通过 API 查看服务状态
   ```bash
   curl -H "X-Project-Key: sk-proj-xxxx" \
     https://ai.a889.cloud/v1/stats
   ```

3. **联系管理员**：
   - Email：admin@example.com
   - 钉钉群：审核服务支持组

---

## 📋 对接检查清单

对接前，确保你已经有：

- [ ] ✅ 获得了 API Key（`sk-proj-xxxx`）
- [ ] ✅ 获得了 API 地址（`https://ai.a889.cloud`）
- [ ] ✅ 了解了你的速率限制（例如 300/分钟）
- [ ] ✅ 当前默认按同步 `/v1/moderate` 对接
- [ ] ✅ 如后续启用异步，再准备 Webhook 端点接收回调
- [ ] ✅ 实现了错误处理和重试逻辑
- [ ] ✅ 测试了与生产环境的连接

---

**创建时间**：2026-03-23
**版本**：1.0.0
