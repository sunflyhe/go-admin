-- 000011_config.up.sql: 系统参数 —— 键值参数表。
-- 说明:
--   1. 表名列名避开 MySQL 保留字:key 是保留字,列名用 config_key;
--   2. 菜单与按钮权限码不在本文件:最初写在这里的 170-173 与文件管理按钮(000006)撞主键,
--      INSERT IGNORE 被静默跳过;菜单改由 000013 以空闲的 180 段插入。

CREATE TABLE IF NOT EXISTS `sys_config` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`       VARCHAR(64)  NOT NULL COMMENT '参数名,展示用',
    `config_key`   VARCHAR(64)  NOT NULL COMMENT '参数键,业务模块按它读取,唯一',
    `value`      VARCHAR(512) NOT NULL DEFAULT '' COMMENT '参数值',
    `remark`     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
    `builtin`    TINYINT      NOT NULL DEFAULT 0 COMMENT '1=内置参数:可改值,不可删、不可改键名',
    `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sys_config_key` (`config_key`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
