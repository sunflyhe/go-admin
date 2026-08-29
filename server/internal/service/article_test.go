// 文章服务测试:入参校验、分类归属校验、列表过滤与发布时间语义。
package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
)

func newArticleService(t *testing.T) (*ArticleService, *ArticleCategoryService, context.Context) {
	t.Helper()
	categorySvc, articleSvc, ctx := newArticleCategoryService(t)
	return articleSvc, categorySvc, ctx
}

func createCategory(t *testing.T, svc *ArticleCategoryService, ctx context.Context, name string) int64 {
	t.Helper()
	category, err := svc.Create(ctx, name, 0)
	if err != nil {
		t.Fatal(err)
	}
	return category.ID
}

func TestArticleCreateRecordsAuthorAndPublishTime(t *testing.T) {
	articleSvc, categorySvc, ctx := newArticleService(t)
	categoryID := createCategory(t, categorySvc, ctx, "公司动态")
	actor := Actor{ID: 7, Username: "editor"}

	article, err := articleSvc.Create(ctx, actor, ArticleSaveInput{
		CategoryID: categoryID, Title: "  第一篇  ", Summary: "摘要",
		Content: "<p>正文</p>", Status: model.ArticleStatusPublished,
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if article.Title != "第一篇" {
		t.Fatalf("标题首尾空白应被去掉: %q", article.Title)
	}
	if article.Author != "editor" || article.AuthorID != 7 {
		t.Fatalf("应记录创建者快照: %+v", article)
	}
	if article.PublishedAt == nil {
		t.Fatal("创建即发布应记录发布时间")
	}

	// 草稿不记发布时间
	draft, err := articleSvc.Create(ctx, actor, ArticleSaveInput{
		Title: "草稿篇", Content: "<p>正文</p>", Status: model.ArticleStatusDraft,
	})
	if err != nil {
		t.Fatalf("未分类文章应可创建(0=未分类): %v", err)
	}
	if draft.PublishedAt != nil {
		t.Fatal("草稿不应有发布时间")
	}
	if draft.CategoryID != 0 {
		t.Fatalf("未传分类应为未分类: %d", draft.CategoryID)
	}
}

func TestArticleCreateValidation(t *testing.T) {
	articleSvc, categorySvc, ctx := newArticleService(t)
	actor := Actor{ID: 1, Username: "admin"}

	cases := []struct {
		label string
		input ArticleSaveInput
		code  int
	}{
		{"空标题", ArticleSaveInput{Title: "  ", Content: "<p>x</p>", Status: 1}, 400},
		{"超长标题", ArticleSaveInput{Title: strings.Repeat("长", 129), Content: "<p>x</p>", Status: 1}, 400},
		{"空正文", ArticleSaveInput{Title: "t", Content: "  ", Status: 1}, 400},
		{"超长摘要", ArticleSaveInput{Title: "t", Content: "<p>x</p>", Summary: strings.Repeat("长", 256), Status: 1}, 400},
		{"非法状态", ArticleSaveInput{Title: "t", Content: "<p>x</p>", Status: 9}, 400},
		{"不存在的分类", ArticleSaveInput{Title: "t", Content: "<p>x</p>", CategoryID: 999, Status: 1}, 404},
	}
	for _, c := range cases {
		_, err := articleSvc.Create(ctx, actor, c.input)
		assertHTTPStatus(t, err, c.code, c.label)
	}

	// 边界:128 字标题、255 字摘要应通过
	if _, err := articleSvc.Create(ctx, actor, ArticleSaveInput{
		Title: strings.Repeat("长", 128), Summary: strings.Repeat("长", 255),
		Content: "<p>x</p>", Status: model.ArticleStatusDraft,
	}); err != nil {
		t.Fatalf("边界长度应可通过: %v", err)
	}
	_ = categorySvc
}

// TestArticleUpdateKeepsAuthorAndFirstPublishTime 编辑不改署名;首发时间不随再次编辑变动;
// 从草稿转发布时补记发布时间。
func TestArticleUpdateKeepsAuthorAndFirstPublishTime(t *testing.T) {
	articleSvc, categorySvc, ctx := newArticleService(t)
	actor := Actor{ID: 7, Username: "editor"}

	article, err := articleSvc.Create(ctx, actor, ArticleSaveInput{
		Title: "初稿", Content: "<p>v1</p>", Status: model.ArticleStatusDraft,
	})
	if err != nil {
		t.Fatal(err)
	}
	otherID := createCategory(t, categorySvc, ctx, "新分类")

	if err := articleSvc.Update(ctx, article.ID, ArticleSaveInput{
		CategoryID: otherID, Title: "修订版", Content: "<p>v2</p>", Status: model.ArticleStatusDraft,
	}); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	detail, err := articleSvc.Get(ctx, article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Author != "editor" {
		t.Fatalf("编辑不得改署名: %+v", detail.ArticleItem)
	}
	if detail.CategoryID != otherID || detail.CategoryName != "新分类" || detail.Title != "修订版" {
		t.Fatalf("更新未生效: %+v", detail.ArticleItem)
	}
	if detail.PublishedAt != nil {
		t.Fatal("草稿阶段不应有发布时间")
	}

	// 转发布 → 记录首发时间;再改回草稿再发布,首发时间不覆盖
	if err := articleSvc.Update(ctx, article.ID, ArticleSaveInput{
		CategoryID: otherID, Title: "发布版", Content: "<p>v3</p>", Status: model.ArticleStatusPublished,
	}); err != nil {
		t.Fatal(err)
	}
	published, err := articleSvc.Get(ctx, article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if published.PublishedAt == nil {
		t.Fatal("转发布应补记发布时间")
	}
	firstAt := *published.PublishedAt

	if err := articleSvc.Update(ctx, article.ID, ArticleSaveInput{
		CategoryID: otherID, Title: "发布版", Content: "<p>v4</p>", Status: model.ArticleStatusDraft,
	}); err != nil {
		t.Fatal(err)
	}
	if err := articleSvc.Update(ctx, article.ID, ArticleSaveInput{
		CategoryID: otherID, Title: "发布版", Content: "<p>v5</p>", Status: model.ArticleStatusPublished,
	}); err != nil {
		t.Fatal(err)
	}
	republished, err := articleSvc.Get(ctx, article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if republished.PublishedAt == nil || !republished.PublishedAt.Equal(firstAt) {
		t.Fatalf("首发时间不得被覆盖: %v vs %v", republished.PublishedAt, firstAt)
	}

	assertHTTPStatus(t, articleSvc.Update(ctx, 999, ArticleSaveInput{
		Title: "t", Content: "<p>x</p>", Status: 1,
	}), 404, "改不存在的文章")
	assertHTTPStatus(t, articleSvc.Update(ctx, article.ID, ArticleSaveInput{
		Title: "t", Content: "<p>x</p>", CategoryID: 999, Status: 1,
	}), 404, "更新指向不存在的分类")
}

// TestArticleListFilters 钉住列表口径:不含正文、分类名回退"未分类"、筛选条件各自生效。
func TestArticleListFilters(t *testing.T) {
	articleSvc, categorySvc, ctx := newArticleService(t)
	actor := Actor{ID: 1, Username: "admin"}
	annID := createCategory(t, categorySvc, ctx, "公告")

	seed := []ArticleSaveInput{
		{CategoryID: annID, Title: "公告一", Summary: "s1", Content: "<p>c1</p>", Status: model.ArticleStatusPublished},
		{CategoryID: annID, Title: "公告二", Content: "<p>c2</p>", Status: model.ArticleStatusDraft},
		{Title: "通知一", Content: "<p>c3</p>", Status: model.ArticleStatusDraft},
	}
	for i := range seed {
		if _, err := articleSvc.Create(ctx, actor, seed[i]); err != nil {
			t.Fatal(err)
		}
	}

	all, err := articleSvc.List(ctx, &ArticleListInput{})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if all.Total != 3 || len(all.List.([]ArticleItem)) != 3 {
		t.Fatalf("总数应为 3: %+v", all)
	}
	// 新文章在前
	items := all.List.([]ArticleItem)
	if items[0].Title != "通知一" {
		t.Fatalf("列表应按 id 倒序: %+v", items)
	}
	// 列表不带正文,分类名正确回退
	if items[2].CategoryName != "公告" {
		t.Fatalf("应带出分类名: %+v", items[2])
	}
	if items[0].CategoryName != "未分类" {
		t.Fatalf("无分类文章应显示未分类: %+v", items[0])
	}
	if detail, err := articleSvc.Get(ctx, items[0].ID); err != nil || detail.Content != "<p>c3</p>" {
		t.Fatalf("详情应含正文: %v %+v", err, detail)
	}

	// 标题模糊
	byTitle, err := articleSvc.List(ctx, &ArticleListInput{Title: "公告"})
	if err != nil || byTitle.Total != 2 {
		t.Fatalf("标题筛选应为 2 条: %v %+v", err, byTitle)
	}
	// 指定分类
	byCategory, err := articleSvc.List(ctx, &ArticleListInput{CategoryID: &annID})
	if err != nil || byCategory.Total != 2 {
		t.Fatalf("分类筛选应为 2 条: %v %+v", err, byCategory)
	}
	unfiled := int64(0)
	byUnfiled, err := articleSvc.List(ctx, &ArticleListInput{CategoryID: &unfiled})
	if err != nil || byUnfiled.Total != 1 {
		t.Fatalf("未分类筛选应为 1 条: %v %+v", err, byUnfiled)
	}
	// 状态筛选
	byStatus, err := articleSvc.List(ctx, &ArticleListInput{Status: model.ArticleStatusPublished})
	if err != nil || byStatus.Total != 1 {
		t.Fatalf("已发布筛选应为 1 条: %v %+v", err, byStatus)
	}
	// 非法状态必须报错,而不是静默按全部
	_, err = articleSvc.List(ctx, &ArticleListInput{Status: 9})
	assertHTTPStatus(t, err, 400, "非法状态筛选")
	// 分页
	paged, err := articleSvc.List(ctx, &ArticleListInput{Query: page.Query{Page: 2, PageSize: 2}})
	if err != nil || paged.Total != 3 || len(paged.List.([]ArticleItem)) != 1 {
		t.Fatalf("第二页应只剩 1 条: %v %+v", err, paged)
	}
}

func TestArticleDelete(t *testing.T) {
	articleSvc, _, ctx := newArticleService(t)
	actor := Actor{ID: 1, Username: "admin"}

	article, err := articleSvc.Create(ctx, actor, ArticleSaveInput{
		Title: "待删", Content: "<p>x</p>", Status: model.ArticleStatusDraft,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := articleSvc.Delete(ctx, article.ID); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	_, err = articleSvc.Get(ctx, article.ID)
	assertHTTPStatus(t, err, 404, "删除后详情")
	assertHTTPStatus(t, articleSvc.Delete(ctx, 999), 404, "删不存在的文章")
	assertHTTPStatus(t, articleSvc.Delete(ctx, 0), 400, "伪 id")
}
