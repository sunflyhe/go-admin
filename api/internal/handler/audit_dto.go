// 操作审计日志接口的 HTTP 绑定 DTO 与到 Service 输入的显式转换。
package handler

import (
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

// AuditListQuery GET /api/v1/audit-logs 查询参数。
type AuditListQuery struct {
	page.Query
	Username  string `form:"username"`
	Path      string `form:"path"`
	Status    int    `form:"status"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
}

func (q *AuditListQuery) toInput() *service.AuditListInput {
	return &service.AuditListInput{
		Query:     q.Query,
		Username:  q.Username,
		Path:      q.Path,
		Status:    q.Status,
		StartTime: q.StartTime,
		EndTime:   q.EndTime,
	}
}
