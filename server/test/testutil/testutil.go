// Package testutil 提供测试基础设施:内存 SQLite + 种子数据。
// 说明:生产库结构由 SQL migration 管理;测试库为隔离的一次性环境,可用 AutoMigrate。
package testutil

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/auth"
	"gorm.io/gorm"
)

// NewTestDB 创建测试专属的内存数据库并建表(按测试名隔离)。
func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.SysUser{}, &model.SysRole{}, &model.SysMenu{},
		&model.SysUserRole{}, &model.SysRoleMenu{},
		&model.SysLoginLog{}, &model.SysAuditLog{},
		&model.SysFile{}, &model.SysFileGroup{}, &model.SysRefreshToken{},
		&model.ArticleCategory{}, &model.Article{}, &model.SysConfig{},
		&model.SysDictType{}, &model.SysDictItem{},
	); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	return db
}

// SeedSuperAdmin 写入超级管理员(id=1,密码 12345678)与内置角色。
func SeedSuperAdmin(t *testing.T, db *gorm.DB) model.SysUser {
	t.Helper()
	hash, err := auth.HashPassword("12345678")
	if err != nil {
		t.Fatal(err)
	}
	user := model.SysUser{ID: 1, Username: "admin", Password: hash, Nickname: "超级管理员", Status: model.StatusEnabled}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("种子用户失败: %v", err)
	}
	role := model.SysRole{ID: 1, Name: "超级管理员", Code: "super_admin", Builtin: true, Status: model.StatusEnabled}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("种子角色失败: %v", err)
	}
	if err := db.Create(&model.SysUserRole{UserID: 1, RoleID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	return user
}
