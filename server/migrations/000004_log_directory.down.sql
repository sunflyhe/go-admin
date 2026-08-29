-- 000004_log_directory.down.sql: 回滚「日志管理」目录,恢复登录日志/操作日志直属系统管理。

UPDATE `sys_menu` SET `parent_id` = 100, `sort` = 4 WHERE `id` = 140;
UPDATE `sys_menu` SET `parent_id` = 100, `sort` = 5 WHERE `id` = 150;

DELETE FROM `sys_role_menu` WHERE `menu_id` = 145;
DELETE FROM `sys_menu` WHERE `id` = 145;
