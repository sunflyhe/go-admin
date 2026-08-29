-- 000013_config_menu.up.sql: 系统参数的菜单与按钮权限码(修正 ID 段)。
-- 背景:000011 曾用 170-173,但该段已被文件管理按钮占用(000006:移动文件/分组管理),
-- INSERT IGNORE 撞主键被静默跳过,导致表与种子就绪而菜单缺失;本迁移改用空闲的 180 段。
-- 菜单沿用固定主键 + INSERT IGNORE;挂在「系统管理」(100) 下;
-- 新菜单对既有角色默认不可见,由管理员按需在「角色管理」里分配,
-- 超级管理员在服务端按内置规则全量放行,无需补授权。

INSERT IGNORE INTO `sys_menu` (`id`, `parent_id`, `name`, `type`, `path`, `component`, `permission`, `icon`, `sort`) VALUES
(180, 100, '系统参数', 2, '/system/config', 'system/config/index', 'system:config:list', 'SetUp', 7),
(181, 180, '创建参数', 3, '', '', 'system:config:create', '', 1),
(182, 180, '更新参数', 3, '', '', 'system:config:update', '', 2),
(183, 180, '删除参数', 3, '', '', 'system:config:delete', '', 3);
