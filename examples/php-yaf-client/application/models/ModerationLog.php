<?php

use Illuminate\Database\Eloquent\Model;

/**
 * class ModerationLogModel
 * 内容审核日志表，记录所有内容的审核请求和结果
 *
 * @property int    $id
 * @property string $request_id      审核请求ID（用于幂等，md5(type_contentId)）
 * @property string $content_type    内容类型：video|usercontents|post|comment|postcomment
 * @property int    $content_id      内容ID
 * @property int    $user_id         用户ID/AFF
 * @property string $status          审核状态：pending|checking|passed|rejected|error
 * @property string $violation_types 违规类型，多个用逗号分隔（spam|abuse|politics|adult|fraud|violence）
 * @property string $request_data    请求的原始数据（JSON）
 * @property string $response_data   审核返回的详细结果（JSON）
 * @property string $error_msg       错误信息
 * @property int    $created_at      创建时间戳
 * @property int    $updated_at      更新时间戳
 * @property int    $reviewed_at     审核完成时间戳
 * @property string $reviewed_by     审核人员/系统标记
 * @property string $remark          备注（异步模式下存储 task_id）
 *
 * @mixin \Eloquent
 */
class ModerationLogModel extends BaseModel
{
    protected $table      = 'moderation_logs';
    protected $primaryKey = 'id';
    public $timestamps    = false;

    protected $fillable = [
        'id',
        'request_id',
        'content_type',
        'content_id',
        'user_id',
        'status',
        'violation_types',
        'request_data',
        'response_data',
        'error_msg',
        'created_at',
        'updated_at',
        'reviewed_at',
        'reviewed_by',
        'remark',
    ];

    // ── 审核状态常量 ─────────────────────────────────────────────────────────

    const STATUS_PENDING  = 'pending';   // 待审核
    const STATUS_CHECKING = 'checking';  // 审核中（异步等待回调）
    const STATUS_PASSED   = 'passed';    // 审核通过
    const STATUS_REJECTED = 'rejected';  // 审核不通过
    const STATUS_ERROR    = 'error';     // 审核出错

    const STATUSES = [
        self::STATUS_PENDING  => '待审核',
        self::STATUS_CHECKING => '审核中',
        self::STATUS_PASSED   => '审核通过',
        self::STATUS_REJECTED => '审核不通过',
        self::STATUS_ERROR    => '审核出错',
    ];

    // ── 常用查询方法 ─────────────────────────────────────────────────────────

    /**
     * 根据内容 ID 和类型查询最新的审核记录。
     *
     * @param int    $contentId
     * @param string $contentType
     * @return static|null
     */
    public static function findByContent(int $contentId, string $contentType): ?self
    {
        return static::where('content_id', $contentId)
            ->where('content_type', $contentType)
            ->orderBy('id', 'desc')
            ->first();
    }

    /**
     * 更新审核状态和结果。
     *
     * @param string $requestId
     * @param string $status       ModerationLogModel::STATUS_*
     * @param array  $result       审核结果数组（来自 API 响应）
     * @return int   受影响行数
     */
    public static function updateStatus(string $requestId, string $status, array $result = []): int
    {
        $data = [
            'status'     => $status,
            'updated_at' => time(),
        ];

        if (!empty($result)) {
            $data['response_data']   = json_encode($result, JSON_UNESCAPED_UNICODE);
            $data['violation_types'] = ($result['category'] ?? 'none') !== 'none' ? ($result['category'] ?? '') : '';
            $data['reviewed_at']     = time();
            $data['reviewed_by']     = 'system';
        }

        return static::where('request_id', $requestId)->update($data);
    }

    /**
     * 分页获取审核记录列表，支持多条件过滤。
     *
     * @param array $filters  支持 status, content_type, content_id, user_id
     * @param int   $page     从 1 开始
     * @param int   $pageSize
     * @return array{items: static[], total: int}
     */
    public static function listPage(array $filters = [], int $page = 1, int $pageSize = 20): array
    {
        $query = static::query()->orderBy('id', 'desc');

        if (!empty($filters['status'])) {
            $query->where('status', $filters['status']);
        }
        if (!empty($filters['content_type'])) {
            $query->where('content_type', $filters['content_type']);
        }
        if (!empty($filters['content_id'])) {
            $query->where('content_id', (int) $filters['content_id']);
        }
        if (!empty($filters['user_id'])) {
            $query->where('user_id', (int) $filters['user_id']);
        }

        $total = $query->count();
        $items = $query->offset(($page - 1) * $pageSize)->limit($pageSize)->get();

        return ['items' => $items, 'total' => $total];
    }

    // ── 辅助方法 ─────────────────────────────────────────────────────────────

    /** 获取状态的中文描述 */
    public function getStatusLabel(): string
    {
        return self::STATUSES[$this->status] ?? $this->status;
    }

    /** 解析 response_data JSON */
    public function getResponseArray(): array
    {
        if (!$this->response_data) return [];
        $data = json_decode($this->response_data, true);
        return is_array($data) ? $data : [];
    }

    /** 获取置信度百分比（0~100） */
    public function getConfidencePercent(): ?int
    {
        $resp = $this->getResponseArray();
        if (!isset($resp['confidence'])) return null;
        return (int) round($resp['confidence'] * 100);
    }
}
