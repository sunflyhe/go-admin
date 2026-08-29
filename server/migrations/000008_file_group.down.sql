-- 000008_file_group.down.sql: 回滚文件分组能力。
-- 按项目约定 down 迁移不由程序自动执行,仅人工回滚时使用。
-- 注意:DROP COLUMN 会连带丢弃 group_id 上的归属信息,回滚前请先确认已无按分组管理的文件。

DELETE FROM `sys_role_menu` WHERE `menu_id` IN (170, 171, 172, 173);
DELETE FROM `sys_menu` WHERE `id` IN (170, 171, 172, 173);

DROP INDEX `idx_sys_file_group` ON `sys_file`;
ALTER TABLE `sys_file` DROP COLUMN `group_id`;

DROP TABLE IF EXISTS `sys_file_group`;
