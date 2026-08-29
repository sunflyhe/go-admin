-- 000017_dict_item_fields.up.sql: 字典项补充「描述」与「标签类型」两列。
-- 说明:
--   1. description 是子项的业务描述,与 remark(纯备注)分列,列序对齐管理端展示习惯;
--   2. tag_type 为前端 el-tag 预留的配色标记,取值:空/primary/success/warning/danger/info,
--      服务层校验取值;业务经 /dict-data 读取后可按它渲染彩色标签;
--   3. 列位置按 value/label/description/sort/tag_type/remark 排布,与页面列序一致。

ALTER TABLE `sys_dict_item`
    ADD COLUMN `description` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '业务描述' AFTER `label`,
    ADD COLUMN `tag_type`    VARCHAR(32)  NOT NULL DEFAULT '' COMMENT '前端标签配色:空/primary/success/warning/danger/info' AFTER `sort`;
