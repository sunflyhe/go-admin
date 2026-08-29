-- 000015_dict_type.up.sql: 字典模块 —— 字典类型表、字典项表、菜单与权限码。
-- 说明:
--   1. 与「系统参数」(sys_config,单键值) 分立:参数是业务可调的单值配置,
--      字典是"一个类型下多条子项"的枚举集(如 性别 → 男/女/未知);
--   2. 字典项的 value 为字符串,兼容非数字编码;同一类型内 value 唯一
--      (MySQL utf8mb4_unicode_ci 排序规则下大小写不敏感,服务层另用 LOWER 比较兜底);
--   3. 菜单沿用固定主键 + INSERT IGNORE,一级菜单取空闲的 190 段,sort=6(系统参数 5 之后);
--      按钮写权限与类型共用 system:dict:create/update/delete(一个页面统一管理);
--   4. 新菜单对既有角色默认不可见,由管理员按需在「角色管理」里分配;
--      超级管理员在服务端按内置规则全量放行,无需补授权。

CREATE TABLE IF NOT EXISTS `sys_dict_type` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`       VARCHAR(64)  NOT NULL COMMENT '字典名,展示用',
    `dict_key`   VARCHAR(64)  NOT NULL COMMENT '字典键,业务模块按它读取,唯一',
    `remark`     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
    `builtin`    TINYINT      NOT NULL DEFAULT 0 COMMENT '1=内置字典:子项可维护,不可删、不可改键名',
    `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_sys_dict_type_key` (`dict_key`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sys_dict_item` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `type_id`    BIGINT UNSIGNED NOT NULL COMMENT '所属字典类型',
    `label`      VARCHAR(64)  NOT NULL COMMENT '展示文本',
    `value`      VARCHAR(128) NOT NULL COMMENT '业务存储值',
    `sort`       INT          NOT NULL DEFAULT 0,
    `status`     TINYINT      NOT NULL DEFAULT 1 COMMENT '1=启用 2=停用,停用后 /dict-data 不下发',
    `remark`     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
    `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_dict_item_type_value` (`type_id`, `value`),
    KEY `idx_dict_item_type` (`type_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

INSERT IGNORE INTO `sys_menu` (`id`, `parent_id`, `name`, `type`, `path`, `component`, `permission`, `icon`, `sort`) VALUES
(190, 0, '字典管理', 2, '/system/dict', 'system/dict/index', 'system:dict:list', 'Collection', 6),
(191, 190, '创建字典', 3, '', '', 'system:dict:create', '', 1),
(192, 190, '更新字典', 3, '', '', 'system:dict:update', '', 2),
(193, 190, '删除字典', 3, '', '', 'system:dict:delete', '', 3);
