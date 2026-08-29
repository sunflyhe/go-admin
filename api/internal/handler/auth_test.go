package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/middleware"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/service"
	pkgauth "github.com/hesunfly/hesunfly-admin-go/api/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/api/test/testutil"
)

func newAuthEnv(t *testing.T) (*gin.Engine, *pkgauth.Manager, *model.SysUser) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	super := testutil.SeedSuperAdmin(t, db)
	jwt := pkgauth.NewManager("test-secret-1234567890", 30*time.Minute, 24*time.Hour, "go-admin")
	svc := service.NewAuthService(db, jwt, discardLogger())
	fileSvc := service.NewFileService(db, mustLocalStorage(t), 5)
	h := NewAuthHandler(svc, service.NewProfileService(db, fileSvc, "/files"))
	authn := &middleware.Authn{DB: db, JWT: jwt}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	authed := r.Group("")
	authed.Use(authn.Require())
	authed.POST("/auth/logout", h.Logout)
	authed.GET("/auth/me", h.Me)
	authed.PUT("/auth/profile", h.UpdateProfile)
	authed.POST("/auth/password", h.ChangePassword)
	authed.POST("/auth/avatar", h.UploadAvatar)
	return r, jwt, &super
}

func mustLocalStorage(t *testing.T) service.Storage {
	t.Helper()
	storage, err := service.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return storage
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
	// 本用例只注册 login/me,不触及个人中心路由,Profile 依赖留空
	h := NewAuthHandler(svc, nil)
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

// 个人中心三条自助路由同样必须要求登录。
func TestProfileRoutesRequireLogin(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	for _, req := range []struct{ method, path string }{
		{"PUT", "/auth/profile"},
		{"POST", "/auth/password"},
		{"POST", "/auth/avatar"},
	} {
		w, _ := doJSON(t, r, req.method, req.path, "", map[string]string{"nickname": "x"})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s 无凭据应 401: %d", req.method, req.path, w.Code)
		}
	}
}

func TestProfileUpdateIgnoresPrivilegedFields(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	_, out := login(t, r, "admin", "12345678")
	token := out["data"].(map[string]interface{})["accessToken"].(string)

	// 请求体里夹带 username/status:DTO 没有这两个字段,应被静默丢弃
	w, _ := doJSON(t, r, "PUT", "/auth/profile", token, map[string]interface{}{
		"nickname": "爆爆龙宝宝", "signature": "广阔天地,大有作为.",
		"username": "hacker", "status": 2,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("改资料应成功: %d %s", w.Code, w.Body.String())
	}

	_, me := doJSON(t, r, "GET", "/auth/me", token, nil)
	user := me["data"].(map[string]interface{})["user"].(map[string]interface{})
	if user["nickname"] != "爆爆龙宝宝" || user["signature"] != "广阔天地,大有作为." {
		t.Fatalf("自助字段应生效: %+v", user)
	}
	if user["username"] != "admin" {
		t.Fatalf("账号不得被自助接口改动: %v", user["username"])
	}
}

func TestAvatarUploadOverHTTP(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	_, out := login(t, r, "admin", "12345678")
	token := out["data"].(map[string]interface{})["accessToken"].(string)

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	part, err := mw.CreateFormFile("file", "me.png")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	part.Write(bytes.Repeat([]byte{0}, 32))
	mw.Close()

	req := httptest.NewRequest("POST", "/auth/avatar", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("头像上传应成功: %d %s", w.Code, w.Body.String())
	}

	_, me := doJSON(t, r, "GET", "/auth/me", token, nil)
	user := me["data"].(map[string]interface{})["user"].(map[string]interface{})
	if avatar, _ := user["avatar"].(string); !strings.HasPrefix(avatar, "/files/") {
		t.Fatalf("/me 应带回公开头像地址: %+v", user)
	}
}

func TestChangePasswordOverHTTPKillsSession(t *testing.T) {
	r, _, _ := newAuthEnv(t)
	_, out := login(t, r, "admin", "12345678")
	token := out["data"].(map[string]interface{})["accessToken"].(string)

	w, _ := doJSON(t, r, "POST", "/auth/password", token, map[string]string{
		"oldPassword": "12345678", "newPassword": "brand-new-pass1",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("改密应成功: %d %s", w.Code, w.Body.String())
	}
	// 不是只"版本号变了",而是旧凭据在下一个请求上立刻不可用
	w2, _ := doJSON(t, r, "GET", "/auth/me", token, nil)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("改密后旧 access token 应立即失效: %d", w2.Code)
	}
	w3, _ := login(t, r, "admin", "brand-new-pass1")
	if w3.Code != http.StatusOK {
		t.Fatalf("新密码应能登录: %d %s", w3.Code, w3.Body.String())
	}
}
