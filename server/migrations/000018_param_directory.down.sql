-- 000018_param_directory.down.sql: 拆散「参数字典」目录,两个页面恢复为一级菜单,原排序还原。

UPDATE `sys_menu` SET `parent_id` = 0, `sort` = 5 WHERE `id` = 180 AND `parent_id` = 185;
UPDATE `sys_menu` SET `parent_id` = 0, `sort` = 6 WHERE `id` = 190 AND `parent_id` = 185;
DELETE FROM `sys_menu` WHERE `id` = 185;
