// 个人中心服务:当前登录用户自助维护资料与头像。
// 与 UserService 的边界:这里只允许操作者改自己的少数几个字段,入参里根本没有
// username/status/roles/password —— 不提供一个"什么都能改"的自助接口。
package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

// MaxAvatarBytes 头像体积上限:头像不需要高清原图,收紧上限同时压住公开目录的存储与带宽。
// 导出是为了 Handler 设置请求体上限时复用同一个数值,不在两层各写一遍。
const MaxAvatarBytes = 2 << 20

// avatarExts 是头像专用白名单,比全局上传白名单更窄(只放图片)。
// 全局白名单 fileExtKind 不因头像而放宽;测试断言本集合必须是图片归类的子集。
var avatarExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// ProfileInput 自助资料更新输入。
type ProfileInput struct {
	Nickname  string
	Email     string
	Phone     string
	Signature string
}

// AvatarInput 头像上传输入:与 HTTP 无关,由 Handler 从 multipart 构造。
type AvatarInput struct {
	FileName string
	Size     int64
	Content  io.Reader
}

type ProfileService struct {
	DB              *gorm.DB
	Files           *FileService
	PublicURLPrefix string
}

func NewProfileService(db *gorm.DB, files *FileService, publicURLPrefix string) *ProfileService {
	return &ProfileService{DB: db, Files: files, PublicURLPrefix: publicURLPrefix}
}

// clampRunes 按列长度截断。必须按 rune 而不是字节切,否则中文昵称/签名会被切成乱码。
func clampRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}

// UpdateProfile 更新本人资料。按列长度裁剪,避免超长值在 MySQL 严格模式下直接报错。
func (s *ProfileService) UpdateProfile(ctx context.Context, actor Actor, in ProfileInput) error {
	updates := map[string]interface{}{
		"nickname":  clampRunes(in.Nickname, 64),
		"email":     clampRunes(in.Email, 128),
		"phone":     clampRunes(in.Phone, 32),
		"signature": clampRunes(in.Signature, 255),
	}
	if err := s.DB.WithContext(ctx).Model(&model.SysUser{ID: actor.ID}).Updates(updates).Error; err != nil {
		return errs.Internal("更新个人资料失败").WithCause(err)
	}
	return nil
}

// UploadAvatar 上传并绑定本人头像。
// 头像必须存成公开文件:<img> 带不了 Authorization,私有头像在页面上渲染不出来;
// 路径含随机 UUID 不可枚举,且落盘/元数据仍由 FileService 统一负责。
func (s *ProfileService) UploadAvatar(ctx context.Context, actor Actor, in AvatarInput) (string, error) {
	if in.Size <= 0 || in.Size > MaxAvatarBytes {
		return "", errs.InvalidParam(fmt.Sprintf("头像大小需小于 %dMB", MaxAvatarBytes>>20))
	}
	ext := strings.ToLower(filepath.Ext(in.FileName))
	if !avatarExts[ext] {
		return "", errs.InvalidParam("头像仅支持 jpg/jpeg/png/gif/webp")
	}
	// 真实 MIME 嗅探、落盘与 sys_file 记录都在 FileService 内完成:
	// 公开访问要求 is_public 行为真(见 GetPublicFile),所以这里必须走完整上传。
	result, err := s.Files.Upload(ctx, actor, &FileUploadInput{
		FileName: in.FileName,
		Size:     in.Size,
		Content:  in.Content,
		IsPublic: true,
	})
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(result.MIME, "image/") {
		// 扩展名与内容不一致:回收刚落盘的文件,不把它绑成头像
		_ = s.Files.Delete(ctx, result.ID)
		return "", errs.InvalidParam("头像内容不是有效图片")
	}

	var user model.SysUser
	if err := s.DB.WithContext(ctx).First(&user, actor.ID).Error; err != nil {
		return "", errs.Unauthorized("账号不存在或已删除")
	}
	// 先传新图再清旧图:新图失败时旧头像仍然可用
	s.cleanupPreviousAvatar(ctx, user.Avatar)

	url := s.PublicURLPrefix + "/" + result.StorePath
	if err := s.DB.WithContext(ctx).Model(&model.SysUser{ID: actor.ID}).
		UpdateColumn("avatar", url).Error; err != nil {
		return "", errs.Internal("更新头像失败").WithCause(err)
	}
	return url, nil
}

// cleanupPreviousAvatar 只回收由本接口产出的旧头像(前缀匹配且能反查到记录)。
// 历史脏值或外部 URL 一律不动,避免误删他人文件。清理失败不影响新头像生效。
func (s *ProfileService) cleanupPreviousAvatar(ctx context.Context, previous string) {
	prefix := s.PublicURLPrefix + "/"
	if previous == "" || !strings.HasPrefix(previous, prefix) {
		return
	}
	var entry model.SysFile
	if err := s.DB.WithContext(ctx).
		Where("store_path = ?", strings.TrimPrefix(previous, prefix)).
		First(&entry).Error; err != nil {
		return
	}
	_ = s.Files.Delete(ctx, entry.ID)
}
