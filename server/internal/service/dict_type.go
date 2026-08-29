// 字典服务:字典类型与子项的维护,以及面向业务模块的按键读取。
// 一个类型(如 性别)下挂多条子项(男/女/未知);业务页面用 key 换取启用子项做下拉。
package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

const (
	maxDictTypeNameLen   = 64
	maxDictTypeKeyLen    = 64
	maxDictTypeRemarkLen = 255
	maxDictItemLabelLen  = 64
	maxDictItemValueLen  = 128
	maxDictItemDescLen   = 255
)

// tagTypes 是 tag_type 的合法取值:对应前端 el-tag 的 type,空串表示不配标签。
var tagTypes = map[string]bool{"": true, "primary": true, "success": true, "warning": true, "danger": true, "info": true}

// DictTypeItem 类型列表条目,ItemCount 供页面展示与删除前的判断提示。
type DictTypeItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Remark    string    `json:"remark"`
	Builtin   bool      `json:"builtin"`
	ItemCount int64     `json:"itemCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DictItemDetail 子项条目。
type DictItemDetail struct {
	ID          int64     `json:"id"`
	TypeID      int64     `json:"typeId"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	Value       string    `json:"value"`
	Sort        int       `json:"sort"`
	TagType     string    `json:"tagType"`
	Status      int       `json:"status"`
	Remark      string    `json:"remark"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DictOption 业务读取用:展示文本、存储值与标签配色,不下发内部字段。
type DictOption struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	TagType string `json:"tagType"`
}

// DictTypeSaveInput 类型创建/更新输入。
type DictTypeSaveInput struct {
	Name   string
	Key    string
	Remark string
}

// DictItemSaveInput 子项创建/更新输入。类型归属在创建时确定,不提供改属。
type DictItemSaveInput struct {
	Label       string
	Value       string
	Description string
	Sort        int
	TagType     string
	Status      int
	Remark      string
}

type DictTypeService struct {
	DB *gorm.DB
}

func NewDictTypeService(db *gorm.DB) *DictTypeService {
	return &DictTypeService{DB: db}
}

// List 类型列表(含各自子项数)。类型是少量枚举,不分页。
func (s *DictTypeService) List(ctx context.Context) ([]DictTypeItem, error) {
	var types []model.SysDictType
	if err := s.DB.WithContext(ctx).Order("id ASC").Find(&types).Error; err != nil {
		return nil, errs.Internal("查询字典类型失败").WithCause(err)
	}
	type row struct {
		TypeID int64
		Num    int64
	}
	var rows []row
	if err := s.DB.WithContext(ctx).Model(&model.SysDictItem{}).
		Select("type_id AS type_id, COUNT(*) AS num").
		Group("type_id").Scan(&rows).Error; err != nil {
		return nil, errs.Internal("统计字典项失败").WithCause(err)
	}
	byType := make(map[int64]int64, len(rows))
	for _, r := range rows {
		byType[r.TypeID] += r.Num
	}
	items := make([]DictTypeItem, 0, len(types))
	for _, t := range types {
		items = append(items, DictTypeItem{
			ID: t.ID, Name: t.Name, Key: t.DictKey, Remark: t.Remark, Builtin: t.Builtin,
			ItemCount: byType[t.ID], CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		})
	}
	return items, nil
}

func normalizeDictTypeInput(input DictTypeSaveInput) (DictTypeSaveInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return input, errs.InvalidParam("字典名不能为空")
	}
	if len([]rune(input.Name)) > maxDictTypeNameLen {
		return input, errs.InvalidParam("字典名最长 64 个字符")
	}
	input.Key = strings.TrimSpace(input.Key)
	if !configKeyPattern.MatchString(input.Key) {
		return input, errs.InvalidParam("字典键只能包含小写字母、数字、点、中划线、下划线,且以小写字母开头")
	}
	if len(input.Key) > maxDictTypeKeyLen {
		return input, errs.InvalidParam("字典键最长 64 个字符")
	}
	input.Remark = strings.TrimSpace(input.Remark)
	if len([]rune(input.Remark)) > maxDictTypeRemarkLen {
		return input, errs.InvalidParam("备注最长 255 个字符")
	}
	return input, nil
}

// assertDictTypeKeyFree 校验类型键唯一,LOWER 比较兜底 MySQL/SQLite 排序规则差异。
func (s *DictTypeService) assertDictTypeKeyFree(ctx context.Context, key string, excludeID int64) error {
	q := s.DB.WithContext(ctx).Model(&model.SysDictType{}).Where("LOWER(dict_key) = ?", strings.ToLower(key))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return errs.Internal("查询字典类型失败").WithCause(err)
	}
	if count > 0 {
		return errs.Conflict("字典键已存在")
	}
	return nil
}

func (s *DictTypeService) Create(ctx context.Context, input DictTypeSaveInput) (*model.SysDictType, error) {
	input, err := normalizeDictTypeInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.assertDictTypeKeyFree(ctx, input.Key, 0); err != nil {
		return nil, err
	}
	dictType := &model.SysDictType{Name: input.Name, DictKey: input.Key, Remark: input.Remark}
	if err := s.DB.WithContext(ctx).Create(dictType).Error; err != nil {
		return nil, errs.Internal("创建字典类型失败").WithCause(err)
	}
	return dictType, nil
}

// Update 内置字典只允许改名称与备注:键名可能被业务代码硬编码引用,改键等于让读取方静默失联。
func (s *DictTypeService) Update(ctx context.Context, id int64, input DictTypeSaveInput) error {
	if id <= 0 {
		return errs.InvalidParam("字典类型不存在")
	}
	input, err := normalizeDictTypeInput(input)
	if err != nil {
		return err
	}
	var dictType model.SysDictType
	if err := s.DB.WithContext(ctx).First(&dictType, id).Error; err != nil {
		return errs.NotFound("字典类型不存在")
	}
	if dictType.Builtin && !strings.EqualFold(input.Key, dictType.DictKey) {
		return errs.Conflict("内置字典不允许修改键名")
	}
	if err := s.assertDictTypeKeyFree(ctx, input.Key, id); err != nil {
		return err
	}
	if err := s.DB.WithContext(ctx).Model(&model.SysDictType{ID: dictType.ID}).
		Updates(map[string]any{"name": input.Name, "dict_key": input.Key, "remark": input.Remark}).Error; err != nil {
		return errs.Internal("更新字典类型失败").WithCause(err)
	}
	return nil
}

// Delete 删除字典类型。内置字典不可删;仍有子项时直接拒绝 ——
// 删类型不该连带丢子项,由用户先清空子项再删,与文章分类删除同一语义。
func (s *DictTypeService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errs.InvalidParam("字典类型不存在")
	}
	var dictType model.SysDictType
	if err := s.DB.WithContext(ctx).First(&dictType, id).Error; err != nil {
		return errs.NotFound("字典类型不存在")
	}
	if dictType.Builtin {
		return errs.Conflict("内置字典不允许删除")
	}
	var used int64
	if err := s.DB.WithContext(ctx).Model(&model.SysDictItem{}).
		Where("type_id = ?", id).Count(&used).Error; err != nil {
		return errs.Internal("统计字典项失败").WithCause(err)
	}
	if used > 0 {
		return errs.Conflict("字典下仍有子项,请先清空子项")
	}
	if err := s.DB.WithContext(ctx).Delete(&model.SysDictType{}, id).Error; err != nil {
		return errs.Internal("删除字典类型失败").WithCause(err)
	}
	return nil
}

func (s *DictTypeService) assertTypeExists(ctx context.Context, typeID int64) error {
	var count int64
	if err := s.DB.WithContext(ctx).Model(&model.SysDictType{}).Where("id = ?", typeID).Count(&count).Error; err != nil {
		return errs.Internal("查询字典类型失败").WithCause(err)
	}
	if count == 0 {
		return errs.NotFound("字典类型不存在")
	}
	return nil
}

func normalizeDictItemInput(input DictItemSaveInput) (DictItemSaveInput, error) {
	input.Label = strings.TrimSpace(input.Label)
	if input.Label == "" {
		return input, errs.InvalidParam("子项显示名不能为空")
	}
	if len([]rune(input.Label)) > maxDictItemLabelLen {
		return input, errs.InvalidParam("子项显示名最长 64 个字符")
	}
	input.Value = strings.TrimSpace(input.Value)
	if input.Value == "" {
		return input, errs.InvalidParam("子项存储值不能为空")
	}
	if len([]rune(input.Value)) > maxDictItemValueLen {
		return input, errs.InvalidParam("子项存储值最长 128 个字符")
	}
	input.Description = strings.TrimSpace(input.Description)
	if len([]rune(input.Description)) > maxDictItemDescLen {
		return input, errs.InvalidParam("描述最长 255 个字符")
	}
	if !tagTypes[input.TagType] {
		return input, errs.InvalidParam("标签类型取值不合法")
	}
	input.Remark = strings.TrimSpace(input.Remark)
	if len([]rune(input.Remark)) > maxDictTypeRemarkLen {
		return input, errs.InvalidParam("备注最长 255 个字符")
	}
	switch input.Status {
	case model.DictItemStatusEnabled, model.DictItemStatusDisabled:
	default:
		input.Status = model.DictItemStatusEnabled // 0 视为未传,默认启用
	}
	return input, nil
}

// assertItemValueFree 校验同一类型内子项值唯一(LOWER 比较,跨库行为一致)。
func (s *DictTypeService) assertItemValueFree(ctx context.Context, typeID int64, value string, excludeID int64) error {
	q := s.DB.WithContext(ctx).Model(&model.SysDictItem{}).
		Where("type_id = ? AND LOWER(value) = ?", typeID, strings.ToLower(value))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return errs.Internal("查询字典项失败").WithCause(err)
	}
	if count > 0 {
		return errs.Conflict("该字典下已存在同值子项")
	}
	return nil
}

// ListItems 类型下的全部子项(含停用,管理端要能看到再启用);按 sort 升序。
func (s *DictTypeService) ListItems(ctx context.Context, typeID int64) ([]DictItemDetail, error) {
	if err := s.assertTypeExists(ctx, typeID); err != nil {
		return nil, err
	}
	var items []model.SysDictItem
	if err := s.DB.WithContext(ctx).Where("type_id = ?", typeID).
		Order("sort ASC, id ASC").Find(&items).Error; err != nil {
		return nil, errs.Internal("查询字典项失败").WithCause(err)
	}
	list := make([]DictItemDetail, 0, len(items))
	for _, it := range items {
		list = append(list, DictItemDetail{
			ID: it.ID, TypeID: it.TypeID, Label: it.Label, Description: it.Description,
			Value: it.Value, Sort: it.Sort, TagType: it.TagType, Status: it.Status, Remark: it.Remark,
			CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		})
	}
	return list, nil
}

func (s *DictTypeService) CreateItem(ctx context.Context, typeID int64, input DictItemSaveInput) (*DictItemDetail, error) {
	if typeID <= 0 {
		return nil, errs.InvalidParam("字典类型不存在")
	}
	input, err := normalizeDictItemInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.assertTypeExists(ctx, typeID); err != nil {
		return nil, err
	}
	if err := s.assertItemValueFree(ctx, typeID, input.Value, 0); err != nil {
		return nil, err
	}
	item := &model.SysDictItem{
		TypeID: typeID, Label: input.Label, Description: input.Description, Value: input.Value,
		Sort: input.Sort, TagType: input.TagType, Status: input.Status, Remark: input.Remark,
	}
	if err := s.DB.WithContext(ctx).Create(item).Error; err != nil {
		return nil, errs.Internal("创建字典项失败").WithCause(err)
	}
	return &DictItemDetail{
		ID: item.ID, TypeID: item.TypeID, Label: item.Label, Description: item.Description,
		Value: item.Value, Sort: item.Sort, TagType: item.TagType, Status: item.Status, Remark: item.Remark,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}, nil
}

// UpdateItem 子项可改显示名/存储值/排序/状态/备注;类型归属不提供修改,移动语义走删除重建。
func (s *DictTypeService) UpdateItem(ctx context.Context, id int64, input DictItemSaveInput) error {
	if id <= 0 {
		return errs.InvalidParam("字典项不存在")
	}
	input, err := normalizeDictItemInput(input)
	if err != nil {
		return err
	}
	var item model.SysDictItem
	if err := s.DB.WithContext(ctx).First(&item, id).Error; err != nil {
		return errs.NotFound("字典项不存在")
	}
	if err := s.assertItemValueFree(ctx, item.TypeID, input.Value, id); err != nil {
		return err
	}
	if err := s.DB.WithContext(ctx).Model(&model.SysDictItem{ID: item.ID}).
		Updates(map[string]any{
			"label": input.Label, "description": input.Description, "value": input.Value,
			"sort": input.Sort, "tag_type": input.TagType, "status": input.Status, "remark": input.Remark,
		}).Error; err != nil {
		return errs.Internal("更新字典项失败").WithCause(err)
	}
	return nil
}

func (s *DictTypeService) DeleteItem(ctx context.Context, id int64) error {
	if id <= 0 {
		return errs.InvalidParam("字典项不存在")
	}
	res := s.DB.WithContext(ctx).Delete(&model.SysDictItem{}, id)
	if res.Error != nil {
		return errs.Internal("删除字典项失败").WithCause(res.Error)
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("字典项不存在")
	}
	return nil
}

// EnabledByKey 业务读取入口:按类型键返回启用子项(仅 label/value,按 sort 升序)。
// 类型不存在返回空列表而不是 404:业务页面对可选字典的读取不应因字典未配置而报错。
// 子项被停用后立即不再下发;无需重启或清缓存 —— 该接口每次实时查库。
func (s *DictTypeService) EnabledByKey(ctx context.Context, key string) ([]DictOption, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errs.InvalidParam("字典键不能为空")
	}
	var dictType model.SysDictType
	if err := s.DB.WithContext(ctx).Where("dict_key = ?", key).First(&dictType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []DictOption{}, nil
		}
		return nil, errs.Internal("查询字典类型失败").WithCause(err)
	}
	var items []model.SysDictItem
	if err := s.DB.WithContext(ctx).
		Where("type_id = ? AND status = ?", dictType.ID, model.DictItemStatusEnabled).
		Order("sort ASC, id ASC").Find(&items).Error; err != nil {
		return nil, errs.Internal("查询字典项失败").WithCause(err)
	}
	options := make([]DictOption, 0, len(items))
	for _, it := range items {
		options = append(options, DictOption{Label: it.Label, Value: it.Value, TagType: it.TagType})
	}
	return options, nil
}
