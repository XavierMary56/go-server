-- =============================================================
-- 数据库初始化脚本
-- 项目：moderation（AI 内容审核服务）
-- 字符集：utf8mb4 / utf8mb4_unicode_ci
-- 说明：脚本支持重复执行，DROP 前会判断表是否存在
-- =============================================================

CREATE DATABASE IF NOT EXISTS `moderation`
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE `moderation`;

-- -------------------------------------------------------------
-- 1. project_keys —— 项目接入密钥表
--    每个业务项目对应一条记录，通过 X-Project-Key 请求头鉴权
-- -------------------------------------------------------------
DROP TABLE IF EXISTS `project_keys`;
CREATE TABLE `project_keys` (
    `id`         BIGINT       NOT NULL AUTO_INCREMENT                                            COMMENT '自增主键',
    `project_name` VARCHAR(128) NOT NULL                                                           COMMENT '项目标识，如 91prona / 51dm',
    `key`        VARCHAR(256) NOT NULL                                                           COMMENT '接入密钥，请求时通过 X-Project-Key 请求头传入',
    `rate_limit` INT          NOT NULL DEFAULT 60                                                COMMENT '每分钟最大请求数，0 表示不限制',
    `enabled`    TINYINT(1)   NOT NULL DEFAULT 1                                                 COMMENT '是否启用：1=启用 0=禁用',
    `deleted_at` DATETIME              DEFAULT NULL                                              COMMENT '软删除时间，非 NULL 表示已删除',
    `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP                                COMMENT '创建时间',
    `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP    COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_key` (`key`),
    KEY `idx_project_name` (`project_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='项目接入密钥表，每个业务项目对应一条记录';

-- -------------------------------------------------------------
-- 2. anthropic_keys —— Anthropic API Key 池
--    支持多 Key 轮询，自动健康检测与故障切换
-- -------------------------------------------------------------
DROP TABLE IF EXISTS `anthropic_keys`;
CREATE TABLE `anthropic_keys` (
    `id`           BIGINT       NOT NULL AUTO_INCREMENT              COMMENT '自增主键',
    `name`         VARCHAR(128) NOT NULL                             COMMENT 'Key 别名，便于管理识别',
    `key`          VARCHAR(512) NOT NULL                             COMMENT 'Anthropic API Key（sk-ant-...）',
    `enabled`      TINYINT(1)   NOT NULL DEFAULT 1                  COMMENT '是否启用：1=启用 0=禁用',
    `status`       VARCHAR(32)  NOT NULL DEFAULT 'unknown'          COMMENT '健康状态：healthy | unhealthy | unknown',
    `usage_count`  BIGINT       NOT NULL DEFAULT 0                  COMMENT '累计使用次数',
    `last_used_at` DATETIME              DEFAULT NULL               COMMENT '最近一次使用时间',
    `checked_at`   DATETIME              DEFAULT NULL               COMMENT '最近一次健康检测时间',
    `created_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_key` (`key`),
    KEY `idx_enabled_status` (`enabled`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='Anthropic API Key 池，支持多 Key 轮询与故障切换';

-- -------------------------------------------------------------
-- 3. provider_keys —— 第三方供应商 API Key 池（OpenAI / Grok）
--    provider 字段区分供应商，支持多供应商并存
-- -------------------------------------------------------------
DROP TABLE IF EXISTS `provider_keys`;
CREATE TABLE `provider_keys` (
    `id`           BIGINT       NOT NULL AUTO_INCREMENT              COMMENT '自增主键',
    `provider`     VARCHAR(32)  NOT NULL                             COMMENT '供应商标识：openai | grok',
    `name`         VARCHAR(128) NOT NULL                             COMMENT 'Key 别名，便于管理识别',
    `key`          VARCHAR(512) NOT NULL                             COMMENT 'API Key',
    `enabled`      TINYINT(1)   NOT NULL DEFAULT 1                  COMMENT '是否启用：1=启用 0=禁用',
    `status`       VARCHAR(32)  NOT NULL DEFAULT 'unknown'          COMMENT '健康状态：healthy | unhealthy | unknown',
    `usage_count`  BIGINT       NOT NULL DEFAULT 0                  COMMENT '累计使用次数',
    `last_used_at` DATETIME              DEFAULT NULL               COMMENT '最近一次使用时间',
    `checked_at`   DATETIME              DEFAULT NULL               COMMENT '最近一次健康检测时间',
    `created_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_key` (`key`),
    KEY `idx_provider_enabled` (`provider`, `enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='第三方供应商 API Key 池（OpenAI / Grok），provider 字段区分供应商';

-- -------------------------------------------------------------
-- 4. model_configs —— 模型调度配置表
--    控制模型队列的权重与故障转移优先级，支持运行时动态调整
-- -------------------------------------------------------------
DROP TABLE IF EXISTS `model_configs`;
CREATE TABLE `model_configs` (
    `id`       BIGINT       NOT NULL AUTO_INCREMENT        COMMENT '自增主键',
    `model_id` VARCHAR(128) NOT NULL                       COMMENT '模型 ID，如 claude-sonnet-4-20250514 / gpt-4o',
    `name`     VARCHAR(128) NOT NULL                       COMMENT '显示名称',
    `provider` VARCHAR(32)  NOT NULL DEFAULT ''            COMMENT '供应商：anthropic | openai | grok，留空则按 model_id 前缀自动识别',
    `weight`   INT          NOT NULL DEFAULT 50            COMMENT '调度权重（0-100），权重越高被选中概率越大',
    `priority` INT          NOT NULL DEFAULT 1             COMMENT '故障转移优先级，数字越小越优先',
    `enabled`  TINYINT(1)   NOT NULL DEFAULT 1             COMMENT '是否启用：1=启用 0=禁用',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_model_id` (`model_id`),
    KEY `idx_enabled_priority` (`enabled`, `priority`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='模型调度配置表，控制模型队列权重与故障转移优先级';

-- -------------------------------------------------------------
-- 5. admin_settings —— 管理后台轻量设置表（KV 存储）
--    支持运行时修改配置无需重启，当前存储 admin_token_hash 等
-- -------------------------------------------------------------
DROP TABLE IF EXISTS `admin_settings`;
CREATE TABLE `admin_settings` (
    `key`        VARCHAR(128) NOT NULL                                                           COMMENT '配置键，如 admin_token_hash',
    `value`      TEXT         NOT NULL                                                           COMMENT '配置值',
    `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP    COMMENT '最后更新时间',
    PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='管理后台轻量设置表（KV 存储），支持运行时修改无需重启';

-- =============================================================
-- 初始化数据
-- =============================================================

-- 项目密钥
INSERT IGNORE INTO `project_keys` (`id`, `project_name`, `key`, `rate_limit`, `enabled`, `deleted_at`, `created_at`, `updated_at`) VALUES
(1, '91prona', 'proj_91prona_def456',          0, 1, NULL, '2026-03-27 15:06:54', '2026-03-27 15:06:54'),
(2, '51dm',    'sk-proj-51dm-9822cb9cf99f9651', 0, 1, NULL, '2026-03-27 15:10:52', '2026-03-27 15:10:52'),
(3, 'web-91vx',    'sk-proj-91vx-59940fd777477ccc', 0, 1, NULL, '2026-03-27 15:10:52', '2026-03-27 15:10:52'),
(4, 'hlcgw',    'sk-proj-hlcgw-82dc1761b136aabe', 0, 1, NULL, '2026-03-27 15:10:52', '2026-03-27 15:10:52'),
(5, '51kp',    'sk-proj-51kp-3250a16892f2644f', 0, 1, NULL, '2026-03-31 02:06:01', '2026-03-31 02:31:46');

-- Anthropic API Key
INSERT IGNORE INTO `anthropic_keys` (`id`, `name`, `key`, `enabled`, `status`, `usage_count`, `last_used_at`, `checked_at`, `created_at`) VALUES
(1, 'mytk', 'sk-ant-api03-REPLACE_WITH_YOUR_ANTHROPIC_KEY', 1, 'healthy', 0, NULL, '2026-03-31 10:21:55', '2026-03-31 02:20:42');

-- 第三方供应商 API Key
INSERT IGNORE INTO `provider_keys` (`id`, `provider`, `name`, `key`, `enabled`, `status`, `usage_count`, `last_used_at`, `checked_at`, `created_at`) VALUES
(1, 'openai', 'mys', 'sk-svcacct-REPLACE_WITH_YOUR_OPENAI_KEY', 1, 'healthy', 0, NULL, '2026-03-31 10:21:57', '2026-03-31 02:23:55');

-- 模型配置
INSERT IGNORE INTO `model_configs` (`id`, `model_id`, `name`, `provider`, `weight`, `priority`, `enabled`) VALUES
(1, 'claude-3-5',          'Claude 3.5 Sonnet', 'anthropic', 30, 0, 1),
(3, 'gpt-5.1-codex-mini',  'GPT-5.1 Codex mini', 'openai',  20, 1, 1),
(4, 'gpt-4o-mini',         'GPT-4o mini',         'openai',  20, 1, 1);

-- 管理设置
INSERT IGNORE INTO `admin_settings` (`key`, `value`, `updated_at`) VALUES
('admin_token_hash', '636fcbfa9864f6ac0ef292b952051ba4e7215dc018f995ced21dcfc8969e497a', '2026-03-31 02:16:29');

-- =============================================================
-- 用户权限配置
-- =============================================================
-- 删除旧用户（如果存在）
DROP USER IF EXISTS 'moderation'@'localhost';
DROP USER IF EXISTS 'moderation'@'%';
DROP USER IF EXISTS 'moderation'@'172.22.0.1';
DROP USER IF EXISTS 'moderation'@'172.22.0.%';

-- 创建新用户并授予权限
CREATE USER 'moderation'@'localhost' IDENTIFIED BY 'moderation123';
CREATE USER 'moderation'@'%' IDENTIFIED BY 'moderation123';
CREATE USER 'moderation'@'172.22.0.%' IDENTIFIED BY 'moderation123';

GRANT ALL PRIVILEGES ON moderation.* TO 'moderation'@'localhost';
GRANT ALL PRIVILEGES ON moderation.* TO 'moderation'@'%';
GRANT ALL PRIVILEGES ON moderation.* TO 'moderation'@'172.22.0.%';

FLUSH PRIVILEGES;
