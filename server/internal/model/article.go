// 文章资讯域的 GORM 模型。表结构以 SQL migration 为准,模型仅作查询映射。
package model

import (
	"time"
)

// 文章状态。
const (
	ArticleStatusDraft     = 1 // 草稿:仅后台可见
	ArticleStatusPublished = 2 // 已发布
)

// ArticleCategory 文章分类。只做一级分类,不开放嵌套。
type ArticleCategory struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;uniqueIndex" json:"name"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
}

func (ArticleCategory) TableName() string { return "article_category" }

// Article 文章。Content 为富文本 HTML:列表接口不回传,详情接口才返回,避免整页长文拖垮列表。
type Article struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	CategoryID  int64      `gorm:"index" json:"categoryId"` // 0=未分类
	Title       string     `gorm:"size:128" json:"title"`
	Summary     string     `gorm:"size:255" json:"summary"`
	Content     string     `gorm:"type:mediumtext" json:"content"`
	Status      int        `gorm:"default:1" json:"status"`
	AuthorID    int64      `json:"authorId"`
	Author      string     `gorm:"size:64" json:"author"` // 创建者用户名快照
	PublishedAt *time.Time `json:"publishedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (Article) TableName() string { return "article" }
