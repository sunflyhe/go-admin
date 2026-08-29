// 系统参数服务:键值参数的维护与内置参数保护。
// 参数是少量配置项,业务模块按 key 读取;列表支持按名称/键名模糊搜索。
package service

import (
	"context"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

const (
	maxConfigNameLen   = 64
	maxConfigKeyLen    = 64
	maxConfigValueLen  = 512
	maxConfigRemarkLen = 255
)

// 参数键约束为小写字母开头,只含小写字母、数字、点、中划线、下划线:
// 预留点号做分组前缀(如 site.name),同时保证键在 URL/日志里无需转义。
var configKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)

// ConfigListInput 列表查询输入。Keyword 同时匹配参数名与键名,空串表示不筛选。
type ConfigListInput struct {
	page.Query
	Keyword string
}

// ConfigSaveInput 创建/更新共用的业务输入,与 HTTP 框架无关。
type ConfigSaveInput struct {
	Name   string
	Key    string
	Value  string
	Remark string
}

type ConfigService struct {
	DB *gorm.DB
}

func NewConfigService(db *gorm.DB) *ConfigService {
	return &ConfigService{DB: db}
}

// List 分页列表,按更新时间倒序:最近调整过的参数排在最上面,方便核对刚改的值。
func (s *ConfigService) List(ctx context.Context, input *ConfigListInput) (*page.Result, error) {
	if err := input.Normalize(); err != nil {
		return nil, err
	}
	query := s.DB.WithContext(ctx).Model(&model.SysConfig{})
	if kw := strings.TrimSpace(input.Keyword); kw != "" {
		like := "%" + kw + "%"
		query = query.Where("name LIKE ? OR config_key LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errs.Internal("查询参数失败").WithCause(err)
	}
	var configs []model.SysConfig
	if err := query.Order("updated_at DESC, id DESC").
		Limit(input.PageSize).Offset(input.Offset()).Find(&configs).Error; err != nil {
		return nil, errs.Internal("查询参数失败").WithCause(err)
	}
	result := page.NewResult(configs, total, input.Query)
	return &result, nil
}

func normalizeConfigInput(input ConfigSaveInput) (ConfigSaveInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return input, errs.InvalidParam("参数名不能为空")
	}
	if len([]rune(input.Name)) > maxConfigNameLen {
		return input, errs.InvalidParam("参数名最长 64 个字符")
	}
	input.Key = strings.TrimSpace(input.Key)
	if !configKeyPattern.MatchString(input.Key) {
		return input, errs.InvalidParam("参数键只能包含小写字母、数字、点、中划线、下划线,且以小写字母开头")
	}
	if len(input.Key) > maxConfigKeyLen {
		return input, errs.InvalidParam("参数键最长 64 个字符")
	}
	if len([]rune(input.Value)) > maxConfigValueLen {
		return input, errs.InvalidParam("参数值最长 512 个字符")
	}
	input.Remark = strings.TrimSpace(input.Remark)
	if len([]rune(input.Remark)) > maxConfigRemarkLen {
		return input, errs.InvalidParam("备注最长 255 个字符")
	}
	return input, nil
}

// assertConfigKeyFree 校验键名唯一。用 LOWER 比较:MySQL 排序规则大小写不敏感而 SQLite 敏感,
// 不降小写会出现"同一份数据两套行为"(与文章分类同名查重同因)。
func (s *ConfigService) assertConfigKeyFree(ctx context.Context, key string, excludeID int64) error {
	q := s.DB.WithContext(ctx).Model(&model.SysConfig{}).Where("LOWER(config_key) = ?", strings.ToLower(key))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return errs.Internal("查询参数失败").WithCause(err)
	}
	if count > 0 {
		return errs.Conflict("参数键已存在")
	}
	return nil
}

func (s *ConfigService) Create(ctx context.Context, input ConfigSaveInput) (*model.SysConfig, error) {
	input, err := normalizeConfigInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.assertConfigKeyFree(ctx, input.Key, 0); err != nil {
		return nil, err
	}
	cfg := &model.SysConfig{Name: input.Name, ConfigKey: input.Key, Value: input.Value, Remark: input.Remark}
	if err := s.DB.WithContext(ctx).Create(cfg).Error; err != nil {
		return nil, errs.Internal("创建参数失败").WithCause(err)
	}
	return cfg, nil
}

// Update 内置参数只允许改名称、值与备注:键名可能被业务代码硬编码引用,
// 改键名等于让读取方静默失联,必须通过删除重建走显式确认。
func (s *ConfigService) Update(ctx context.Context, id int64, input ConfigSaveInput) error {
	if id <= 0 {
		return errs.InvalidParam("参数不存在")
	}
	input, err := normalizeConfigInput(input)
	if err != nil {
		return err
	}
	var cfg model.SysConfig
	if err := s.DB.WithContext(ctx).First(&cfg, id).Error; err != nil {
		return errs.NotFound("参数不存在")
	}
	if cfg.Builtin && !strings.EqualFold(input.Key, cfg.ConfigKey) {
		return errs.Conflict("内置参数不允许修改键名")
	}
	if err := s.assertConfigKeyFree(ctx, input.Key, id); err != nil {
		return err
	}
	if err := s.DB.WithContext(ctx).Model(&model.SysConfig{ID: cfg.ID}).
		Updates(map[string]any{
			"name": input.Name, "config_key": input.Key,
			"value": input.Value, "remark": input.Remark,
		}).Error; err != nil {
		return errs.Internal("更新参数失败").WithCause(err)
	}
	return nil
}

// Delete 内置参数不可删除。底座自带的参数可能被框架或业务模块引用,
// 删掉后读取方只能拿到空值,这种破坏必须挡在服务层而不是靠前端隐藏按钮。
func (s *ConfigService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errs.InvalidParam("参数不存在")
	}
	var cfg model.SysConfig
	if err := s.DB.WithContext(ctx).First(&cfg, id).Error; err != nil {
		return errs.NotFound("参数不存在")
	}
	if cfg.Builtin {
		return errs.Conflict("内置参数不允许删除")
	}
	if err := s.DB.WithContext(ctx).Delete(&model.SysConfig{}, id).Error; err != nil {
		return errs.Internal("删除参数失败").WithCause(err)
	}
	return nil
}
