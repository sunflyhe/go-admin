// 文章分类服务测试:命名唯一性、排序与删除边界。
// 分类是文章的归属枚举,非空分类必须拒绝删除,否则删分类会连带孤立一批文章。
package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/test/testutil"
)

func newArticleCategoryService(t *testing.T) (*ArticleCategoryService, *ArticleService, context.Context) {
	t.Helper()
	db := testutil.NewTestDB(t)
	return NewArticleCategoryService(db), NewArticleService(db), context.Background()
}

func TestArticleCategoryListOrderedWithCounts(t *testing.T) {
	categorySvc, _, ctx := newArticleCategoryService(t)

	if _, err := categorySvc.Create(ctx, "公司动态", 10); err != nil {
		t.Fatal(err)
	}
	if _, err := categorySvc.Create(ctx, "产品发布", 1); err != nil {
		t.Fatal(err)
	}

	items, err := categorySvc.List(ctx)
	if err != nil {
		t.Fatalf("查询分类失败: %v", err)
	}
	// 排序按 sort 升序,同 sort 按 id:先建的不一定在前
	if len(items) != 2 || items[0].Name != "产品发布" || items[1].Name != "公司动态" {
		t.Fatalf("分类应按 sort 排序: %+v", items)
	}
	if items[0].Count != 0 || items[1].Count != 0 {
		t.Fatalf("空分类计数应为 0: %+v", items)
	}

	// 建两篇文章挂在"产品发布"下,计数应跟上(草稿也算)
	categoryID := items[0].ID
	for _, title := range []string{"第一篇", "第二篇"} {
		if err := categorySvc.DB.Create(&model.Article{
			CategoryID: categoryID, Title: title, Content: "<p>x</p>",
			Status: model.ArticleStatusDraft, AuthorID: 1, Author: "admin",
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	items, err = categorySvc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Count != 2 || items[1].Count != 0 {
		t.Fatalf("分类计数不符: %+v", items)
	}
}

func TestArticleCategoryCreateNormalizesName(t *testing.T) {
	categorySvc, _, ctx := newArticleCategoryService(t)

	category, err := categorySvc.Create(ctx, "  公司动态  ", 0)
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if category.Name != "公司动态" {
		t.Fatalf("首尾空白应被去掉: %q", category.Name)
	}

	_, err = categorySvc.Create(ctx, "   ", 0)
	assertHTTPStatus(t, err, 400, "纯空白分类名")
	_, err = categorySvc.Create(ctx, strings.Repeat("长", 65), 0)
	assertHTTPStatus(t, err, 400, "超长分类名")
	if _, err := categorySvc.Create(ctx, strings.Repeat("长", 64), 0); err != nil {
		t.Fatalf("64 字符应是上界而不是越界: %v", err)
	}
}

// TestArticleCategoryRejectsDuplicateName 同名必须大小写不敏感:
// MySQL 排序规则本身不区分,测试库 SQLite 区分,不显式 LOWER 就会出现同一份数据两套行为。
func TestArticleCategoryRejectsDuplicateName(t *testing.T) {
	categorySvc, _, ctx := newArticleCategoryService(t)
	if _, err := categorySvc.Create(ctx, "News", 0); err != nil {
		t.Fatal(err)
	}

	_, err := categorySvc.Create(ctx, "news", 0)
	assertHTTPStatus(t, err, 409, "仅大小写不同的同名分类")

	_, err = categorySvc.Create(ctx, "  News ", 0)
	assertHTTPStatus(t, err, 409, "去空白后同名的分类")

	if _, err := categorySvc.Create(ctx, "公告", 0); err != nil {
		t.Fatalf("不同名分类应能创建: %v", err)
	}
}

func TestArticleCategoryUpdate(t *testing.T) {
	categorySvc, _, ctx := newArticleCategoryService(t)
	category, err := categorySvc.Create(ctx, "旧名", 3)
	if err != nil {
		t.Fatal(err)
	}
	other, err := categorySvc.Create(ctx, "其他", 4)
	if err != nil {
		t.Fatal(err)
	}

	// 改成自己当前的名字不得被判成同名(excludeID 生效)
	if err := categorySvc.Update(ctx, category.ID, "旧名", 3); err != nil {
		t.Fatalf("保持原名应成功: %v", err)
	}
	if err := categorySvc.Update(ctx, category.ID, " 新名 ", 1); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	var renamed model.ArticleCategory
	if err := categorySvc.DB.First(&renamed, category.ID).Error; err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "新名" || renamed.Sort != 1 {
		t.Fatalf("更新未生效: %+v", renamed)
	}

	assertHTTPStatus(t, categorySvc.Update(ctx, other.ID, "新名", 0), 409, "改成已有同名")
	assertHTTPStatus(t, categorySvc.Update(ctx, 999, "x", 0), 404, "改不存在的分类")
	assertHTTPStatus(t, categorySvc.Update(ctx, other.ID, "  ", 0), 400, "空分类名")
}

func TestArticleCategoryDelete(t *testing.T) {
	categorySvc, articleSvc, ctx := newArticleCategoryService(t)
	actor := Actor{ID: 1, Username: "admin"}

	full, err := categorySvc.Create(ctx, "有文章", 0)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := categorySvc.Create(ctx, "空分类", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := articleSvc.Create(ctx, actor, ArticleSaveInput{
		CategoryID: full.ID, Title: "文章", Content: "<p>正文</p>", Status: model.ArticleStatusDraft,
	}); err != nil {
		t.Fatal(err)
	}

	assertHTTPStatus(t, categorySvc.Delete(ctx, full.ID), 409, "非空分类")
	var stillThere model.ArticleCategory
	if err := categorySvc.DB.First(&stillThere, full.ID).Error; err != nil {
		t.Fatalf("被拒的删除不得动数据: %v", err)
	}

	if err := categorySvc.Delete(ctx, empty.ID); err != nil {
		t.Fatalf("空分类应可删除: %v", err)
	}
	var count int64
	if err := categorySvc.DB.Model(&model.ArticleCategory{}).Where("id = ?", empty.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("空分类应已被删除")
	}

	assertHTTPStatus(t, categorySvc.Delete(ctx, 999), 404, "不存在的分类")
	assertHTTPStatus(t, categorySvc.Delete(ctx, 0), 400, "伪节点未分类不得被删")
}
