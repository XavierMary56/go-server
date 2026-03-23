-- 内容审核记录表结构示例
CREATE TABLE `moderation_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `content_id` BIGINT UNSIGNED NOT NULL COMMENT '内容ID',
  `content_type` VARCHAR(32) NOT NULL COMMENT '内容类型',
  `status` TINYINT NOT NULL COMMENT '审核状态',
  `result` TEXT COMMENT '审核结果原始数据',
  `violation_type` VARCHAR(32) DEFAULT NULL COMMENT '违规类型',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_content_id_type` (`content_id`, `content_type`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='内容审核记录表';
