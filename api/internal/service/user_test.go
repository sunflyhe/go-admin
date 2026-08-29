package service

import (
	"context"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/api/test/testutil"
)

// Service 已脱离 Gin:直接使用标准库 context 调用。
func newEnv(t *testing.T) (*UserService, context.Context) {
	t.Helper()
	return NewUserService(testutil.NewTestDB(t)), context.Background()
}

func TestCreateAndDuplicate(t *testing.T) {
	svc, ctx := newEnv(t)
	item, err := svc.Create(ctx, &UserSaveInput{Username: "alice", Password: "alice123456", Nickname: "Alice"})
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if item.ID == 0 {
		t.Fatal("应返回用户 ID")
	}
	if _, err := svc.Create(ctx, &UserSaveInput{Username: "alice", Password: "alice123456"}); err == nil {
		t.Fatal("重复用户名应报冲突")
	}
}

func TestSuperAdminProtected(t *testing.T) {
	svc, ctx := newEnv(t)
	testutil.SeedSuperAdmin(t, svc.DB)
	if err := svc.Delete(ctx, 1); err == nil {
		t.Fatal("内置超管账号不允许删除")
	}
	if err := svc.SetStatus(ctx, 1, &UserSetStatusInput{Status: model.StatusDisabled}); err == nil {
		t.Fatal("内置超管账号不允许停用")
	}
}

func TestDeleteCleansRelations(t *testing.T) {
	svc, ctx := newEnv(t)
	testutil.SeedSuperAdmin(t, svc.DB)
	item, err := svc.Create(ctx, &UserSaveInput{Username: "bob", Password: "bob123456"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AssignRoles(ctx, item.ID, &UserAssignRolesInput{RoleIDs: []int64{1}}); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}
	if err := svc.Delete(ctx, item.ID); err != nil {
		t.Fatalf("删除用户失败: %v", err)
	}
	var count int64
	svc.DB.Model(&model.SysUserRole{}).Where("user_id = ?", item.ID).Count(&count)
	if count != 0 {
		t.Fatal("用户角色关联应被清理")
	}
}

func TestResetPasswordInvalidatesToken(t *testing.T) {
	svc, ctx := newEnv(t)
	testutil.SeedSuperAdmin(t, svc.DB)
	var u model.SysUser
	svc.DB.First(&u, 1)
	before := u.TokenVersion
	if err := svc.ResetPassword(ctx, 1, &UserResetPasswordInput{Password: "reset123456"}); err != nil {
		t.Fatalf("重置密码失败: %v", err)
	}
	svc.DB.First(&u, 1)
	if u.TokenVersion != before+1 {
		t.Fatal("重置密码后 token_version 应自增")
	}
}
