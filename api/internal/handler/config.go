// 系统参数控制器:参数的列表、增删改。
// 权限判定全部在路由上挂 RequirePerm,本层只做绑定与转调。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/validate"
)

type ConfigHandler struct {
	Svc *service.ConfigService
}

func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{Svc: svc}
}

// List GET /api/v1/configs
func (h *ConfigHandler) List(c *gin.Context) {
	var query ConfigListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		resp.Fail(c, errs.InvalidParam("分页参数错误"))
		return
	}
	result, err := h.Svc.List(c.Request.Context(), query.toInput())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Create POST /api/v1/configs
func (h *ConfigHandler) Create(c *gin.Context) {
	var req ConfigSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 参数名与参数键必填"))
		return
	}
	cfg, err := h.Svc.Create(c.Request.Context(), req.toInput())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, cfg)
}

// Update PUT /api/v1/configs/:id
func (h *ConfigHandler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req ConfigSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 参数名与参数键必填"))
		return
	}
	if err := h.Svc.Update(c.Request.Context(), id, req.toInput()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Delete DELETE /api/v1/configs/:id
func (h *ConfigHandler) Delete(c *gin.Context) {
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
