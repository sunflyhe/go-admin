-- 000014_config_menu_top_level.down.sql: 参数字典恢复挂回「系统管理」(100) 下,排序还原为 7。
-- WHERE 带上 parent_id = 0:若客户已自行调整过归属,不覆盖他们的改动。

UPDATE `sys_menu` SET `parent_id` = 100, `sort` = 7 WHERE `id` = 180 AND `parent_id` = 0;
