// 系统参数服务测试:键格式与唯一性、分页搜索与内置参数保护。
package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
	"github.com/hesunfly/hesunfly-admin-go/server/test/testutil"
)

func newConfigService(t *testing.T) (*ConfigService, context.Context) {
	t.Helper()
	db := testutil.NewTestDB(t)
	return NewConfigService(db), context.Background()
}

// createConfigErr 把 Create 的双返回值收敛成 error,便于与 assertHTTPStatus 连用。
func createConfigErr(s *ConfigService, ctx context.Context, in ConfigSaveInput) error {
	_, err := s.Create(ctx, in)
	return err
}

func TestConfigListPaginationAndKeyword(t *testing.T) {
	configSvc, ctx := newConfigService(t)
	for _, in := range []ConfigSaveInput{
		{Name: "站点名称", Key: "site.name", Value: "Hesunfly Admin"},
		{Name: "登录页欢迎语", Key: "site.welcome", Value: "欢迎使用"},
		{Name: "上传大小上限", Key: "upload.max-size", Value: "10"},
	} {
		if _, err := configSvc.Create(ctx, in); err != nil {
			t.Fatal(err)
		}
	}

	result, err := configSvc.List(ctx, &ConfigListInput{})
	if err != nil {
		t.Fatalf("查询参数失败: %v", err)
	}
	if result.Total != 3 || len(result.List.([]model.SysConfig)) != 3 {
		t.Fatalf("应返回全部 3 条: %+v", result)
	}

	// 关键字同时匹配参数名与键名:"welcome" 命中键,"欢迎" 命中名,都只该有 1 条
	for _, kw := range []string{"welcome", "欢迎"} {
		result, err = configSvc.List(ctx, &ConfigListInput{Keyword: kw})
		if err != nil {
			t.Fatal(err)
		}
		list := result.List.([]model.SysConfig)
		if result.Total != 1 || len(list) != 1 || list[0].ConfigKey != "site.welcome" {
			t.Fatalf("关键字 %q 应只命中 site.welcome: %+v", kw, result)
		}
	}

	// 分页:pageSize=2 时第一页 2 条、第二页 1 条,total 始终为 3
	result, err = configSvc.List(ctx, &ConfigListInput{Query: page.Query{Page: 1, PageSize: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 3 || len(result.List.([]model.SysConfig)) != 2 {
		t.Fatalf("第一页应 2 条且 total 为 3: %+v", result)
	}
	result, err = configSvc.List(ctx, &ConfigListInput{Query: page.Query{Page: 2, PageSize: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 3 || len(result.List.([]model.SysConfig)) != 1 {
		t.Fatalf("第二页应只剩 1 条: %+v", result)
	}
}

func TestConfigCreateValidatesInput(t *testing.T) {
	configSvc, ctx := newConfigService(t)

	cfg, err := configSvc.Create(ctx, ConfigSaveInput{Name: "  站点名称  ", Key: " site.name ", Value: "x"})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if cfg.Name != "站点名称" || cfg.ConfigKey != "site.name" {
		t.Fatalf("首尾空白应被去掉: %+v", cfg)
	}

	// 空值合法:参数值允许为空(如"不展示"语义)
	if _, err := configSvc.Create(ctx, ConfigSaveInput{Name: "页脚版权", Key: "site.copyright"}); err != nil {
		t.Fatalf("空参数值应可创建: %v", err)
	}

	assertHTTPStatus(t, createConfigErr(configSvc, ctx, ConfigSaveInput{Name: "  ", Key: "a.b"}), 400, "空参数名")
	assertHTTPStatus(t, createConfigErr(configSvc, ctx, ConfigSaveInput{Name: "x", Key: "Site.Name"}), 400, "大写键名")
	assertHTTPStatus(t, createConfigErr(configSvc, ctx, ConfigSaveInput{Name: "x", Key: "1abc"}), 400, "数字开头的键名")
	assertHTTPStatus(t, createConfigErr(configSvc, ctx, ConfigSaveInput{Name: "x", Key: "含中文"}), 400, "非 ASCII 键名")
	assertHTTPStatus(t, createConfigErr(configSvc, ctx, ConfigSaveInput{Name: "x", Key: "a b"}), 400, "含空格的键名")
	assertHTTPStatus(t, createConfigErr(configSvc, ctx, ConfigSaveInput{Name: "x", Key: "a.b", Value: strings.Repeat("值", 513)}), 400, "超长参数值")
	if _, err := configSvc.Create(ctx, ConfigSaveInput{Name: "x", Key: "a." + strings.Repeat("b", 62)}); err != nil {
		t.Fatalf("64 字符键应是上界而不是越界: %v", err)
	}
}

// TestConfigRejectsDuplicateKey 键名唯一:大写等非法键会被格式校验先拦成 400,
// 合法键的重复创建必须 409;唯一性比较用 LOWER 兜底 MySQL/SQLite 排序规则差异。
func TestConfigRejectsDuplicateKey(t *testing.T) {
	configSvc, ctx := newConfigService(t)
	if _, err := configSvc.Create(ctx, ConfigSaveInput{Name: "站点名称", Key: "site.name"}); err != nil {
		t.Fatal(err)
	}

	// 键的格式校验已把大写拦在 400,能到这里撞唯一键的必然是合法小写键;
	// 去空白后同键也要被拦(LOWER 比较兜底 SQLite/MySQL 排序规则差异)
	assertHTTPStatus(t, createConfigErr(configSvc, ctx, ConfigSaveInput{Name: "另一个", Key: "site.name"}), 409, "同键创建")
	assertHTTPStatus(t, createConfigErr(configSvc, ctx, ConfigSaveInput{Name: "另一个", Key: " site.name "}), 409, "去空白后同键创建")

	// 更新时改成别人的键也要被拦;改成自己当前的键不得误判(excludeID 生效)
	cfg, err := configSvc.Create(ctx, ConfigSaveInput{Name: "欢迎语", Key: "site.welcome"})
	if err != nil {
		t.Fatal(err)
	}
	assertHTTPStatus(t, configSvc.Update(ctx, cfg.ID, ConfigSaveInput{Name: "欢迎语", Key: "site.name"}), 409, "改成已有键名")
	if err := configSvc.Update(ctx, cfg.ID, ConfigSaveInput{Name: "欢迎语改", Key: "site.welcome", Value: "v2"}); err != nil {
		t.Fatalf("保持原键应成功: %v", err)
	}
}

func TestConfigUpdate(t *testing.T) {
	configSvc, ctx := newConfigService(t)
	cfg, err := configSvc.Create(ctx, ConfigSaveInput{Name: "旧名", Key: "old.key", Value: "v1", Remark: "备注"})
	if err != nil {
		t.Fatal(err)
	}

	if err := configSvc.Update(ctx, cfg.ID, ConfigSaveInput{Name: "新名", Key: "old.key", Value: "v2", Remark: "新备注"}); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	var updated model.SysConfig
	if err := configSvc.DB.First(&updated, cfg.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Name != "新名" || updated.Value != "v2" || updated.Remark != "新备注" {
		t.Fatalf("更新未生效: %+v", updated)
	}

	assertHTTPStatus(t, configSvc.Update(ctx, 999, ConfigSaveInput{Name: "x", Key: "a.b"}), 404, "改不存在的参数")
	assertHTTPStatus(t, configSvc.Update(ctx, cfg.ID, ConfigSaveInput{Name: "  ", Key: "a.b"}), 400, "空参数名")
	assertHTTPStatus(t, configSvc.Update(ctx, 0, ConfigSaveInput{Name: "x", Key: "a.b"}), 400, "非法 ID")
}

// TestConfigBuiltinProtection 内置参数可改值,不可删、不可改键名;
// 保护必须落在服务层,前端隐藏按钮不是安全措施。
func TestConfigBuiltinProtection(t *testing.T) {
	configSvc, ctx := newConfigService(t)
	builtin := &model.SysConfig{Name: "上传大小上限", ConfigKey: "upload.max-size", Value: "10", Builtin: true}
	if err := configSvc.DB.Create(builtin).Error; err != nil {
		t.Fatal(err)
	}

	// 可改值(键保持不变)
	if err := configSvc.Update(ctx, builtin.ID, ConfigSaveInput{Name: "上传大小上限", Key: "upload.max-size", Value: "20"}); err != nil {
		t.Fatalf("内置参数应可改值: %v", err)
	}
	var after model.SysConfig
	if err := configSvc.DB.First(&after, builtin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Value != "20" {
		t.Fatalf("改值未生效: %+v", after)
	}

	assertHTTPStatus(t, configSvc.Update(ctx, builtin.ID, ConfigSaveInput{Name: "x", Key: "other.key"}), 409, "内置参数改键名")
	assertHTTPStatus(t, configSvc.Delete(ctx, builtin.ID), 409, "内置参数删除")

	// 普通参数不受限:可改键、可删
	plain, err := configSvc.Create(ctx, ConfigSaveInput{Name: "临时参数", Key: "tmp.flag"})
	if err != nil {
		t.Fatal(err)
	}
	if err := configSvc.Update(ctx, plain.ID, ConfigSaveInput{Name: "临时参数", Key: "tmp.flag2"}); err != nil {
		t.Fatalf("普通参数应可改键: %v", err)
	}
	if err := configSvc.Delete(ctx, plain.ID); err != nil {
		t.Fatalf("普通参数应可删除: %v", err)
	}

	assertHTTPStatus(t, configSvc.Delete(ctx, 999), 404, "删不存在的参数")
	assertHTTPStatus(t, configSvc.Delete(ctx, 0), 400, "非法 ID")
}
