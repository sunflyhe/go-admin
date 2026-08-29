-- 000009_article.down.sql: 回滚文章资讯模块。菜单关联先删,再删菜单与业务表;
-- 文章与分类数据随表丢弃,由使用方自行确认后再执行 down。

DELETE FROM `sys_role_menu` WHERE `menu_id` BETWEEN 200 AND 224;
DELETE FROM `sys_menu` WHERE `id` BETWEEN 200 AND 224;

DROP TABLE IF EXISTS `article`;
DROP TABLE IF EXISTS `article_category`;
