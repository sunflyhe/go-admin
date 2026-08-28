// 认证控制器:登录、刷新、退出、当前用户。
// Handler 是唯一依赖 Gin 的 HTTP 层:绑定请求体、读取 IP/User-Agent、
// 转换为 service.LoginMeta / service.Actor 后调用 Service。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
)

type AuthHandler struct {
	Svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{Svc: svc}
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
