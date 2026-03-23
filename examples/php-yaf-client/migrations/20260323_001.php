<?php

use Illuminate\Database\Capsule\Manager as DB;
use Illuminate\Database\Schema\Blueprint;

class Migration20260323_001
{
    public function up()
    {
        if (!DB::schema()->hasTable('moderation_logs')) {
            DB::schema()->create('moderation_logs', function (Blueprint $table) {
                $table->bigIncrements('id');
                $table->bigInteger('content_id')->unsigned()->comment('内容ID');
                $table->string('content_type', 32)->comment('内容类型');
                $table->tinyInteger('status')->comment('审核状态');
                $table->text('result')->nullable()->comment('审核结果原始数据');
                $table->string('violation_type', 32)->nullable()->comment('违规类型');
                $table->dateTime('created_at')->default(DB::raw('CURRENT_TIMESTAMP'));
                $table->dateTime('updated_at')->default(DB::raw('CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP'));
                $table->index(['content_id', 'content_type'], 'idx_content_id_type');
                $table->index(['status'], 'idx_status');
            });
        }
    }

    public function down()
    {
        if (DB::schema()->hasTable('moderation_logs')) {
            DB::schema()->drop('moderation_logs');
        }
    }
}
