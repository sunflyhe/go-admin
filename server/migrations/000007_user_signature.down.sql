-- 000007_user_signature.down.sql: 移除「个人签名」列。
-- 按项目约定 down 迁移不由程序自动执行,仅人工回滚时使用。

ALTER TABLE `sys_user` DROP COLUMN `signature`;
