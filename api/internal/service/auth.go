// 登录认证服务:登录、刷新、退出、当前用户信息。
// 安全规则:
//   - 密码仅保存 bcrypt 哈希;登录失败不区分“用户不存在/密码错误”
//   - 登录失败按 用户名+IP 限流,防止暴力破解
//   - refresh token 采用轮换 + 吊销登记,旧刷新令牌不可复用
//   - 退出后 token_version 自增,该用户所有已签发 access token 立即失效
package service

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/hesunfly/hesunfly-admin-go/api/internal/model"
	pkgauth "github.com/hesunfly/hesunfly-admin-go/api/pkg/auth"
	"github.com/hesunfly/hesunfly-admin-go/api/pkg/errs"
)

// AuthService 登录认证域服务。
// 两级失败限流(均为进程内存实现,与 Limiter 同一前提:单实例足够):
//   - Limiter 按 用户名+IP:防单点暴力破解;
//   - AccountLimiter 仅按用户名:防换 IP 分布式爆破。只对真实存在的账号计数,
//     避免任意用户名可被第三方恶意锁定(对不存在账号的探测由 IP 维度拦截)。
type AuthService struct {
	DB             *gorm.DB
	JWT            *pkgauth.Manager
	Limiter        *Limiter
	AccountLimiter *Limiter
	Logger         *slog.Logger
}

func NewAuthService(db *gorm.DB, jwt *pkgauth.Manager, logger *slog.Logger) *AuthService {
	return &AuthService{
		DB:             db,
		JWT:            jwt,
		Limiter:        NewLimiter(5, 15*time.Minute, 15*time.Minute),
		AccountLimiter: NewLimiter(10, 15*time.Minute, 15*time.Minute),
		Logger:         logger,
	}
}

// ---- 输出 DTO(输入为方法参数:账号密码、LoginMeta、Actor)----

type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type UserProfile struct {
	ID        int64    `json:"id"`
	Username  string   `json:"username"`
	Nickname  string   `json:"nickname"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	Avatar    string   `json:"avatar"`
	Signature string   `json:"signature"`
	Roles     []string `json:"roles"`
	Super     bool     `json:"super"`
}

type LoginResp struct {
	TokenPair
	User UserProfile `json:"user"`
}

type MeResp struct {
	User        UserProfile `json:"user"`
	Permissions []string    `json:"permissions"`
	Menus       []MenuNode  `json:"menus"`
}

// ---- Service ----

// Login 账号密码登录:校验限流、验证密码、签发 token、记录登录日志。
// HTTP 元数据由 Handler 通过 meta 显式传入,Service 不感知 Gin。
func (s *AuthService) Login(ctx context.Context, username, password string, meta LoginMeta) (*LoginResp, error) {
	key := username + "|" + meta.IP
	if locked, _ := s.Limiter.Locked(key); locked {
		s.recordLoginLog(ctx, username, false, "尝试次数过多被限流", meta)
		return nil, errs.TooManyRequests("登录失败次数过多,请稍后再试")
	}
	if locked, _ := s.AccountLimiter.Locked(username); locked {
		s.recordLoginLog(ctx, username, false, "账号连续失败被锁定", meta)
		return nil, errs.TooManyRequests("该账号登录失败次数过多,已被临时锁定,请稍后再试")
	}

	var user model.SysUser
	err := s.DB.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		s.Limiter.Fail(key)
		s.recordLoginLog(ctx, username, false, "账号或密码错误", meta)
		return nil, errs.Unauthorized("账号或密码错误")
	}
	if !pkgauth.VerifyPassword(user.Password, password) {
		s.Limiter.Fail(key)
		s.AccountLimiter.Fail(username)
		s.recordLoginLog(ctx, username, false, "账号或密码错误", meta)
		return nil, errs.Unauthorized("账号或密码错误")
	}
	if user.Status != model.StatusEnabled {
		s.recordLoginLog(ctx, username, false, "账号已停用", meta)
		return nil, errs.Forbidden("账号已被停用,请联系管理员")
	}

	access, refresh, err := s.JWT.IssuePair(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		return nil, errs.Internal("签发凭据失败").WithCause(err)
	}
	if err := s.registerRefreshToken(ctx, user.ID, refresh); err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.DB.WithContext(ctx).Model(&user).Update("last_login_at", &now).Error; err != nil {
		s.Logger.Warn("更新最后登录时间失败", "error", err)
	}
	s.Limiter.Reset(key)
	s.AccountLimiter.Reset(username)
	s.recordLoginLog(ctx, username, true, "", meta)

	return &LoginResp{
		TokenPair: TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresAt:    time.Now().Add(s.JWT.AccessTTL()),
		},
		User: s.profile(ctx, &user),
	}, nil
}

// registerRefreshToken 将 refresh token 的 jti 登记入库,用于轮换与吊销。
func (s *AuthService) registerRefreshToken(ctx context.Context, userID int64, refresh string) error {
	claims, err := s.JWT.Parse(refresh, pkgauth.TokenTypeRefresh)
	if err != nil {
		return errs.Internal("解析刷新令牌失败").WithCause(err)
	}
	rt := model.SysRefreshToken{
		JTI:       claims.JTI,
		UserID:    userID,
		ExpiresAt: claims.ExpiresAt.Time,
	}
	if err := s.DB.WithContext(ctx).Create(&rt).Error; err != nil {
		return errs.Internal("登记刷新令牌失败").WithCause(err)
	}
	return nil
}

// Refresh 轮换刷新令牌:校验旧令牌登记后立即吊销,签发新 access + refresh。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.JWT.Parse(refreshToken, pkgauth.TokenTypeRefresh)
	if err != nil {
		return nil, err
	}
	var rt model.SysRefreshToken
	if err := s.DB.WithContext(ctx).Where("jti = ?", claims.JTI).First(&rt).Error; err != nil {
		return nil, errs.Unauthorized("刷新凭据无效,请重新登录")
	}
	if rt.Revoked {
		// 已吊销令牌被复用,视为泄露风险,吊销该用户全部刷新令牌。
		s.revokeAll(ctx, claims.UserID)
		return nil, errs.Unauthorized("刷新凭据已失效,请重新登录")
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, errs.Unauthorized("刷新凭据已过期,请重新登录")
	}

	var user model.SysUser
	if err := s.DB.WithContext(ctx).First(&user, claims.UserID).Error; err != nil {
		return nil, errs.Unauthorized("账号不存在或已删除")
	}
	if user.Status != model.StatusEnabled {
		return nil, errs.Forbidden("账号已被停用,请联系管理员")
	}
	if user.TokenVersion != claims.TokenVersion {
		return nil, errs.Unauthorized("凭据已失效,请重新登录")
	}

	now := time.Now()
	// 必须带 revoked=false 条件更新：两个并发 refresh 都完成前置读取时，
	// 仅允许其中一个请求消费旧令牌并签发新的 token pair。
	res := s.DB.WithContext(ctx).Model(&model.SysRefreshToken{}).
		Where("jti = ? AND revoked = ?", claims.JTI, false).
		Updates(map[string]interface{}{"revoked": true, "revoked_at": now})
	if res.Error != nil {
		return nil, errs.Internal("吊销旧令牌失败").WithCause(res.Error)
	}
	if res.RowsAffected != 1 {
		s.revokeAll(ctx, claims.UserID)
		return nil, errs.Unauthorized("刷新凭据已失效,请重新登录")
	}
	access, refresh, err := s.JWT.IssuePair(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		return nil, errs.Internal("签发凭据失败").WithCause(err)
	}
	if err := s.registerRefreshToken(ctx, user.ID, refresh); err != nil {
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
func (s *AuthService) Logout(ctx context.Context, actor Actor) error {
	s.revokeAll(ctx, actor.ID)
	if err := s.DB.WithContext(ctx).Model(&model.SysUser{ID: actor.ID}).
		UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		return errs.Internal("退出登录失败").WithCause(err)
	}
	return nil
}

// ChangePassword 自助改密:必须验证原密码,以此证明"是本人"。
// 与管理员重置(UserService.ResetPassword)的差别只在于这道证明;失效已签发凭据的语义一致 ——
// token_version 自增让 access token 立刻不可用,revokeAll 让 refresh token 无法再续期。
func (s *AuthService) ChangePassword(ctx context.Context, actor Actor, oldPassword, newPassword string) error {
	var user model.SysUser
	if err := s.DB.WithContext(ctx).First(&user, actor.ID).Error; err != nil {
		return errs.Unauthorized("账号不存在或已删除")
	}
	if !pkgauth.VerifyPassword(user.Password, oldPassword) {
		return errs.InvalidParam("原密码不正确")
	}
	if oldPassword == newPassword {
		return errs.InvalidParam("新密码不能与原密码相同")
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := pkgauth.HashPassword(newPassword)
	if err != nil {
		return errs.Internal("密码加密失败").WithCause(err)
	}
	if err := s.DB.WithContext(ctx).Model(&model.SysUser{ID: user.ID}).
		Updates(map[string]interface{}{"password": hash, "token_version": gorm.Expr("token_version + 1")}).Error; err != nil {
		return errs.Internal("修改密码失败").WithCause(err)
	}
	s.revokeAll(ctx, user.ID)
	return nil
}

func (s *AuthService) revokeAll(ctx context.Context, userID int64) {
	now := time.Now()
	if err := s.DB.WithContext(ctx).Model(&model.SysRefreshToken{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Updates(map[string]interface{}{"revoked": true, "revoked_at": now}).Error; err != nil {
		s.Logger.Warn("吊销刷新令牌失败", "error", err)
	}
}

// Me 当前用户:基本信息、角色码、权限码集合与可见菜单树。
// 用户状态与版本校验由认证中间件完成,这里按 actor.ID 重新加载资料。
func (s *AuthService) Me(ctx context.Context, actor Actor) (*MeResp, error) {
	var user model.SysUser
	if err := s.DB.WithContext(ctx).First(&user, actor.ID).Error; err != nil {
		return nil, errs.Unauthorized("账号不存在或已删除")
	}
	perms, err := s.permissions(ctx, &user)
	if err != nil {
		return nil, err
	}
	menus, err := LoadUserMenus(ctx, s.DB, &user)
	if err != nil {
		return nil, err
	}
	return &MeResp{
		User:        s.profile(ctx, &user),
		Permissions: perms,
		Menus:       menus,
	}, nil
}

func (s *AuthService) permissions(ctx context.Context, user *model.SysUser) ([]string, error) {
	if user.IsSuperAdmin() {
		return []string{"*"}, nil
	}
	var perms []string
	err := s.DB.WithContext(ctx).
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

func (s *AuthService) profile(ctx context.Context, user *model.SysUser) UserProfile {
	var roleCodes []string
	_ = s.DB.WithContext(ctx).
		Table("sys_role").
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role.id").
		Where("sys_user_role.user_id = ? AND sys_role.status = ?", user.ID, model.StatusEnabled).
		Pluck("sys_role.code", &roleCodes).Error
	if roleCodes == nil {
		roleCodes = []string{}
	}
	return UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Phone:     user.Phone,
		Avatar:    user.Avatar,
		Signature: user.Signature,
		Roles:     roleCodes,
		Super:     user.IsSuperAdmin(),
	}
}

func (s *AuthService) recordLoginLog(ctx context.Context, username string, success bool, reason string, meta LoginMeta) {
	entry := model.SysLoginLog{
		Username:   username,
		Success:    success,
		FailReason: reason,
		IP:         meta.IP,
		UserAgent:  truncate(meta.UserAgent, 255),
	}
	if err := s.DB.WithContext(ctx).Create(&entry).Error; err != nil {
		s.Logger.Error("记录登录日志失败", "error", err)
	}
}
