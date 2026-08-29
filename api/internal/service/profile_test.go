// 个人中心服务测试:自助改资料的字段边界,与头像上传的窄白名单/公开落库/旧图回收。
package service

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	pkgauth "github.com/hesunfly/hesunfly-admin-go/api/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/api/test/testutil"
)

func newProfileService(t *testing.T) (*ProfileService, context.Context) {
	t.Helper()
	db := testutil.NewTestDB(t)
	testutil.SeedSuperAdmin(t, db)
	storage, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	files := NewFileService(db, storage, 5)
	return &ProfileService{DB: db, Files: files, PublicURLPrefix: "/files"}, context.Background()
}

var profileActor = Actor{ID: 1, Username: "admin", IsSuper: true}

func TestUpdateProfileWritesOnlyEditableFields(t *testing.T) {
	svc, ctx := newProfileService(t)

	if err := svc.UpdateProfile(ctx, profileActor, ProfileInput{
		Nickname:  "爆爆龙宝宝",
		Email:     "admin@example.com",
		Phone:     "16858888988",
		Signature: "广阔天地,大有作为.",
	}); err != nil {
		t.Fatalf("更新资料失败: %v", err)
	}

	var user model.SysUser
	if err := svc.DB.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.Nickname != "爆爆龙宝宝" || user.Email != "admin@example.com" ||
		user.Phone != "16858888988" || user.Signature != "广阔天地,大有作为." {
		t.Fatalf("四个自助字段应完整落库: %+v", user)
	}
	// 账号/状态/密码/凭据版本都不在自助接口能力范围内
	if user.Username != "admin" || user.Status != model.StatusEnabled || user.TokenVersion != 0 {
		t.Fatalf("自助改资料不得改动账号/状态/凭据版本: %+v", user)
	}
	if !pkgauth.VerifyPassword(user.Password, "12345678") {
		t.Fatal("自助改资料不得改动密码哈希")
	}
}

func TestUpdateProfileClampsWithoutBreakingUTF8(t *testing.T) {
	svc, ctx := newProfileService(t)
	long := strings.Repeat("签", 300)

	if err := svc.UpdateProfile(ctx, profileActor, ProfileInput{Nickname: strings.Repeat("名", 100), Signature: long}); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	var user model.SysUser
	if err := svc.DB.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if got := []rune(user.Nickname); len(got) != 64 {
		t.Fatalf("昵称应按 rune 裁到 64,实际 %d", len(got))
	}
	if user.Signature != strings.Repeat("签", 255) {
		t.Fatal("签名裁剪必须按 rune,按字节切会产出乱码")
	}
}

// 头像白名单必须严格是"图片上传归类"的子集:个人中心不得成为放宽全局上传的入口。
func TestAvatarWhitelistIsSubsetOfImageCategory(t *testing.T) {
	imageExts := map[string]bool{}
	for _, ext := range extsInKind(kindImage) {
		imageExts[ext] = true
	}
	for ext := range avatarExts {
		if !imageExts[ext] {
			t.Fatalf("头像扩展名 %s 不在图片分类内,等于绕过全局白名单放宽上传", ext)
		}
	}
	for _, ext := range []string{".zip", ".pdf", ".docx", ".txt"} {
		if avatarExts[ext] {
			t.Fatalf("头像不应允许 %s", ext)
		}
	}
}

func pngBytes() []byte {
	return append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0}, 32)...)
}

func TestUploadAvatarStoresPublicFileAndBindsUser(t *testing.T) {
	svc, ctx := newProfileService(t)
	content := pngBytes()

	url, err := svc.UploadAvatar(ctx, profileActor, AvatarInput{
		FileName: "me.png", Size: int64(len(content)), Content: bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("头像上传失败: %v", err)
	}
	if !strings.HasPrefix(url, "/files/") || filepath.Ext(url) != ".png" {
		t.Fatalf("应返回公开前缀地址: %s", url)
	}

	var user model.SysUser
	if err := svc.DB.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.Avatar != url {
		t.Fatalf("sys_user.avatar 应指向新地址: %s vs %s", user.Avatar, url)
	}
	// 公开访问依赖 sys_file 的 is_public(见 GetPublicFile),缺一行头像就渲染不出来
	var entry model.SysFile
	storePath := strings.TrimPrefix(url, "/files/")
	if err := svc.DB.Where("store_path = ? AND is_public = ?", storePath, true).First(&entry).Error; err != nil {
		t.Fatalf("头像必须登记为公开文件,否则 /files 取不到: %v", err)
	}
}

func TestUploadAvatarRejectsNonImageExtension(t *testing.T) {
	svc, ctx := newProfileService(t)
	content := []byte("just some text")

	if _, err := svc.UploadAvatar(ctx, profileActor, AvatarInput{
		FileName: "notes.txt", Size: int64(len(content)), Content: bytes.NewReader(content),
	}); err == nil {
		t.Fatal("非图片扩展名应拒绝")
	}
	var user model.SysUser
	if err := svc.DB.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.Avatar != "" {
		t.Fatalf("失败的上传不得改动现有头像: %s", user.Avatar)
	}
}

func TestUploadAvatarRejectsOversize(t *testing.T) {
	svc, ctx := newProfileService(t)
	content := append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0}, int(MaxAvatarBytes)+1)...)

	if _, err := svc.UploadAvatar(ctx, profileActor, AvatarInput{
		FileName: "big.png", Size: int64(len(content)), Content: bytes.NewReader(content),
	}); err == nil {
		t.Fatal("超过头像体积上限应拒绝")
	}
}

// 伪装成 .png 的非图片内容:由 FileService 的真实 MIME 嗅探拦住,头像不被污染。
func TestUploadAvatarRejectsFakeImage(t *testing.T) {
	svc, ctx := newProfileService(t)
	content := []byte(strings.Repeat("not an image at all", 40))

	if _, err := svc.UploadAvatar(ctx, profileActor, AvatarInput{
		FileName: "fake.png", Size: int64(len(content)), Content: bytes.NewReader(content),
	}); err == nil {
		t.Fatal("内容与扩展名不符应拒绝")
	}
	var count int64
	svc.DB.Model(&model.SysFile{}).Count(&count)
	if count != 0 {
		t.Fatalf("被拒的头像不应留下文件记录: %d", count)
	}
}

func TestUploadAvatarReplacesAndCleansPreviousFile(t *testing.T) {
	svc, ctx := newProfileService(t)
	content := pngBytes()

	first, err := svc.UploadAvatar(ctx, profileActor, AvatarInput{
		FileName: "a.png", Size: int64(len(content)), Content: bytes.NewReader(content),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.UploadAvatar(ctx, profileActor, AvatarInput{
		FileName: "b.png", Size: int64(len(content)), Content: bytes.NewReader(content),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("两次上传应产出不同路径")
	}

	var count int64
	svc.DB.Model(&model.SysFile{}).Where("store_path = ?", strings.TrimPrefix(first, "/files/")).Count(&count)
	if count != 0 {
		t.Fatal("旧头像文件与记录应被回收,否则换头像会无限堆积文件")
	}
	var total int64
	svc.DB.Model(&model.SysFile{}).Count(&total)
	if total != 1 {
		t.Fatalf("只应保留当前头像: %d", total)
	}
}

// 不是本模块产出的头像值(外部 URL)不得被当作孤儿清理掉。
func TestCleanupLeavesForeignAvatarValueAlone(t *testing.T) {
	svc, ctx := newProfileService(t)
	if err := svc.DB.Model(&model.SysUser{ID: 1}).
		UpdateColumn("avatar", "https://cdn.example.com/x.png").Error; err != nil {
		t.Fatal(err)
	}
	content := pngBytes()
	if _, err := svc.UploadAvatar(ctx, profileActor, AvatarInput{
		FileName: "c.png", Size: int64(len(content)), Content: bytes.NewReader(content),
	}); err != nil {
		t.Fatal(err)
	}
	var user model.SysUser
	if err := svc.DB.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(user.Avatar, "/files/") {
		t.Fatalf("新头像应生效: %s", user.Avatar)
	}
}
