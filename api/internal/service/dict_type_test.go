// 字典服务测试:类型键唯一、内置保护、子项同类型内值唯一、按键读取的停用过滤。
package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/api/test/testutil"
)

func newDictTypeService(t *testing.T) (*DictTypeService, context.Context) {
	t.Helper()
	db := testutil.NewTestDB(t)
	return NewDictTypeService(db), context.Background()
}

// createTypeErr / createItemErr 把双返回值收敛成 error,便于与 assertHTTPStatus 连用。
func createTypeErr(s *DictTypeService, ctx context.Context, in DictTypeSaveInput) error {
	_, err := s.Create(ctx, in)
	return err
}

func createItemErr(s *DictTypeService, ctx context.Context, typeID int64, in DictItemSaveInput) error {
	_, err := s.CreateItem(ctx, typeID, in)
	return err
}

func TestDictTypeListOrderedWithCounts(t *testing.T) {
	svc, ctx := newDictTypeService(t)
	gender, err := svc.Create(ctx, DictTypeSaveInput{Name: "性别", Key: "gender"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, DictTypeSaveInput{Name: "是否", Key: "yes_no"}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.CreateItem(ctx, gender.ID, DictItemSaveInput{Label: "男", Value: "1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateItem(ctx, gender.ID, DictItemSaveInput{Label: "女", Value: "2"}); err != nil {
		t.Fatal(err)
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("查询字典类型失败: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("应返回 2 个类型: %+v", items)
	}
	if items[0].ItemCount != 2 || items[1].ItemCount != 0 {
		t.Fatalf("子项计数不符: %+v", items)
	}
}

func TestDictTypeValidateAndDuplicateKey(t *testing.T) {
	svc, ctx := newDictTypeService(t)

	typ, err := svc.Create(ctx, DictTypeSaveInput{Name: "  性别  ", Key: " gender "})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if typ.Name != "性别" || typ.DictKey != "gender" {
		t.Fatalf("首尾空白应被去掉: %+v", typ)
	}

	assertHTTPStatus(t, createTypeErr(svc, ctx, DictTypeSaveInput{Name: "  ", Key: "a.b"}), 400, "空字典名")
	assertHTTPStatus(t, createTypeErr(svc, ctx, DictTypeSaveInput{Name: "x", Key: "Gender"}), 400, "大写键名")
	assertHTTPStatus(t, createTypeErr(svc, ctx, DictTypeSaveInput{Name: "x", Key: "含中文"}), 400, "非 ASCII 键名")
	assertHTTPStatus(t, createTypeErr(svc, ctx, DictTypeSaveInput{Name: "x", Key: "gender"}), 409, "同键创建")
	assertHTTPStatus(t, createTypeErr(svc, ctx, DictTypeSaveInput{Name: "x", Key: " gender "}), 409, "去空白后同键创建")

	// 更新:同键放行(excludeID 生效),撞他人键被拦
	if err := svc.Update(ctx, typ.ID, DictTypeSaveInput{Name: "性别", Key: "gender"}); err != nil {
		t.Fatalf("保持原键应成功: %v", err)
	}
	other, err := svc.Create(ctx, DictTypeSaveInput{Name: "是否", Key: "yes_no"})
	if err != nil {
		t.Fatal(err)
	}
	_ = other
	assertHTTPStatus(t, svc.Update(ctx, typ.ID, DictTypeSaveInput{Name: "性别", Key: "yes_no"}), 409, "改成已有键名")
}

// TestDictTypeBuiltinProtection 内置字典可维护子项、可改名,不可删、不可改键名。
func TestDictTypeBuiltinProtection(t *testing.T) {
	svc, ctx := newDictTypeService(t)
	builtin := &model.SysDictType{Name: "性别", DictKey: "gender", Builtin: true}
	if err := svc.DB.Create(builtin).Error; err != nil {
		t.Fatal(err)
	}

	// 键保持不变时可正常更新
	if err := svc.Update(ctx, builtin.ID, DictTypeSaveInput{Name: "人员性别", Key: "gender"}); err != nil {
		t.Fatalf("内置字典应可改名: %v", err)
	}
	assertHTTPStatus(t, svc.Update(ctx, builtin.ID, DictTypeSaveInput{Name: "x", Key: "other.key"}), 409, "内置字典改键名")
	assertHTTPStatus(t, svc.Delete(ctx, builtin.ID), 409, "内置字典删除")

	// 内置字典的子项正常维护(内置限制的是类型本身,不是运营数据)
	if _, err := svc.CreateItem(ctx, builtin.ID, DictItemSaveInput{Label: "男", Value: "1"}); err != nil {
		t.Fatalf("内置字典应可加子项: %v", err)
	}
}

func TestDictTypeDeleteBlockedWithItems(t *testing.T) {
	svc, ctx := newDictTypeService(t)
	typ, err := svc.Create(ctx, DictTypeSaveInput{Name: "性别", Key: "gender"})
	if err != nil {
		t.Fatal(err)
	}
	item, err := svc.CreateItem(ctx, typ.ID, DictItemSaveInput{Label: "男", Value: "1"})
	if err != nil {
		t.Fatal(err)
	}

	assertHTTPStatus(t, svc.Delete(ctx, typ.ID), 409, "仍有子项时删除类型")
	if err := svc.DeleteItem(ctx, item.ID); err != nil {
		t.Fatalf("删除子项失败: %v", err)
	}
	if err := svc.Delete(ctx, typ.ID); err != nil {
		t.Fatalf("清空子项后应可删除类型: %v", err)
	}
	assertHTTPStatus(t, svc.Delete(ctx, 999), 404, "删不存在的类型")
	assertHTTPStatus(t, svc.Delete(ctx, 0), 400, "非法 ID")
}

func TestDictItemCRUD(t *testing.T) {
	svc, ctx := newDictTypeService(t)
	gender, err := svc.Create(ctx, DictTypeSaveInput{Name: "性别", Key: "gender"})
	if err != nil {
		t.Fatal(err)
	}
	yn, err := svc.Create(ctx, DictTypeSaveInput{Name: "是否", Key: "yes_no"})
	if err != nil {
		t.Fatal(err)
	}

	item, err := svc.CreateItem(ctx, gender.ID, DictItemSaveInput{Label: " 男 ", Value: " 1 ", Sort: 2, TagType: "primary"})
	if err != nil {
		t.Fatalf("创建子项失败: %v", err)
	}
	if item.Label != "男" || item.Value != "1" {
		t.Fatalf("首尾空白应被去掉: %+v", item)
	}

	// 描述与标签类型:非法配色被拦,合法值透传
	assertHTTPStatus(t, createItemErr(svc, ctx, gender.ID, DictItemSaveInput{Label: "x", Value: "9", TagType: "red"}), 400, "非法标签类型")
	assertHTTPStatus(t, createItemErr(svc, ctx, gender.ID, DictItemSaveInput{Label: "x", Value: "9", Description: strings.Repeat("述", 256)}), 400, "超长描述")
	full, err := svc.CreateItem(ctx, gender.ID, DictItemSaveInput{
		Label: "双性别", Value: "3", Description: " 性别未知 ", TagType: "info",
	})
	if err != nil {
		t.Fatal(err)
	}
	if full.Description != "性别未知" || full.TagType != "info" {
		t.Fatalf("描述/标签类型未按预期保存: %+v", full)
	}

	// 同类型内同值被拦;不同类型同值合法
	assertHTTPStatus(t, createItemErr(svc, ctx, gender.ID, DictItemSaveInput{Label: "男性", Value: "1"}), 409, "同类型同值")
	if _, err := svc.CreateItem(ctx, yn.ID, DictItemSaveInput{Label: "男", Value: "1"}); err != nil {
		t.Fatalf("不同类型同值应成功: %v", err)
	}

	// status=0 视为未传,默认启用
	hidden, err := svc.CreateItem(ctx, gender.ID, DictItemSaveInput{Label: "未知", Value: "0"})
	if err != nil {
		t.Fatal(err)
	}
	if hidden.Status != model.DictItemStatusEnabled {
		t.Fatalf("未传状态应默认启用: %+v", hidden)
	}

	// 更新:子项显示名/描述/存储值/排序/配色;同值查重排除自身
	if err := svc.UpdateItem(ctx, item.ID, DictItemSaveInput{Label: "男性", Value: "1", Sort: 5, TagType: "success", Status: model.DictItemStatusDisabled}); err != nil {
		t.Fatalf("更新子项失败: %v", err)
	}
	var updated model.SysDictItem
	if err := svc.DB.First(&updated, item.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Label != "男性" || updated.Sort != 5 || updated.Status != model.DictItemStatusDisabled || updated.TagType != "success" {
		t.Fatalf("更新未生效: %+v", updated)
	}
	assertHTTPStatus(t, svc.UpdateItem(ctx, item.ID, DictItemSaveInput{Label: "x", Value: "0"}), 409, "更新撞他人值")
	assertHTTPStatus(t, svc.UpdateItem(ctx, 999, DictItemSaveInput{Label: "x", Value: "9"}), 404, "改不存在的子项")

	// 类型下子项列表按 sort 升序(男 sort=5 已停用仍显示,管理端可见)
	items, err := svc.ListItems(ctx, gender.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("应返回 3 个子项: %+v", items)
	}
}

func TestDictEnabledByKey(t *testing.T) {
	svc, ctx := newDictTypeService(t)
	gender, err := svc.Create(ctx, DictTypeSaveInput{Name: "性别", Key: "gender"})
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []DictItemSaveInput{
		{Label: "未知", Value: "0", Sort: 3},
		{Label: "男", Value: "1", Sort: 1, TagType: "primary"},
		{Label: "女", Value: "2", Sort: 2, Status: model.DictItemStatusDisabled},
	}
	created := make(map[string]int64, len(fixtures))
	for _, in := range fixtures {
		item, err := svc.CreateItem(ctx, gender.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		created[in.Value] = item.ID
	}

	// 只返回启用项,按 sort 升序;停用项(女)不下发
	options, err := svc.EnabledByKey(ctx, "gender")
	if err != nil {
		t.Fatalf("按键读取失败: %v", err)
	}
	if len(options) != 2 || options[0].Value != "1" || options[1].Value != "0" {
		t.Fatalf("应只含启用的 男/未知 且按 sort 排序: %+v", options)
	}
	if options[0].TagType != "primary" {
		t.Fatalf("业务读取应携带标签配色: %+v", options)
	}

	// 重新启用后立即下发
	if err := svc.UpdateItem(ctx, created["2"], DictItemSaveInput{Label: "女", Value: "2", Status: model.DictItemStatusEnabled}); err != nil {
		t.Fatal(err)
	}
	options, err = svc.EnabledByKey(ctx, "gender")
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 3 {
		t.Fatalf("启用后应下发 3 项: %+v", options)
	}

	// 不存在的键返回空列表而非报错:业务页面对未配置字典不应报 404
	options, err = svc.EnabledByKey(ctx, "not-exist")
	if err != nil {
		t.Fatalf("不存在的键应返回空列表: %v", err)
	}
	if len(options) != 0 {
		t.Fatalf("不存在的键应返回空列表: %+v", options)
	}
	assertHTTPStatus(t, func() error { _, err := svc.EnabledByKey(ctx, "  "); return err }(), 400, "空键名")

	// 空格与超长 value 校验
	assertHTTPStatus(t, func() error {
		_, err := svc.CreateItem(ctx, gender.ID, DictItemSaveInput{Label: "x", Value: strings.Repeat("v", 129)})
		return err
	}(), 400, "超长存储值")
	assertHTTPStatus(t, func() error {
		_, err := svc.CreateItem(ctx, gender.ID, DictItemSaveInput{Label: "x", Value: " "})
		return err
	}(), 400, "空白存储值")
}
