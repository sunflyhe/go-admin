-- 000009_article.up.sql: 文章资讯模块 —— 文章分类表、文章表、菜单与权限码。
-- 说明:
--   1. 菜单沿用固定主键 + INSERT IGNORE;200 段为新一级目录「文章资讯」预留;
--   2. 正文 content 用 MEDIUMTEXT 存富文本 HTML,列表接口不回传,详情接口才返回;
--   3. 文章配图走 POST /api/v1/article-images(权限码 article:article:upload-image),
--      复用文件上传白名单与真实 MIME 校验,且强制 is_public=true ——
--      正文 <img> 无法携带 Authorization,图片必须公开可访问;
--   4. 新模块对既有角色默认不可见,由管理员按需在「角色管理」里分配;
--      超级管理员在服务端按内置规则全量放行,无需补授权。

CREATE TABLE IF NOT EXISTS `article_category` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`       VARCHAR(64) NOT NULL COMMENT '分类名,唯一',
    `sort`       INT         NOT NULL DEFAULT 0,
    `created_at` DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_article_category_name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `article` (
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `category_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属分类,0=未分类',
    `title`        VARCHAR(128) NOT NULL,
    `summary`      VARCHAR(255) NOT NULL DEFAULT '' COMMENT '列表摘要,空=列表不展示',
    `content`      MEDIUMTEXT   NOT NULL COMMENT '富文本 HTML',
    `status`       TINYINT      NOT NULL DEFAULT 1 COMMENT '1=草稿 2=已发布',
    `author_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建者用户 ID',
    `author`       VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '创建者用户名快照',
    `published_at` DATETIME     NULL COMMENT '首次发布时间,保留不随再次编辑变动',
    `created_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_article_category` (`category_id`),
    KEY `idx_article_status` (`status`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

INSERT IGNORE INTO `sys_menu` (`id`, `parent_id`, `name`, `type`, `path`, `component`, `permission`, `icon`, `sort`) VALUES
(200, 0, '文章资讯', 1, '/article', '', '', 'Notebook', 4),
(210, 200, '文章分类', 2, '/article/category', 'article/category/index', 'article:category:list', 'Collection', 1),
(211, 210, '创建分类', 3, '', '', 'article:category:create', '', 1),
(212, 210, '更新分类', 3, '', '', 'article:category:update', '', 2),
(213, 210, '删除分类', 3, '', '', 'article:category:delete', '', 3),
(220, 200, '文章管理', 2, '/article/article', 'article/article/index', 'article:article:list', 'Document', 2),
(221, 220, '创建文章', 3, '', '', 'article:article:create', '', 1),
(222, 220, '更新文章', 3, '', '', 'article:article:update', '', 2),
(223, 220, '删除文章', 3, '', '', 'article:article:delete', '', 3),
(224, 220, '上传配图', 3, '', '', 'article:article:upload-image', '', 4);
