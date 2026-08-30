// 登录日志接口的 HTTP 绑定 DTO 与到 Service 输入的显式转换。
package handler

import (
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

// LoginLogListQuery GET /admin-api/login-logs 查询参数。
type LoginLogListQuery struct {
	page.Query
	Username  string `form:"username"`
	Success   *bool  `form:"success"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
}

func (q *LoginLogListQuery) toInput() *service.LoginLogListInput {
	return &service.LoginLogListInput{
		Query:     q.Query,
		Username:  q.Username,
		Success:   q.Success,
		StartTime: q.StartTime,
		EndTime:   q.EndTime,
	}
}
