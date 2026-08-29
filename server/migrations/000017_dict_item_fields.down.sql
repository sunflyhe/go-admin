-- 000017_dict_item_fields.down.sql: 移除字典项的描述与标签类型列。

ALTER TABLE `sys_dict_item`
    DROP COLUMN `description`,
    DROP COLUMN `tag_type`;
