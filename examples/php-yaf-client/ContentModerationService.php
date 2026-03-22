<?php
/**
 * GoModerationService.php
 * ─────────────────────────────────────────────────────────
 * PHP YAF 项目对接 Go 审核服务的客户端
 * 放到 application/services/ 目录，替换原 ContentModerationService.php
 *
 * 与原 PHP 版本接口完全兼容，切换时只需修改配置文件中的 endpoint 即可
 */
class ContentModerationService
{
    /** @var string */
    private $endpoint;

    /** @var string */
    private $projectKey;

    /** @var int */
    private $timeout;

    /** @var bool */
    private $async;

    /** @var string */
    private $webhookUrl;

    /** @var string */
    private $strictness;

    public function __construct()
    {
        $config = Yaf_Registry::get('config');
        $mod    = $config->moderation;

        $this->endpoint   = rtrim($mod->endpoint ?? 'http://localhost:8080', '/');
        $this->projectKey = $mod->api_key        ?? '';
        $this->timeout    = (int) ($mod->timeout  ?? 5);
        $this->async      = (bool)($mod->async    ?? false);
        $this->webhookUrl = $mod->webhook_url     ?? '';
        $this->strictness = $mod->strictness      ?? 'standard';
    }

    /**
     * 审核内容（主入口，与原 PHP 版本接口完全一致）
     *
     * @param  string $content 待审内容
     * @param  string $type    post | comment
     * @param  array  $context 附加信息
     * @return array  ['verdict' => 'approved|flagged|rejected', 'reason' => '...', ...]
     */
    public function moderate(string $content, string $type = 'post', array $context = []): array
    {
        if ($this->async) {
            return $this->moderateAsync($content, $type, $context);
        }

        return $this->callGoServer('/v1/moderate', [
            'content'    => $content,
            'type'       => $type,
            'model'      => 'auto',
            'strictness' => $this->strictness,
            'context'    => $context,
        ]);
    }

    /**
     * 异步审核（不阻塞请求，结果由 Webhook 回调）
     */
    public function moderateAsync(string $content, string $type, array $context = []): array
    {
        $result = $this->callGoServer('/v1/moderate/async', [
            'content'     => $content,
            'type'        => $type,
            'model'       => 'auto',
            'strictness'  => $this->strictness,
            'webhook_url' => $this->webhookUrl,
            'context'     => $context,
        ]);

        // 异步模式下先标记 pending，等回调更新
        return array_merge($result, [
            'verdict' => 'flagged',
            'reason'  => '审核中，通过后自动发布',
            'async'   => true,
        ]);
    }

    /**
     * 健康检查（可用于 PHP 项目启动时校验连通性）
     */
    public function ping(): bool
    {
        try {
            $ch = curl_init($this->endpoint . '/v1/health');
            curl_setopt_array($ch, [
                CURLOPT_RETURNTRANSFER => true,
                CURLOPT_TIMEOUT        => 3,
            ]);
            $body = curl_exec($ch);
            curl_close($ch);
            $data = json_decode($body, true);
            return ($data['status'] ?? '') === 'ok';
        } catch (Throwable $e) {
            return false;
        }
    }

    /**
     * 发送 HTTP 请求到 Go 审核服务
     */
    private function callGoServer(string $path, array $data): array
    {
        $url     = $this->endpoint . $path;
        $payload = json_encode($data, JSON_UNESCAPED_UNICODE);

        $headers = [
            'Content-Type: application/json',
            'Accept: application/json',
        ];
        if ($this->projectKey !== '') {
            $headers[] = 'X-Project-Key: ' . $this->projectKey;
        }

        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_POST           => true,
            CURLOPT_POSTFIELDS     => $payload,
            CURLOPT_HTTPHEADER     => $headers,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT        => $this->timeout,
        ]);

        $body    = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $curlErr  = curl_error($ch);
        curl_close($ch);

        if ($curlErr) {
            return $this->safeFallback('网络异常: ' . $curlErr);
        }

        $result = json_decode($body, true);
        if (!is_array($result)) {
            return $this->safeFallback("HTTP {$httpCode}: 响应解析失败");
        }

        return $result;
    }

    /**
     * 安全降级：所有异常都转 pending，不误拦用户
     */
    private function safeFallback(string $reason): array
    {
        return [
            'verdict'    => 'flagged',
            'category'   => 'none',
            'confidence' => 0.0,
            'reason'     => '审核服务异常，已转人工',
            'model_used' => 'fallback',
            'fallback'   => true,
            '_error'     => $reason,
        ];
    }
}
