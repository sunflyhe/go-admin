// 路由装配入口:引擎与全局中间件、健康检查、SPA 静态托管在此完成,
// 具体路由按端拆分到同包的 routes_*.go(routes_admin / routes_api / 未来的新端)。
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

// routeWires 路由注册函数共享的已装配依赖,避免各注册函数重复构造或超长参数列表。
type routeWires struct {
	*Deps
	engine     *gin.Engine
	authn      *middleware.Authn
	api        *gin.RouterGroup // /api/v1,含审计记录中间件
	authed     *gin.RouterGroup // api + Require():登录后可访问
	fileSvc    *service.FileService
	profileSvc *service.ProfileService
}

// New 构建 gin 引擎。
func New(d *Deps) *gin.Engine {
	gin.SetMode(d.Cfg.Server.Mode)
	r := gin.New()
	// 默认不信任任何转发头。部署在反向代理后时，必须显式配置可信代理 IP/CIDR，
	// 否则客户端可伪造 X-Forwarded-For，污染审计、登录日志和 IP 限流判断。
	if err := r.SetTrustedProxies(d.Cfg.Server.TrustedProxies); err != nil {
		d.Logger.Error("配置可信代理失败", "error", err)
		return r
	}
	r.Use(middleware.RequestID(), middleware.AccessLog(d.Logger), middleware.Recovery(d.Logger))
	r.MaxMultipartMemory = 8 << 20

	registerHealthRoutes(r, d.DB)

	// 前端静态托管(可选):优先 API,其次静态资源,未命中回退 index.html
	mountWebDir(r, d.Cfg.Server.WebDir)

	// 本地存储与文件服务:文件接口与个人中心头像上传共用同一实例,避免两处初始化漂移。
	storage, err := service.NewLocal(d.Cfg.Upload.Dir)
	if err != nil {
		d.Logger.Error("初始化本地存储失败", "error", err)
		return r
	}
	fileSvc := service.NewFileService(d.DB, storage, d.Cfg.Upload.MaxSizeMB)

	api := r.Group("/api/v1")
	api.Use(d.Recorder.Middleware())
	authn := &middleware.Authn{DB: d.DB, JWT: d.JWT}
	authed := api.Group("")
	authed.Use(authn.Require())

	w := &routeWires{
		Deps:       d,
		engine:     r,
		authn:      authn,
		api:        api,
		authed:     authed,
		fileSvc:    fileSvc,
		profileSvc: service.NewProfileService(d.DB, fileSvc, d.Cfg.Upload.PublicURL),
	}

	registerAdminRoutes(w)
	registerApiRoutes(w)

	return r
}

// registerHealthRoutes healthz 探活;readyz 探数据库可用性。
func registerHealthRoutes(r *gin.Engine, db *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": "db"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": "db"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}

// mountWebDir 可选的 SPA 静态托管:未匹配 API 与真实文件时回退 index.html(history 路由刷新)。
func mountWebDir(r *gin.Engine, webDir string) {
	if webDir == "" {
		return
	}
	abs, err := filepath.Abs(webDir)
	if err != nil {
		return
	}
	if _, err := os.Stat(abs); err != nil {
		return
	}
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

// handlerForFile 公开下载与文件管理共用一个 handler 实例。
func (w *routeWires) fileHandler() *handler.FileHandler {
	return handler.NewFileHandler(w.fileSvc, w.Deps.Cfg.Upload.PublicURL, w.Deps.Cfg.Upload.MaxSizeMB)
}
