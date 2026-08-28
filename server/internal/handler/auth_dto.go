// 认证接口的 HTTP 绑定 DTO。
package handler

// LoginRequest POST /api/v1/auth/login 请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=128"`
}

// RefreshRequest POST /api/v1/auth/refresh 请求体。
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}
