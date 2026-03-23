<?php
/**
 * 管理后台审核记录列表示例
 */
class ModerationLogsController extends Yaf_Controller_Abstract
{
    public function indexAction()
    {
        // $logs = ModerationLogModel::list(...);
        $logs = [];
        $this->getView()->assign('logs', $logs);
    }
    public function detailAction()
    {
        $id = $_GET['id'] ?? null;
        // $log = ModerationLogModel::find($id);
        $log = null;
        $this->getView()->assign('log', $log);
    }
}
