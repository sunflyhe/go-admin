// CORS 跨域支持:前端与 API 跨域直连部署(如前端上 CDN、API 独立域名)时启用。
// 同源部署(推荐形态,经同域 Nginx 反代)保持 allowedOrigins 为空,不输出任何 CORS 头。
// 认证走 Authorization 请求头而非 Cookie,因此无需 Allow-Credentials。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CORS struct {
	allowed  map[string]bool
	wildcard bool
}

// NewCORS allowedOrigins 为精确来源列表(含 scheme,如 https://admin.example.com),
// "*" 表示放行任意来源。传空列表时中间件退化为直通,不产生任何 CORS 响应头。
func NewCORS(allowedOrigins []string) *CORS {
	c := &CORS{allowed: make(map[string]bool, len(allowedOrigins))}
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			c.wildcard = true
		}
		c.allowed[o] = true
	}
	return c
}

func (c *CORS) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin != "" && (c.wildcard || c.allowed[origin]) {
			// 回显具体来源而非 *,便于将来需要携带凭据时无缝收紧
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Vary", "Origin")
			if ctx.Request.Method == http.MethodOptions && ctx.GetHeader("Access-Control-Request-Method") != "" {
				ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
				ctx.Header("Access-Control-Max-Age", "86400")
				ctx.AbortWithStatus(http.StatusNoContent)
				return
			}
		}
		ctx.Next()
	}
}
