// 路由层集成测试:验证不同角色的服务端权限差异(P1 验收项)。
package router

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/config"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/audit"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/model"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/testutil"
)

func newRouterEnv(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	testutil.SeedSuperAdmin(t, db)

	// 审计员角色(id=2)只授予“登录日志”页面
	role := model.SysRole{ID: 2, Name: "审计员", Code: "auditor", Status: model.StatusEnabled}
	db.Create(&role)
	pageMenu := model.SysMenu{ID: 140, ParentID: 100, Name: "登录日志", Type: model.MenuTypePage, Path: "/system/login-log", Permission: "system:loginlog:list", Status: model.StatusEnabled}
	dirMenu := model.SysMenu{ID: 100, ParentID: 0, Name: "系统管理", Type: model.MenuTypeDirectory, Path: "/system", Status: model.StatusEnabled}
	db.Create(&dirMenu)
	db.Create(&pageMenu)
	db.Create(&model.SysRoleMenu{RoleID: 2, MenuID: 100})
	db.Create(&model.SysRoleMenu{RoleID: 2, MenuID: 140})

	hash, _ := auth.HashPassword("12345678")
	auditor := model.SysUser{ID: 2, Username: "auditor", Password: hash, Status: model.StatusEnabled}
	db.Create(&auditor)
	db.Create(&model.SysUserRole{UserID: 2, RoleID: 2})

	cfg := &config.Config{}
	cfg.Server.Mode = "test"
	cfg.MySQL.DSN = "test"
	cfg.JWT.Secret = "test-secret-1234567890"
	cfg.Upload.Dir = t.TempDir()

	return New(&Deps{
		Cfg:      cfg,
		DB:       db,
		Logger:   slog.New(slog.DiscardHandler),
		JWT:      auth.NewManager(cfg.JWT.Secret, 30*time.Minute, 24*time.Hour, "go-admin"),
		Recorder: audit.NewRecorder(db),
	})
}

func bodyToMap(w *httptest.ResponseRecorder) map[string]interface{} {
	m := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

func loginAndGetToken(t *testing.T, r *gin.Engine, username string) (string, map[string]interface{}) {
	t.Helper()
	body := `{"username":"` + username + `","password":"12345678"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("登录失败: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out.Data.AccessToken, bodyToMap(w)
}

func doReq(r *gin.Engine, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSuperAdminCanListUsers(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "admin")
	w := doReq(r, "GET", "/api/v1/users", token)
	if w.Code != http.StatusOK {
		t.Fatalf("超管应可访问用户列表: %d %s", w.Code, w.Body.String())
	}
}

func TestAuditorForbiddenOnUsers(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "auditor")
	w := doReq(r, "GET", "/api/v1/users", token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("审计员访问用户列表应返回 403: %d %s", w.Code, w.Body.String())
	}
	// 但可以访问登录日志
	w = doReq(r, "GET", "/api/v1/login-logs", token)
	if w.Code != http.StatusOK {
		t.Fatalf("审计员应可访问登录日志: %d %s", w.Code, w.Body.String())
	}
}

func TestUnauthenticatedRejected(t *testing.T) {
	r := newRouterEnv(t)
	w := doReq(r, "GET", "/api/v1/users", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未登录应返回 401: %d", w.Code)
	}
}

func TestHealthz(t *testing.T) {
	r := newRouterEnv(t)
	w := doReq(r, "GET", "/healthz", "")
	if w.Code != http.StatusOK {
		t.Fatalf("healthz 应返回 200: %d", w.Code)
	}
}
