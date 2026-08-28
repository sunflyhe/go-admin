-- 000001_init.up.sql: P1 核心系统表(用户、角色、菜单、关联、登录日志)。
-- 字符集统一 utf8mb4,引擎 InnoDB。

CREATE TABLE IF NOT EXISTS `sys_user` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `username`      VARCHAR(64)  NOT NULL,
    `password`      VARCHAR(100) NOT NULL COMMENT 'bcrypt 哈希',
    `nickname`      VARCHAR(64)  NOT NULL DEFAULT '',
    `email`         VARCHAR(128) NOT NULL DEFAULT '',
    `phone`         VARCHAR(32)  NOT NULL DEFAULT '',
    `avatar`        VARCHAR(255) NOT NULL DEFAULT '',
    `status`        TINYINT      NOT NULL DEFAULT 1 COMMENT '1 启用 2 停用',
    `token_version` BIGINT       NOT NULL DEFAULT 0 COMMENT '登出/重置密码后自增使旧 token 失效',
    `last_login_at` DATETIME     NULL,
    `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sys_user_username` (`username`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_role` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(64)  NOT NULL,
    `code`        VARCHAR(64)  NOT NULL,
    `description` VARCHAR(255) NOT NULL DEFAULT '',
    `builtin`     TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '内置角色不可删除',
    `status`      TINYINT      NOT NULL DEFAULT 1,
    `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sys_role_code` (`code`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_menu` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `parent_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `name`       VARCHAR(64)  NOT NULL,
    `type`       TINYINT      NOT NULL COMMENT '1 目录 2 页面 3 按钮',
    `path`       VARCHAR(255) NOT NULL DEFAULT '',
    `component`  VARCHAR(255) NOT NULL DEFAULT '',
    `permission` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '权限码,如 system:user:create',
    `icon`       VARCHAR(64)  NOT NULL DEFAULT '',
    `sort`       INT          NOT NULL DEFAULT 0,
    `status`     TINYINT      NOT NULL DEFAULT 1,
    `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_sys_menu_parent` (`parent_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_user_role` (
    `user_id` BIGINT UNSIGNED NOT NULL,
    `role_id` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`user_id`, `role_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_role_menu` (
    `role_id` BIGINT UNSIGNED NOT NULL,
    `menu_id` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`role_id`, `menu_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_login_log` (
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `username`    VARCHAR(64)  NOT NULL,
    `success`     TINYINT(1)   NOT NULL,
    `fail_reason` VARCHAR(255) NOT NULL DEFAULT '',
    `ip`          VARCHAR(64)  NOT NULL DEFAULT '',
    `user_agent`  VARCHAR(255) NOT NULL DEFAULT '',
    `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_login_log_username` (`username`),
    KEY `idx_login_log_created` (`created_at`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
