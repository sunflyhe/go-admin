// 操作审计查询与清理服务。审计中间件(Recorder、请求脱敏)在 internal/middleware/audit.go,
// 它是 HTTP 边界组件,允许依赖 Gin;本文件保持无 Gin 依赖。
// 记录范围:写请求(POST/PUT/DELETE/PATCH);密码、Token、手机号、身份证等敏感字段一律脱敏;
// 响应只记录状态码与错误摘要,不保存完整响应体。
package service

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
)

// truncate 供登录日志等服务截断字符串使用。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// CleanupCleaner 按保留天数定期清理审计日志与登录日志。
func CleanupCleaner(db *gorm.DB, retentionDays int, logger *slog.Logger) (stop func()) {
	if retentionDays <= 0 {
		logger.Info("审计日志保留策略: 永久保留")
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				deadline := time.Now().AddDate(0, 0, -retentionDays)
				if err := db.WithContext(ctx).Where("created_at < ?", deadline).Delete(&model.SysAuditLog{}).Error; err != nil {
					logger.Error("清理审计日志失败", "error", err)
				}
				if err := db.WithContext(ctx).Where("created_at < ?", deadline).Delete(&model.SysLoginLog{}).Error; err != nil {
					logger.Error("清理登录日志失败", "error", err)
				}
			}
		}
	}()
	return cancel
}

type AuditService struct{ DB *gorm.DB }

func NewAuditService(db *gorm.DB) *AuditService { return &AuditService{DB: db} }

type AuditListInput struct {
	page.Query
	Username  string ``
	Path      string ``
	Status    int    ``
	StartTime string ``
	EndTime   string ``
}

func (s *AuditService) List(ctx context.Context, req *AuditListInput) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(ctx).Model(&model.SysAuditLog{})
	if req.Username != "" {
		q = q.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.Path != "" {
		q = q.Where("path LIKE ?", "%"+req.Path+"%")
	}
	if req.Status != 0 {
		q = q.Where("status = ?", req.Status)
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
		return nil, errs.Internal("查询审计日志失败").WithCause(err)
	}
	var logs []model.SysAuditLog
	if err := q.Order("id DESC").Offset(req.Offset()).Limit(req.PageSize).Find(&logs).Error; err != nil {
		return nil, errs.Internal("查询审计日志失败").WithCause(err)
	}
	return &page.Result{List: logs, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}
