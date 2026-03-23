<?php
/**
 * 管理后台 —— 审核记录列表与详情
 *
 * 路由（Admin 模块）：
 *   GET /admin/moderationlogs/index   审核记录列表（支持过滤/分页）
 *   GET /admin/moderationlogs/detail  单条审核记录详情
 *
 * 视图文件路径（示例）：
 *   application/modules/Admin/views/moderationlogs/index.phtml
 *   application/modules/Admin/views/moderationlogs/detail.phtml
 */
class ModerationLogsController extends Yaf_Controller_Abstract
{
    /**
     * GET /admin/moderationlogs/index
     *
     * 支持的 GET 参数：
     *   status       审核状态过滤，见 ModerationLogModel::STATUS_*
     *   content_type 内容类型过滤，如 post|comment
     *   content_id   精确匹配内容 ID
     *   user_id      精确匹配用户 ID
     *   page         页码，默认 1
     *   page_size    每页条数，默认 20，最大 100
     *
     * 传递给视图的变量：
     *   $logs        当前页记录列表（ModerationLogModel 实例数组）
     *   $total       总条数
     *   $page        当前页码
     *   $pageSize    每页条数
     *   $totalPages  总页数
     *   $filters     当前过滤条件数组
     *   $statuses    所有状态及中文标签（用于下拉）
     */
    public function indexAction(): void
    {
        $filters = [
            'status'       => $_GET['status']       ?? '',
            'content_type' => $_GET['content_type'] ?? '',
            'content_id'   => $_GET['content_id']   ?? '',
            'user_id'      => $_GET['user_id']       ?? '',
        ];
        // 过滤空值，避免无效 where 条件
        $filters = array_filter($filters, fn($v) => $v !== '');

        $page     = max(1, (int) ($_GET['page']      ?? 1));
        $pageSize = min(100, max(1, (int) ($_GET['page_size'] ?? 20)));

        $result     = ModerationLogModel::listPage($filters, $page, $pageSize);
        $total      = $result['total'];
        $totalPages = (int) ceil($total / $pageSize);

        $view = $this->getView();
        $view->assign('logs',       $result['items']);
        $view->assign('total',      $total);
        $view->assign('page',       $page);
        $view->assign('pageSize',   $pageSize);
        $view->assign('totalPages', $totalPages);
        $view->assign('filters',    $filters);
        $view->assign('statuses',   ModerationLogModel::STATUSES);
    }

    /**
     * GET /admin/moderationlogs/detail?id=123
     *
     * 传递给视图的变量：
     *   $log         ModerationLogModel 实例（含解析后的 response）
     *   $response    审核 API 返回的原始结果数组
     */
    public function detailAction(): void
    {
        $id  = isset($_GET['id']) ? (int) $_GET['id'] : 0;
        $log = $id > 0 ? ModerationLogModel::find($id) : null;

        if (!$log) {
            // 记录不存在，跳转回列表
            $this->redirect('/admin/moderationlogs/index');
            return;
        }

        $view = $this->getView();
        $view->assign('log',      $log);
        $view->assign('response', $log->getResponseArray());
    }

    // ── 工具方法 ─────────────────────────────────────────────────────────────

    private function redirect(string $url): void
    {
        $response = $this->getResponse();
        $response->setRedirect($url);
    }
}
