-- 000005_permission_directory.down.sql: 恢复目录名称为「系统管理」。

UPDATE `sys_menu` SET `name` = '系统管理' WHERE `id` = 100 AND `name` = '权限管理';
