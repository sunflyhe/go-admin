// Package role 角色管理域:CRUD、角色-菜单分配。
// 内置角色 super_admin 不可删除、不可停用。
package service

import (
	"context"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
)

type RoleService struct{ DB *gorm.DB }

func NewRoleService(db *gorm.DB) *RoleService { return &RoleService{DB: db} }

// ---- DTO ----

type RoleItem struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Description string  `json:"description"`
	Builtin     bool    `json:"builtin"`
	Status      int     `json:"status"`
	UserCount   int64   `json:"userCount"`
	MenuIDs     []int64 `json:"menuIds,omitempty"`
}

type RoleListInput struct {
	page.Query
	Name   string ``
	Status int    ``
}

type RoleSaveInput struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

type RoleAssignMenusInput struct {
	MenuIDs []int64 `json:"menuIds"`
}

// ---- RoleService ----

func (s *RoleService) List(ctx context.Context, req *RoleListInput) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(ctx).Model(&model.SysRole{})
	if req.Name != "" {
		q = q.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Status != 0 {
		q = q.Where("status = ?", req.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errs.Internal("查询角色失败").WithCause(err)
	}
	var roles []model.SysRole
	if err := q.Order("id ASC").Offset(req.Offset()).Limit(req.PageSize).Find(&roles).Error; err != nil {
		return nil, errs.Internal("查询角色失败").WithCause(err)
	}
	items := make([]RoleItem, 0, len(roles))
	for i := range roles {
		items = append(items, s.toItem(ctx, &roles[i]))
	}
	return &page.Result{List: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

func (s *RoleService) toItem(ctx context.Context, r *model.SysRole) RoleItem {
	var count int64
	_ = s.DB.WithContext(ctx).Model(&model.SysUserRole{}).Where("role_id = ?", r.ID).Count(&count).Error
	return RoleItem{
		ID: r.ID, Name: r.Name, Code: r.Code, Description: r.Description,
		Builtin: r.Builtin, Status: r.Status, UserCount: count,
		MenuIDs: []int64{},
	}
}

func (s *RoleService) Create(ctx context.Context, req *RoleSaveInput) (*RoleItem, error) {
	var count int64
	if err := s.DB.WithContext(ctx).Model(&model.SysRole{}).Where("code = ?", req.Code).Count(&count).Error; err != nil {
		return nil, errs.Internal("查询失败").WithCause(err)
	}
	if count > 0 {
		return nil, errs.Conflict("角色编码已存在")
	}
	status := req.Status
	if status == 0 {
		status = model.StatusEnabled
	}
	r := &model.SysRole{Name: req.Name, Code: req.Code, Description: req.Description, Status: status}
	if err := s.DB.WithContext(ctx).Create(r).Error; err != nil {
		return nil, errs.Internal("创建角色失败").WithCause(err)
	}
	item := s.toItem(ctx, r)
	return &item, nil
}

func (s *RoleService) Update(ctx context.Context, id int64, req *RoleSaveInput) (*RoleItem, error) {
	var r model.SysRole
	if err := s.DB.WithContext(ctx).First(&r, id).Error; err != nil {
		return nil, errs.NotFound("角色不存在")
	}
	var count int64
	if err := s.DB.WithContext(ctx).Model(&model.SysRole{}).Where("code = ? AND id <> ?", req.Code, id).Count(&count).Error; err != nil {
		return nil, errs.Internal("查询失败").WithCause(err)
	}
	if count > 0 {
		return nil, errs.Conflict("角色编码已存在")
	}
	r.Name, r.Description = req.Name, req.Description
	if req.Status != 0 {
		if r.IsSuperAdminRole() && req.Status == model.StatusDisabled {
			return nil, errs.InvalidParam("内置超级管理员角色不允许停用")
		}
		r.Status = req.Status
	}
	// 角色编码创建后不可修改,避免权限配置漂移
	if err := s.DB.WithContext(ctx).Model(&r).Select("name", "description", "status").Updates(r).Error; err != nil {
		return nil, errs.Internal("更新角色失败").WithCause(err)
	}
	item := s.toItem(ctx, &r)
	return &item, nil
}

func (s *RoleService) Delete(ctx context.Context, id int64) error {
	var r model.SysRole
	if err := s.DB.WithContext(ctx).First(&r, id).Error; err != nil {
		return errs.NotFound("角色不存在")
	}
	if r.Builtin {
		return errs.InvalidParam("内置角色不允许删除")
	}
	var count int64
	if err := s.DB.WithContext(ctx).Model(&model.SysUserRole{}).Where("role_id = ?", id).Count(&count).Error; err != nil {
		return errs.Internal("查询失败").WithCause(err)
	}
	if count > 0 {
		return errs.InvalidParam("仍有用户持有该角色,请先解除分配")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&r).Error; err != nil {
			return errs.Internal("删除角色失败").WithCause(err)
		}
		if err := tx.Where("role_id = ?", id).Delete(&model.SysRoleMenu{}).Error; err != nil {
			return errs.Internal("清理角色菜单关联失败").WithCause(err)
		}
		return nil
	})
}

func (s *RoleService) Menus(ctx context.Context, id int64) ([]int64, error) {
	var ids []int64
	if err := s.DB.WithContext(ctx).Model(&model.SysRoleMenu{}).
		Where("role_id = ?", id).Pluck("menu_id", &ids).Error; err != nil {
		return nil, errs.Internal("查询角色菜单失败").WithCause(err)
	}
	if ids == nil {
		ids = []int64{}
	}
	return ids, nil
}

func (s *RoleService) AssignMenus(ctx context.Context, id int64, req *RoleAssignMenusInput) error {
	var r model.SysRole
	if err := s.DB.WithContext(ctx).First(&r, id).Error; err != nil {
		return errs.NotFound("角色不存在")
	}
	if len(req.MenuIDs) > 0 {
		var valid int64
		if err := s.DB.WithContext(ctx).Model(&model.SysMenu{}).
			Where("id IN ? AND status = ?", req.MenuIDs, model.StatusEnabled).Count(&valid).Error; err != nil {
			return errs.Internal("查询菜单失败").WithCause(err)
		}
		if int(valid) != len(req.MenuIDs) {
			return errs.InvalidParam("存在无效或停用的菜单")
		}
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&model.SysRoleMenu{}).Error; err != nil {
			return errs.Internal("清理旧菜单关联失败").WithCause(err)
		}
		for _, mid := range req.MenuIDs {
			if err := tx.Create(&model.SysRoleMenu{RoleID: id, MenuID: mid}).Error; err != nil {
				return errs.Internal("分配菜单失败").WithCause(err)
			}
		}
		return nil
	})
}
