-- 000003_seed.down.sql: 仅人工在明确风险下执行;将删除种子数据(含超管账号)。
DELETE FROM `sys_role_menu` WHERE `role_id` IN (1, 2);
DELETE FROM `sys_user_role` WHERE `user_id` = 1;
DELETE FROM `sys_menu` WHERE `id` IN (100,101,110,111,112,113,114,115,116,120,121,122,123,124,130,131,132,133,140,150,160,161,162);
DELETE FROM `sys_role` WHERE `id` IN (1, 2);
DELETE FROM `sys_user` WHERE `id` = 1;
