<?php

use Illuminate\Database\Eloquent\Model;

/**
 * class ModerationLogModel
 * 内容审核日志表，记录所有内容的审核请求和结果
 *
 * @property int $id
 * @property string $request_id 审核请求ID（用于幂等性）
 * @property string $content_type 内容类型：video|usercontents|post|comment|postcomment
 * @property int $content_id 内容ID
 * @property int $user_id 用户ID/AFF
 * @property string $status 审核状态：pending|checking|passed|rejected|error
 * @property string $violation_types 违规类型，多个用逗号分隔（色情|暴力|广告等）
 * @property string $request_data 请求的原始数据（JSON）
 * @property string $response_data 审核返回的详细结果（JSON）
 * @property string $error_msg 错误信息
 * @property int $created_at 创建时间戳
 * @property int $updated_at 更新时间戳
 * @property int $reviewed_at 审核完成时间戳
 * @property string $reviewed_by 审核人员/系统标记
 * @property string $remark 备注
 *
 * @mixin \Eloquent
 */
class ModerationLogModel extends BaseModel
{
    protected $table = 'moderation_logs';
    protected $primaryKey = 'id';
    public $timestamps = false;
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

    // 审核状态常量
    const STATUS_PENDING = 'pending';      // 待审核
    const STATUS_CHECKING = 'checking';    // 审核中
    const STATUS_PASSED = 'passed';        // 审核通过
    const STATUS_REJECTED = 'rejected';    // 审核不通过
    const STATUS_ERROR = 'error';          // 审核出错

    const STATUSES = [
        self::STATUS_PENDING => '待审核',
        self::STATUS_CHECKING => '审核中',
        self::STATUS_PASSED => '审核通过',
        self::STATUS_REJECTED => '审核不通过',
        self::STATUS_ERROR => '审核出错',
    ];
}
