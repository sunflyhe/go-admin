-- 000011_config.down.sql: 删除系统参数表。
-- 菜单归 000013 管理(180-183),由 000013 的 down 删除;本文件不再触碰 sys_menu。

DROP TABLE IF EXISTS `sys_config`;
