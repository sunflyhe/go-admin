-- 000002_ops.up.sql: P2 表(操作审计日志、文件元数据、刷新令牌登记)。

CREATE TABLE IF NOT EXISTS `sys_audit_log` (
    `id`               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `username`         VARCHAR(64)  NOT NULL DEFAULT '',
    `method`           VARCHAR(8)   NOT NULL DEFAULT '',
    `path`             VARCHAR(255) NOT NULL DEFAULT '',
    `status`           INT          NOT NULL DEFAULT 0 COMMENT 'HTTP 状态码',
    `latency_ms`       BIGINT       NOT NULL DEFAULT 0,
    `ip`               VARCHAR(64)  NOT NULL DEFAULT '',
    `user_agent`       VARCHAR(255) NOT NULL DEFAULT '',
    `request_summary`  TEXT         NULL COMMENT '脱敏后的请求摘要',
    `response_summary` TEXT         NULL COMMENT '响应摘要/错误信息',
    `created_at`       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_audit_log_user` (`user_id`),
    KEY `idx_audit_log_created` (`created_at`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_file` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `origin_name` VARCHAR(255) NOT NULL,
    `store_path`  VARCHAR(255) NOT NULL COMMENT '相对存储根目录路径',
    `size`        BIGINT       NOT NULL DEFAULT 0,
    `mime`        VARCHAR(128) NOT NULL DEFAULT '',
    `ext`         VARCHAR(16)  NOT NULL DEFAULT '',
    `is_public`   TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '公开文件可直接访问,私有文件需鉴权下载',
    `uploader_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `uploader`    VARCHAR(64)  NOT NULL DEFAULT '',
    `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sys_file_store_path` (`store_path`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_refresh_token` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `jti`        VARCHAR(64) NOT NULL,
    `user_id`    BIGINT UNSIGNED NOT NULL,
    `expires_at` DATETIME    NOT NULL,
    `revoked`    TINYINT(1)  NOT NULL DEFAULT 0,
    `revoked_at` DATETIME    NULL,
    `created_at` DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_refresh_token_jti` (`jti`),
    KEY `idx_refresh_token_user` (`user_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
