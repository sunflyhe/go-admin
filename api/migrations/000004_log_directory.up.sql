-- 000004_log_directory.up.sql: 新增一级「日志管理」目录,收纳登录日志与操作日志。
-- 说明:本迁移会调整 140/150 两条种子菜单的归属(这是本次变更的目的);
-- 页面路由(/system/login-log、/system/audit-log)、组件与权限码均不变,前端无需改动。
-- 角色绑定:凡可见 140/150 的角色都补授目录 145,避免非内置角色丢失日志入口。

INSERT IGNORE INTO `sys_menu` (`id`, `parent_id`, `name`, `type`, `path`, `component`, `permission`, `icon`, `sort`) VALUES
(145, 0, '日志管理', 1, '/log', '', '', 'Tickets', 2);

UPDATE `sys_menu` SET `parent_id` = 145, `sort` = 1 WHERE `id` = 140;
UPDATE `sys_menu` SET `parent_id` = 145, `sort` = 2 WHERE `id` = 150;

INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
SELECT DISTINCT `rm`.`role_id`, 145
FROM `sys_role_menu` AS `rm`
WHERE `rm`.`menu_id` IN (140, 150)
  AND NOT EXISTS (SELECT 1 FROM `sys_role_menu` WHERE `role_id` = `rm`.`role_id` AND `menu_id` = 145);
