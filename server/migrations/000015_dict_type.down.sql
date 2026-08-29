-- 000015_dict_type.down.sql: 删除字典模块的菜单与两张表。

DELETE FROM `sys_menu` WHERE `id` IN (190, 191, 192, 193);

DROP TABLE IF EXISTS `sys_dict_item`;
DROP TABLE IF EXISTS `sys_dict_type`;
