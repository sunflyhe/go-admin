// 菜单管理控制器。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/validate"
)

type MenuHandler struct {
	Svc *service.MenuService
}

func NewMenuHandler(svc *service.MenuService) *MenuHandler {
	return &MenuHandler{Svc: svc}
}

// Tree GET /api/v1/menus/tree
func (h *MenuHandler) Tree(c *gin.Context) {
	result, err := h.Svc.Tree(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// List GET /api/v1/menus
func (h *MenuHandler) List(c *gin.Context) {
	result, err := h.Svc.ListAll(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Create POST /api/v1/menus
func (h *MenuHandler) Create(c *gin.Context) {
	var req service.MenuSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: "+err.Error()))
		return
	}
	result, err := h.Svc.Create(c, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, result)
}

// Update PUT /api/v1/menus/:id
func (h *MenuHandler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req service.MenuSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: "+err.Error()))
		return
	}
	result, err := h.Svc.Update(c, id, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Delete DELETE /api/v1/menus/:id
func (h *MenuHandler) Delete(c *gin.Context) {
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
