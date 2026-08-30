package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, trustedProxies string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "mysql:\n  dsn: test\njwt:\n  secret: test-secret-1234567890\nserver:\n  mode: test\n"
	if trustedProxies != "" {
		content += "  trustedProxies:\n" + trustedProxies
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadTrustedProxies(t *testing.T) {
	cfg, err := Load(writeConfig(t, "    - 192.0.2.10\n    - 198.51.100.0/24\n"))
	if err != nil {
		t.Fatalf("加载合法可信代理失败: %v", err)
	}
	if got := strings.Join(cfg.Server.TrustedProxies, ","); got != "192.0.2.10,198.51.100.0/24" {
		t.Fatalf("可信代理不符合预期: %q", got)
	}
}

func TestLoadTrustedProxiesFromEnv(t *testing.T) {
	t.Setenv("ADMIN_SERVER_TRUSTED_PROXIES", " 192.0.2.10, 198.51.100.0/24 ,, ")
	cfg, err := Load(writeConfig(t, ""))
	if err != nil {
		t.Fatalf("加载环境变量可信代理失败: %v", err)
	}
	if got := strings.Join(cfg.Server.TrustedProxies, ","); got != "192.0.2.10,198.51.100.0/24" {
		t.Fatalf("环境变量可信代理不符合预期: %q", got)
	}
}

func TestLoadRejectsInvalidTrustedProxy(t *testing.T) {
	_, err := Load(writeConfig(t, "    - not-an-ip\n"))
	if err == nil || !strings.Contains(err.Error(), "trustedProxies") {
		t.Fatalf("无效可信代理应被拒绝,实际: %v", err)
	}
}

func TestLoadWebDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "mysql:\n  dsn: test\njwt:\n  secret: test-secret-1234567890\nserver:\n  mode: test\n  webDirs:\n    app: dist/app\n    admin: dist/admin\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.WebDirs["app"] != "dist/app" || cfg.Server.WebDirs["admin"] != "dist/admin" {
		t.Fatalf("webDirs 解析错误: %v", cfg.Server.WebDirs)
	}
}

func TestLoadCORSAllowedOriginsFromEnv(t *testing.T) {
	path := writeConfig(t, "")
	t.Setenv("ADMIN_SERVER_CORS_ALLOWED_ORIGINS", "https://admin.example.com, https://app.example.com")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://admin.example.com", "https://app.example.com"}
	if len(cfg.Server.CORSAllowedOrigins) != len(want) {
		t.Fatalf("CORS 来源解析错误: %v", cfg.Server.CORSAllowedOrigins)
	}
	for i, o := range want {
		if cfg.Server.CORSAllowedOrigins[i] != o {
			t.Fatalf("CORS 来源解析错误: %v", cfg.Server.CORSAllowedOrigins)
		}
	}
}
