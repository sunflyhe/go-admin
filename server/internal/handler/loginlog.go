// 登录日志控制器。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
)

type LoginLogHandler struct {
	Svc *service.LoginLogService
}

func NewLoginLogHandler(svc *service.LoginLogService) *LoginLogHandler {
	return &LoginLogHandler{Svc: svc}
}

// List GET /api/v1/login-logs
func (h *LoginLogHandler) List(c *gin.Context) {
	var req service.LoginLogListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("分页参数错误"))
		return
	}
	result, err := h.Svc.List(c, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}
