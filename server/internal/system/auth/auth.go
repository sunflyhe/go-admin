// Package auth 登录认证域:登录、刷新、退出、当前用户信息。
// 安全规则:
//   - 密码仅保存 bcrypt 哈希;登录失败不区分“用户不存在/密码错误”
//   - 登录失败按 用户名+IP 限流,防止暴力破解
//   - refresh token 采用轮换 + 吊销登记,旧刷新令牌不可复用
//   - 退出后 token_version 自增,该用户所有已签发 access token 立即失效
package auth

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/middleware"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/menu"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/resp"
)

type Service struct {
	DB      *gorm.DB
	JWT     *auth.Manager
	Limiter *Limiter
	Logger  *slog.Logger
}

func NewService(db *gorm.DB, jwt *auth.Manager, logger *slog.Logger) *Service {
	return &Service{
		DB:      db,
		JWT:     jwt,
		Limiter: NewLimiter(5, 15*time.Minute, 15*time.Minute),
		Logger:  logger,
	}
}

// ---- DTO ----

type LoginReq struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=128"`
}

type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type UserProfile struct {
	ID       int64    `json:"id"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Email    string   `json:"email"`
	Phone    string   `json:"phone"`
	Avatar   string   `json:"avatar"`
	Roles    []string `json:"roles"`
	Super    bool     `json:"super"`
}

type LoginResp struct {
	TokenPair
	User UserProfile `json:"user"`
}

type RefreshReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type MeResp struct {
	User        UserProfile           `json:"user"`
	Permissions []string              `json:"permissions"`
	Menus       []menu.MenuNode       `json:"menus"`
}

// ---- Service ----

// Login 账号密码登录:校验限流、验证密码、签发 token、记录登录日志。
func (s *Service) Login(c *gin.Context, req *LoginReq) (*LoginResp, error) {
	ip := c.ClientIP()
	key := req.Username + "|" + ip
	if locked, _ := s.Limiter.Locked(key); locked {
		s.recordLoginLog(req.Username, false, "尝试次数过多被限流", c)
		return nil, errs.TooManyRequests("登录失败次数过多,请稍后再试")
	}

	var user model.SysUser
	err := s.DB.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		s.Limiter.Fail(key)
		s.recordLoginLog(req.Username, false, "账号或密码错误", c)
		return nil, errs.Unauthorized("账号或密码错误")
	}
	if !auth.VerifyPassword(user.Password, req.Password) {
		s.Limiter.Fail(key)
		s.recordLoginLog(req.Username, false, "账号或密码错误", c)
		return nil, errs.Unauthorized("账号或密码错误")
	}
	if user.Status != model.StatusEnabled {
		s.recordLoginLog(req.Username, false, "账号已停用", c)
		return nil, errs.Forbidden("账号已被停用,请联系管理员")
	}

	access, refresh, err := s.JWT.IssuePair(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		return nil, errs.Internal("签发凭据失败").WithCause(err)
	}
	if err := s.registerRefreshToken(c, user.ID, refresh); err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.DB.Model(&user).Update("last_login_at", &now).Error; err != nil {
		s.Logger.Warn("更新最后登录时间失败", "error", err)
	}
	s.Limiter.Reset(key)
	s.recordLoginLog(req.Username, true, "", c)

	return &LoginResp{
		TokenPair: TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresAt:    time.Now().Add(s.JWT.AccessTTL()),
		},
		User: s.profile(c, &user),
	}, nil
}

// registerRefreshToken 将 refresh token 的 jti 登记入库,用于轮换与吊销。
func (s *Service) registerRefreshToken(c *gin.Context, userID int64, refresh string) error {
	claims, err := s.JWT.Parse(refresh, auth.TokenTypeRefresh)
	if err != nil {
		return errs.Internal("解析刷新令牌失败").WithCause(err)
	}
	rt := model.SysRefreshToken{
		JTI:       claims.JTI,
		UserID:    userID,
		ExpiresAt: claims.ExpiresAt.Time,
	}
	if err := s.DB.WithContext(c).Create(&rt).Error; err != nil {
		return errs.Internal("登记刷新令牌失败").WithCause(err)
	}
	return nil
}

// Refresh 轮换刷新令牌:校验旧令牌登记后立即吊销,签发新 access + refresh。
func (s *Service) Refresh(c *gin.Context, req *RefreshReq) (*TokenPair, error) {
	claims, err := s.JWT.Parse(req.RefreshToken, auth.TokenTypeRefresh)
	if err != nil {
		return nil, err
	}
	var rt model.SysRefreshToken
	if err := s.DB.Where("jti = ?", claims.JTI).First(&rt).Error; err != nil {
		return nil, errs.Unauthorized("刷新凭据无效,请重新登录")
	}
	if rt.Revoked {
		// 已吊销令牌被复用,视为泄露风险,吊销该用户全部刷新令牌。
		s.revokeAll(c, claims.UserID)
		return nil, errs.Unauthorized("刷新凭据已失效,请重新登录")
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, errs.Unauthorized("刷新凭据已过期,请重新登录")
	}

	var user model.SysUser
	if err := s.DB.First(&user, claims.UserID).Error; err != nil {
		return nil, errs.Unauthorized("账号不存在或已删除")
	}
	if user.Status != model.StatusEnabled {
		return nil, errs.Forbidden("账号已被停用,请联系管理员")
	}
	if user.TokenVersion != claims.TokenVersion {
		return nil, errs.Unauthorized("凭据已失效,请重新登录")
	}

	now := time.Now()
	if err := s.DB.Model(&rt).Updates(map[string]interface{}{"revoked": true, "revoked_at": now}).Error; err != nil {
		return nil, errs.Internal("吊销旧令牌失败").WithCause(err)
	}
	access, refresh, err := s.JWT.IssuePair(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		return nil, errs.Internal("签发凭据失败").WithCause(err)
	}
	if err := s.registerRefreshToken(c, user.ID, refresh); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    time.Now().Add(s.JWT.AccessTTL()),
	}, nil
}

// Logout 退出登录:吊销该用户全部刷新令牌并自增 token_version,
// 旧 access token 与 refresh token 随即全部失效。
func (s *Service) Logout(c *gin.Context) error {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		return errs.Unauthorized("未登录")
	}
	s.revokeAll(c, user.ID)
	if err := s.DB.Model(&model.SysUser{ID: user.ID}).
		UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		return errs.Internal("退出登录失败").WithCause(err)
	}
	return nil
}

func (s *Service) revokeAll(c *gin.Context, userID int64) {
	now := time.Now()
	if err := s.DB.WithContext(c).Model(&model.SysRefreshToken{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Updates(map[string]interface{}{"revoked": true, "revoked_at": now}).Error; err != nil {
		s.Logger.Warn("吊销刷新令牌失败", "error", err)
	}
}

// Me 当前用户:基本信息、角色码、权限码集合与可见菜单树。
func (s *Service) Me(c *gin.Context) (*MeResp, error) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		return nil, errs.Unauthorized("未登录")
	}
	perms, err := s.permissions(c, user)
	if err != nil {
		return nil, err
	}
	menus, err := menu.LoadUserMenus(s.DB.WithContext(c), user)
	if err != nil {
		return nil, err
	}
	return &MeResp{
		User:        s.profile(c, user),
		Permissions: perms,
		Menus:       menus,
	}, nil
}

func (s *Service) permissions(c *gin.Context, user *model.SysUser) ([]string, error) {
	if user.IsSuperAdmin() {
		return []string{"*"}, nil
	}
	var perms []string
	err := s.DB.WithContext(c).
		Table("sys_menu").
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role_menu.role_id").
		Joins("JOIN sys_role ON sys_role.id = sys_role_menu.role_id").
		Where("sys_user_role.user_id = ?", user.ID).
		Where("sys_menu.status = ? AND sys_menu.permission <> ''", model.StatusEnabled).
		Where("sys_role.status = ?", model.StatusEnabled).
		Distinct().Pluck("sys_menu.permission", &perms).Error
	if err != nil {
		return nil, errs.Internal("加载权限失败").WithCause(err)
	}
	return perms, nil
}

func (s *Service) profile(c *gin.Context, user *model.SysUser) UserProfile {
	var roleCodes []string
	_ = s.DB.WithContext(c).
		Table("sys_role").
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role.id").
		Where("sys_user_role.user_id = ? AND sys_role.status = ?", user.ID, model.StatusEnabled).
		Pluck("sys_role.code", &roleCodes).Error
	if roleCodes == nil {
		roleCodes = []string{}
	}
	return UserProfile{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Email:    user.Email,
		Phone:    user.Phone,
		Avatar:   user.Avatar,
		Roles:    roleCodes,
		Super:    user.IsSuperAdmin(),
	}
}

func (s *Service) recordLoginLog(username string, success bool, reason string, c *gin.Context) {
	entry := model.SysLoginLog{
		Username:   username,
		Success:    success,
		FailReason: reason,
		IP:         c.ClientIP(),
		UserAgent:  truncate(c.Request.UserAgent(), 255),
	}
	if err := s.DB.Create(&entry).Error; err != nil {
		s.Logger.Error("记录登录日志失败", "error", err)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ---- Handler ----

type Handler struct {
	Svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{Svc: svc} }

// Login POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: 用户名和密码必填"))
		return
	}
	result, err := h.Svc.Login(c, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Refresh POST /api/v1/auth/refresh
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errs.InvalidParam("参数错误: refreshToken 必填"))
		return
	}
	result, err := h.Svc.Refresh(c, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}

// Logout POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	if err := h.Svc.Logout(c); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Me GET /api/v1/auth/me
func (h *Handler) Me(c *gin.Context) {
	result, err := h.Svc.Me(c)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, result)
}
