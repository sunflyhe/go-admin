// 路由注册:对应 Hyperf 的 config/routes.php,集中声明 API 路由与中间件。
package router

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/config"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/handler"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/middleware"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	pkgauth "github.com/hesunfly/hesunfly-admin-go/api/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
)

// Deps 路由所需依赖。
type Deps struct {
	Cfg      *config.Config
	DB       *gorm.DB
	Logger   *slog.Logger
	JWT      *pkgauth.Manager
	Recorder *middleware.Recorder
}

// New 构建 gin 引擎。
func New(d *Deps) *gin.Engine {
	gin.SetMode(d.Cfg.Server.Mode)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.AccessLog(d.Logger), middleware.Recovery(d.Logger))
	r.MaxMultipartMemory = 8 << 20

	// 健康检查:healthz 探活;readyz 探数据库可用性
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		if sqlDB, err := d.DB.DB(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": "db"})
			return
		} else if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": "db"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// 前端静态托管(可选):优先 API,其次静态资源,未命中回退 index.html
	webDir := d.Cfg.Server.WebDir
	if webDir != "" {
		if abs, err := filepath.Abs(webDir); err == nil {
			if _, err := os.Stat(abs); err == nil {
				r.NoRoute(func(c *gin.Context) {
					if strings.HasPrefix(c.Request.URL.Path, "/api") {
						c.JSON(http.StatusNotFound, gin.H{"code": errs.CodeNotFound, "message": "接口不存在", "data": nil})
						return
					}
					full := filepath.Join(abs, filepath.Clean("/"+c.Request.URL.Path))
					if info, err := os.Stat(full); err == nil && !info.IsDir() {
						c.File(full)
						return
					}
					c.File(filepath.Join(abs, "index.html"))
				})
			}
		}
	}

	authn := &middleware.Authn{DB: d.DB, JWT: d.JWT}

	// 本地存储与文件服务:文件接口与个人中心头像上传共用同一实例,避免两处初始化漂移。
	storage, err := service.NewLocal(d.Cfg.Upload.Dir)
	if err != nil {
		d.Logger.Error("初始化本地存储失败", "error", err)
		return r
	}
	fileSvc := service.NewFileService(d.DB, storage, d.Cfg.Upload.MaxSizeMB)

	api := r.Group("/api/v1")
	api.Use(d.Recorder.Middleware())

	// 认证:公开接口
	authSvc := service.NewAuthService(d.DB, d.JWT, d.Logger)
	profileSvc := service.NewProfileService(d.DB, fileSvc, d.Cfg.Upload.PublicURL)
	authHandler := handler.NewAuthHandler(authSvc, profileSvc)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)

	// 以下接口需要登录
	authed := api.Group("")
	authed.Use(authn.Require())
	authed.POST("/auth/logout", authHandler.Logout)
	authed.GET("/auth/me", authHandler.Me)
	// 个人中心:只作用于登录者自身,入参里没有 id,因此与 logout/me 一样只过 Require()。
	// 头像复用文件存储但不要求 system:file:upload —— 否则普通用户无法设置自己的头像。
	authed.PUT("/auth/profile", authHandler.UpdateProfile)
	authed.POST("/auth/password", authHandler.ChangePassword)
	authed.POST("/auth/avatar", authHandler.UploadAvatar)

	// 用户管理
	userSvc := service.NewUserService(d.DB)
	userHandler := handler.NewUserHandler(userSvc, profileSvc)
	ug := authed.Group("/users")
	{
		ug.GET("", authn.RequirePerm("system:user:list"), userHandler.List)
		ug.GET("/export", authn.RequirePerm("system:user:export"), userHandler.Export)
		ug.POST("", authn.RequirePerm("system:user:create"), userHandler.Create)
		ug.PUT("/:id", authn.RequirePerm("system:user:update"), userHandler.Update)
		ug.PUT("/:id/avatar", authn.RequirePerm("system:user:update"), userHandler.UploadAvatar)
		ug.DELETE("/:id", authn.RequirePerm("system:user:delete"), userHandler.Delete)
		ug.PUT("/:id/status", authn.RequirePerm("system:user:update"), userHandler.SetStatus)
		ug.PUT("/:id/password", authn.RequirePerm("system:user:reset-password"), userHandler.ResetPassword)
		ug.PUT("/:id/roles", authn.RequirePerm("system:user:assign-role"), userHandler.AssignRoles)
	}

	// 角色管理
	roleSvc := service.NewRoleService(d.DB)
	roleHandler := handler.NewRoleHandler(roleSvc)
	rg := authed.Group("/roles")
	{
		rg.GET("", authn.RequirePerm("system:role:list"), roleHandler.List)
		rg.POST("", authn.RequirePerm("system:role:create"), roleHandler.Create)
		rg.PUT("/:id", authn.RequirePerm("system:role:update"), roleHandler.Update)
		rg.DELETE("/:id", authn.RequirePerm("system:role:delete"), roleHandler.Delete)
		rg.GET("/:id/menus", authn.RequirePerm("system:role:list"), roleHandler.Menus)
		rg.PUT("/:id/menus", authn.RequirePerm("system:role:assign-menu"), roleHandler.AssignMenus)
	}

	// 菜单管理
	menuSvc := service.NewMenuService(d.DB)
	menuHandler := handler.NewMenuHandler(menuSvc)
	mg := authed.Group("/menus")
	{
		mg.GET("", authn.RequirePerm("system:menu:list"), menuHandler.List)
		mg.GET("/tree", authn.RequirePerm("system:menu:list"), menuHandler.Tree)
		mg.POST("", authn.RequirePerm("system:menu:create"), menuHandler.Create)
		mg.PUT("/:id", authn.RequirePerm("system:menu:update"), menuHandler.Update)
		mg.DELETE("/:id", authn.RequirePerm("system:menu:delete"), menuHandler.Delete)
	}

	// 日志
	auditSvc := service.NewAuditService(d.DB)
	auditHandler := handler.NewAuditHandler(auditSvc)
	authed.GET("/audit-logs", authn.RequirePerm("system:auditlog:list"), auditHandler.List)

	llSvc := service.NewLoginLogService(d.DB)
	llHandler := handler.NewLoginLogHandler(llSvc)
	authed.GET("/login-logs", authn.RequirePerm("system:loginlog:list"), llHandler.List)

	// 文件（storage 与 fileSvc 已在认证段之前构造,供个人中心头像共用）
	fileHandler := handler.NewFileHandler(fileSvc, d.Cfg.Upload.PublicURL, d.Cfg.Upload.MaxSizeMB)
	fileGroupHandler := handler.NewFileGroupHandler(service.NewFileGroupService(d.DB))
	// 公开文件也通过数据库的 is_public 标记校验后再输出。不能把整个上传目录
	// 直接静态暴露，否则私有文件可按 store_path 绕过鉴权访问。
	if strings.HasPrefix(d.Cfg.Upload.PublicURL, "/") {
		r.GET(d.Cfg.Upload.PublicURL+"/*storePath", fileHandler.PublicDownload)
	}
	fg := authed.Group("/files")
	{
		fg.POST("", authn.RequirePerm("system:file:upload"), fileHandler.Upload)
		fg.GET("", authn.RequirePerm("system:file:list"), fileHandler.List)
		fg.DELETE("/:id", authn.RequirePerm("system:file:delete"), fileHandler.Delete)
		fg.GET("/:id/download", authn.RequirePerm("system:file:list"), fileHandler.Download)
		// 批量操作用固定的集合端点:批量删除走 POST 而不是带请求体的 DELETE
		fg.PUT("/group", authn.RequirePerm("system:file:move"), fileHandler.Move)
		fg.POST("/batch-delete", authn.RequirePerm("system:file:delete"), fileHandler.BatchDelete)
	}
	fgh := authed.Group("/file-groups")
	{
		fgh.GET("", authn.RequirePerm("system:file:list"), fileGroupHandler.List)
		fgh.POST("", authn.RequirePerm("system:filegroup:create"), fileGroupHandler.Create)
		fgh.PUT("/:id", authn.RequirePerm("system:filegroup:update"), fileGroupHandler.Update)
		fgh.DELETE("/:id", authn.RequirePerm("system:filegroup:delete"), fileGroupHandler.Delete)
	}

	// 系统参数
	configHandler := handler.NewConfigHandler(service.NewConfigService(d.DB))
	cg := authed.Group("/configs")
	{
		cg.GET("", authn.RequirePerm("system:config:list"), configHandler.List)
		cg.POST("", authn.RequirePerm("system:config:create"), configHandler.Create)
		cg.PUT("/:id", authn.RequirePerm("system:config:update"), configHandler.Update)
		cg.DELETE("/:id", authn.RequirePerm("system:config:delete"), configHandler.Delete)
	}

	// 字典:类型与子项维护(system:dict:*),业务按键读取 /dict-data 仅需登录
	dictTypeHandler := handler.NewDictTypeHandler(service.NewDictTypeService(d.DB))
	dt := authed.Group("/dict-types")
	{
		dt.GET("", authn.RequirePerm("system:dict:list"), dictTypeHandler.List)
		dt.POST("", authn.RequirePerm("system:dict:create"), dictTypeHandler.Create)
		dt.PUT("/:id", authn.RequirePerm("system:dict:update"), dictTypeHandler.Update)
		dt.DELETE("/:id", authn.RequirePerm("system:dict:delete"), dictTypeHandler.Delete)
		dt.GET("/:id/items", authn.RequirePerm("system:dict:list"), dictTypeHandler.ListItems)
		dt.POST("/:id/items", authn.RequirePerm("system:dict:create"), dictTypeHandler.CreateItem)
	}
	di := authed.Group("/dict-items")
	{
		di.PUT("/:id", authn.RequirePerm("system:dict:update"), dictTypeHandler.UpdateItem)
		di.DELETE("/:id", authn.RequirePerm("system:dict:delete"), dictTypeHandler.DeleteItem)
	}
	authed.GET("/dict-data", dictTypeHandler.DictData)

	// 文章资讯:分类、文章与富文本配图上传
	articleCategoryHandler := handler.NewArticleCategoryHandler(service.NewArticleCategoryService(d.DB))
	acg := authed.Group("/article-categories")
	{
		acg.GET("", authn.RequirePerm("article:category:list"), articleCategoryHandler.List)
		acg.POST("", authn.RequirePerm("article:category:create"), articleCategoryHandler.Create)
		acg.PUT("/:id", authn.RequirePerm("article:category:update"), articleCategoryHandler.Update)
		acg.DELETE("/:id", authn.RequirePerm("article:category:delete"), articleCategoryHandler.Delete)
	}

	articleSvc := service.NewArticleService(d.DB)
	articleHandler := handler.NewArticleHandler(articleSvc, fileSvc, d.Cfg.Upload.PublicURL, d.Cfg.Upload.MaxSizeMB)
	ag := authed.Group("/articles")
	{
		ag.GET("", authn.RequirePerm("article:article:list"), articleHandler.List)
		ag.GET("/:id", authn.RequirePerm("article:article:list"), articleHandler.Get)
		ag.POST("", authn.RequirePerm("article:article:create"), articleHandler.Create)
		ag.PUT("/:id", authn.RequirePerm("article:article:update"), articleHandler.Update)
		ag.DELETE("/:id", authn.RequirePerm("article:article:delete"), articleHandler.Delete)
	}
	// 配图上传独立权限码:能写文章不代表能进文件中心,反之亦然
	authed.POST("/article-images", authn.RequirePerm("article:article:upload-image"), articleHandler.UploadImage)

	return r
}
