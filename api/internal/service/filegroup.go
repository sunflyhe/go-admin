// 文件分组服务:文件中心左栏的目录。本轮只做一级分组,parent_id 留位不开放嵌套。
package service

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
)

const maxGroupNameLen = 64

// FileGroupItem 分组条目,Count 为该组文件数(供左栏显示)。
type FileGroupItem struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// FileGroupTree 当前类型标签下左栏需要的完整信息:分组列表 + 两个伪节点的文件数。
// 未分组(group_id=0)与全部都不落库,所以计数单独回传,前端不必自己猜。
// 三个数字都限定在 List 传入的 category 内 —— 分组是类型之下的层级,口径必须跟着类型走。
type FileGroupTree struct {
	Groups  []FileGroupItem `json:"groups"`
	Unfiled int64           `json:"unfiled"`
	Total   int64           `json:"total"`
}

type FileGroupService struct {
	DB *gorm.DB
}

func NewFileGroupService(db *gorm.DB) *FileGroupService {
	return &FileGroupService{DB: db}
}

// List 取回分组与各组文件数,一次 GROUP BY 完成,避免按分组逐个 COUNT。
// category 是最外层导航:计数必须落在它之内,与右侧文件列表用同一套谓词(applyCategory)。
func (s *FileGroupService) List(ctx context.Context, rawCategory string) (*FileGroupTree, error) {
	category, err := normalizeCategory(rawCategory)
	if err != nil {
		return nil, err
	}
	var groups []model.SysFileGroup
	if err := s.DB.WithContext(ctx).Order("sort ASC, id ASC").Find(&groups).Error; err != nil {
		return nil, errs.Internal("查询分组失败").WithCause(err)
	}
	type row struct {
		GroupID int64
		Num     int64
	}
	var rows []row
	counts := s.DB.WithContext(ctx).Model(&model.SysFile{}).
		Select("group_id AS group_id, COUNT(*) AS num").Group("group_id")
	counts, err = applyCategory(counts, category)
	if err != nil {
		return nil, err
	}
	if err := counts.Scan(&rows).Error; err != nil {
		return nil, errs.Internal("统计文件失败").WithCause(err)
	}
	byGroup := make(map[int64]int64, len(rows))
	var total int64
	for _, r := range rows {
		byGroup[r.GroupID] += r.Num
		total += r.Num
	}

	// 分组本身不按类型过滤:计数为 0 的组仍要能被选中,否则切个标签左栏就少一个组
	items := make([]FileGroupItem, 0, len(groups))
	for _, g := range groups {
		items = append(items, FileGroupItem{ID: g.ID, Name: g.Name, Count: byGroup[g.ID]})
	}
	return &FileGroupTree{Groups: items, Unfiled: byGroup[0], Total: total}, nil
}

func normalizeGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errs.InvalidParam("分组名不能为空")
	}
	if len([]rune(name)) > maxGroupNameLen {
		return "", errs.InvalidParam("分组名最长 64 个字符")
	}
	return name, nil
}

// assertNameFree 校验同名。用 LOWER 比较:MySQL 排序规则本身大小写不敏感,
// 而 SQLite 敏感,不降小写会出现"同一份数据两套行为"。
func (s *FileGroupService) assertNameFree(ctx context.Context, name string, excludeID int64) error {
	q := s.DB.WithContext(ctx).Model(&model.SysFileGroup{}).Where("LOWER(name) = ?", strings.ToLower(name))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return errs.Internal("查询分组失败").WithCause(err)
	}
	if count > 0 {
		return errs.Conflict("同名分组已存在")
	}
	return nil
}

func (s *FileGroupService) Create(ctx context.Context, name string) (*model.SysFileGroup, error) {
	name, err := normalizeGroupName(name)
	if err != nil {
		return nil, err
	}
	if err := s.assertNameFree(ctx, name, 0); err != nil {
		return nil, err
	}
	// 排在已有分组之后,左栏顺序才稳定
	var maxSort int
	s.DB.WithContext(ctx).Model(&model.SysFileGroup{}).Select("COALESCE(MAX(sort), 0)").Scan(&maxSort)
	group := &model.SysFileGroup{Name: name, Sort: maxSort + 1}
	if err := s.DB.WithContext(ctx).Create(group).Error; err != nil {
		return nil, errs.Internal("创建分组失败").WithCause(err)
	}
	return group, nil
}

func (s *FileGroupService) Update(ctx context.Context, id int64, name string) error {
	if id <= 0 {
		return errs.InvalidParam("分组不存在")
	}
	name, err := normalizeGroupName(name)
	if err != nil {
		return err
	}
	var group model.SysFileGroup
	if err := s.DB.WithContext(ctx).First(&group, id).Error; err != nil {
		return errs.NotFound("分组不存在")
	}
	if err := s.assertNameFree(ctx, name, id); err != nil {
		return err
	}
	if err := s.DB.WithContext(ctx).Model(&model.SysFileGroup{ID: group.ID}).
		Update("name", name).Error; err != nil {
		return errs.Internal("更新分组失败").WithCause(err)
	}
	return nil
}

// Delete 删除空分组。非空分组直接拒绝:分组是"整理"手段,不该顺带删掉文件。
func (s *FileGroupService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errs.InvalidParam("分组不存在")
	}
	var group model.SysFileGroup
	if err := s.DB.WithContext(ctx).First(&group, id).Error; err != nil {
		return errs.NotFound("分组不存在")
	}
	var used int64
	if err := s.DB.WithContext(ctx).Model(&model.SysFile{}).
		Where("group_id = ?", id).Count(&used).Error; err != nil {
		return errs.Internal("统计文件失败").WithCause(err)
	}
	if used > 0 {
		return errs.Conflict("分组下仍有文件,请先移动或删除")
	}
	if err := s.DB.WithContext(ctx).Delete(&model.SysFileGroup{}, id).Error; err != nil {
		return errs.Internal("删除分组失败").WithCause(err)
	}
	return nil
}
