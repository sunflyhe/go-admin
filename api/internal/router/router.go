// 路由装配入口:引擎与全局中间件、健康检查、多端 SPA 静态托管在此完成,
// 具体路由按端拆分到同包的 admin.go / api.go(新端再加新文件)。
// 端前缀约定:app 端接口 /api/*,admin 端接口 /admin-api/*,文件 /files/*。
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
	engine      *gin.Engine
	authn       *middleware.Authn
	appApi      *gin.RouterGroup // /api:app 端与开放接口,含审计记录中间件
	adminApi    *gin.RouterGroup // /admin-api:admin 端,含审计记录中间件
	adminAuthed *gin.RouterGroup // adminApi + Require():登录后可访问
	fileSvc     *service.FileService
	profileSvc  *service.ProfileService
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
	r.Use(
		middleware.RequestID(),
		middleware.AccessLog(d.Logger),
		middleware.Recovery(d.Logger),
		// CORS:默认(未配置来源)为直通,不产生任何跨域头;跨域直连部署时在配置中填写白名单
		middleware.NewCORS(d.Cfg.Server.CORSAllowedOrigins).Middleware(),
	)
	r.MaxMultipartMemory = 8 << 20

	registerHealthRoutes(r, d.DB)

	// 多端 SPA 静态托管(可选):app 挂根 /,admin 挂 /admin/
	mountWebDirs(r, d.Cfg.Server.WebDirs)

	// 本地存储与文件服务:文件接口与个人中心头像上传共用同一实例,避免两处初始化漂移。
	storage, err := service.NewLocal(d.Cfg.Upload.Dir)
	if err != nil {
		d.Logger.Error("初始化本地存储失败", "error", err)
		return r
	}
	fileSvc := service.NewFileService(d.DB, storage, d.Cfg.Upload.MaxSizeMB)

	authn := &middleware.Authn{DB: d.DB, JWT: d.JWT}

	appApi := r.Group("/api")
	appApi.Use(d.Recorder.Middleware())

	adminApi := r.Group("/admin-api")
	adminApi.Use(d.Recorder.Middleware())
	adminAuthed := adminApi.Group("")
	adminAuthed.Use(authn.Require())

	w := &routeWires{
		Deps:        d,
		engine:      r,
		authn:       authn,
		appApi:      appApi,
		adminApi:    adminApi,
		adminAuthed: adminAuthed,
		fileSvc:     fileSvc,
		profileSvc:  service.NewProfileService(d.DB, fileSvc, d.Cfg.Upload.PublicURL),
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

// apiPrefixes 由后端自身服务的路径前缀:SPA 回退不得吞掉这些前缀下的 404。
var apiPrefixes = []string{"/api/", "/admin-api/", "/files/"}

// mountWebDirs 多端 SPA 静态托管:
//   - "app" 目录挂根路径 /:未命中 API 与真实文件时回退 app 的 index.html(history 路由刷新)
//   - "admin" 目录挂 /admin/:未命中真实文件时回退 admin 自己的 index.html
//
// 目录不存在时静默跳过该端,便于只部署其中一个前端。
func mountWebDirs(r *gin.Engine, dirs map[string]string) {
	appDir := resolveDir(dirs["app"])
	adminDir := resolveDir(dirs["admin"])

	if appDir != "" {
		r.NoRoute(func(c *gin.Context) {
			for _, prefix := range apiPrefixes {
				if strings.HasPrefix(c.Request.URL.Path, prefix) {
					c.JSON(http.StatusNotFound, gin.H{"code": errs.CodeNotFound, "message": "接口不存在", "data": nil})
					return
				}
			}
			// /admin 段属于 admin 端页面;app 的根回退不得接管它(即使 admin 端未托管也保持 404 语义)
			if c.Request.URL.Path == "/admin" || strings.HasPrefix(c.Request.URL.Path, "/admin/") {
				c.JSON(http.StatusNotFound, gin.H{"code": errs.CodeNotFound, "message": "页面不存在", "data": nil})
				return
			}
			serveSPA(c, appDir)
		})
	}

	if adminDir != "" {
		// gin 的 /*path 通配即覆盖 /admin/ 本身;/admin(无斜杠)由 gin 自动 301 到 /admin/
		adminSPA := func(c *gin.Context) {
			rel := c.Param("path")
			if rel == "/" || rel == "" {
				serveSPA(c, adminDir)
				return
			}
			full := filepath.Join(adminDir, filepath.Clean(rel))
			if info, err := os.Stat(full); err == nil && !info.IsDir() {
				c.File(full)
				return
			}
			serveSPA(c, adminDir)
		}
		r.GET("/admin/*path", adminSPA)
		r.HEAD("/admin/*path", adminSPA)
	}
}

// serveSPA 输出目录内与请求路径匹配的静态文件,未命中回退该目录的 index.html。
// index.html 强制 no-cache:浏览器必须每次协商重验,否则发版后用户会长期停留在
// 旧 bundle 上(启发式缓存按 Last-Modified 计算有效期,导致"改了代码看不到变化");
// 带 hash 的静态资源文件名即版本,可放心使用默认缓存。
func serveSPA(c *gin.Context, dir string) {
	full := filepath.Join(dir, filepath.Clean("/"+c.Request.URL.Path))
	if info, err := os.Stat(full); err == nil && !info.IsDir() {
		if strings.HasSuffix(full, "index.html") {
			c.Header("Cache-Control", "no-cache")
		}
		c.File(full)
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.File(filepath.Join(dir, "index.html"))
}

// resolveDir 返回存在的目录绝对路径,空串或不存在返回空串。
func resolveDir(dir string) string {
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return ""
	}
	return abs
}

// handlerForFile 公开下载与文件管理共用一个 handler 实例。
func (w *routeWires) fileHandler() *handler.FileHandler {
	return handler.NewFileHandler(w.fileSvc, w.Deps.Cfg.Upload.PublicURL, w.Deps.Cfg.Upload.MaxSizeMB)
}
