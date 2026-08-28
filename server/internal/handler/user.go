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
	var query UserListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		resp.Fail(c, errs.InvalidParam("分页参数错误: pageSize 不能超过 100"))
		return
	}
	result, err := h.Svc.List(c.Request.Context(), query.toInput())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Create POST /api/v1/users
func (h *UserHandler) Create(c *gin.Context) {
	var req UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 用户名必填(3-64 位),密码至少 8 位"))
		return
	}
	result, err := h.Svc.Create(c.Request.Context(), req.toInput())
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
	var req UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误"))
		return
	}
	result, err := h.Svc.Update(c.Request.Context(), id, req.toInput())
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
	if err := h.Svc.Delete(c.Request.Context(), id); err != nil {
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
	var req UserSetStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: status 必须为 1(启用)或 2(停用)"))
		return
	}
	if err := h.Svc.SetStatus(c.Request.Context(), id, &service.UserSetStatusInput{Status: req.Status}); err != nil {
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
	var req UserResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 密码至少 8 位"))
		return
	}
	if err := h.Svc.ResetPassword(c.Request.Context(), id, &service.UserResetPasswordInput{Password: req.Password}); err != nil {
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
	var req UserAssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: roleIds 必填"))
		return
	}
	if err := h.Svc.AssignRoles(c.Request.Context(), id, &service.UserAssignRolesInput{RoleIDs: req.RoleIDs}); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Export GET /api/v1/users/export
func (h *UserHandler) Export(c *gin.Context) {
	f, filename, err := h.Svc.Export(c.Request.Context())
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
