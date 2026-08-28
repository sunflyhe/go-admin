package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/test/testutil"
)

func newEnv(t *testing.T) (*UserService, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	return NewUserService(testutil.NewTestDB(t)), c
}

func TestCreateAndDuplicate(t *testing.T) {
	svc, c := newEnv(t)
	item, err := svc.Create(c, &UserSaveReq{Username: "alice", Password: "12345678", Nickname: "Alice"})
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if item.ID == 0 {
		t.Fatal("应返回用户 ID")
	}
	if _, err := svc.Create(c, &UserSaveReq{Username: "alice", Password: "12345678"}); err == nil {
		t.Fatal("重复用户名应报冲突")
	}
}

func TestSuperAdminProtected(t *testing.T) {
	svc, c := newEnv(t)
	testutil.SeedSuperAdmin(t, svc.DB)
	if err := svc.Delete(c, 1); err == nil {
		t.Fatal("内置超管账号不允许删除")
	}
	if err := svc.SetStatus(c, 1, &UserSetStatusReq{Status: model.StatusDisabled}); err == nil {
		t.Fatal("内置超管账号不允许停用")
	}
}

func TestDeleteCleansRelations(t *testing.T) {
	svc, c := newEnv(t)
	testutil.SeedSuperAdmin(t, svc.DB)
	item, err := svc.Create(c, &UserSaveReq{Username: "bob", Password: "12345678"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AssignRoles(c, item.ID, &UserAssignRolesReq{RoleIDs: []int64{1}}); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}
	if err := svc.Delete(c, item.ID); err != nil {
		t.Fatalf("删除用户失败: %v", err)
	}
	var count int64
	svc.DB.Model(&model.SysUserRole{}).Where("user_id = ?", item.ID).Count(&count)
	if count != 0 {
		t.Fatal("用户角色关联应被清理")
	}
}

func TestResetPasswordInvalidatesToken(t *testing.T) {
	svc, c := newEnv(t)
	testutil.SeedSuperAdmin(t, svc.DB)
	var u model.SysUser
	svc.DB.First(&u, 1)
	before := u.TokenVersion
	if err := svc.ResetPassword(c, 1, &UserResetPasswordReq{Password: "87654321"}); err != nil {
		t.Fatalf("重置密码失败: %v", err)
	}
	svc.DB.First(&u, 1)
	if u.TokenVersion != before+1 {
		t.Fatal("重置密码后 token_version 应自增")
	}
}
