<?php
/**
 * 内容审核服务 - PHP Yaf 示例
 *
 * 该示例专注于对接 go-server 的 /v2/moderations 接口。
 */
class ContentModerationService
{
    private static function getConfig(): array
    {
        $cfg = null;
        if (class_exists('Yaf_Registry')) {
            $config = Yaf_Registry::get('config');
            if ($config && isset($config->moderation)) {
                $cfg = $config->moderation;
            }
        }

        return [
            'endpoint'    => $cfg->endpoint ?? 'https://zyaokkmo.cc',
            'api_key'     => $cfg->api_key ?? '',
            'timeout'     => (int) ($cfg->timeout ?? 5),
            'strictness'  => $cfg->strictness ?? 'standard',
        ];
    }

    /**
     * 同步审核内容 (V2)
     *
     * @param string $content 待审核文本
     * @param string $type 内容类型：comment | post
     * @param array $context 业务上下文
     * @return array|null 审核结果或 null (服务异常)
     */
    public static function moderate(string $content, string $type = 'comment', array $context = []): ?array
    {
        $config = self::getConfig();
        if (empty($config['api_key'])) {
            return null;
        }

        $url = rtrim($config['endpoint'], '/') . '/v2/moderations';
        $payload = [
            'content'    => $content,
            'type'       => $type,
            'strictness' => $config['strictness'],
            'context'    => $context,
        ];

        $response = self::httpPost($url, $payload, $config['api_key'], $config['timeout']);

        // V2 响应结构在 data 字段下
        if (isset($response['code']) && $response['code'] === 200 && isset($response['data'])) {
            return $response['data']['result'] ?? null;
        }

        return null;
    }

    private static function httpPost(string $url, array $data, string $apiKey, int $timeout): ?array
    {
        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_POST           => true,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_POSTFIELDS     => json_encode($data),
            CURLOPT_TIMEOUT        => $timeout,
            CURLOPT_HTTPHEADER     => [
                'Content-Type: application/json',
                'X-Project-Key: ' . $apiKey,
            ],
        ]);

        $raw = curl_exec($ch);
        $errno = curl_errno($ch);
        curl_close($ch);

        if ($errno || $raw === false) {
            return null;
        }

        return json_decode($raw, true);
    }
}
