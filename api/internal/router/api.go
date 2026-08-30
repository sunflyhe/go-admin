// app 端(C 端/开放接口)路由:统一挂在 /api/portal 下,与管理端路由完全隔离。
// 公开只读接口不挂权限码;未来若需要 C 端登录态,在此文件内另行设计轻量鉴权,
// 不复用 admin 的 RequirePerm 模型。
package router

import (
	"github.com/hesunfly/hesunfly-admin-go/api/internal/handler"
)

func registerApiRoutes(w *routeWires) {
	portal := w.appApi.Group("/portal")
	portalHandler := handler.NewPortalHandler()

	// 演示接口:portal 组的形态参考,真实业务接口落地时可一并删除
	portal.GET("/demo", portalHandler.Demo)
}
