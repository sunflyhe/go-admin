// 认证接口的 HTTP 绑定 DTO。
package handler

import "github.com/hesunfly/hesunfly-admin-go/api/internal/service"

// LoginRequest POST /api/v1/auth/login 请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=128"`
}

// RefreshRequest POST /api/v1/auth/refresh 请求体。
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// ProfileUpdateRequest PUT /api/v1/auth/profile 请求体。
// 只开放本人资料四个字段:username/status/roles 不在其中,密码与头像各有专用接口,
// 避免开出一个"自助什么都能改"的入口。
type ProfileUpdateRequest struct {
	Nickname  string `json:"nickname" binding:"max=64"`
	Email     string `json:"email" binding:"max=128"`
	Phone     string `json:"phone" binding:"max=32"`
	Signature string `json:"signature" binding:"max=255"`
}

func (r *ProfileUpdateRequest) toInput() service.ProfileInput {
	return service.ProfileInput{
		Nickname:  r.Nickname,
		Email:     r.Email,
		Phone:     r.Phone,
		Signature: r.Signature,
	}
}

// ChangePasswordRequest POST /api/v1/auth/password 请求体。
// 长度规则与管理员重置保持一致(min=8),不额外发明密码策略。
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required,max=128"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
}

// AvatarResponse POST /api/v1/auth/avatar 响应:头像最终访问地址。
type AvatarResponse struct {
	Avatar string `json:"avatar"`
}
