-- 内容审核记录表
-- 与 ModerationLogModel.php 字段完全对应
CREATE TABLE `moderation_logs` (
  `id`              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `request_id`      VARCHAR(64)      NOT NULL            COMMENT '审核请求幂等ID（md5(type_contentId)）',
  `content_type`    VARCHAR(32)      NOT NULL            COMMENT '内容类型：post|comment|video 等',
  `content_id`      BIGINT UNSIGNED  NOT NULL            COMMENT '业务内容ID',
  `user_id`         BIGINT UNSIGNED  NOT NULL DEFAULT 0  COMMENT '用户ID',
  `status`          VARCHAR(16)      NOT NULL DEFAULT 'pending'
                                                         COMMENT '审核状态：pending|checking|passed|rejected|error',
  `violation_types` VARCHAR(255)     NOT NULL DEFAULT ''  COMMENT '违规类型，逗号分隔；无违规为空',
  `request_data`    TEXT                                 COMMENT '提交给审核服务的请求体 JSON',
  `response_data`   TEXT                                 COMMENT '审核服务返回的结果 JSON',
  `error_msg`       VARCHAR(500)     NOT NULL DEFAULT ''  COMMENT '出错时的错误信息',
  `created_at`      INT UNSIGNED     NOT NULL DEFAULT 0  COMMENT '创建时间戳',
  `updated_at`      INT UNSIGNED     NOT NULL DEFAULT 0  COMMENT '更新时间戳',
  `reviewed_at`     INT UNSIGNED              DEFAULT NULL COMMENT '审核完成时间戳',
  `reviewed_by`     VARCHAR(64)      NOT NULL DEFAULT ''  COMMENT '审核方标记：system 或人工账号',
  `remark`          VARCHAR(255)     NOT NULL DEFAULT ''  COMMENT '备注，异步模式下存储 task_id',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_request_id`   (`request_id`),
  KEY       `idx_content`        (`content_id`, `content_type`),
  KEY       `idx_status`         (`status`),
  KEY       `idx_user_id`        (`user_id`),
  KEY       `idx_created_at`     (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='内容审核日志表';
