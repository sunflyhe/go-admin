// 操作审计日志控制器。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/resp"
)

type AuditHandler struct {
	Svc *service.AuditService
}

func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{Svc: svc}
}

// List GET /api/v1/audit-logs
func (h *AuditHandler) List(c *gin.Context) {
	var query AuditListQuery
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
