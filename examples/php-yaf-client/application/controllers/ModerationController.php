<?php
/**
 * 内容审核 Demo 控制器
 */
class ModerationController extends Yaf_Controller_Abstract
{
    /**
     * 示例：POST /moderation/check
     */
    public function checkAction(): false
    {
        $content = $this->getRequest()->getPost('content');
        if (empty($content)) {
            $this->sendJson(['code' => 1, 'msg' => 'content is required']);
            return false;
        }

        $result = ContentModerationService::moderate($content, 'comment', [
            'user_id' => '1001'
        ]);

        if ($result === null) {
            $this->sendJson(['code' => 500, 'msg' => 'moderation service unavailable']);
            return false;
        }

        $this->sendJson([
            'code' => 0,
            'data' => [
                'verdict'  => $result['verdict'],
                'category' => $result['category'],
                'reason'   => $result['reason'] ?? '',
            ]
        ]);
        return false;
    }

    private function sendJson(array $data): void
    {
        $this->getResponse()->setHeader('Content-Type', 'application/json; charset=utf-8');
        $this->getResponse()->setBody(json_encode($data, JSON_UNESCAPED_UNICODE));
    }
}
