// 文章资讯模块的 HTTP 绑定 DTO 与到 Service 输入的显式转换。
package handler

import (
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

// ---- 文章分类 ----

// ArticleCategorySaveRequest POST/PUT /admin-api/article-categories 请求体。
type ArticleCategorySaveRequest struct {
	Name string `json:"name" binding:"required,max=64"`
	Sort int    `json:"sort"`
}

// ---- 文章 ----

// ArticleListQuery GET /admin-api/articles 查询参数。
// categoryId 用指针区分"没传"(全部分类)与"传了 0"(未分类),语义不同不能合并。
type ArticleListQuery struct {
	page.Query
	Title      string `form:"title"`
	CategoryID *int64 `form:"categoryId"`
	Status     int    `form:"status"` // 0=全部
}

func (q *ArticleListQuery) toInput() *service.ArticleListInput {
	return &service.ArticleListInput{
		Query:      q.Query,
		Title:      q.Title,
		CategoryID: q.CategoryID,
		Status:     q.Status,
	}
}

// ArticleSaveRequest POST/PUT /admin-api/articles 请求体。
// categoryId=0 表示未分类,是合法取值,故不加 required;
// status 的取值合法性由 Service 裁决,binding 只挡明显乱传。
type ArticleSaveRequest struct {
	CategoryID int64  `json:"categoryId"`
	Title      string `json:"title" binding:"required,max=128"`
	Summary    string `json:"summary" binding:"max=255"`
	Content    string `json:"content" binding:"required"`
	Status     int    `json:"status" binding:"required,oneof=1 2"`
}

func (r *ArticleSaveRequest) toInput() service.ArticleSaveInput {
	return service.ArticleSaveInput{
		CategoryID: r.CategoryID,
		Title:      r.Title,
		Summary:    r.Summary,
		Content:    r.Content,
		Status:     r.Status,
	}
}
