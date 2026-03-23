<?php
/**
 * 内容审核服务
 *
 * 对接 go-server 内容审核 API，支持同步/异步两种审核模式。
 *
 * API 文档：
 *   POST /v1/moderate        同步审核，直接返回结果
 *   POST /v1/moderate/async  异步审核，返回 task_id，结果通过 webhook 回调
 *   GET  /v1/task/{task_id}  查询异步任务状态
 *   GET  /v1/health          健康检查
 *
 * 鉴权：请求头 X-Project-Key: <your_project_key>
 *
 * 使用示例：
 *   // 同步审核
 *   $result = ContentModerationService::submitForModeration(123, 'post', ['content' => '文本内容']);
 *   if ($result && $result['verdict'] === 'rejected') { ... }
 *
 *   // 异步审核（结果通过 webhook 回调）
 *   $result = ContentModerationService::submitForModerationAsync(123, 'post', ['content' => '...']);
 *   // $result['task_id'] 可保存，后续通过 /v1/task/{id} 查询
 */
class ContentModerationService
{
    // ── 配置读取 ────────────────────────────────────────────────────────────

    /**
     * 从 Yaf 配置或环境变量中读取审核服务配置。
     *
     * @return array{endpoint:string, api_key:string, timeout:int, async:bool, webhook_url:string, strictness:string}
     */
    private static function getConfig(): array
    {
        // 优先从 Yaf_Registry 读取（application.ini [moderation] 节）
        $cfg = null;
        if (class_exists('Yaf_Registry')) {
            $config = Yaf_Registry::get('config');
            if ($config && isset($config->moderation)) {
                $cfg = $config->moderation;
            }
        }

        return [
            'endpoint'    => self::cfgGet($cfg, 'endpoint',    'MODERATION_ENDPOINT',    'http://moderation-api.example.com'),
            'api_key'     => self::cfgGet($cfg, 'api_key',     'MODERATION_API_KEY',     ''),
            'timeout'     => (int) self::cfgGet($cfg, 'timeout', 'MODERATION_TIMEOUT',   '5'),
            'async'       => filter_var(self::cfgGet($cfg, 'async', 'MODERATION_ASYNC', 'false'), FILTER_VALIDATE_BOOLEAN),
            'webhook_url' => self::cfgGet($cfg, 'webhook_url', 'MODERATION_WEBHOOK_URL', ''),
            'strictness'  => self::cfgGet($cfg, 'strictness',  'MODERATION_STRICTNESS',  'standard'),
        ];
    }

    /** 按优先级：Yaf config -> 环境变量 -> 默认值 */
    private static function cfgGet($cfg, string $key, string $envKey, string $default): string
    {
        if ($cfg && isset($cfg->{$key})) {
            return (string) $cfg->{$key};
        }
        $env = getenv($envKey);
        return ($env !== false && $env !== '') ? $env : $default;
    }

    // ── 同步审核 ────────────────────────────────────────────────────────────

    /**
     * 提交内容进行同步审核，阻塞等待结果。
     *
     * @param int    $contentId   内容 ID
     * @param string $type        内容类型，如 post | comment | video
     * @param array  $data        必须包含 content（文本）字段
     * @param string $userId      可选，用于日志记录
     * @return array|null         审核结果，格式见 ModerateResult，失败返回 null
     */
    public static function submitForModeration(int $contentId, string $type, array $data, string $userId = ''): ?array
    {
        $cfg = self::getConfig();

        $requestId = self::generateRequestId($contentId, $type);
        $content   = $data['content'] ?? '';

        // 幂等检查：同一 request_id 已有终态记录则直接返回
        $existing = ModerationLogModel::where('request_id', $requestId)
            ->whereIn('status', [ModerationLogModel::STATUS_PASSED, ModerationLogModel::STATUS_REJECTED])
            ->first();
        if ($existing) {
            return json_decode($existing->response_data, true);
        }

        $payload = [
            'content'    => $content,
            'type'       => $type,
            'strictness' => $cfg['strictness'],
            'model'      => 'auto',
            'context'    => ['content_id' => $contentId, 'user_id' => $userId],
        ];

        // 写入 pending 记录
        $log = ModerationLogModel::create([
            'request_id'   => $requestId,
            'content_id'   => $contentId,
            'content_type' => $type,
            'user_id'      => $userId ?: 0,
            'status'       => ModerationLogModel::STATUS_PENDING,
            'request_data' => json_encode($payload, JSON_UNESCAPED_UNICODE),
            'created_at'   => time(),
            'updated_at'   => time(),
        ]);

        // 调用同步审核接口
        $response = self::httpPost($cfg['endpoint'] . '/v1/moderate', $payload, $cfg['api_key'], $cfg['timeout']);

        if ($response === null) {
            self::updateLog($log, ModerationLogModel::STATUS_ERROR, null, '', 'request failed');
            return null;
        }

        $verdict   = $response['verdict']   ?? 'approved';
        $category  = $response['category']  ?? 'none';
        $status    = ($verdict === 'rejected') ? ModerationLogModel::STATUS_REJECTED : ModerationLogModel::STATUS_PASSED;

        self::updateLog($log, $status, $response, $category);
        return $response;
    }

    // ── 异步审核 ────────────────────────────────────────────────────────────

    /**
     * 提交内容进行异步审核，立即返回 task_id，审核完成后服务端会调用 webhook_url 回调。
     *
     * @param int    $contentId
     * @param string $type
     * @param array  $data
     * @param string $userId
     * @return array|null  ['task_id' => '...', 'message' => '...']，失败返回 null
     */
    public static function submitForModerationAsync(int $contentId, string $type, array $data, string $userId = ''): ?array
    {
        $cfg = self::getConfig();

        $requestId  = self::generateRequestId($contentId, $type);
        $content    = $data['content'] ?? '';

        $payload = [
            'content'     => $content,
            'type'        => $type,
            'strictness'  => $cfg['strictness'],
            'model'       => 'auto',
            'webhook_url' => $cfg['webhook_url'],
            'context'     => ['content_id' => $contentId, 'user_id' => $userId, 'request_id' => $requestId],
        ];

        // 写入 checking 记录
        ModerationLogModel::updateOrCreate(
            ['request_id' => $requestId],
            [
                'content_id'   => $contentId,
                'content_type' => $type,
                'user_id'      => $userId ?: 0,
                'status'       => ModerationLogModel::STATUS_CHECKING,
                'request_data' => json_encode($payload, JSON_UNESCAPED_UNICODE),
                'created_at'   => time(),
                'updated_at'   => time(),
            ]
        );

        $response = self::httpPost($cfg['endpoint'] . '/v1/moderate/async', $payload, $cfg['api_key'], $cfg['timeout']);

        if ($response === null || empty($response['task_id'])) {
            ModerationLogModel::where('request_id', $requestId)
                ->update(['status' => ModerationLogModel::STATUS_ERROR, 'error_msg' => 'async request failed', 'updated_at' => time()]);
            return null;
        }

        // 保存 task_id 到 remark 备用
        ModerationLogModel::where('request_id', $requestId)
            ->update(['remark' => 'task_id:' . $response['task_id'], 'updated_at' => time()]);

        return $response;
    }

    // ── Webhook 回调处理 ────────────────────────────────────────────────────

    /**
     * 处理来自 go-server 的审核结果 webhook 回调。
     *
     * 回调 body 格式：
     * {
     *   "task_id":    "task_xxx",
     *   "status":     "done",
     *   "verdict":    "approved|flagged|rejected",
     *   "category":   "none|spam|abuse|politics|adult|fraud|violence",
     *   "confidence": 0.95,
     *   "reason":     "...",
     *   "model_used": "...",
     *   "latency_ms": 200
     * }
     *
     * context 中可携带 request_id 用于匹配本地记录（提交时放入 context）。
     *
     * @param array $callbackData  解析后的回调 JSON
     * @return bool
     */
    public static function handleModerationCallback(array $callbackData): bool
    {
        $taskId    = $callbackData['task_id']   ?? '';
        $verdict   = $callbackData['verdict']   ?? 'approved';
        $category  = $callbackData['category']  ?? 'none';
        $status    = ($verdict === 'rejected') ? ModerationLogModel::STATUS_REJECTED : ModerationLogModel::STATUS_PASSED;

        // 优先通过 context.request_id 匹配
        $requestId = $callbackData['context']['request_id'] ?? '';

        $query = ModerationLogModel::query();
        if ($requestId) {
            $log = $query->where('request_id', $requestId)->first();
        } else {
            // fallback：通过 remark 中保存的 task_id 查找
            $log = $query->where('remark', 'like', 'task_id:' . $taskId . '%')->first();
        }

        if (!$log) {
            // 记录找不到，忽略
            return false;
        }

        $log->status        = $status;
        $log->violation_types = ($category !== 'none') ? $category : '';
        $log->response_data = json_encode($callbackData, JSON_UNESCAPED_UNICODE);
        $log->reviewed_at   = time();
        $log->reviewed_by   = 'system';
        $log->updated_at    = time();
        $log->save();

        return true;
    }

    // ── 任务状态查询 ────────────────────────────────────────────────────────

    /**
     * 主动查询异步任务状态（可用于轮询或调试）。
     *
     * @param string $taskId
     * @return array|null
     */
    public static function queryTask(string $taskId): ?array
    {
        $cfg = self::getConfig();
        return self::httpGet($cfg['endpoint'] . '/v1/task/' . urlencode($taskId), $cfg['api_key'], $cfg['timeout']);
    }

    // ── 内部工具方法 ────────────────────────────────────────────────────────

    private static function generateRequestId(int $contentId, string $type): string
    {
        return md5($type . '_' . $contentId);
    }

    private static function updateLog($log, string $status, ?array $response, string $category, string $errorMsg = ''): void
    {
        if (!$log) return;
        $log->status          = $status;
        $log->violation_types = ($category && $category !== 'none') ? $category : '';
        $log->response_data   = $response ? json_encode($response, JSON_UNESCAPED_UNICODE) : null;
        $log->error_msg       = $errorMsg;
        $log->reviewed_at     = time();
        $log->reviewed_by     = 'system';
        $log->updated_at      = time();
        $log->save();
    }

    /**
     * 发送 POST 请求。
     *
     * @return array|null  解析后的 JSON，失败返回 null
     */
    private static function httpPost(string $url, array $payload, string $apiKey, int $timeout): ?array
    {
        $body = json_encode($payload, JSON_UNESCAPED_UNICODE);

        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_POST           => true,
            CURLOPT_POSTFIELDS     => $body,
            CURLOPT_TIMEOUT        => $timeout,
            CURLOPT_CONNECTTIMEOUT => 3,
            CURLOPT_HTTPHEADER     => [
                'Content-Type: application/json',
                'X-Project-Key: ' . $apiKey,
            ],
        ]);

        $raw  = curl_exec($ch);
        $errno = curl_errno($ch);
        curl_close($ch);

        if ($errno || $raw === false) {
            return null;
        }

        $decoded = json_decode($raw, true);
        return is_array($decoded) ? $decoded : null;
    }

    /**
     * 发送 GET 请求。
     *
     * @return array|null
     */
    private static function httpGet(string $url, string $apiKey, int $timeout): ?array
    {
        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT        => $timeout,
            CURLOPT_CONNECTTIMEOUT => 3,
            CURLOPT_HTTPHEADER     => [
                'X-Project-Key: ' . $apiKey,
            ],
        ]);

        $raw  = curl_exec($ch);
        $errno = curl_errno($ch);
        curl_close($ch);

        if ($errno || $raw === false) {
            return null;
        }

        $decoded = json_decode($raw, true);
        return is_array($decoded) ? $decoded : null;
    }
}
