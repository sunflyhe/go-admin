// Package loginlog 登录日志查询域。
package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/page"
)

type LoginLogService struct{ DB *gorm.DB }

func NewLoginLogService(db *gorm.DB) *LoginLogService { return &LoginLogService{DB: db} }

type LoginLogListInput struct {
	page.Query
	Username  string ``
	Success   *bool  ``
	StartTime string ``
	EndTime   string ``
}

func (s *LoginLogService) List(ctx context.Context, req *LoginLogListInput) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(ctx).Model(&model.SysLoginLog{})
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
