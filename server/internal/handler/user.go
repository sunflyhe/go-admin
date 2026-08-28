// 用户管理控制器。
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/validate"
)

type UserHandler struct {
	Svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{Svc: svc}
}

// List GET /api/v1/users
func (h *UserHandler) List(c *gin.Context) {
	var req service.UserListReq
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

// Create POST /api/v1/users
func (h *UserHandler) Create(c *gin.Context) {
	var req service.UserSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 用户名必填(3-64 位),密码至少 8 位"))
		return
	}
	result, err := h.Svc.Create(c, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Created(c, result)
}

// Update PUT /api/v1/users/:id
func (h *UserHandler) Update(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req service.UserSaveReq
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

// Delete DELETE /api/v1/users/:id
func (h *UserHandler) Delete(c *gin.Context) {
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

// SetStatus PUT /api/v1/users/:id/status
func (h *UserHandler) SetStatus(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req service.UserSetStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: status 必须为 1(启用)或 2(停用)"))
		return
	}
	if err := h.Svc.SetStatus(c, id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ResetPassword PUT /api/v1/users/:id/password
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req service.UserResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 密码至少 8 位"))
		return
	}
	if err := h.Svc.ResetPassword(c, id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// AssignRoles PUT /api/v1/users/:id/roles
func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var req service.UserAssignRolesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: roleIds 必填"))
		return
	}
	if err := h.Svc.AssignRoles(c, id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Export GET /api/v1/users/export
func (h *UserHandler) Export(c *gin.Context) {
	f, filename, err := h.Svc.Export(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		resp.Fail(c, errs.Internal("生成导出文件失败").WithCause(err))
		return
	}
	defer f.Close()
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}
