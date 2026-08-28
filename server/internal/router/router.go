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

	"github.com/hesunfly/hesunfly-admin-go/server/internal/config"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/handler"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/middleware"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	pkgauth "github.com/hesunfly/hesunfly-admin-go/server/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
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

	api := r.Group("/api/v1")
	api.Use(d.Recorder.Middleware())

	// 认证:公开接口
	authSvc := service.NewAuthService(d.DB, d.JWT, d.Logger)
	authHandler := handler.NewAuthHandler(authSvc)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)

	// 以下接口需要登录
	authed := api.Group("")
	authed.Use(authn.Require())
	authed.POST("/auth/logout", authHandler.Logout)
	authed.GET("/auth/me", authHandler.Me)

	// 用户管理
	userSvc := service.NewUserService(d.DB)
	userHandler := handler.NewUserHandler(userSvc)
	ug := authed.Group("/users")
	{
		ug.GET("", authn.RequirePerm("system:user:list"), userHandler.List)
		ug.GET("/export", authn.RequirePerm("system:user:export"), userHandler.Export)
		ug.POST("", authn.RequirePerm("system:user:create"), userHandler.Create)
		ug.PUT("/:id", authn.RequirePerm("system:user:update"), userHandler.Update)
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

	// 文件
	storage, err := service.NewLocal(d.Cfg.Upload.Dir)
	if err != nil {
		d.Logger.Error("初始化本地存储失败", "error", err)
		return r
	}
	fileSvc := service.NewFileService(d.DB, storage, d.Cfg.Upload.MaxSizeMB)
	fileHandler := handler.NewFileHandler(fileSvc, d.Cfg.Upload.PublicURL, d.Cfg.Upload.MaxSizeMB)
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
	}

	return r
}
