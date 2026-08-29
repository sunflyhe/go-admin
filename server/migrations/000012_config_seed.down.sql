-- 000012_config_seed.down.sql: 清除系统参数的演示种子数据。
-- 只按固定主键删除,客户在原 ID 上新建或改名过的数据不受影响。

DELETE FROM `sys_config` WHERE `id` IN (1, 2, 3);
