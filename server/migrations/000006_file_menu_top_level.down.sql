-- 000006_file_menu_top_level.down.sql: 把「文件管理」放回「权限管理」目录,恢复种子中的排序。

UPDATE `sys_menu` SET `parent_id` = 100, `sort` = 6 WHERE `id` = 160 AND `parent_id` = 0;
