// admin 管理端路由:认证、系统管理、内容业务全部在此注册(文件名即端名,与 api.go 平级)。
// 鉴权模型统一为 authed(登录) + RequirePerm(权限码),权限码与菜单种子数据一一对应。
// 路由自上而下线性声明,按序阅读即是完整的后台能力清单。
package router

import (
	"strings"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/handler"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
)

func registerAdminRoutes(w *routeWires) {
	// ---- 认证:登录/刷新公开;logout/me/个人中心仅需登录——
	// 只作用于登录者自身,入参里没有 id,因此与 logout/me 一样只过 Require()。
	// 头像复用文件存储但不要求 system:file:upload,否则普通用户无法设置自己的头像。
	authHandler := handler.NewAuthHandler(service.NewAuthService(w.DB, w.JWT, w.Logger), w.profileSvc)
	w.adminApi.POST("/auth/login", authHandler.Login)
	w.adminApi.POST("/auth/refresh", authHandler.Refresh)
	w.adminAuthed.POST("/auth/logout", authHandler.Logout)
	w.adminAuthed.GET("/auth/me", authHandler.Me)
	w.adminAuthed.PUT("/auth/profile", authHandler.UpdateProfile)
	w.adminAuthed.POST("/auth/password", authHandler.ChangePassword)
	w.adminAuthed.POST("/auth/avatar", authHandler.UploadAvatar)

	// ---- 用户管理
	userHandler := handler.NewUserHandler(service.NewUserService(w.DB), w.profileSvc)
	ug := w.adminAuthed.Group("/users")
	{
		ug.GET("", w.authn.RequirePerm("system:user:list"), userHandler.List)
		ug.GET("/export", w.authn.RequirePerm("system:user:export"), userHandler.Export)
		ug.POST("", w.authn.RequirePerm("system:user:create"), userHandler.Create)
		ug.PUT("/:id", w.authn.RequirePerm("system:user:update"), userHandler.Update)
		ug.PUT("/:id/avatar", w.authn.RequirePerm("system:user:update"), userHandler.UploadAvatar)
		ug.DELETE("/:id", w.authn.RequirePerm("system:user:delete"), userHandler.Delete)
		ug.PUT("/:id/status", w.authn.RequirePerm("system:user:update"), userHandler.SetStatus)
		ug.PUT("/:id/password", w.authn.RequirePerm("system:user:reset-password"), userHandler.ResetPassword)
		ug.PUT("/:id/roles", w.authn.RequirePerm("system:user:assign-role"), userHandler.AssignRoles)
	}

	// ---- 角色管理
	roleHandler := handler.NewRoleHandler(service.NewRoleService(w.DB))
	rg := w.adminAuthed.Group("/roles")
	{
		rg.GET("", w.authn.RequirePerm("system:role:list"), roleHandler.List)
		rg.POST("", w.authn.RequirePerm("system:role:create"), roleHandler.Create)
		rg.PUT("/:id", w.authn.RequirePerm("system:role:update"), roleHandler.Update)
		rg.DELETE("/:id", w.authn.RequirePerm("system:role:delete"), roleHandler.Delete)
		rg.GET("/:id/menus", w.authn.RequirePerm("system:role:list"), roleHandler.Menus)
		rg.PUT("/:id/menus", w.authn.RequirePerm("system:role:assign-menu"), roleHandler.AssignMenus)
	}

	// ---- 菜单管理
	menuHandler := handler.NewMenuHandler(service.NewMenuService(w.DB))
	mg := w.adminAuthed.Group("/menus")
	{
		mg.GET("", w.authn.RequirePerm("system:menu:list"), menuHandler.List)
		mg.GET("/tree", w.authn.RequirePerm("system:menu:list"), menuHandler.Tree)
		mg.POST("", w.authn.RequirePerm("system:menu:create"), menuHandler.Create)
		mg.PUT("/:id", w.authn.RequirePerm("system:menu:update"), menuHandler.Update)
		mg.DELETE("/:id", w.authn.RequirePerm("system:menu:delete"), menuHandler.Delete)
	}

	// ---- 日志:审计与登录日志,只读,各自独立权限码
	auditHandler := handler.NewAuditHandler(service.NewAuditService(w.DB))
	w.adminAuthed.GET("/audit-logs", w.authn.RequirePerm("system:auditlog:list"), auditHandler.List)

	llHandler := handler.NewLoginLogHandler(service.NewLoginLogService(w.DB))
	w.adminAuthed.GET("/login-logs", w.authn.RequirePerm("system:loginlog:list"), llHandler.List)

	// ---- 文件中心:公开文件也通过数据库 is_public 标记校验后再输出。
	// 不能把整个上传目录直接静态暴露,否则私有文件可按 store_path 绕过鉴权访问。
	fileHandler := w.fileHandler()
	fileGroupHandler := handler.NewFileGroupHandler(service.NewFileGroupService(w.DB))
	if strings.HasPrefix(w.Cfg.Upload.PublicURL, "/") {
		w.engine.GET(w.Cfg.Upload.PublicURL+"/*storePath", fileHandler.PublicDownload)
	}

	fg := w.adminAuthed.Group("/files")
	{
		fg.POST("", w.authn.RequirePerm("system:file:upload"), fileHandler.Upload)
		fg.GET("", w.authn.RequirePerm("system:file:list"), fileHandler.List)
		fg.DELETE("/:id", w.authn.RequirePerm("system:file:delete"), fileHandler.Delete)
		fg.GET("/:id/download", w.authn.RequirePerm("system:file:list"), fileHandler.Download)
		// 批量操作用固定的集合端点:批量删除走 POST 而不是带请求体的 DELETE
		fg.PUT("/group", w.authn.RequirePerm("system:file:move"), fileHandler.Move)
		fg.POST("/batch-delete", w.authn.RequirePerm("system:file:delete"), fileHandler.BatchDelete)
	}
	fgh := w.adminAuthed.Group("/file-groups")
	{
		fgh.GET("", w.authn.RequirePerm("system:file:list"), fileGroupHandler.List)
		fgh.POST("", w.authn.RequirePerm("system:filegroup:create"), fileGroupHandler.Create)
		fgh.PUT("/:id", w.authn.RequirePerm("system:filegroup:update"), fileGroupHandler.Update)
		fgh.DELETE("/:id", w.authn.RequirePerm("system:filegroup:delete"), fileGroupHandler.Delete)
	}

	// ---- 系统参数
	configHandler := handler.NewConfigHandler(service.NewConfigService(w.DB))
	cg := w.adminAuthed.Group("/configs")
	{
		cg.GET("", w.authn.RequirePerm("system:config:list"), configHandler.List)
		cg.POST("", w.authn.RequirePerm("system:config:create"), configHandler.Create)
		cg.PUT("/:id", w.authn.RequirePerm("system:config:update"), configHandler.Update)
		cg.DELETE("/:id", w.authn.RequirePerm("system:config:delete"), configHandler.Delete)
	}

	// ---- 字典:类型与子项维护走 system:dict:*;
	// 业务侧按键读取 /dict-data 仅需登录(任意登录用户都可能用到字典数据)。
	dictTypeHandler := handler.NewDictTypeHandler(service.NewDictTypeService(w.DB))
	dt := w.adminAuthed.Group("/dict-types")
	{
		dt.GET("", w.authn.RequirePerm("system:dict:list"), dictTypeHandler.List)
		dt.POST("", w.authn.RequirePerm("system:dict:create"), dictTypeHandler.Create)
		dt.PUT("/:id", w.authn.RequirePerm("system:dict:update"), dictTypeHandler.Update)
		dt.DELETE("/:id", w.authn.RequirePerm("system:dict:delete"), dictTypeHandler.Delete)
		dt.GET("/:id/items", w.authn.RequirePerm("system:dict:list"), dictTypeHandler.ListItems)
		dt.POST("/:id/items", w.authn.RequirePerm("system:dict:create"), dictTypeHandler.CreateItem)
	}
	di := w.adminAuthed.Group("/dict-items")
	{
		di.PUT("/:id", w.authn.RequirePerm("system:dict:update"), dictTypeHandler.UpdateItem)
		di.DELETE("/:id", w.authn.RequirePerm("system:dict:delete"), dictTypeHandler.DeleteItem)
	}
	w.adminAuthed.GET("/dict-data", dictTypeHandler.DictData)

	// ---- 文章资讯:分类、文章与富文本配图上传,权限码前缀 article:*
	categoryHandler := handler.NewArticleCategoryHandler(service.NewArticleCategoryService(w.DB))
	acg := w.adminAuthed.Group("/article-categories")
	{
		acg.GET("", w.authn.RequirePerm("article:category:list"), categoryHandler.List)
		acg.POST("", w.authn.RequirePerm("article:category:create"), categoryHandler.Create)
		acg.PUT("/:id", w.authn.RequirePerm("article:category:update"), categoryHandler.Update)
		acg.DELETE("/:id", w.authn.RequirePerm("article:category:delete"), categoryHandler.Delete)
	}

	articleHandler := handler.NewArticleHandler(
		service.NewArticleService(w.DB),
		w.fileSvc,
		w.Cfg.Upload.PublicURL,
		w.Cfg.Upload.MaxSizeMB,
	)
	ag := w.adminAuthed.Group("/articles")
	{
		ag.GET("", w.authn.RequirePerm("article:article:list"), articleHandler.List)
		ag.GET("/:id", w.authn.RequirePerm("article:article:list"), articleHandler.Get)
		ag.POST("", w.authn.RequirePerm("article:article:create"), articleHandler.Create)
		ag.PUT("/:id", w.authn.RequirePerm("article:article:update"), articleHandler.Update)
		ag.DELETE("/:id", w.authn.RequirePerm("article:article:delete"), articleHandler.Delete)
	}
	// 配图上传独立权限码:能写文章不代表能进文件中心,反之亦然
	w.adminAuthed.POST("/article-images", w.authn.RequirePerm("article:article:upload-image"), articleHandler.UploadImage)
}
