// 路由层集成测试:验证不同角色的服务端权限差异(P1 验收项)。
package router

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/config"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/middleware"
	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	pkgauth "github.com/hesunfly/hesunfly-admin-go/api/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/api/test/testutil"
)

func newRouterEnv(t *testing.T) *gin.Engine {
	return newRouterEnvWithTrustedProxies(t, nil)
}

func newRouterEnvWithTrustedProxies(t *testing.T, trustedProxies []string) *gin.Engine {
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

	hash, _ := pkgauth.HashPassword("12345678")
	auditor := model.SysUser{ID: 2, Username: "auditor", Password: hash, Status: model.StatusEnabled}
	db.Create(&auditor)
	db.Create(&model.SysUserRole{UserID: 2, RoleID: 2})

	cfg := &config.Config{}
	cfg.Server.Mode = "test"
	cfg.Server.TrustedProxies = trustedProxies
	cfg.MySQL.DSN = "test"
	cfg.JWT.Secret = "test-secret-1234567890"
	cfg.Upload.Dir = t.TempDir()
	// MaxSizeMB 必须显式给出:为 0 时 Handler 的 MaxBytesReader 上限是 0 字节,
	// 任何 multipart 请求都会在绑定阶段就被拒,上传链路就永远测不到。
	cfg.Upload.MaxSizeMB = 5
	cfg.Upload.PublicURL = "/files"

	return New(&Deps{
		Cfg:      cfg,
		DB:       db,
		Logger:   slog.New(slog.DiscardHandler),
		JWT:      pkgauth.NewManager(cfg.JWT.Secret, 30*time.Minute, 24*time.Hour, "go-admin"),
		Recorder: middleware.NewRecorder(db),
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

// TestConfigPermission 系统参数走同一套服务端权限码:
// 无 system:dict:list 的账号必须被 403,超管可完成建改删闭环。
func TestConfigPermission(t *testing.T) {
	r := newRouterEnv(t)
	auditorToken, _ := loginAndGetToken(t, r, "auditor")
	w := doReq(r, "GET", "/api/v1/configs", auditorToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("审计员访问参数列表应返回 403: %d %s", w.Code, w.Body.String())
	}

	adminToken, _ := loginAndGetToken(t, r, "admin")
	w = doJSON(t, r, "POST", "/api/v1/configs", adminToken, `{"name":"站点名称","key":"site.name","value":"Hesunfly Admin"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("超管创建参数应 201: %d %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := int(created["data"].(map[string]interface{})["id"].(float64))

	w = doJSON(t, r, "PUT", "/api/v1/configs/"+strconv.Itoa(id), adminToken, `{"name":"站点名称","key":"site.name","value":"改名后的值"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("超管更新参数应 200: %d %s", w.Code, w.Body.String())
	}
	w = doReq(r, "GET", "/api/v1/configs?keyword=site.name", adminToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "改名后的值") {
		t.Fatalf("列表应能搜到更新后的参数: %d %s", w.Code, w.Body.String())
	}
	w = doReq(r, "DELETE", "/api/v1/configs/"+strconv.Itoa(id), adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("超管删除参数应 200: %d %s", w.Code, w.Body.String())
	}
}

// TestDictTypePermission 字典类型/子项维护走 system:dict:* 权限码;
// 业务读取 /dict-data 只要求登录 —— 无任何菜单权限的审计员也应能取下拉数据。
func TestDictTypePermission(t *testing.T) {
	r := newRouterEnv(t)
	auditorToken, _ := loginAndGetToken(t, r, "auditor")
	w := doReq(r, "GET", "/api/v1/dict-types", auditorToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("审计员访问字典类型列表应返回 403: %d %s", w.Code, w.Body.String())
	}

	adminToken, _ := loginAndGetToken(t, r, "admin")
	w = doJSON(t, r, "POST", "/api/v1/dict-types", adminToken, `{"name":"性别","key":"gender"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("超管创建字典类型应 201: %d %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	typeID := int(created["data"].(map[string]interface{})["id"].(float64))

	w = doJSON(t, r, "POST", "/api/v1/dict-types/"+strconv.Itoa(typeID)+"/items", adminToken, `{"label":"男","value":"1"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("超管创建字典项应 201: %d %s", w.Code, w.Body.String())
	}

	// 业务读取:审计员没有 system:dict:list,但 /dict-data 只要求登录
	w = doReq(r, "GET", "/api/v1/dict-data?key=gender", auditorToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"label":"男"`) {
		t.Fatalf("登录用户按键读取字典应 200 且含子项: %d %s", w.Code, w.Body.String())
	}

	w = doReq(r, "DELETE", "/api/v1/dict-types/"+strconv.Itoa(typeID), adminToken)
	if w.Code != http.StatusConflict {
		t.Fatalf("仍有子项时删除类型应 409: %d %s", w.Code, w.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	r := newRouterEnv(t)
	w := doReq(r, "GET", "/healthz", "")
	if w.Code != http.StatusOK {
		t.Fatalf("healthz 应返回 200: %d", w.Code)
	}
}

func TestTrustedProxyClientIP(t *testing.T) {
	r := newRouterEnvWithTrustedProxies(t, []string{"192.0.2.0/24"})
	r.GET("/test-client-ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	req := httptest.NewRequest(http.MethodGet, "/test-client-ip", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.24")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Body.String(); got != "203.0.113.24" {
		t.Fatalf("应使用可信代理提供的客户端 IP,实际: %q", got)
	}
}

func TestUntrustedProxyCannotSetClientIP(t *testing.T) {
	r := newRouterEnv(t)
	r.GET("/test-client-ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	req := httptest.NewRequest(http.MethodGet, "/test-client-ip", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.24")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Body.String(); got != "192.0.2.10" {
		t.Fatalf("非可信代理不能覆盖客户端 IP,实际: %q", got)
	}
}

// TestFileListCategoryDoesNotBypassPermission 新增查询参数不得成为权限旁路:
// 前端隐藏按钮不是安全措施,带 category 的请求仍必须被服务端权限码拦住。
func TestFileListCategoryDoesNotBypassPermission(t *testing.T) {
	r := newRouterEnv(t)
	auditorToken, _ := loginAndGetToken(t, r, "auditor")
	w := doReq(r, "GET", "/api/v1/files?category=image", auditorToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("无 system:file:list 的账号带 category 仍应 403: %d %s", w.Code, w.Body.String())
	}

	adminToken, _ := loginAndGetToken(t, r, "admin")
	w = doReq(r, "GET", "/api/v1/files?category=image", adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("超管按分类查询应 200: %d %s", w.Code, w.Body.String())
	}
}

func TestFileListRejectsUnknownCategory(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "admin")
	w := doReq(r, "GET", "/api/v1/files?category=video", token)
	if w.Code != http.StatusOK {
		t.Fatalf("视频是合法标签,应 200: %d %s", w.Code, w.Body.String())
	}

	// document/archive/other 是旧的上传归类,已不是对外标签
	w = doReq(r, "GET", "/api/v1/files?category=document", token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("未知分类应 400,而不是静默返回全部: %d %s", w.Code, w.Body.String())
	}
}

func doJSON(t *testing.T, r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// fileCenterEndpoints 是文件中心新增的写端点:每一个都必须挂权限码。
var fileCenterEndpoints = []struct {
	method, path, body string
}{
	{"PUT", "/api/v1/files/group", `{"ids":[1],"groupId":0}`},
	{"POST", "/api/v1/files/batch-delete", `{"ids":[1]}`},
	{"GET", "/api/v1/file-groups", ""},
	{"POST", "/api/v1/file-groups", `{"name":"素材"}`},
	{"PUT", "/api/v1/file-groups/1", `{"name":"素材2"}`},
	{"DELETE", "/api/v1/file-groups/1", ""},
}

// TestFileCenterEndpointsNotPermissionBypass 新增端点不得成为权限旁路:
// 审计员只有登录日志权限,访问这些端点必须全部 403,而不是 404/200。
func TestFileCenterEndpointsNotPermissionBypass(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "auditor")
	for _, e := range fileCenterEndpoints {
		w := doJSON(t, r, e.method, e.path, token, e.body)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s %s 应 403: %d %s", e.method, e.path, w.Code, w.Body.String())
		}
	}
}

func TestFileCenterEndpointsRequireAuth(t *testing.T) {
	r := newRouterEnv(t)
	for _, e := range fileCenterEndpoints {
		w := doJSON(t, r, e.method, e.path, "", e.body)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s 未登录应 401: %d %s", e.method, e.path, w.Code, w.Body.String())
		}
	}
}

// TestFileGroupCRUDForSuperAdmin 端到端串一遍分组路由:确认真的是接通到 Service,
// 而不是只注册了中间件就返回 403/404。
func TestFileGroupCRUDForSuperAdmin(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "admin")

	w := doJSON(t, r, "POST", "/api/v1/file-groups", token, `{"name":"素材"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("创建分组应 201: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Data.ID <= 0 || created.Data.Name != "素材" {
		t.Fatalf("响应应回传新分组: %s", w.Body.String())
	}
	groupPath := "/api/v1/file-groups/" + strconv.FormatInt(created.Data.ID, 10)

	// 同名(仅大小写不同也一样)必须冲突:Service 侧断言,这里确认错误能穿透到 HTTP 层
	w = doJSON(t, r, "POST", "/api/v1/file-groups", token, `{"name":"素材"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("同名分组应 409: %d %s", w.Code, w.Body.String())
	}

	if w = doJSON(t, r, "PUT", groupPath, token, `{"name":"素材图"}`); w.Code != http.StatusOK {
		t.Fatalf("重命名应 200: %d %s", w.Code, w.Body.String())
	}
	if w = doReq(r, "GET", "/api/v1/file-groups", token); w.Code != http.StatusOK {
		t.Fatalf("分组列表应 200: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "素材图") {
		t.Fatalf("列表应包含重命名后的分组: %s", w.Body.String())
	}
	if w = doReq(r, "DELETE", groupPath, token); w.Code != http.StatusOK {
		t.Fatalf("删除空分组应 200: %d %s", w.Code, w.Body.String())
	}
}

// TestFileGroupDeleteRejectsNonEmpty 删非空分组必须是 409,而不是连带删掉文件。
func TestFileGroupDeleteRejectsNonEmpty(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "admin")

	w := doJSON(t, r, "POST", "/api/v1/file-groups", token, `{"name":"有文件"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("创建分组失败: %d %s", w.Code, w.Body.String())
	}
	var groupResp struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &groupResp); err != nil || groupResp.Data.ID <= 0 {
		t.Fatalf("未解析出分组 id: %v %s", err, w.Body.String())
	}
	groupPath := "/api/v1/file-groups/" + strconv.FormatInt(groupResp.Data.ID, 10)

	if w = uploadFileToGroup(t, r, token, "in-group.png", groupResp.Data.ID); w.Code != http.StatusCreated {
		t.Fatalf("上传应 201: %d %s", w.Code, w.Body.String())
	}

	w = doReq(r, "DELETE", groupPath, token)
	if w.Code != http.StatusConflict {
		t.Fatalf("非空分组删除应 409: %d %s", w.Code, w.Body.String())
	}
	// 文件必须还在
	w = doReq(r, "GET", "/api/v1/files", token)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "in-group.png") {
		t.Fatalf("被拒的分组删除不得影响文件: %d %s", w.Code, w.Body.String())
	}
	// groupId 过滤应能定位到它,且落到"未分组"的计数为 0
	w = doReq(r, "GET", "/api/v1/files?groupId="+strconv.FormatInt(groupResp.Data.ID, 10), token)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatalf("按分组筛选应命中 1 条: %d %s", w.Code, w.Body.String())
	}
	w = doReq(r, "GET", "/api/v1/files?groupId=0", token)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"total":0`) {
		t.Fatalf("未分组应为 0 条: %d %s", w.Code, w.Body.String())
	}
}

// TestFileGroupCountsFollowCategory 层级是"类型标签 → 分组 → 文件":
// 同一批数据下换标签,左栏计数必须跟着变,不能恒等于全部文件数。
func TestFileGroupCountsFollowCategory(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "admin")

	w := doJSON(t, r, "POST", "/api/v1/file-groups", token, `{"name":"素材"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("创建分组失败: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Data.ID <= 0 {
		t.Fatalf("未解析出分组 id: %v %s", err, w.Body.String())
	}
	if w = uploadFileToGroup(t, r, token, "shot.png", resp.Data.ID); w.Code != http.StatusCreated {
		t.Fatalf("上传应 201: %d %s", w.Code, w.Body.String())
	}

	totalOf := func(query string) string {
		t.Helper()
		w := doReq(r, "GET", "/api/v1/file-groups"+query, token)
		if w.Code != http.StatusOK {
			t.Fatalf("%s 应 200: %d %s", query, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	if got := totalOf("?category=image"); !strings.Contains(got, `"total":1`) {
		t.Fatalf("图片标签下总数应为 1: %s", got)
	}
	if got := totalOf("?category=video"); !strings.Contains(got, `"total":0`) {
		t.Fatalf("视频标签下总数应为 0: %s", got)
	}
	// 未知标签必须 400,而不是静默按全部统计
	if w := doReq(r, "GET", "/api/v1/file-groups?category=document", token); w.Code != http.StatusBadRequest {
		t.Fatalf("未知标签应 400: %d %s", w.Code, w.Body.String())
	}
}

// uploadFileToGroup 构造一次真实 multipart 上传,验证 Handler 会透传 groupId。
func uploadFileToGroup(t *testing.T, r *gin.Engine, token, filename string, groupID int64) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	// PNG 文件头 + 填充:通过真实 MIME 嗅探
	content := append(append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, 32)...), '\x00')
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("groupId", strconv.FormatInt(groupID, 10)); err != nil {
		t.Fatal(err)
	}
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/files", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// articleEndpoints 是文章资讯新增的端点:每一个都必须挂权限码。
var articleEndpoints = []struct {
	method, path, body string
}{
	{"GET", "/api/v1/article-categories", ""},
	{"POST", "/api/v1/article-categories", `{"name":"公告"}`},
	{"PUT", "/api/v1/article-categories/1", `{"name":"公告2"}`},
	{"DELETE", "/api/v1/article-categories/1", ""},
	{"GET", "/api/v1/articles", ""},
	{"GET", "/api/v1/articles/1", ""},
	{"POST", "/api/v1/articles", `{"title":"t","content":"<p>x</p>","status":1}`},
	{"PUT", "/api/v1/articles/1", `{"title":"t","content":"<p>x</p>","status":1}`},
	{"DELETE", "/api/v1/articles/1", ""},
	{"POST", "/api/v1/article-images", ""},
}

// TestArticleEndpointsNotPermissionBypass 新增端点不得成为权限旁路:
// 审计员只有登录日志权限,访问文章资讯端点必须全部 403,而不是 404/200。
func TestArticleEndpointsNotPermissionBypass(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "auditor")
	for _, e := range articleEndpoints {
		w := doJSON(t, r, e.method, e.path, token, e.body)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s %s 应 403: %d %s", e.method, e.path, w.Code, w.Body.String())
		}
	}
}

func TestArticleEndpointsRequireAuth(t *testing.T) {
	r := newRouterEnv(t)
	for _, e := range articleEndpoints {
		w := doJSON(t, r, e.method, e.path, "", e.body)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s 未登录应 401: %d %s", e.method, e.path, w.Code, w.Body.String())
		}
	}
}

// TestArticleModuleCRUDForSuperAdmin 端到端串一遍文章资讯:建分类 → 配图 → 发文章 →
// 列表/详情 → 更新 → 删除,确认真的是接通到 Service,而不是只注册了中间件。
func TestArticleModuleCRUDForSuperAdmin(t *testing.T) {
	r := newRouterEnv(t)
	token, _ := loginAndGetToken(t, r, "admin")

	// 建分类
	w := doJSON(t, r, "POST", "/api/v1/article-categories", token, `{"name":"公告","sort":1}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("创建分类应 201: %d %s", w.Code, w.Body.String())
	}
	var categoryResp struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &categoryResp); err != nil || categoryResp.Data.ID <= 0 {
		t.Fatalf("未解析出分类 id: %v %s", err, w.Body.String())
	}
	categoryID := strconv.FormatInt(categoryResp.Data.ID, 10)

	// 同名分类必须冲突
	w = doJSON(t, r, "POST", "/api/v1/article-categories", token, `{"name":"公告"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("同名分类应 409: %d %s", w.Code, w.Body.String())
	}

	// 富文本配图:复用文件上传校验,响应必须给出可公开访问的 URL
	imageResp := uploadArticleImage(t, r, token, "cover.png")
	if imageResp.Code != http.StatusCreated {
		t.Fatalf("配图上传应 201: %d %s", imageResp.Code, imageResp.Body)
	}
	if !strings.Contains(imageResp.Body.String(), `"/files/`) {
		t.Fatalf("配图 URL 应走公开前缀: %s", imageResp.Body.String())
	}

	// 发文章(发布)
	body := `{"categoryId":` + categoryID + `,"title":"第一篇","summary":"摘要","content":"<p>正文</p>","status":2}`
	if w = doJSON(t, r, "POST", "/api/v1/articles", token, body); w.Code != http.StatusCreated {
		t.Fatalf("创建文章应 201: %d %s", w.Code, w.Body.String())
	}
	var articleResp struct {
		Data struct {
			ID      int64  `json:"id"`
			Author  string `json:"author"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &articleResp); err != nil || articleResp.Data.ID <= 0 {
		t.Fatalf("未解析出文章 id: %v %s", err, w.Body.String())
	}
	if articleResp.Data.Author != "admin" {
		t.Fatalf("应记录创建者: %s", w.Body.String())
	}
	articlePath := "/api/v1/articles/" + strconv.FormatInt(articleResp.Data.ID, 10)

	// 分类不存在必须 404
	badCategory := `{"categoryId":999,"title":"x","content":"<p>x</p>","status":1}`
	if w = doJSON(t, r, "POST", "/api/v1/articles", token, badCategory); w.Code != http.StatusNotFound {
		t.Fatalf("指向不存在分类应 404: %d %s", w.Code, w.Body.String())
	}

	// 列表带分类名与状态,且不回传正文
	if w = doReq(r, "GET", "/api/v1/articles", token); w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"total":1`) || !strings.Contains(w.Body.String(), "公告") {
		t.Fatalf("列表应带出分类名: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "正文") {
		t.Fatalf("列表不得回传正文: %s", w.Body.String())
	}

	// 详情含正文(Go 的 JSON 编码会把 < 转义成 \u003c,按内容断言而不是按标签断言)
	if w = doReq(r, "GET", articlePath, token); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "正文") {
		t.Fatalf("详情应含正文: %d %s", w.Code, w.Body.String())
	}

	// 更新
	updateBody := `{"categoryId":` + categoryID + `,"title":"修订","summary":"","content":"<p>v2</p>","status":1}`
	if w = doJSON(t, r, "PUT", articlePath, token, updateBody); w.Code != http.StatusOK {
		t.Fatalf("更新文章应 200: %d %s", w.Code, w.Body.String())
	}

	// 非空分类删除必须 409
	if w = doReq(r, "DELETE", "/api/v1/article-categories/"+categoryID, token); w.Code != http.StatusConflict {
		t.Fatalf("非空分类删除应 409: %d %s", w.Code, w.Body.String())
	}

	// 删除文章后分类可删
	if w = doReq(r, "DELETE", articlePath, token); w.Code != http.StatusOK {
		t.Fatalf("删除文章应 200: %d %s", w.Code, w.Body.String())
	}
	if w = doReq(r, "DELETE", "/api/v1/article-categories/"+categoryID, token); w.Code != http.StatusOK {
		t.Fatalf("空分类删除应 200: %d %s", w.Code, w.Body.String())
	}
}

// uploadArticleImage 构造一次真实 multipart 配图上传,验证 Handler 会强制公开并返回 URL。
func uploadArticleImage(t *testing.T, r *gin.Engine, token, filename string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	// PNG 文件头 + 填充:通过真实 MIME 嗅探
	content := append(append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, 32)...), '\x00')
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/article-images", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
