-- 000018_param_directory.up.sql: 「系统参数」(180) 与「字典管理」(190) 归入一级目录「参数字典」。
-- 说明:
--   1. 新建目录菜单 185(空闲段,一级 sort=5,接替原「系统参数」的位置),
--      path=/params 仅作目录路由占位;两个子页面的 path/组件/权限码均不变,前端零改动;
--   2. 子页面在目录内重新排序:系统参数 1、字典管理 2;
--   3. WHERE 带上原 parent_id:若客户已自行调整过归属,不覆盖他们的改动;
--   4. 可见性影响:菜单树装配时父级未授权的子节点本会被提升为根节点(见 internal/service/menu.go
--      buildTree),只被授予 180/190 而未授予 185 的角色入口不丢,本次只是把隐式结果变成显式数据。

INSERT IGNORE INTO `sys_menu` (`id`, `parent_id`, `name`, `type`, `path`, `component`, `permission`, `icon`, `sort`) VALUES
(185, 0, '参数字典', 1, '/params', '', '', 'Operation', 5);

UPDATE `sys_menu` SET `parent_id` = 185, `sort` = 1 WHERE `id` = 180 AND `parent_id` = 0;
UPDATE `sys_menu` SET `parent_id` = 185, `sort` = 2 WHERE `id` = 190 AND `parent_id` = 0;
