// Package router 组装 HTTP 路由与中间件。
package router

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/config"
	mw "github.com/hesunfly/hesunfly-admin-go/server/internal/platform/middleware"
	sysauth "github.com/hesunfly/hesunfly-admin-go/server/internal/system/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/audit"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/file"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/loginlog"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/menu"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/role"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/user"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

// Deps 路由所需依赖。
type Deps struct {
	Cfg      *config.Config
	DB       *gorm.DB
	Logger   *slog.Logger
	JWT      *auth.Manager
	Recorder *audit.Recorder
}

// New 构建 gin 引擎。
func New(d *Deps) *gin.Engine {
	gin.SetMode(d.Cfg.Server.Mode)
	r := gin.New()
	r.Use(mw.RequestID(), mw.AccessLog(d.Logger), mw.Recovery(d.Logger))
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

	// 公开文件静态访问(存储由 Local Storage 管理,http.Dir 自带防穿越)
	if local, ok := fileStorageDir(d); ok && strings.HasPrefix(d.Cfg.Upload.PublicURL, "/") {
		r.Static(d.Cfg.Upload.PublicURL, local)
	}

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

	authn := &mw.Authn{DB: d.DB, JWT: d.JWT}

	api := r.Group("/api/v1")
	api.Use(d.Recorder.Middleware())

	// 认证:公开接口
	authSvc := sysauth.NewService(d.DB, d.JWT, d.Logger)
	authHandler := sysauth.NewHandler(authSvc)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)

	// 以下接口需要登录
	authed := api.Group("")
	authed.Use(authn.Require())
	authed.POST("/auth/logout", authHandler.Logout)
	authed.GET("/auth/me", authHandler.Me)

	// 用户管理
	userSvc := user.NewService(d.DB)
	userHandler := user.NewHandler(userSvc)
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
	roleSvc := role.NewService(d.DB)
	roleHandler := role.NewHandler(roleSvc)
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
	menuSvc := menu.NewService(d.DB)
	menuHandler := menu.NewHandler(menuSvc)
	mg := authed.Group("/menus")
	{
		mg.GET("", authn.RequirePerm("system:menu:list"), menuHandler.List)
		mg.GET("/tree", authn.RequirePerm("system:menu:list"), menuHandler.Tree)
		mg.POST("", authn.RequirePerm("system:menu:create"), menuHandler.Create)
		mg.PUT("/:id", authn.RequirePerm("system:menu:update"), menuHandler.Update)
		mg.DELETE("/:id", authn.RequirePerm("system:menu:delete"), menuHandler.Delete)
	}

	// 日志
	auditSvc := audit.NewService(d.DB)
	auditHandler := audit.NewHandler(auditSvc)
	authed.GET("/audit-logs", authn.RequirePerm("system:auditlog:list"), auditHandler.List)

	llSvc := loginlog.NewService(d.DB)
	llHandler := loginlog.NewHandler(llSvc)
	authed.GET("/login-logs", authn.RequirePerm("system:loginlog:list"), llHandler.List)

	// 文件
	storage, err := file.NewLocal(d.Cfg.Upload.Dir)
	if err != nil {
		d.Logger.Error("初始化本地存储失败", "error", err)
		return r
	}
	fileSvc := file.NewService(d.DB, storage, d.Cfg.Upload.MaxSizeMB)
	fileHandler := file.NewHandler(fileSvc, d.Cfg.Upload.PublicURL)
	fg := authed.Group("/files")
	{
		fg.POST("", authn.RequirePerm("system:file:upload"), fileHandler.Upload)
		fg.GET("", authn.RequirePerm("system:file:list"), fileHandler.List)
		fg.DELETE("/:id", authn.RequirePerm("system:file:delete"), fileHandler.Delete)
		fg.GET("/:id/download", fileHandler.Download)
	}

	return r
}

func fileStorageDir(d *Deps) (string, bool) {
	abs, err := filepath.Abs(d.Cfg.Upload.Dir)
	if err != nil {
		return "", false
	}
	if _, err := os.Stat(abs); err != nil {
		_ = os.MkdirAll(abs, 0o755)
	}
	return abs, true
}
