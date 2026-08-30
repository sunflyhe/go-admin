// 菜单管理控制器。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/validate"
)

type MenuHandler struct {
	Svc *service.MenuService
}

func NewMenuHandler(svc *service.MenuService) *MenuHandler {
	return &MenuHandler{Svc: svc}
}

// Tree GET /admin-api/menus/tree
func (h *MenuHandler) Tree(c *gin.Context) {
	result, err := h.Svc.Tree(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// List GET /admin-api/menus
func (h *MenuHandler) List(c *gin.Context) {
	result, err := h.Svc.ListAll(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Create POST /admin-api/menus
func (h *MenuHandler) Create(c *gin.Context) {
	var req MenuSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: "+err.Error()))
		return
	}
	result, err := h.Svc.Create(c.Request.Context(), req.toInput())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, result)
}

// Update PUT /admin-api/menus/:id
func (h *MenuHandler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req MenuSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: "+err.Error()))
		return
	}
	result, err := h.Svc.Update(c.Request.Context(), id, req.toInput())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Delete DELETE /admin-api/menus/:id
func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if err := h.Svc.Delete(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
