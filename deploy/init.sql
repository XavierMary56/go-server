CREATE DATABASE IF NOT EXISTS moderation CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE moderation;

CREATE TABLE IF NOT EXISTS project_keys (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    project_id VARCHAR(128) NOT NULL,
    `key`      VARCHAR(256) NOT NULL UNIQUE,
    rate_limit INT          NOT NULL DEFAULT 60,
    enabled    TINYINT(1)   NOT NULL DEFAULT 1,
    deleted_at DATETIME     DEFAULT NULL,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS anthropic_keys (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    `key`        VARCHAR(512) NOT NULL UNIQUE,
    enabled      TINYINT(1)   NOT NULL DEFAULT 1,
    status       VARCHAR(32)  NOT NULL DEFAULT 'unknown',
    usage_count  BIGINT       NOT NULL DEFAULT 0,
    last_used_at DATETIME     DEFAULT NULL,
    checked_at   DATETIME     DEFAULT NULL,
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS provider_keys (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    provider     VARCHAR(32)  NOT NULL,
    name         VARCHAR(128) NOT NULL,
    `key`        VARCHAR(512) NOT NULL UNIQUE,
    enabled      TINYINT(1)   NOT NULL DEFAULT 1,
    status       VARCHAR(32)  NOT NULL DEFAULT 'unknown',
    usage_count  BIGINT       NOT NULL DEFAULT 0,
    last_used_at DATETIME     DEFAULT NULL,
    checked_at   DATETIME     DEFAULT NULL,
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS model_configs (
    id       BIGINT AUTO_INCREMENT PRIMARY KEY,
    model_id VARCHAR(128) NOT NULL UNIQUE,
    name     VARCHAR(128) NOT NULL,
    provider VARCHAR(32)  NOT NULL DEFAULT '',
    weight   INT          NOT NULL DEFAULT 50,
    priority INT          NOT NULL DEFAULT 1,
    enabled  TINYINT(1)   NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS admin_settings (
    `key`      VARCHAR(128) PRIMARY KEY,
    value      TEXT         NOT NULL,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 迁移数据
INSERT IGNORE INTO project_keys (id, project_id, `key`, rate_limit, enabled, deleted_at, created_at, updated_at) VALUES
(1, '91prona', 'proj_91prona_def456', 0, 1, NULL, '2026-03-27 15:06:54', '2026-03-27 15:06:54'),
(2, '51dm', 'sk-proj-51dm-1c184bbbd7a8faaf', 0, 1, NULL, '2026-03-27 15:10:52', '2026-03-27 15:10:52'),
(3, '51kp', 'sk-proj-51kp-3250a16892f2644f', 0, 1, NULL, '2026-03-31 02:06:01', '2026-03-31 02:31:46');

INSERT IGNORE INTO anthropic_keys (id, name, `key`, enabled, status, usage_count, last_used_at, checked_at, created_at) VALUES
(1, 'mytk', 'sk-ant-api03-REPLACE_WITH_YOUR_ANTHROPIC_KEY', 1, 'healthy', 0, NULL, '2026-03-31 10:21:55', '2026-03-31 02:20:42');

INSERT IGNORE INTO provider_keys (id, provider, name, `key`, enabled, status, usage_count, last_used_at, checked_at, created_at) VALUES
(1, 'openai', 'mys', 'sk-svcacct-REPLACE_WITH_YOUR_OPENAI_KEY', 1, 'healthy', 0, NULL, '2026-03-31 10:21:57', '2026-03-31 02:23:55');

INSERT IGNORE INTO model_configs (id, model_id, name, provider, weight, priority, enabled) VALUES
(1, 'claude-3-5', 'Claude 3.5 Sonnet', 'anthropic', 30, 0, 1),
(3, 'gpt-5.1-codex-mini', 'GPT-5.1 Codex mini', 'openai', 20, 1, 1),
(4, 'gpt-4o-mini', 'GPT-4o mini', 'openai', 20, 1, 1);

INSERT IGNORE INTO admin_settings (`key`, value, updated_at) VALUES
('admin_token_hash', '636fcbfa9864f6ac0ef292b952051ba4e7215dc018f995ced21dcfc8969e497a', '2026-03-31 02:16:29');
