<?php
/**
 * 内容审核回调与状态查询接口示例
 */
class ModerationCallbackController extends Yaf_Controller_Abstract
{
    public function callbackAction()
    {
        $data = file_get_contents('php://input');
        $json = json_decode($data, true);
        // TODO: 校验签名、参数等
        // TODO: 处理审核结果，更新 ModerationLog 状态
        // ModerationLogModel::updateStatus($json['content_id'], $json['status'], $json['result']);
        echo json_encode(['code' => 0, 'msg' => 'ok']);
        return false;
    }
    public function statusAction()
    {
        $contentId = $_GET['content_id'] ?? null;
        $type = $_GET['type'] ?? null;
        // $log = ModerationLogModel::findByContent($contentId, $type);
        $log = null;
        if ($log) {
            echo json_encode(['code' => 0, 'data' => $log]);
        } else {
            echo json_encode(['code' => 1, 'msg' => 'not found']);
        }
        return false;
    }
}
