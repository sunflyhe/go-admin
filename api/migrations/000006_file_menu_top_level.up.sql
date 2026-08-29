-- 000006_file_menu_top_level.up.sql: 「文件管理」(160) 从「权限管理」目录移到一级菜单。
-- 页面 path(/system/file)、组件、权限码(system:file:*)与其下按钮(161/162)均不变,前端无需改动。
-- 可见性影响:菜单树装配时,父级未授权给某角色的子节点本就会被提升为根节点(见 internal/service/menu.go buildTree),
-- 所以"只被授予 160、未被授予 100"的角色看到的入口不变;本次只是把隐式结果变成显式数据。
-- 一级菜单现有排序:仪表盘 0、权限管理 1、日志管理 2,故文件管理取 sort = 3。
-- WHERE 带上 parent_id = 100:若客户已自行调整过归属,不覆盖他们的改动。

UPDATE `sys_menu` SET `parent_id` = 0, `sort` = 3 WHERE `id` = 160 AND `parent_id` = 100;
