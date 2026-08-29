-- 000013_config_menu.down.sql: 删除系统参数菜单(按固定主键)。

DELETE FROM `sys_menu` WHERE `id` IN (180, 181, 182, 183);
