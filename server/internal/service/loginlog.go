// Package loginlog 登录日志查询域。
package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
)

type LoginLogService struct{ DB *gorm.DB }

func NewLoginLogService(db *gorm.DB) *LoginLogService { return &LoginLogService{DB: db} }

type LoginLogListReq struct {
	page.Query
	Username  string `form:"username"`
	Success   *bool  `form:"success"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
}

func (s *LoginLogService) List(c *gin.Context, req *LoginLogListReq) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(c).Model(&model.SysLoginLog{})
	if req.Username != "" {
		q = q.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.Success != nil {
		q = q.Where("success = ?", *req.Success)
	}
	if req.StartTime != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", req.StartTime, time.Local); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", req.EndTime, time.Local); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errs.Internal("查询登录日志失败").WithCause(err)
	}
	var logs []model.SysLoginLog
	if err := q.Order("id DESC").Offset(req.Offset()).Limit(req.PageSize).Find(&logs).Error; err != nil {
		return nil, errs.Internal("查询登录日志失败").WithCause(err)
	}
	return &page.Result{List: logs, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}
