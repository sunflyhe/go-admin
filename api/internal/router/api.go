// app 端(C 端/开放接口)路由:统一挂在 /api/v1/portal 下,与管理端路由完全隔离。
// 公开只读接口不挂权限码;未来若需要 C 端登录态,在此文件内另行设计轻量鉴权,
// 不复用 admin 的 RequirePerm 模型。当前暂无接口,第一个接口落地时在此展开。
package router

func registerApiRoutes(*routeWires) {}
