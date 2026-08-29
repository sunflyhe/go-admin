-- 000014_config_menu_top_level.up.sql: 「系统参数」(180) 从「系统管理」目录移到一级菜单。
-- 页面 path(/system/config)、组件、权限码(system:config:*)与其下按钮(181-183)均不变,前端无需改动。
-- 一级菜单现有排序:仪表盘 0、权限管理 1、日志管理 2、文件管理 3、文章资讯 4,故系统参数取 sort = 5。
-- WHERE 带上 parent_id = 100:若客户已自行调整过归属,不覆盖他们的改动。

UPDATE `sys_menu` SET `parent_id` = 0, `sort` = 5 WHERE `id` = 180 AND `parent_id` = 100;
