-- 000001_init.down.sql: 仅人工在明确风险下执行;将删除 P1 核心系统表及全部数据。
DROP TABLE IF EXISTS `sys_login_log`;
DROP TABLE IF EXISTS `sys_role_menu`;
DROP TABLE IF EXISTS `sys_user_role`;
DROP TABLE IF EXISTS `sys_menu`;
DROP TABLE IF EXISTS `sys_role`;
DROP TABLE IF EXISTS `sys_user`;
