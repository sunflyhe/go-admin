-- 000007_user_signature.up.sql: sys_user 增加「个人签名」列,供个人中心展示与编辑。
-- avatar 列在 000001 已存在,本次只补 signature;个人中心头像写入复用现有列。

ALTER TABLE `sys_user`
    ADD COLUMN `signature` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '个人签名' AFTER `avatar`;
