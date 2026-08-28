// 菜单管理接口的 HTTP 绑定 DTO 与到 Service 输入的显式转换。
package handler

import (
	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
)

// MenuSaveRequest 菜单创建/更新请求体。
type MenuSaveRequest struct {
	ParentID   int64  `json:"parentId"`
	Name       string `json:"name" binding:"required,max=64"`
	Type       int    `json:"type" binding:"required,oneof=1 2 3"`
	Path       string `json:"path" binding:"max=255"`
	Component  string `json:"component" binding:"max=255"`
	Permission string `json:"permission" binding:"max=128"`
	Icon       string `json:"icon" binding:"max=64"`
	Sort       int    `json:"sort"`
	Status     int    `json:"status" binding:"required,oneof=1 2"`
}

func (r *MenuSaveRequest) toInput() *service.MenuSaveInput {
	return &service.MenuSaveInput{
		ParentID:   r.ParentID,
		Name:       r.Name,
		Type:       r.Type,
		Path:       r.Path,
		Component:  r.Component,
		Permission: r.Permission,
		Icon:       r.Icon,
		Sort:       r.Sort,
		Status:     r.Status,
	}
}
