-- 000019_user_remark.down.sql: 移除「备注」列。
ALTER TABLE `sys_user` DROP COLUMN `remark`;
