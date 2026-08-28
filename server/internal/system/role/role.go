// Package role 角色管理域:CRUD、角色-菜单分配。
// 内置角色 super_admin 不可删除、不可停用。
package role

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/validate"
)

type Service struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{DB: db} }

// ---- DTO ----

type Item struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Description string  `json:"description"`
	Builtin     bool    `json:"builtin"`
	Status      int     `json:"status"`
	UserCount   int64   `json:"userCount"`
	MenuIDs     []int64 `json:"menuIds,omitempty"`
}

type ListReq struct {
	page.Query
	Name   string `form:"name"`
	Status int    `form:"status" binding:"omitempty,oneof=1 2"`
}

type SaveReq struct {
	Name        string `json:"name" binding:"required,max=64"`
	Code        string `json:"code" binding:"required,min=2,max=64"`
	Description string `json:"description" binding:"max=255"`
	Status      int    `json:"status" binding:"omitempty,oneof=1 2"`
}

type AssignMenusReq struct {
	MenuIDs []int64 `json:"menuIds" binding:"required"`
}

// ---- Service ----

func (s *Service) List(c *gin.Context, req *ListReq) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(c).Model(&model.SysRole{})
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
	items := make([]Item, 0, len(roles))
	for i := range roles {
		items = append(items, s.toItem(c, &roles[i]))
	}
	return &page.Result{List: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

func (s *Service) toItem(c *gin.Context, r *model.SysRole) Item {
	var count int64
	_ = s.DB.WithContext(c).Model(&model.SysUserRole{}).Where("role_id = ?", r.ID).Count(&count).Error
	return Item{
		ID: r.ID, Name: r.Name, Code: r.Code, Description: r.Description,
		Builtin: r.Builtin, Status: r.Status, UserCount: count,
		MenuIDs: []int64{},
	}
}

func (s *Service) Create(c *gin.Context, req *SaveReq) (*Item, error) {
	var count int64
	if err := s.DB.WithContext(c).Model(&model.SysRole{}).Where("code = ?", req.Code).Count(&count).Error; err != nil {
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
	if err := s.DB.WithContext(c).Create(r).Error; err != nil {
		return nil, errs.Internal("创建角色失败").WithCause(err)
	}
	item := s.toItem(c, r)
	return &item, nil
}

func (s *Service) Update(c *gin.Context, id int64, req *SaveReq) (*Item, error) {
	var r model.SysRole
	if err := s.DB.WithContext(c).First(&r, id).Error; err != nil {
		return nil, errs.NotFound("角色不存在")
	}
	var count int64
	if err := s.DB.WithContext(c).Model(&model.SysRole{}).Where("code = ? AND id <> ?", req.Code, id).Count(&count).Error; err != nil {
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
	if err := s.DB.WithContext(c).Model(&r).Select("name", "description", "status").Updates(r).Error; err != nil {
		return nil, errs.Internal("更新角色失败").WithCause(err)
	}
	item := s.toItem(c, &r)
	return &item, nil
}

func (s *Service) Delete(c *gin.Context, id int64) error {
	var r model.SysRole
	if err := s.DB.WithContext(c).First(&r, id).Error; err != nil {
		return errs.NotFound("角色不存在")
	}
	if r.Builtin {
		return errs.InvalidParam("内置角色不允许删除")
	}
	var count int64
	if err := s.DB.WithContext(c).Model(&model.SysUserRole{}).Where("role_id = ?", id).Count(&count).Error; err != nil {
		return errs.Internal("查询失败").WithCause(err)
	}
	if count > 0 {
		return errs.InvalidParam("仍有用户持有该角色,请先解除分配")
	}
	return s.DB.WithContext(c).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&r).Error; err != nil {
			return errs.Internal("删除角色失败").WithCause(err)
		}
		if err := tx.Where("role_id = ?", id).Delete(&model.SysRoleMenu{}).Error; err != nil {
			return errs.Internal("清理角色菜单关联失败").WithCause(err)
		}
		return nil
	})
}

func (s *Service) Menus(c *gin.Context, id int64) ([]int64, error) {
	var ids []int64
	if err := s.DB.WithContext(c).Model(&model.SysRoleMenu{}).
		Where("role_id = ?", id).Pluck("menu_id", &ids).Error; err != nil {
		return nil, errs.Internal("查询角色菜单失败").WithCause(err)
	}
	if ids == nil {
		ids = []int64{}
	}
	return ids, nil
}

func (s *Service) AssignMenus(c *gin.Context, id int64, req *AssignMenusReq) error {
	var r model.SysRole
	if err := s.DB.WithContext(c).First(&r, id).Error; err != nil {
		return errs.NotFound("角色不存在")
	}
	if len(req.MenuIDs) > 0 {
		var valid int64
		if err := s.DB.WithContext(c).Model(&model.SysMenu{}).
			Where("id IN ? AND status = ?", req.MenuIDs, model.StatusEnabled).Count(&valid).Error; err != nil {
			return errs.Internal("查询菜单失败").WithCause(err)
		}
		if int(valid) != len(req.MenuIDs) {
			return errs.InvalidParam("存在无效或停用的菜单")
		}
	}
	return s.DB.WithContext(c).Transaction(func(tx *gorm.DB) error {
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

// ---- Handler ----

type Handler struct{ Svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{Svc: svc} }

// List GET /api/v1/roles
func (h *Handler) List(c *gin.Context) {
	var req ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("分页参数错误: pageSize 不能超过 100"))
		return
	}
	result, err := h.Svc.List(c, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Create POST /api/v1/roles
func (h *Handler) Create(c *gin.Context) {
	var req SaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 名称与编码必填,编码 2-64 位"))
		return
	}
	result, err := h.Svc.Create(c, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, result)
}

// Update PUT /api/v1/roles/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req SaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误"))
		return
	}
	result, err := h.Svc.Update(c, id, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Delete DELETE /api/v1/roles/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if err := h.Svc.Delete(c, id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Menus GET /api/v1/roles/:id/menus
func (h *Handler) Menus(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	result, err := h.Svc.Menus(c, id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// AssignMenus PUT /api/v1/roles/:id/menus
func (h *Handler) AssignMenus(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req AssignMenusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: menuIds 必填"))
		return
	}
	if err := h.Svc.AssignMenus(c, id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
