// Handler 内共用的登录态转换辅助。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/middleware"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
)

// middlewareActor 从认证中间件注入的登录态转换为业务层 Actor。
func middlewareActor(c *gin.Context) (service.Actor, bool) {
	return middleware.ActorFrom(c)
}
