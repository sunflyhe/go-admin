-- 000010_article_seed.down.sql: 清除文章资讯演示种子数据。
-- 按 000010 固定的主键精确删除,不触碰用户此后新建的其他分类/文章。
-- 若用户恰好在 1..5 主键上建了自己的数据(仅在种子未先行时可能),
-- 为避免误删,删除前限定 author_id = 1(种子的作者恒为 admin)。

DELETE FROM `article` WHERE `id` BETWEEN 1 AND 5 AND `author_id` = 1;
DELETE FROM `article_category` WHERE `id` BETWEEN 1 AND 3;
