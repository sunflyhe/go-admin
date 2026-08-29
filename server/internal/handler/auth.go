// 认证控制器:登录、刷新、退出、当前用户,以及个人中心(改资料/改密码/改头像)。
// Handler 是唯一依赖 Gin 的 HTTP 层:绑定请求体、读取 IP/User-Agent、解析 multipart、
// 转换为 service.LoginMeta / service.Actor / service.ProfileInput 后调用 Service。
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
)

type AuthHandler struct {
	Svc *service.AuthService
	// Profile 个人中心:与认证同属 /auth/* 命名空间,共用一个控制器。
	Profile *service.ProfileService
}

func NewAuthHandler(svc *service.AuthService, profile *service.ProfileService) *AuthHandler {
	return &AuthHandler{Svc: svc, Profile: profile}
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 用户名和密码必填"))
		return
	}
	result, err := h.Svc.Login(c.Request.Context(), req.Username, req.Password, service.LoginMeta{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Refresh POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: refreshToken 必填"))
		return
	}
	result, err := h.Svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Logout POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
	if err := h.Svc.Logout(c.Request.Context(), actor); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Me GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
	result, err := h.Svc.Me(c.Request.Context(), actor)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// UpdateProfile PUT /api/v1/auth/profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 昵称/邮箱/手机/签名长度超限"))
		return
	}
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
	if err := h.Profile.UpdateProfile(c.Request.Context(), actor, req.toInput()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ChangePassword POST /api/v1/auth/password
// 成功后当前 access token 立即失效,前端必须清凭据并跳登录页。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 原密码与新密码必填,新密码至少 8 位"))
		return
	}
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
	if err := h.Svc.ChangePassword(c.Request.Context(), actor, req.OldPassword, req.NewPassword); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// UploadAvatar POST /api/v1/auth/avatar (multipart 字段名 file)
func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	actor, ok := middlewareActor(c)
	if !ok {
		resp.Fail(c, errs.Unauthorized("未登录"))
		return
	}
	// 留出 multipart 边界开销的余量,否则刚好 2MB 的图片会被误判超限
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxAvatarBytes+64<<10)
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, errs.InvalidParam(fmt.Sprintf("上传失败: 未找到 file 字段或文件超过 %dMB 限制", service.MaxAvatarBytes>>20)))
		return
	}
	src, err := fh.Open()
	if err != nil {
		resp.Fail(c, errs.Internal("读取头像失败").WithCause(err))
		return
	}
	defer src.Close()

	url, err := h.Profile.UploadAvatar(c.Request.Context(), actor, service.AvatarInput{
		FileName: fh.Filename,
		Size:     fh.Size,
		Content:  src,
	})
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, AvatarResponse{Avatar: url})
}
