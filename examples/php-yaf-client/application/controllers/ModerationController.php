<?php
/**
 * 内容审核回调与状态查询接口
 *
 * 路由配置示例（Bootstrap.php 或 routes.php）：
 *   // 审核结果 webhook 回调
 *   $router->addRoute('moderation_callback', new Yaf_Route_Static('/moderation/callback'));
 *   // 审核状态查询
 *   $router->addRoute('moderation_status',  new Yaf_Route_Static('/moderation/status'));
 *
 * 对应 URL：
 *   POST /moderation/callback  —— go-server 回调此接口通知审核结果
 *   GET  /moderation/status    —— 查询指定内容当前审核状态
 */
class ModerationController extends Yaf_Controller_Abstract
{
    /**
     * POST /moderation/callback
     *
     * go-server 异步审核完成后，向此接口推送结果。
     *
     * 请求 body（JSON）：
     * {
     *   "task_id":    "task_1234567890",
     *   "status":     "done",
     *   "verdict":    "approved|flagged|rejected",
     *   "category":   "none|spam|abuse|politics|adult|fraud|violence",
     *   "confidence": 0.95,
     *   "reason":     "...",
     *   "model_used": "claude-3-haiku",
     *   "latency_ms": 320
     * }
     *
     * 响应：{"code":0,"msg":"ok"}
     */
    public function callbackAction(): false
    {
        $raw  = file_get_contents('php://input');
        $data = json_decode($raw, true);

        if (!is_array($data)) {
            $this->sendJson(['code' => 1, 'msg' => 'invalid json body']);
            return false;
        }

        // 必要字段校验
        if (empty($data['task_id']) || empty($data['verdict'])) {
            $this->sendJson(['code' => 2, 'msg' => 'missing required fields: task_id, verdict']);
            return false;
        }

        $ok = ContentModerationService::handleModerationCallback($data);

        if (!$ok) {
            // 找不到对应记录，也返回 ok 避免 go-server 重试
            $this->sendJson(['code' => 0, 'msg' => 'ok (record not found, ignored)']);
            return false;
        }

        $this->sendJson(['code' => 0, 'msg' => 'ok']);
        return false;
    }

    /**
     * GET /moderation/status?content_id=123&type=post
     *
     * 查询指定内容最新的审核记录状态，用于前端轮询或后台排查。
     *
     * 响应（找到）：
     * {
     *   "code": 0,
     *   "data": {
     *     "status":          "passed",
     *     "status_label":    "审核通过",
     *     "verdict":         "approved",
     *     "category":        "none",
     *     "confidence":      95,
     *     "violation_types": "",
     *     "reviewed_at":     1711180800
     *   }
     * }
     *
     * 响应（未找到）：{"code": 1, "msg": "not found"}
     */
    public function statusAction(): false
    {
        $contentId   = isset($_GET['content_id']) ? (int) $_GET['content_id'] : 0;
        $contentType = $_GET['type'] ?? '';

        if ($contentId <= 0 || $contentType === '') {
            $this->sendJson(['code' => 2, 'msg' => 'content_id and type are required']);
            return false;
        }

        $log = ModerationLogModel::findByContent($contentId, $contentType);

        if (!$log) {
            $this->sendJson(['code' => 1, 'msg' => 'not found']);
            return false;
        }

        $resp = $log->getResponseArray();

        $this->sendJson([
            'code' => 0,
            'data' => [
                'status'          => $log->status,
                'status_label'    => $log->getStatusLabel(),
                'verdict'         => $resp['verdict']    ?? '',
                'category'        => $resp['category']   ?? '',
                'confidence'      => $log->getConfidencePercent(),
                'violation_types' => $log->violation_types,
                'reviewed_at'     => $log->reviewed_at,
            ],
        ]);
        return false;
    }

    // ── 工具方法 ─────────────────────────────────────────────────────────────

    private function sendJson(array $data): void
    {
        $response = $this->getResponse();
        $response->setHeader('Content-Type', 'application/json; charset=utf-8');
        $response->setBody(json_encode($data, JSON_UNESCAPED_UNICODE));
    }
}
