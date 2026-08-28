// Package audit 操作审计域。
// 记录范围:写请求(POST/PUT/DELETE/PATCH);密码、Token、手机号、身份证等敏感字段一律脱敏;
// 响应只记录状态码与错误摘要,不保存完整响应体。
// 落库异步执行,不阻塞请求;提供按保留天数清理的策略。
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/app/middleware"
	"github.com/hesunfly/hesunfly-admin-go/server/app/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
)

const (
	skipHeader      = "X-Audit-Skip"
	maxBodyCapture  = 2 * 1024 // 请求/响应摘要捕获上限
	maxChannelQueue = 1024
)

// sensitiveKeys 值将被替换为 ***。匹配按小写包含判断。
var sensitiveKeys = []string{
	"password", "passwd", "pwd", "token", "secret", "authorization",
	"phone", "mobile", "idcard", "id_card", "email",
}

func maskValue(key string) bool {
	key = strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(key, s) {
			return true
		}
	}
	return false
}

// MaskJSON 对 JSON 字符串中的敏感字段做脱敏;无法解析为 JSON 对象的请求体不落库,返回占位符,
// 避免敏感信息绕过脱敏(非 application/json 的请求本就不会进入审计)。
func MaskJSON(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return "{...}"
	}
	maskMap(m)
	out, err := json.Marshal(m)
	if err != nil {
		return "{...}"
	}
	return truncate(string(out), maxBodyCapture)
}

func maskMap(m map[string]interface{}) {
	for k, v := range m {
		if maskValue(k) {
			m[k] = "***"
			continue
		}
		if sub, ok := v.(map[string]interface{}); ok {
			maskMap(sub)
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Recorder 异步审计写入器。
type Recorder struct {
	db   *gorm.DB
	ch   chan model.SysAuditLog
	wg   sync.WaitGroup
	stop chan struct{}
}

func NewRecorder(db *gorm.DB) *Recorder {
	r := &Recorder{db: db, ch: make(chan model.SysAuditLog, maxChannelQueue), stop: make(chan struct{})}
	r.wg.Add(1)
	go r.loop()
	return r
}

func (r *Recorder) loop() {
	defer r.wg.Done()
	for {
		select {
		case entry := <-r.ch:
			if err := r.db.Create(&entry).Error; err != nil {
				// 日志失败不影响业务,仅记录错误
				slog.Error("写入审计日志失败", "error", err)
			}
		case <-r.stop:
			return
		}
	}
}

// Close 停止写入器并等待队列排空。
func (r *Recorder) Close() {
	close(r.stop)
	r.wg.Wait()
}

// Middleware 审计中间件:记录操作者、接口、方法、状态码、耗时、IP、UA 与脱敏请求摘要。
// 通过设置请求头 X-Audit-Skip: 1 可跳过(如文件上传等无法解析体或已单独记录的场景)。
func (r *Recorder) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case "POST", "PUT", "DELETE", "PATCH":
		default:
			c.Next()
			return
		}
		if c.GetHeader(skipHeader) != "" || c.ContentType() != "application/json" {
			c.Next()
			return
		}
		start := time.Now()
		var reqSummary string
		if c.Request.Body != nil {
			buf := &bytes.Buffer{}
			_, _ = buf.ReadFrom(io.LimitReader(c.Request.Body, maxBodyCapture))
			reqSummary = MaskJSON(buf.String())
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf.Bytes()), c.Request.Body))
		}
		c.Next()
		user, _ := middleware.CurrentUser(c)
		entry := model.SysAuditLog{
			Method:         c.Request.Method,
			Path:           c.Request.URL.Path,
			Status:         c.Writer.Status(),
			LatencyMs:      time.Since(start).Milliseconds(),
			IP:             c.ClientIP(),
			UserAgent:      truncate(c.Request.UserAgent(), 255),
			RequestSummary: reqSummary,
		}
		if user != nil {
			entry.UserID, entry.Username = user.ID, user.Username
		}
		// 失败请求记录归一化错误信息作为响应摘要
		if c.Writer.Status() >= 400 {
			if e := c.Errors.Last(); e != nil {
				entry.ResponseSummary = truncate(e.Error(), 512)
			}
		}
		select {
		case r.ch <- entry:
		default:
			// 队列满时丢弃并记录,保证接口可用性优先
			slog.Warn("审计队列已满,丢弃审计记录", "path", entry.Path)
		}
	}
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

// ---- 查询服务与 Handler ----

type AuditService struct{ DB *gorm.DB }

func NewAuditService(db *gorm.DB) *AuditService { return &AuditService{DB: db} }

type AuditListReq struct {
	page.Query
	Username  string `form:"username"`
	Path      string `form:"path"`
	Status    int    `form:"status"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
}

func (s *AuditService) List(c *gin.Context, req *AuditListReq) (*page.Result, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	q := s.DB.WithContext(c).Model(&model.SysAuditLog{})
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
