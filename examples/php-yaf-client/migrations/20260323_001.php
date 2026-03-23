<?php

use Illuminate\Database\Capsule\Manager as DB;
use Illuminate\Database\Schema\Blueprint;

/**
 * 创建内容审核日志表 moderation_logs
 *
 * 字段说明：
 *   request_id      审核请求幂等 ID（md5(type_contentId)），用于防重
 *   content_type    内容类型：post | comment | video | usercontents
 *   content_id      业务内容 ID
 *   user_id         内容所属用户 ID
 *   status          审核状态：pending | checking | passed | rejected | error
 *   violation_types 违规类型，多个用逗号分隔，无违规为空
 *   request_data    请求体 JSON（content 等）
 *   response_data   API 返回的审核结果 JSON（verdict/category/confidence/reason 等）
 *   error_msg       出错时的错误信息
 *   reviewed_at     审核完成时间戳
 *   reviewed_by     审核方标记（'system' 为自动，其他为人工）
 *   remark          备注，异步模式下存储 task_id
 */
class Migration20260323_001
{
    public function up(): void
    {
        if (!DB::schema()->hasTable('moderation_logs')) {
            DB::schema()->create('moderation_logs', function (Blueprint $table) {
                $table->bigIncrements('id');
                $table->string('request_id', 64)->unique()->comment('审核请求幂等ID');
                $table->string('content_type', 32)->comment('内容类型：post|comment|video 等');
                $table->unsignedBigInteger('content_id')->comment('业务内容ID');
                $table->unsignedBigInteger('user_id')->default(0)->comment('用户ID');
                $table->string('status', 16)->default('pending')->comment('审核状态：pending|checking|passed|rejected|error');
                $table->string('violation_types', 255)->nullable()->default('')->comment('违规类型，逗号分隔');
                $table->text('request_data')->nullable()->comment('请求体JSON');
                $table->text('response_data')->nullable()->comment('API审核结果JSON');
                $table->string('error_msg', 500)->nullable()->default('')->comment('错误信息');
                $table->unsignedInteger('created_at')->default(0)->comment('创建时间戳');
                $table->unsignedInteger('updated_at')->default(0)->comment('更新时间戳');
                $table->unsignedInteger('reviewed_at')->nullable()->comment('审核完成时间戳');
                $table->string('reviewed_by', 64)->nullable()->default('')->comment('审核人/system');
                $table->string('remark', 255)->nullable()->default('')->comment('备注，异步时存task_id');

                $table->index(['content_id', 'content_type'], 'idx_content');
                $table->index('status',      'idx_status');
                $table->index('user_id',     'idx_user_id');
                $table->index('created_at',  'idx_created_at');
            });
        }
    }

    public function down(): void
    {
        DB::schema()->dropIfExists('moderation_logs');
    }
}
