// Package menu 菜单域:菜单树构建、按用户过滤可见菜单、菜单 CRUD。
package service

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

// MenuNode 前端菜单树节点(目录与页面;按钮不进树,以权限码形式下发)。
type MenuNode struct {
	ID        int64       `json:"id"`
	ParentID  int64       `json:"parentId"`
	Name      string      `json:"name"`
	Type      int         `json:"type"`
	Path      string      `json:"path"`
	Component string      `json:"component"`
	Icon      string      `json:"icon"`
	Sort      int         `json:"sort"`
	Children  []*MenuNode `json:"children,omitempty"`
}

// LoadUserMenus 返回用户可见的菜单树(目录/页面,仅启用)。
// 超级管理员返回全部;普通用户按角色授权过滤。
func LoadUserMenus(db *gorm.DB, user *model.SysUser) ([]MenuNode, error) {
	var menus []model.SysMenu
	q := db.Table("sys_menu").
		Where("sys_menu.type IN ? AND sys_menu.status = ?", []int{model.MenuTypeDirectory, model.MenuTypePage}, model.StatusEnabled)
	if !user.IsSuperAdmin() {
		q = q.Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
			Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role_menu.role_id").
			Joins("JOIN sys_role ON sys_role.id = sys_role_menu.role_id").
			Where("sys_user_role.user_id = ? AND sys_role.status = ?", user.ID, model.StatusEnabled).
			Distinct()
	}
	if err := q.Order("sys_menu.sort ASC, sys_menu.id ASC").Find(&menus).Error; err != nil {
		return nil, errs.Internal("加载菜单失败").WithCause(err)
	}
	return buildTree(menus), nil
}

// LoadVisibleTree 管理端菜单维护使用的完整树(含按钮,仅启用或全部由参数控制)。
func LoadVisibleTree(db *gorm.DB, withButtons bool) ([]MenuNode, error) {
	q := db
	if withButtons {
		q = db.Where("status = ?", model.StatusEnabled)
	} else {
		q = db.Where("type IN ? AND status = ?", []int{model.MenuTypeDirectory, model.MenuTypePage}, model.StatusEnabled)
	}
	var menus []model.SysMenu
	if err := q.Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		return nil, errs.Internal("加载菜单失败").WithCause(err)
	}
	return buildTree(menus), nil
}

// LoadAll 返回全部菜单(含停用,管理端需要)。
func LoadAll(db *gorm.DB) ([]model.SysMenu, error) {
	var menus []model.SysMenu
	if err := db.Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		return nil, errs.Internal("加载菜单失败").WithCause(err)
	}
	return menus, nil
}

// buildTree 先按指针挂接父子关系,全部挂接完成后再转换输出,避免子节点丢失。
func buildTree(menus []model.SysMenu) []MenuNode {
	nodes := make(map[int64]*MenuNode, len(menus))
	var rootPtrs []*MenuNode
	for i := range menus {
		m := menus[i]
		nodes[m.ID] = &MenuNode{
			ID: m.ID, ParentID: m.ParentID, Name: m.Name, Type: m.Type,
			Path: m.Path, Component: m.Component, Icon: m.Icon, Sort: m.Sort,
		}
	}
	for i := range menus {
		m := menus[i]
		node := nodes[m.ID]
		if parent, ok := nodes[m.ParentID]; ok && m.ParentID != m.ID {
			parent.Children = append(parent.Children, node)
		} else {
			rootPtrs = append(rootPtrs, node)
		}
	}
	for _, n := range nodes {
		sortNodesPtr(n.Children)
	}
	sortNodesPtr(rootPtrs)
	roots := make([]MenuNode, 0, len(rootPtrs))
	for _, p := range rootPtrs {
		roots = append(roots, *p)
	}
	return roots
}

func sortNodesPtr(nodes []*MenuNode) {
	for i := 1; i < len(nodes); i++ {
		for j := i; j > 0 && nodes[j].Sort < nodes[j-1].Sort; j-- {
			nodes[j], nodes[j-1] = nodes[j-1], nodes[j]
		}
	}
}

// ---- MenuService:菜单 CRUD ----

type MenuService struct{ DB *gorm.DB }

func NewMenuService(db *gorm.DB) *MenuService { return &MenuService{DB: db} }

type MenuSaveReq struct {
	ParentID   int64  `json:"parentId"`
	Name       string `json:"name" binding:"required,max=64"`
	Type       int    `json:"type" binding:"required,oneof=1 2 3"`
	Path       string `json:"path" binding:"max=255"`
	Component  string `json:"component" binding:"max=255"`
	Permission string `json:"permission" binding:"max=128"`
	Icon       string `json:"icon" binding:"max=64"`
	Sort       int    `json:"sort"`
	Status     int    `json:"status" binding:"oneof=1 2"`
}

func (s *MenuService) Tree(c *gin.Context) ([]MenuNode, error) {
	return LoadVisibleTree(s.DB.WithContext(c), true)
}

func (s *MenuService) ListAll(c *gin.Context) ([]model.SysMenu, error) {
	return LoadAll(s.DB.WithContext(c))
}

func (s *MenuService) Create(c *gin.Context, req *MenuSaveReq) (*model.SysMenu, error) {
	if req.ParentID != 0 {
		if err := s.checkParent(c, req.ParentID, req.Type); err != nil {
			return nil, err
		}
	}
	m := &model.SysMenu{
		ParentID: req.ParentID, Name: req.Name, Type: req.Type,
		Path: req.Path, Component: req.Component, Permission: req.Permission,
		Icon: req.Icon, Sort: req.Sort, Status: req.Status,
	}
	if err := s.DB.WithContext(c).Create(m).Error; err != nil {
		return nil, errs.Internal("创建菜单失败").WithCause(err)
	}
	return m, nil
}

func (s *MenuService) Update(c *gin.Context, id int64, req *MenuSaveReq) (*model.SysMenu, error) {
	var m model.SysMenu
	if err := s.DB.WithContext(c).First(&m, id).Error; err != nil {
		return nil, errs.NotFound("菜单不存在")
	}
	if req.ParentID == id {
		return nil, errs.InvalidParam("父级不能是自身")
	}
	if req.ParentID != 0 {
		if err := s.checkParent(c, req.ParentID, req.Type); err != nil {
			return nil, err
		}
		if isDescendant, _ := s.isDescendant(c, id, req.ParentID); isDescendant {
			return nil, errs.InvalidParam("父级不能是自身的子节点")
		}
	}
	m.ParentID, m.Name, m.Type = req.ParentID, req.Name, req.Type
	m.Path, m.Component, m.Permission = req.Path, req.Component, req.Permission
	m.Icon, m.Sort, m.Status = req.Icon, req.Sort, req.Status
	if err := s.DB.WithContext(c).Save(&m).Error; err != nil {
		return nil, errs.Internal("更新菜单失败").WithCause(err)
	}
	return &m, nil
}

func (s *MenuService) Delete(c *gin.Context, id int64) error {
	var count int64
	if err := s.DB.WithContext(c).Model(&model.SysMenu{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return errs.Internal("查询失败").WithCause(err)
	}
	if count > 0 {
		return errs.InvalidParam("请先删除子菜单")
	}
	return s.DB.WithContext(c).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.SysMenu{}, id).Error; err != nil {
			return errs.Internal("删除菜单失败").WithCause(err)
		}
		if err := tx.Where("menu_id = ?", id).Delete(&model.SysRoleMenu{}).Error; err != nil {
			return errs.Internal("清理角色菜单关联失败").WithCause(err)
		}
		return nil
	})
}

func (s *MenuService) checkParent(c *gin.Context, parentID int64, childType int) error {
	var parent model.SysMenu
	if err := s.DB.WithContext(c).First(&parent, parentID).Error; err != nil {
		return errs.NotFound("父级菜单不存在")
	}
	if parent.Type == model.MenuTypeButton {
		return errs.InvalidParam("按钮下不能再挂子级")
	}
	if childType == model.MenuTypeDirectory && parent.Type != model.MenuTypeDirectory {
		return errs.InvalidParam("目录只能挂在目录下")
	}
	return nil
}

// isDescendant 逐层向上查 candidate 的祖先链,最多 20 层,防止把自身后代设为父级形成环。
func (s *MenuService) isDescendant(c *gin.Context, ancestor, candidate int64) (bool, error) {
	cur := candidate
	for i := 0; i < 20; i++ {
		var m model.SysMenu
		if err := s.DB.WithContext(c).Select("id", "parent_id").First(&m, cur).Error; err != nil {
			return false, nil
		}
		if m.ParentID == ancestor {
			return true, nil
		}
		if m.ParentID == 0 {
			return false, nil
		}
		cur = m.ParentID
	}
	return false, nil
}
