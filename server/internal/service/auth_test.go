// 认证服务测试:不构造任何 Gin 上下文,直接以标准库 context 调用,
// 同时验证登录日志仍记录 IP 与 User-Agent(来自 Handler 显式传入的 LoginMeta)。
package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	pkgauth "github.com/hesunfly/hesunfly-admin-go/server/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/test/testutil"
)

func newAuthService(t *testing.T) (*AuthService, context.Context) {
	t.Helper()
	db := testutil.NewTestDB(t)
	testutil.SeedSuperAdmin(t, db)
	jwt := pkgauth.NewManager("test-secret-1234567890", 30*time.Minute, 24*time.Hour, "go-admin")
	svc := NewAuthService(db, jwt, slog.New(slog.DiscardHandler))
	return svc, context.Background()
}

func TestLoginRecordsIPAndUserAgent(t *testing.T) {
	svc, ctx := newAuthService(t)
	meta := LoginMeta{IP: "9.9.9.9", UserAgent: "pytest-agent/1.0"}
	resp, err := svc.Login(ctx, "admin", "12345678", meta)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("应返回 token 对")
	}
	var entry model.SysLoginLog
	if err := svc.DB.Where("username = ? AND success = ?", "admin", true).
		Order("id DESC").First(&entry).Error; err != nil {
		t.Fatalf("登录日志缺失: %v", err)
	}
	if entry.IP != "9.9.9.9" {
		t.Fatalf("登录日志应记录 Handler 传入的 IP: %s", entry.IP)
	}
	if entry.UserAgent != "pytest-agent/1.0" {
		t.Fatalf("登录日志应记录 Handler 传入的 User-Agent: %s", entry.UserAgent)
	}
}

func TestFailedLoginRecordsReasonAndMeta(t *testing.T) {
	svc, ctx := newAuthService(t)
	meta := LoginMeta{IP: "8.8.8.8", UserAgent: "fail-agent/2.0"}
	if _, err := svc.Login(ctx, "admin", "wrong-password", meta); err == nil {
		t.Fatal("错误密码应登录失败")
	}
	var entry model.SysLoginLog
	if err := svc.DB.Where("username = ? AND success = ?", "admin", false).
		Order("id DESC").First(&entry).Error; err != nil {
		t.Fatalf("失败日志缺失: %v", err)
	}
	if entry.IP != "8.8.8.8" || entry.UserAgent != "fail-agent/2.0" {
		t.Fatalf("失败日志也应记录 IP/UA: %+v", entry)
	}
	if entry.FailReason != "账号或密码错误" {
		t.Fatalf("失败原因不应区分账号存在与否: %s", entry.FailReason)
	}
}

// Service 脱离 Gin 后可直接复用:整个刷新轮换流程无任何 HTTP 语义。
func TestRefreshRotationWithoutGin(t *testing.T) {
	svc, ctx := newAuthService(t)
	resp, err := svc.Login(ctx, "admin", "12345678", LoginMeta{IP: "127.0.0.1", UserAgent: "cli"})
	if err != nil {
		t.Fatal(err)
	}
	oldRefresh := resp.RefreshToken

	pair, err := svc.Refresh(ctx, oldRefresh)
	if err != nil {
		t.Fatalf("首次刷新应成功: %v", err)
	}
	if pair.RefreshToken == oldRefresh {
		t.Fatal("刷新应轮换出新的 refresh token")
	}
	// 旧 refresh 已吊销,复用应失败
	if _, err := svc.Refresh(ctx, oldRefresh); err == nil {
		t.Fatal("旧 refresh 复用应失败")
	}
	// 登出后新 refresh 也失效
	if err := svc.Logout(ctx, Actor{ID: 1, Username: "admin", IsSuper: true}); err != nil {
		t.Fatalf("登出失败: %v", err)
	}
	if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Fatal("登出后 refresh 应失效")
	}
}

func TestMeWithoutGin(t *testing.T) {
	svc, ctx := newAuthService(t)
	if _, err := svc.Login(ctx, "admin", "12345678", LoginMeta{IP: "127.0.0.1", UserAgent: "cli"}); err != nil {
		t.Fatal(err)
	}
	me, err := svc.Me(ctx, Actor{ID: 1, Username: "admin", IsSuper: true})
	if err != nil {
		t.Fatalf("Me 调用失败: %v", err)
	}
	if !me.User.Super {
		t.Fatal("超管标记应成立")
	}
	if len(me.Permissions) != 1 || me.Permissions[0] != "*" {
		t.Fatalf("超管应获得通配权限: %v", me.Permissions)
	}
}
