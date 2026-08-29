-- 000008_file_group.up.sql: 文件中心分组能力 —— 分组表、文件归属列、分组/移动相关权限码。
-- 说明:
--   1. group_id = 0 是伪节点「未分组」,不写任何种子分组行,客户环境保持干净;
--   2. 本轮只做一级分组,parent_id 先留位,不开放嵌套;
--   3. 菜单按钮沿用固定主键 + INSERT IGNORE;授权规则与 000004 一致 ——
--      凡已拥有「文件管理」(160) 的角色补授新按钮,避免非超管账号看不到入口。

CREATE TABLE IF NOT EXISTS `sys_file_group` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `parent_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '预留父级,本轮只做一级分组',
    `name`       VARCHAR(64)  NOT NULL,
    `sort`       INT          NOT NULL DEFAULT 0,
    `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_sys_file_group_parent` (`parent_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

ALTER TABLE `sys_file`
    ADD COLUMN `group_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属分组,0=未分组' AFTER `ext`;

CREATE INDEX `idx_sys_file_group` ON `sys_file` (`group_id`);

INSERT IGNORE INTO `sys_menu` (`id`, `parent_id`, `name`, `type`, `path`, `component`, `permission`, `icon`, `sort`) VALUES
(170, 160, '移动文件', 3, '', '', 'system:file:move', '', 3),
(171, 160, '新增分组', 3, '', '', 'system:filegroup:create', '', 4),
(172, 160, '更新分组', 3, '', '', 'system:filegroup:update', '', 5),
(173, 160, '删除分组', 3, '', '', 'system:filegroup:delete', '', 6);

INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
SELECT DISTINCT `rm`.`role_id`, `m`.`id`
FROM `sys_role_menu` AS `rm`
JOIN `sys_menu` AS `m` ON `m`.`id` IN (170, 171, 172, 173)
WHERE `rm`.`menu_id` = 160
  AND NOT EXISTS (SELECT 1 FROM `sys_role_menu` WHERE `role_id` = `rm`.`role_id` AND `menu_id` = `m`.`id`);
