<?php
/**
 * 内容审核服务
 *
 * 当前示例支持对接 go-server 的 V1 / V2 审核接口，推荐新接入优先使用 V2。
 * 真实地址和项目密钥必须通过 application.ini 或环境变量提供，不应在代码中硬编码。
 *
 * API 文档：
 *   POST /v2/moderations        同步审核（推荐）
 *   POST /v2/moderations/async  异步审核（推荐）
 *   GET  /v2/tasks/{task_id}    查询异步任务状态
 *   POST /v1/moderate           兼容旧版同步接口
 *   POST /v1/moderate/async     兼容旧版异步接口
 *   GET  /v1/task/{task_id}     兼容旧版任务查询
 *   GET  /v1/health             健康检查
 *   GET  /v2/health             健康检查
 *
 * 鉴权：请求头 X-Project-Key: <your_project_key>
 *
 * 使用示例：
 *   $result = ContentModerationService::submitForModeration(123, 'post', ['content' => '文本内容']);
 *   if ($result && $result['verdict'] === 'rejected') { ... }
 */
class ContentModerationService
{
    // ── 配置读取 ────────────────────────────────────────────────────────────

    /**
     * 从 Yaf 配置或环境变量中读取审核服务配置。
     *
     * @return array{endpoint:string, api_key:string, timeout:int, async:bool, webhook_url:string, strictness:string, api_version:string}
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
            'endpoint'    => self::cfgGet($cfg, 'endpoint',    'MODERATION_ENDPOINT',    ''),
            'api_key'     => self::cfgGet($cfg, 'api_key',     'MODERATION_API_KEY',     ''),
            'timeout'     => (int) self::cfgGet($cfg, 'timeout', 'MODERATION_TIMEOUT',   '5'),
            'async'       => filter_var(self::cfgGet($cfg, 'async', 'MODERATION_ASYNC', 'false'), FILTER_VALIDATE_BOOLEAN),
            'webhook_url' => self::cfgGet($cfg, 'webhook_url', 'MODERATION_WEBHOOK_URL', ''),
            'strictness'  => self::cfgGet($cfg, 'strictness',  'MODERATION_STRICTNESS',  'standard'),
            'api_version' => strtolower(self::cfgGet($cfg, 'api_version', 'MODERATION_API_VERSION', 'v2')),
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

        // 同时兼容 V1 / V2，默认优先走 V2
        $response = self::httpPost(self::buildModerationEndpoint($cfg, false), $payload, $cfg['api_key'], $cfg['timeout']);

        if ($response === null) {
            self::updateLog($log, ModerationLogModel::STATUS_ERROR, null, '', 'request failed');
            return null;
        }

        $normalized = self::normalizeModerationResponse($response);
        if ($normalized === null) {
            self::updateLog($log, ModerationLogModel::STATUS_ERROR, null, '', 'invalid moderation response');
            return null;
        }

        $verdict   = $normalized['verdict']  ?? 'approved';
        $category  = $normalized['category'] ?? 'none';
        $status    = ($verdict === 'rejected') ? ModerationLogModel::STATUS_REJECTED : ModerationLogModel::STATUS_PASSED;

        self::updateLog($log, $status, $normalized, $category);
        return $normalized;
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

        $response = self::httpPost(self::buildModerationEndpoint($cfg, true), $payload, $cfg['api_key'], $cfg['timeout']);

        $taskId = self::extractTaskId($response);
        if ($response === null || $taskId === '') {
            ModerationLogModel::where('request_id', $requestId)
                ->update(['status' => ModerationLogModel::STATUS_ERROR, 'error_msg' => 'async request failed', 'updated_at' => time()]);
            return null;
        }

        // 保存 task_id 到 remark 备用
        ModerationLogModel::where('request_id', $requestId)
            ->update(['remark' => 'task_id:' . $taskId, 'updated_at' => time()]);

        return [
            'task_id' => $taskId,
            'message' => $response['message'] ?? 'task accepted',
            'raw'     => $response,
        ];
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
     * 当前 go-server 默认 webhook 只回传 task_id / status / verdict / category 等结果字段，不回传原始 context；本地记录应优先通过已保存的 task_id 关联，context.request_id 仅作为兼容兜底。
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

        // 若业务方自行补传了 context.request_id 可优先使用；当前默认主要靠 task_id 回查
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
        $response = self::httpGet(self::buildTaskEndpoint($cfg, $taskId), $cfg['api_key'], $cfg['timeout']);
        if ($response === null) {
            return null;
        }

        if (isset($response['data']) && is_array($response['data'])) {
            return $response['data'];
        }

        return $response;
    }

    private static function buildModerationEndpoint(array $cfg, bool $async): string
    {
        $version = ($cfg['api_version'] ?? 'v2') === 'v1' ? 'v1' : 'v2';
        $base = rtrim($cfg['endpoint'], '/');

        if ($version === 'v1') {
            return $base . ($async ? '/v1/moderate/async' : '/v1/moderate');
        }

        return $base . ($async ? '/v2/moderations/async' : '/v2/moderations');
    }

    private static function buildTaskEndpoint(array $cfg, string $taskId): string
    {
        $version = ($cfg['api_version'] ?? 'v2') === 'v1' ? 'v1' : 'v2';
        $base = rtrim($cfg['endpoint'], '/');
        $path = $version === 'v1' ? '/v1/task/' : '/v2/tasks/';
        return $base . $path . urlencode($taskId);
    }

    private static function extractTaskId(?array $response): string
    {
        if (!$response) {
            return '';
        }
        if (!empty($response['task_id'])) {
            return (string) $response['task_id'];
        }
        if (isset($response['data']['task_id'])) {
            return (string) $response['data']['task_id'];
        }
        return '';
    }

    private static function normalizeModerationResponse(array $response): ?array
    {
        if (isset($response['data']['result']) && is_array($response['data']['result'])) {
            return $response['data']['result'];
        }

        if (isset($response['verdict'])) {
            return $response;
        }

        return null;
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
