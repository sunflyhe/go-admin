// 文件分组服务测试:左栏目录的命名唯一性、计数与删除边界。
// 分组是"整理"手段而非文件生命周期,非空分组必须拒绝删除,否则删目录会连带删文件。
package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
)

func newFileGroupService(t *testing.T) (*FileGroupService, *FileService, context.Context) {
	t.Helper()
	fileSvc, ctx := newFileService(t)
	return NewFileGroupService(fileSvc.DB), fileSvc, ctx
}

func TestFileGroupListCounts(t *testing.T) {
	groupSvc, fileSvc, ctx := newFileGroupService(t)
	a, err := groupSvc.Create(ctx, "素材")
	if err != nil {
		t.Fatal(err)
	}
	b, err := groupSvc.Create(ctx, "合同")
	if err != nil {
		t.Fatal(err)
	}
	seedFiles(t, fileSvc,
		fileSeed{ext: ".png", mime: "image/png", groupID: a.ID},
		fileSeed{ext: ".jpg", mime: "image/jpeg", groupID: a.ID},
		fileSeed{ext: ".pdf", mime: "application/pdf", groupID: b.ID},
		fileSeed{ext: ".txt", mime: "text/plain"}, // 未分组
	)

	tree, err := groupSvc.List(ctx, "")
	if err != nil {
		t.Fatalf("查询分组失败: %v", err)
	}
	if tree.Total != 4 || tree.Unfiled != 1 {
		t.Fatalf("总数/未分组不符: total=%d unfiled=%d", tree.Total, tree.Unfiled)
	}
	// 创建顺序即左栏顺序(sort 递增),不能因 map 遍历而乱序
	if len(tree.Groups) != 2 || tree.Groups[0].Name != "素材" || tree.Groups[1].Name != "合同" {
		t.Fatalf("分组顺序应按 sort: %+v", tree.Groups)
	}
	if tree.Groups[0].Count != 2 || tree.Groups[1].Count != 1 {
		t.Fatalf("分组计数不符: %+v", tree.Groups)
	}

	// 空分组也要出现在列表里(计数为 0),否则新建后左栏看不到它
	empty, err := groupSvc.Create(ctx, "空组")
	if err != nil {
		t.Fatal(err)
	}
	tree, err = groupSvc.List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Groups) != 3 || tree.Groups[2].ID != empty.ID || tree.Groups[2].Count != 0 {
		t.Fatalf("空分组应出现在列表且计数为 0: %+v", tree.Groups)
	}
}

// TestFileGroupListCountsScopedByCategory 钉住层级:类型标签是最高层导航,分组在其之下。
// 切到"图片"时左栏数字只能是图片数,否则点进一个显示 3 的分组却只看到 2 个文件。
func TestFileGroupListCountsScopedByCategory(t *testing.T) {
	groupSvc, fileSvc, ctx := newFileGroupService(t)
	group, err := groupSvc.Create(ctx, "素材")
	if err != nil {
		t.Fatal(err)
	}
	seedFiles(t, fileSvc,
		fileSeed{ext: ".png", mime: "image/png", groupID: group.ID},
		fileSeed{ext: ".jpg", mime: "image/jpeg", groupID: group.ID},
		fileSeed{ext: ".mp4", mime: "video/mp4", groupID: group.ID},
		fileSeed{ext: ".pdf", mime: "application/pdf"},
	)

	counts := func(category string) (inGroup, unfiled, total int64) {
		t.Helper()
		tree, err := groupSvc.List(ctx, category)
		if err != nil {
			t.Fatalf("%q 查询失败: %v", category, err)
		}
		// 计数为 0 的分组也必须还在列表里,否则切个标签分组就凭空消失
		if len(tree.Groups) != 1 || tree.Groups[0].ID != group.ID {
			t.Fatalf("%q 下分组列表异常: %+v", category, tree.Groups)
		}
		return tree.Groups[0].Count, tree.Unfiled, tree.Total
	}

	if g, u, tt := counts(""); g != 3 || u != 1 || tt != 4 {
		t.Fatalf("全部标签应为 组3/未分组1/总4,实际 %d/%d/%d", g, u, tt)
	}
	if g, u, tt := counts("image"); g != 2 || u != 0 || tt != 2 {
		t.Fatalf("图片标签应为 组2/未分组0/总2,实际 %d/%d/%d", g, u, tt)
	}
	if g, u, tt := counts("video"); g != 1 || u != 0 || tt != 1 {
		t.Fatalf("视频标签应为 组1/未分组0/总1,实际 %d/%d/%d", g, u, tt)
	}
	if g, u, tt := counts("file"); g != 0 || u != 1 || tt != 1 {
		t.Fatalf("文件标签应为 组0/未分组1/总1,实际 %d/%d/%d", g, u, tt)
	}

	_, err = groupSvc.List(ctx, "document")
	assertHTTPStatus(t, err, http.StatusBadRequest, "未知标签")
}

func TestFileGroupCreateNormalizesName(t *testing.T) {
	groupSvc, _, ctx := newFileGroupService(t)

	group, err := groupSvc.Create(ctx, "  素材 A  ")
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if group.Name != "素材 A" {
		t.Fatalf("首尾空白应被去掉: %q", group.Name)
	}

	_, err = groupSvc.Create(ctx, "   ")
	assertHTTPStatus(t, err, http.StatusBadRequest, "纯空白分组名")
	_, err = groupSvc.Create(ctx, strings.Repeat("长", 65))
	assertHTTPStatus(t, err, http.StatusBadRequest, "超长分组名")
	if _, err := groupSvc.Create(ctx, strings.Repeat("长", 64)); err != nil {
		t.Fatalf("64 字符应是上界而不是越界: %v", err)
	}
}

// TestFileGroupCreateRejectsDuplicateName 同名必须大小写不敏感:
// MySQL 排序规则本身不区分,测试库 SQLite 区分,不显式 LOWER 就会出现同一份数据两套行为。
func TestFileGroupCreateRejectsDuplicateName(t *testing.T) {
	groupSvc, _, ctx := newFileGroupService(t)
	if _, err := groupSvc.Create(ctx, "Logo"); err != nil {
		t.Fatal(err)
	}

	_, err := groupSvc.Create(ctx, "logo")
	assertHTTPStatus(t, err, http.StatusConflict, "仅大小写不同的同名分组")

	// 去空白后等价,避免 "素材" 与 " 素材 " 两组并存
	_, err = groupSvc.Create(ctx, "  Logo ")
	assertHTTPStatus(t, err, http.StatusConflict, "去空白后同名的分组")

	if _, err := groupSvc.Create(ctx, "素材"); err != nil {
		t.Fatalf("不同名分组应能创建: %v", err)
	}
}

func TestFileGroupUpdateRenames(t *testing.T) {
	groupSvc, _, ctx := newFileGroupService(t)
	group, err := groupSvc.Create(ctx, "旧名")
	if err != nil {
		t.Fatal(err)
	}
	other, err := groupSvc.Create(ctx, "其他组")
	if err != nil {
		t.Fatal(err)
	}

	// 改成自己当前的名字不得被判成同名(excludeID 生效)
	if err := groupSvc.Update(ctx, group.ID, "旧名"); err != nil {
		t.Fatalf("保持原名应成功: %v", err)
	}
	if err := groupSvc.Update(ctx, group.ID, " 新名 "); err != nil {
		t.Fatalf("重命名失败: %v", err)
	}
	var renamed model.SysFileGroup
	if err := groupSvc.DB.First(&renamed, group.ID).Error; err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "新名" {
		t.Fatalf("重命名未生效: %q", renamed.Name)
	}
	// sort 不能被重命名打乱(注意:GORM 会复用结构体里的主键,必须换新变量查另一行)
	var otherRow model.SysFileGroup
	if err := groupSvc.DB.First(&otherRow, other.ID).Error; err != nil {
		t.Fatal(err)
	}
	if otherRow.Sort <= renamed.Sort {
		t.Fatalf("重命名后顺序应保持创建序: other=%d renamed=%d", otherRow.Sort, renamed.Sort)
	}

	assertHTTPStatus(t, groupSvc.Update(ctx, other.ID, "新名"), http.StatusConflict, "改成已有同名")
	assertHTTPStatus(t, groupSvc.Update(ctx, 999, "x"), http.StatusNotFound, "改不存在的分组")
	assertHTTPStatus(t, groupSvc.Update(ctx, other.ID, "  "), http.StatusBadRequest, "空分组名")
}

func TestFileGroupDelete(t *testing.T) {
	groupSvc, fileSvc, ctx := newFileGroupService(t)
	full, err := groupSvc.Create(ctx, "有文件")
	if err != nil {
		t.Fatal(err)
	}
	empty, err := groupSvc.Create(ctx, "空组")
	if err != nil {
		t.Fatal(err)
	}
	seedFiles(t, fileSvc, fileSeed{ext: ".png", mime: "image/png", groupID: full.ID})

	assertHTTPStatus(t, groupSvc.Delete(ctx, full.ID), http.StatusConflict, "非空分组")
	var stillThere model.SysFileGroup
	if err := groupSvc.DB.First(&stillThere, full.ID).Error; err != nil {
		t.Fatalf("被拒的删除不得动数据: %v", err)
	}

	if err := groupSvc.Delete(ctx, empty.ID); err != nil {
		t.Fatalf("空分组应可删除: %v", err)
	}
	var count int64
	if err := groupSvc.DB.Model(&model.SysFileGroup{}).Where("id = ?", empty.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("空分组应已被删除")
	}

	assertHTTPStatus(t, groupSvc.Delete(ctx, 999), http.StatusNotFound, "不存在的分组")
	assertHTTPStatus(t, groupSvc.Delete(ctx, 0), http.StatusBadRequest, "伪节点未分组不得被删")
}

// mustErr 让"应当失败"的调用可以写成一行断言。
func mustErr[T any](_ T, err error) error {
	return err
}
