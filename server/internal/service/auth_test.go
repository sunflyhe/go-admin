// 认证服务测试:不构造任何 Gin 上下文,直接以标准库 context 调用,
// 同时验证登录日志仍记录 IP 与 User-Agent(来自 Handler 显式传入的 LoginMeta)。
package service

import (
	"context"
	"log/slog"
	"strconv"
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

func TestChangePasswordRequiresCorrectOldPassword(t *testing.T) {
	svc, ctx := newAuthService(t)
	actor := Actor{ID: 1, Username: "admin", IsSuper: true}

	if err := svc.ChangePassword(ctx, actor, "wrong-old-password", "brand-new-pass1"); err == nil {
		t.Fatal("原密码不正确时必须拒绝,否则拿到 access token 就能改密")
	}
	var user model.SysUser
	if err := svc.DB.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if pkgauth.VerifyPassword(user.Password, "brand-new-pass1") {
		t.Fatal("被拒的改密不得写入新密码")
	}
	if err := svc.ChangePassword(ctx, actor, "12345678", "12345678"); err == nil {
		t.Fatal("新旧密码相同应拒绝,避免误以为已修改")
	}
}

func TestChangePasswordInvalidatesIssuedCredentials(t *testing.T) {
	svc, ctx := newAuthService(t)
	actor := Actor{ID: 1, Username: "admin", IsSuper: true}
	pair, err := svc.Login(ctx, "admin", "12345678", LoginMeta{IP: "10.0.0.1", UserAgent: "cli"})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.ChangePassword(ctx, actor, "12345678", "brand-new-pass1"); err != nil {
		t.Fatalf("改密失败: %v", err)
	}

	// access token 的失效发生在认证中间件的版本比对处,服务层验证版本确实已自增
	claims, err := svc.JWT.Parse(pair.AccessToken, pkgauth.TokenTypeAccess)
	if err != nil {
		t.Fatalf("旧 access token 本身应仍可解析: %v", err)
	}
	var user model.SysUser
	if err := svc.DB.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.TokenVersion == claims.TokenVersion {
		t.Fatal("改密后 token_version 必须自增,否则旧 access token 在到期前仍然可用")
	}

	// 刷新令牌全部吊销,旧凭据无法再续期
	if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Fatal("改密后旧刷新令牌必须不可再用")
	}
	if _, err := svc.Login(ctx, "admin", "brand-new-pass1", LoginMeta{IP: "10.0.0.2", UserAgent: "cli"}); err != nil {
		t.Fatalf("新密码应能登录: %v", err)
	}
}

// TestAccountLockAcrossIPs 按账号维度锁定:换 IP 不影响计数,
// 连续失败达到阈值后即使密码正确也拒绝 —— 否则分布式爆破每 IP 5 次可无限叠加。
// 对不存在的账号不启用账号级锁定,避免任意用户名可被第三方恶意锁死。
func TestAccountLockAcrossIPs(t *testing.T) {
	svc, ctx := newAuthService(t)

	// 10 次失败,每次换一个 IP:IP 维度限流永不触发,只有账号维度在累积
	for i := 0; i < 10; i++ {
		ip := "10.1.0." + strconv.Itoa(i+1)
		if _, err := svc.Login(ctx, "admin", "wrong-password", LoginMeta{IP: ip, UserAgent: "spray"}); err == nil {
			t.Fatalf("第 %d 次错误密码应失败", i+1)
		}
	}
	if _, err := svc.Login(ctx, "admin", "12345678", LoginMeta{IP: "10.1.0.99", UserAgent: "victim"}); err == nil {
		t.Fatal("账号锁定后正确密码也应被拒绝")
	}
	var entry model.SysLoginLog
	if err := svc.DB.Where("username = ? AND success = ?", "admin", false).
		Order("id DESC").First(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if entry.FailReason != "账号连续失败被锁定" {
		t.Fatalf("锁定期间的失败日志原因应为账号锁定: %s", entry.FailReason)
	}

	// 不存在的账号:失败计数只走 IP 维度,换个新 IP 即可继续探测(账号级不锁)
	if _, err := svc.Login(ctx, "ghost", "wrong-password", LoginMeta{IP: "10.2.0.1", UserAgent: "probe"}); err == nil {
		t.Fatal("不存在的账号也应登录失败")
	}
}
