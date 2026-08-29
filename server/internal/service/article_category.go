// 文章分类服务:文章资讯的枚举维护。只做一级分类,不开放嵌套。
package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

const maxCategoryNameLen = 64

// ArticleCategoryItem 分类条目,Count 为分类下文章数(供页面展示与删除前的判断提示)。
type ArticleCategoryItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Sort      int       `json:"sort"`
	Count     int64     `json:"count"`
	CreatedAt time.Time `json:"createdAt"`
}

type ArticleCategoryService struct {
	DB *gorm.DB
}

func NewArticleCategoryService(db *gorm.DB) *ArticleCategoryService {
	return &ArticleCategoryService{DB: db}
}

// List 取回全部分类与各自文章数,一次 GROUP BY 完成;分类是少量枚举,不分页。
// 计数包含草稿与已发布:删除前的冲突判断必须按全量口径,否则"看着是空的"分类会删不掉。
func (s *ArticleCategoryService) List(ctx context.Context) ([]ArticleCategoryItem, error) {
	var categories []model.ArticleCategory
	if err := s.DB.WithContext(ctx).Order("sort ASC, id ASC").Find(&categories).Error; err != nil {
		return nil, errs.Internal("查询分类失败").WithCause(err)
	}
	type row struct {
		CategoryID int64
		Num        int64
	}
	var rows []row
	if err := s.DB.WithContext(ctx).Model(&model.Article{}).
		Select("category_id AS category_id, COUNT(*) AS num").
		Group("category_id").Scan(&rows).Error; err != nil {
		return nil, errs.Internal("统计文章失败").WithCause(err)
	}
	byCategory := make(map[int64]int64, len(rows))
	for _, r := range rows {
		byCategory[r.CategoryID] += r.Num
	}
	items := make([]ArticleCategoryItem, 0, len(categories))
	for _, c := range categories {
		items = append(items, ArticleCategoryItem{ID: c.ID, Name: c.Name, Sort: c.Sort, Count: byCategory[c.ID], CreatedAt: c.CreatedAt})
	}
	return items, nil
}

func normalizeCategoryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errs.InvalidParam("分类名不能为空")
	}
	if len([]rune(name)) > maxCategoryNameLen {
		return "", errs.InvalidParam("分类名最长 64 个字符")
	}
	return name, nil
}

// assertCategoryNameFree 校验同名。用 LOWER 比较:MySQL 排序规则本身大小写不敏感,
// 而 SQLite 敏感,不降小写会出现"同一份数据两套行为"。
func (s *ArticleCategoryService) assertCategoryNameFree(ctx context.Context, name string, excludeID int64) error {
	q := s.DB.WithContext(ctx).Model(&model.ArticleCategory{}).Where("LOWER(name) = ?", strings.ToLower(name))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return errs.Internal("查询分类失败").WithCause(err)
	}
	if count > 0 {
		return errs.Conflict("同名分类已存在")
	}
	return nil
}

func (s *ArticleCategoryService) Create(ctx context.Context, name string, sort int) (*model.ArticleCategory, error) {
	name, err := normalizeCategoryName(name)
	if err != nil {
		return nil, err
	}
	if err := s.assertCategoryNameFree(ctx, name, 0); err != nil {
		return nil, err
	}
	category := &model.ArticleCategory{Name: name, Sort: sort}
	if err := s.DB.WithContext(ctx).Create(category).Error; err != nil {
		return nil, errs.Internal("创建分类失败").WithCause(err)
	}
	return category, nil
}

func (s *ArticleCategoryService) Update(ctx context.Context, id int64, name string, sort int) error {
	if id <= 0 {
		return errs.InvalidParam("分类不存在")
	}
	name, err := normalizeCategoryName(name)
	if err != nil {
		return err
	}
	var category model.ArticleCategory
	if err := s.DB.WithContext(ctx).First(&category, id).Error; err != nil {
		return errs.NotFound("分类不存在")
	}
	if err := s.assertCategoryNameFree(ctx, name, id); err != nil {
		return err
	}
	if err := s.DB.WithContext(ctx).Model(&model.ArticleCategory{ID: category.ID}).
		Updates(map[string]any{"name": name, "sort": sort}).Error; err != nil {
		return errs.Internal("更新分类失败").WithCause(err)
	}
	return nil
}

// Delete 删除空分类。分类下仍有文章时直接拒绝:删分类不该连带丢文章,
// 文章所属分类是内容资产的一部分,由用户先处理文章再删分类。
func (s *ArticleCategoryService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errs.InvalidParam("分类不存在")
	}
	var category model.ArticleCategory
	if err := s.DB.WithContext(ctx).First(&category, id).Error; err != nil {
		return errs.NotFound("分类不存在")
	}
	var used int64
	if err := s.DB.WithContext(ctx).Model(&model.Article{}).
		Where("category_id = ?", id).Count(&used).Error; err != nil {
		return errs.Internal("统计文章失败").WithCause(err)
	}
	if used > 0 {
		return errs.Conflict("分类下仍有文章,请先调整文章分类")
	}
	if err := s.DB.WithContext(ctx).Delete(&model.ArticleCategory{}, id).Error; err != nil {
		return errs.Internal("删除分类失败").WithCause(err)
	}
	return nil
}
