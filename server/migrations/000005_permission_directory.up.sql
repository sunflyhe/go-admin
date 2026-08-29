-- 000005_permission_directory.up.sql: 一级目录「系统管理」(100) 更名为「权限管理」。
-- 只改展示名称:目录 path(/system)、子菜单 path、组件与权限码(system:*)全部保持不变,
-- 因此已签发凭据、角色-菜单绑定、前端动态路由都不受影响。
-- WHERE 带上 name = '系统管理':若客户已自行改名,不覆盖他们的改动(种子数据幂等规则)。

UPDATE `sys_menu` SET `name` = '权限管理' WHERE `id` = 100 AND `name` = '系统管理';
