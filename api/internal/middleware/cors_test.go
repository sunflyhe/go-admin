package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func corsRouter(t *testing.T, origins []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewCORS(origins).Middleware())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestCORSAllowedOrigin(t *testing.T) {
	r := corsRouter(t, []string{"https://admin.example.com"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("预检请求应返回 204: %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://admin.example.com" {
		t.Fatalf("预检应带 Allow-Origin: %v", w.Header())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://admin.example.com" {
		t.Fatalf("实际请求应带 Allow-Origin: %v", w.Header())
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	r := corsRouter(t, []string{"https://admin.example.com"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("未列入白名单的来源不应拿到 Allow-Origin")
	}
}

func TestCORSEmptyIsPassthrough(t *testing.T) {
	r := corsRouter(t, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("未启用 CORS 时不应有任何 CORS 头")
	}
}

func TestCORSWildcard(t *testing.T) {
	r := corsRouter(t, []string{"*"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://any.example.com")
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://any.example.com" {
		t.Fatal("通配 * 应放行任意来源")
	}
}
