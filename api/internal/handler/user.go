// 用户管理控制器。
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/resp"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/validate"
)

type UserHandler struct {
	Svc     *service.UserService
	Profile *service.ProfileService
}

func NewUserHandler(svc *service.UserService, profile *service.ProfileService) *UserHandler {
	return &UserHandler{Svc: svc, Profile: profile}
}

// List GET /admin-api/users
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

// Create POST /admin-api/users
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

// Update PUT /admin-api/users/:id
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

// UploadAvatar PUT /admin-api/users/:id/avatar (multipart 字段名 file)
// 管理员替指定用户更换头像,复用个人中心头像上传链路;权限沿用 system:user:update。
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	id, err := validate.PathID(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
	in, closer, ok := parseAvatarUpload(c)
	if !ok {
		return
	}
	defer closer.Close()

	url, err := h.Profile.UploadAvatarFor(c.Request.Context(), actor, id, *in)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, AvatarResponse{Avatar: url})
}

// Delete DELETE /admin-api/users/:id
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

// SetStatus PUT /admin-api/users/:id/status
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

// ResetPassword PUT /admin-api/users/:id/password
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

// AssignRoles PUT /admin-api/users/:id/roles
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

// Export GET /admin-api/users/export
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
