-- 000019_user_remark.up.sql: sys_user 增加「备注」列,供用户管理页管理员维护备注信息。
-- 个人签名(signature)归个人中心自助维护,备注归管理员维护,两者语义不同,不互相复用。
ALTER TABLE `sys_user`
    ADD COLUMN `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注' AFTER `signature`;
