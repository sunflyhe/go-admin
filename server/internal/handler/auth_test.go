package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/middleware"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/service"
	pkgauth "github.com/hesunfly/hesunfly-admin-go/server/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/test/testutil"
)

func newAuthEnv(t *testing.T) (*gin.Engine, *pkgauth.Manager, *model.SysUser) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	super := testutil.SeedSuperAdmin(t, db)
	jwt := pkgauth.NewManager("test-secret-1234567890", 30*time.Minute, 24*time.Hour, "go-admin")
	svc := service.NewAuthService(db, jwt, discardLogger())
	h := NewAuthHandler(svc)
	authn := &middleware.Authn{DB: db, JWT: jwt}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	authed := r.Group("")
	authed.Use(authn.Require())
	authed.POST("/auth/logout", h.Logout)
	authed.GET("/auth/me", h.Me)
	return r, jwt, &super
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func doJSON(t *testing.T, r *gin.Engine, method, path, token string, body interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func login(t *testing.T, r *gin.Engine, username, password string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	return doJSON(t, r, "POST", "/auth/login", "", map[string]string{"username": username, "password": password})
}

func TestLoginSuccessAndMe(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	w, out := login(t, r, "admin", "12345678")
	if w.Code != http.StatusOK {
		t.Fatalf("登录应成功: %d %s", w.Code, w.Body.String())
	}
	data := out["data"].(map[string]interface{})
	token := data["accessToken"].(string)

	w2, out2 := doJSON(t, r, "GET", "/auth/me", token, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("/me 应成功: %d", w2.Code)
	}
	me := out2["data"].(map[string]interface{})
	user := me["user"].(map[string]interface{})
	if user["username"] != "admin" {
		t.Fatalf("用户名不匹配: %v", user)
	}
	perms := me["permissions"].([]interface{})
	if len(perms) != 1 || perms[0] != "*" {
		t.Fatalf("超管应获得通配权限: %v", perms)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	w, out := login(t, r, "admin", "wrong-password")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("错误密码应返回 401: %d", w.Code)
	}
	if out["message"] != "账号或密码错误" {
		t.Fatalf("错误提示不应区分账号存在与否: %v", out["message"])
	}
}

func TestLoginDisabledAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	testutil.SeedSuperAdmin(t, db)
	jwt := pkgauth.NewManager("test-secret-1234567890", 30*time.Minute, 24*time.Hour, "go-admin")
	svc := service.NewAuthService(db, jwt, discardLogger())
	h := NewAuthHandler(svc)
	authn := &middleware.Authn{DB: db, JWT: jwt}
	r := gin.New()
	r.POST("/auth/login", h.Login)
	authed := r.Group("")
	authed.Use(authn.Require())
	authed.GET("/auth/me", h.Me)

	// 直接改库模拟停用(超管在服务层受保护,这里只验证停用后的登录拒绝逻辑)
	if err := db.Model(&model.SysUser{ID: 1}).Update("status", model.StatusDisabled).Error; err != nil {
		t.Fatal(err)
	}
	w, _ := login(t, r, "admin", "12345678")
	if w.Code != http.StatusForbidden {
		t.Fatalf("停用账号登录应返回 403: %d", w.Code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	for i := 0; i < 5; i++ {
		w, _ := login(t, r, "admin", "wrong-password")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("第 %d 次失败应返回 401", i+1)
		}
	}
	w, out := login(t, r, "admin", "wrong-password")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("超过失败阈值应返回 429: %d %s", w.Code, w.Body.String())
	}
	_ = out
	// 即使密码正确,锁定期间也应拒绝
	w, _ = login(t, r, "admin", "12345678")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("锁定期间正确密码也应被拒绝: %d", w.Code)
	}
}

func TestRefreshRotationAndLogout(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	_, out := login(t, r, "admin", "12345678")
	data := out["data"].(map[string]interface{})
	refresh := data["refreshToken"].(string)

	// 第一次刷新成功并获得新 token
	w, out2 := doJSON(t, r, "POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh})
	if w.Code != http.StatusOK {
		t.Fatalf("首次刷新应成功: %d %s", w.Code, w.Body.String())
	}
	data2 := out2["data"].(map[string]interface{})
	newRefresh := data2["refreshToken"].(string)
	newAccess := data2["accessToken"].(string)

	// 旧 refresh 已被轮换吊销,复用应失败
	w, _ = doJSON(t, r, "POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("旧 refresh 复用应返回 401: %d", w.Code)
	}

	// 退出后新 access 也失效
	w, _ = doJSON(t, r, "POST", "/auth/logout", newAccess, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("退出应成功: %d", w.Code)
	}
	w, _ = doJSON(t, r, "GET", "/auth/me", newAccess, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("退出后旧 access 应失效: %d", w.Code)
	}
	w, _ = doJSON(t, r, "POST", "/auth/refresh", "", map[string]string{"refreshToken": newRefresh})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("退出后 refresh 应失效: %d", w.Code)
	}
}

func TestUnauthorizedWithoutToken(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	w, _ := doJSON(t, r, "GET", "/auth/me", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无凭据应返回 401: %d", w.Code)
	}
}
