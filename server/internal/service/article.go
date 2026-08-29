// 文章服务:文章资讯的列表、详情与增删改。正文为富文本 HTML,由前端编辑器产出。
package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
)

const (
	maxArticleTitleLen   = 128
	maxArticleSummaryLen = 255
)

// ArticleItem 列表条目:不含正文,列表页不需要整页 HTML。
type ArticleItem struct {
	ID           int64      `json:"id"`
	CategoryID   int64      `json:"categoryId"`
	CategoryName string     `json:"categoryName"`
	Title        string     `json:"title"`
	Summary      string     `json:"summary"`
	Status       int        `json:"status"`
	Author       string     `json:"author"`
	PublishedAt  *time.Time `json:"publishedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// ArticleDetail 详情:在列表字段之上多一个正文。
type ArticleDetail struct {
	ArticleItem
	Content string `json:"content"`
}

// ArticleSaveInput 创建/更新共用的业务输入,与 HTTP 框架无关。
type ArticleSaveInput struct {
	CategoryID int64
	Title      string
	Summary    string
	Content    string
	Status     int
}

// ArticleListInput 列表查询输入。CategoryID 用指针区分"没传"(全部分类)与"传了 0"(未分类);
// Status 0 表示不筛选。
type ArticleListInput struct {
	page.Query
	Title      string
	CategoryID *int64
	Status     int
}

type ArticleService struct {
	DB *gorm.DB
}

func NewArticleService(db *gorm.DB) *ArticleService {
	return &ArticleService{DB: db}
}

// normalizeArticleInput 校验保存入参,返回规范化后的副本。
func normalizeArticleInput(input ArticleSaveInput) (ArticleSaveInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return input, errs.InvalidParam("文章标题不能为空")
	}
	if len([]rune(input.Title)) > maxArticleTitleLen {
		return input, errs.InvalidParam("文章标题最长 128 个字符")
	}
	input.Summary = strings.TrimSpace(input.Summary)
	if len([]rune(input.Summary)) > maxArticleSummaryLen {
		return input, errs.InvalidParam("摘要最长 255 个字符")
	}
	if strings.TrimSpace(input.Content) == "" {
		return input, errs.InvalidParam("文章正文不能为空")
	}
	if input.CategoryID < 0 {
		return input, errs.InvalidParam("文章分类不存在")
	}
	switch input.Status {
	case model.ArticleStatusDraft, model.ArticleStatusPublished:
	default:
		return input, errs.InvalidParam("文章状态取值不合法")
	}
	return input, nil
}

func (s *ArticleService) assertCategoryExists(ctx context.Context, categoryID int64) error {
	if categoryID == 0 {
		return nil // 0 = 未分类,合法
	}
	var count int64
	if err := s.DB.WithContext(ctx).Model(&model.ArticleCategory{}).
		Where("id = ?", categoryID).Count(&count).Error; err != nil {
		return errs.Internal("查询分类失败").WithCause(err)
	}
	if count == 0 {
		return errs.NotFound("文章分类不存在")
	}
	return nil
}

// List 分页列表:LEFT JOIN 分类拿名称,已删分类的文章回退为"未分类"。
func (s *ArticleService) List(ctx context.Context, input *ArticleListInput) (*page.Result, error) {
	if err := input.Normalize(); err != nil {
		return nil, err
	}
	if input.Status != 0 {
		switch input.Status {
		case model.ArticleStatusDraft, model.ArticleStatusPublished:
		default:
			return nil, errs.InvalidParam("文章状态取值不合法")
		}
	}
	q := s.DB.WithContext(ctx).Model(&model.Article{}).
		Select("article.id, article.category_id, article.title, article.summary, article.status, "+
			"article.author, article.published_at, article.created_at, "+
			"IFNULL(article_category.name, '未分类') AS category_name").
		Joins("LEFT JOIN article_category ON article_category.id = article.category_id")
	if input.Title != "" {
		q = q.Where("article.title LIKE ?", "%"+input.Title+"%")
	}
	if input.CategoryID != nil {
		q = q.Where("article.category_id = ?", *input.CategoryID)
	}
	if input.Status != 0 {
		q = q.Where("article.status = ?", input.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errs.Internal("查询文章失败").WithCause(err)
	}
	var items []ArticleItem
	if err := q.Order("article.id DESC").Offset(input.Offset()).Limit(input.PageSize).
		Scan(&items).Error; err != nil {
		return nil, errs.Internal("查询文章失败").WithCause(err)
	}
	return &page.Result{List: items, Total: total, Page: input.Page, PageSize: input.PageSize}, nil
}

// Get 详情,含正文。
func (s *ArticleService) Get(ctx context.Context, id int64) (*ArticleDetail, error) {
	if id <= 0 {
		return nil, errs.InvalidParam("文章不存在")
	}
	var article model.Article
	if err := s.DB.WithContext(ctx).First(&article, id).Error; err != nil {
		return nil, errs.NotFound("文章不存在")
	}
	categoryName := "未分类"
	if article.CategoryID > 0 {
		var category model.ArticleCategory
		if err := s.DB.WithContext(ctx).First(&category, article.CategoryID).Error; err == nil {
			categoryName = category.Name
		}
	}
	return &ArticleDetail{
		ArticleItem: ArticleItem{
			ID: article.ID, CategoryID: article.CategoryID, CategoryName: categoryName,
			Title: article.Title, Summary: article.Summary, Status: article.Status,
			Author: article.Author, PublishedAt: article.PublishedAt,
			CreatedAt: article.CreatedAt,
		},
		Content: article.Content,
	}, nil
}

func (s *ArticleService) Create(ctx context.Context, actor Actor, input ArticleSaveInput) (*model.Article, error) {
	input, err := normalizeArticleInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.assertCategoryExists(ctx, input.CategoryID); err != nil {
		return nil, err
	}
	article := &model.Article{
		CategoryID: input.CategoryID,
		Title:      input.Title,
		Summary:    input.Summary,
		Content:    input.Content,
		Status:     input.Status,
		AuthorID:   actor.ID,
		Author:     actor.Username,
	}
	// 创建即发布时记录发布时间
	if input.Status == model.ArticleStatusPublished {
		now := time.Now()
		article.PublishedAt = &now
	}
	if err := s.DB.WithContext(ctx).Create(article).Error; err != nil {
		return nil, errs.Internal("创建文章失败").WithCause(err)
	}
	return article, nil
}

// Update 更新文章。作者以创建者为准,编辑不改署名;
// 首次转为发布状态时补记 published_at,再次编辑不覆盖首发时间。
func (s *ArticleService) Update(ctx context.Context, id int64, input ArticleSaveInput) error {
	if id <= 0 {
		return errs.InvalidParam("文章不存在")
	}
	input, err := normalizeArticleInput(input)
	if err != nil {
		return err
	}
	var article model.Article
	if err := s.DB.WithContext(ctx).First(&article, id).Error; err != nil {
		return errs.NotFound("文章不存在")
	}
	if err := s.assertCategoryExists(ctx, input.CategoryID); err != nil {
		return err
	}
	updates := map[string]any{
		"category_id": input.CategoryID,
		"title":       input.Title,
		"summary":     input.Summary,
		"content":     input.Content,
		"status":      input.Status,
	}
	if input.Status == model.ArticleStatusPublished && article.PublishedAt == nil {
		now := time.Now()
		updates["published_at"] = &now
	}
	if err := s.DB.WithContext(ctx).Model(&model.Article{ID: article.ID}).
		Updates(updates).Error; err != nil {
		return errs.Internal("更新文章失败").WithCause(err)
	}
	return nil
}

func (s *ArticleService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errs.InvalidParam("文章不存在")
	}
	var article model.Article
	if err := s.DB.WithContext(ctx).First(&article, id).Error; err != nil {
		return errs.NotFound("文章不存在")
	}
	if err := s.DB.WithContext(ctx).Delete(&model.Article{}, id).Error; err != nil {
		return errs.Internal("删除文章失败").WithCause(err)
	}
	return nil
}
