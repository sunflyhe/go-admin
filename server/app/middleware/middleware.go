// Package middleware 提供 Gin 中间件:请求 ID、访问日志、Panic 恢复、认证与权限校验。
package middleware

import (
	"log/slog"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	CtxRequestID = "requestID"
	CtxClaims    = "authClaims"
	CtxUser      = "authUser"
)

// RequestID 为每个请求生成或透传 X-Request-ID,并写入响应头与上下文。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(CtxRequestID, id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}

// AccessLog 输出结构化访问日志:请求 ID、方法、路径、状态码、耗时、IP、UA 与错误信息。
func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()
		attrs := []any{
			"requestID", c.GetString(CtxRequestID),
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", c.Writer.Status(),
			"latencyMs", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"userAgent", c.Request.UserAgent(),
		}
		if errMsg := c.Errors.Last(); errMsg != nil {
			attrs = append(attrs, "error", errMsg.Error())
			logger.Error("access", attrs...)
			return
		}
		logger.Info("access", attrs...)
	}
}

// Recovery 捕获 panic,记录带请求 ID 的错误日志并返回统一 500 响应。
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"requestID", c.GetString(CtxRequestID),
					"path", c.Request.URL.Path,
					"panic", r,
					"stack", string(stack(2048)),
				)
				c.AbortWithStatusJSON(500, gin.H{"code": 50000, "message": "internal server error", "data": nil})
			}
		}()
		c.Next()
	}
}

func stack(max int) []byte {
	buf := make([]byte, max)
	n := runtime.Stack(buf, false)
	return buf[:n]
}
