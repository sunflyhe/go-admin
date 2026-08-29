-- 000003_seed.up.sql: 幂等种子数据。
-- 规则:固定主键 + INSERT IGNORE,已存在的记录不会被覆盖,客户对角色/菜单的修改不受影响。
-- 默认超级管理员账号 admin / Admin@123456(哈希见下),首次部署后必须立即修改。

INSERT IGNORE INTO `sys_user` (`id`, `username`, `password`, `nickname`, `status`) VALUES
(1, 'admin', '$2a$10$cf8c7339Qo7.t2wFU1wy7O3wvqI8eAiPeMSwQy40Wxza5GYNo04u.', '超级管理员', 1);

INSERT IGNORE INTO `sys_role` (`id`, `name`, `code`, `description`, `builtin`, `status`) VALUES
(1, '超级管理员', 'super_admin', '内置角色:拥有全部权限,不可删除', 1, 1),
(2, '审计员', 'auditor', '种子示例角色:仅可查看登录日志与操作日志,用于演示权限差异', 0, 1);

-- 菜单树:目录(1)/页面(2)/按钮(3),permission code 为服务端强制校验依据。
INSERT IGNORE INTO `sys_menu` (`id`, `parent_id`, `name`, `type`, `path`, `component`, `permission`, `icon`, `sort`) VALUES
(101, 0, '仪表盘', 2, '/dashboard', 'dashboard/index', '', 'Odometer', 0),
(100, 0, '系统管理', 1, '/system', '', '', 'Setting', 1),
(110, 100, '用户管理', 2, '/system/user', 'system/user/index', 'system:user:list', 'User', 1),
(111, 110, '创建用户', 3, '', '', 'system:user:create', '', 1),
(112, 110, '更新用户', 3, '', '', 'system:user:update', '', 2),
(113, 110, '删除用户', 3, '', '', 'system:user:delete', '', 3),
(114, 110, '重置密码', 3, '', '', 'system:user:reset-password', '', 4),
(115, 110, '分配角色', 3, '', '', 'system:user:assign-role', '', 5),
(116, 110, '导出用户', 3, '', '', 'system:user:export', '', 6),
(120, 100, '角色管理', 2, '/system/role', 'system/role/index', 'system:role:list', 'Avatar', 2),
(121, 120, '创建角色', 3, '', '', 'system:role:create', '', 1),
(122, 120, '更新角色', 3, '', '', 'system:role:update', '', 2),
(123, 120, '删除角色', 3, '', '', 'system:role:delete', '', 3),
(124, 120, '分配菜单', 3, '', '', 'system:role:assign-menu', '', 4),
(130, 100, '菜单管理', 2, '/system/menu', 'system/menu/index', 'system:menu:list', 'Menu', 3),
(131, 130, '创建菜单', 3, '', '', 'system:menu:create', '', 1),
(132, 130, '更新菜单', 3, '', '', 'system:menu:update', '', 2),
(133, 130, '删除菜单', 3, '', '', 'system:menu:delete', '', 3),
(140, 100, '登录日志', 2, '/system/login-log', 'system/loginlog/index', 'system:loginlog:list', 'Key', 4),
(150, 100, '操作日志', 2, '/system/audit-log', 'system/auditlog/index', 'system:auditlog:list', 'Document', 5),
(160, 100, '文件管理', 2, '/system/file', 'system/file/index', 'system:file:list', 'Files', 6),
(161, 160, '上传文件', 3, '', '', 'system:file:upload', '', 1),
(162, 160, '删除文件', 3, '', '', 'system:file:delete', '', 2);

-- 超级管理员在服务端按内置规则全量放行,无需逐条授权;
-- 这里仍建立关联,便于直接查询其角色对应菜单。
INSERT IGNORE INTO `sys_user_role` (`user_id`, `role_id`) VALUES (1, 1);

INSERT IGNORE INTO `sys_role_menu` (`role_id`, `menu_id`)
SELECT 1, `id` FROM `sys_menu`
WHERE NOT EXISTS (SELECT 1 FROM `sys_role_menu` WHERE `role_id` = 1 AND `menu_id` = `sys_menu`.`id`);

-- 审计员角色:仅授予目录与两个日志页面(含页面上的 list 权限码)。
INSERT IGNORE INTO `sys_role_menu` (`role_id`, `menu_id`) VALUES
(2, 100), (2, 140), (2, 150);
