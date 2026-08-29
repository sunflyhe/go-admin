-- 000016_dict_type_seed.down.sql: 清除字典模块的演示种子数据(按固定主键)。

DELETE FROM `sys_dict_item` WHERE `id` IN (1, 2, 3, 4, 5);
DELETE FROM `sys_dict_type` WHERE `id` IN (1, 2);
