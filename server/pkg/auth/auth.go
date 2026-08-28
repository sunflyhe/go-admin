// Package auth 提供 JWT 签发/校验与密码哈希。
// access token 与 refresh token 使用同一密钥但携带不同的 tok 类型声明,防止混用。
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims access/refresh token 共用的声明结构。
type Claims struct {
	UserID       int64  `json:"uid"`
	Username     string `json:"username"`
	TokenVersion int64  `json:"ver"`
	TokenType    string `json:"typ"`
	JTI          string `json:"jti"`
	jwt.RegisteredClaims
}

// Manager 管理 token 的签发与校验。
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewManager(secret string, accessTTL time.Duration, refreshTTL time.Duration, issuer string) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL, issuer: issuer}
}

func (m *Manager) AccessTTL() time.Duration  { return m.accessTTL }
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// IssuePair 为用户签发新的 access + refresh token。
func (m *Manager) IssuePair(userID int64, username string, version int64) (access, refresh string, err error) {
	access, err = m.issue(userID, username, version, TokenTypeAccess, m.accessTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = m.issue(userID, username, version, TokenTypeRefresh, m.refreshTTL)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (m *Manager) issue(userID int64, username string, version int64, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:       userID,
		Username:     username,
		TokenVersion: version,
		TokenType:    typ,
		JTI:          uuid.NewString(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 解析并校验 token,onlyType 指定必须匹配的 token 类型。
func (m *Manager) Parse(tokenStr, onlyType string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errs.New(errs.CodeTokenExpired, "登录已过期,请重新登录", 401)
		}
		return nil, errs.Unauthorized("无效的凭据")
	}
	if !token.Valid {
		return nil, errs.Unauthorized("无效的凭据")
	}
	if claims.TokenType != onlyType {
		return nil, errs.Unauthorized("凭据类型不正确")
	}
	return claims, nil
}

// HashPassword 使用 bcrypt 生成密码哈希。
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword 校验明文密码与哈希是否匹配。
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
