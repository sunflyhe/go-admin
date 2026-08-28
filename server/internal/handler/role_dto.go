// 角色管理接口的 HTTP 绑定 DTO 与到 Service 输入的显式转换。
package handler

import (
	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
)

// RoleListQuery GET /api/v1/roles 查询参数。
type RoleListQuery struct {
	page.Query
	Name   string `form:"name"`
	Status int    `form:"status" binding:"omitempty,oneof=1 2"`
}

func (q *RoleListQuery) toInput() *service.RoleListInput {
	return &service.RoleListInput{
		Query:  q.Query,
		Name:   q.Name,
		Status: q.Status,
	}
}

// RoleCreateRequest POST /api/v1/roles 请求体。
type RoleCreateRequest struct {
	Name        string `json:"name" binding:"required,max=64"`
	Code        string `json:"code" binding:"required,min=2,max=64"`
	Description string `json:"description" binding:"max=255"`
	Status      int    `json:"status" binding:"omitempty,oneof=1 2"`
}

// RoleUpdateRequest PUT /api/v1/roles/:id 请求体(编码创建后不可修改)。
type RoleUpdateRequest struct {
	Name        string `json:"name" binding:"required,max=64"`
	Code        string `json:"code" binding:"omitempty,min=2,max=64"`
	Description string `json:"description" binding:"max=255"`
	Status      int    `json:"status" binding:"omitempty,oneof=1 2"`
}

// RoleAssignMenusRequest PUT /api/v1/roles/:id/menus 请求体。
type RoleAssignMenusRequest struct {
	MenuIDs []int64 `json:"menuIds" binding:"required"`
}

// toInput 创建:编码必填。
func (r *RoleCreateRequest) toInput() *service.RoleSaveInput {
	return &service.RoleSaveInput{
		Name:        r.Name,
		Code:        r.Code,
		Description: r.Description,
		Status:      r.Status,
	}
}

// toInput 更新:编码由 Service 忽略,这里不透传,避免误改。
func (r *RoleUpdateRequest) toInput() *service.RoleSaveInput {
	return &service.RoleSaveInput{
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
	}
}
