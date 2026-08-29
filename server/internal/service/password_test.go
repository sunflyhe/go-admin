// 密码强度策略测试:长度与字符类别要求,以及它在各设密入口的一致生效。
package service

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/server/test/testutil"
)

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		pw    string
		valid bool
		label string
	}{
		{"abcd1234", true, "字母+数字"},
		{"Abcd1234!@", true, "含符号不额外加分也不拒绝"},
		{"密码密码密码123", true, "中文也算字母类别"},
		{"12345678", false, "纯数字(种子历史密码的形态,新设密不再允许)"},
		{"abcdefgh", false, "纯字母"},
		{"abc123", false, "长度不足"},
		{strings.Repeat("a", 129), false, "超长"},
		{"", false, "空串"},
	}
	for _, c := range cases {
		err := ValidatePassword(c.pw)
		if c.valid && err != nil {
			t.Errorf("%s 应通过,实际: %v", c.label, err)
		}
		if !c.valid && err == nil {
			t.Errorf("%s 应被拒绝", c.label)
		}
	}
}

// TestPasswordPolicyOnAllSetEntries 策略必须落在 Service 层:
// 创建用户、重置密码、自助改密三条入口,绕过 HTTP binding 也要被拦。
func TestPasswordPolicyOnAllSetEntries(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedSuperAdmin(t, db)
	ctx := context.Background()
	userSvc := NewUserService(db)
	authSvc := NewAuthService(db, nil, slog.New(slog.DiscardHandler))

	weak := "12345678"
	if _, err := userSvc.Create(ctx, &UserSaveInput{Username: "alice", Password: weak}); err == nil {
		t.Fatal("创建用户:纯数字密码应被拒绝")
	}
	if err := userSvc.ResetPassword(ctx, 1, &UserResetPasswordInput{Password: weak}); err == nil {
		t.Fatal("重置密码:纯数字密码应被拒绝")
	}
	actor := Actor{ID: 1, Username: "admin", IsSuper: true}
	if err := authSvc.ChangePassword(ctx, actor, "12345678", weak); err == nil {
		t.Fatal("自助改密:纯数字密码应被拒绝")
	}

	// 合法密码在三条入口都正常工作
	strong := "alice123456"
	if _, err := userSvc.Create(ctx, &UserSaveInput{Username: "alice", Password: strong}); err != nil {
		t.Fatalf("创建用户:合法密码不应被拒: %v", err)
	}
	if err := userSvc.ResetPassword(ctx, 1, &UserResetPasswordInput{Password: "reset123456"}); err != nil {
		t.Fatalf("重置密码:合法密码不应被拒: %v", err)
	}
	if err := authSvc.ChangePassword(ctx, actor, "reset123456", "brand-new-pass1"); err != nil {
		t.Fatalf("自助改密:合法密码不应被拒: %v", err)
	}
}
