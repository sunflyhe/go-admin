package auth

import (
	"testing"
	"time"
)

func TestIssueAndParse(t *testing.T) {
	m := NewManager("test-secret-1234567890", 30*time.Minute, 24*time.Hour, "go-admin")
	access, refresh, err := m.IssuePair(42, "alice", 3)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	claims, err := m.Parse(access, TokenTypeAccess)
	if err != nil {
		t.Fatalf("解析 access 失败: %v", err)
	}
	if claims.UserID != 42 || claims.Username != "alice" || claims.TokenVersion != 3 {
		t.Fatalf("claims 不匹配: %+v", claims)
	}
	if _, err := m.Parse(refresh, TokenTypeRefresh); err != nil {
		t.Fatalf("解析 refresh 失败: %v", err)
	}
	// 类型不可混用:refresh 不能当 access 用
	if _, err := m.Parse(refresh, TokenTypeAccess); err == nil {
		t.Fatal("refresh token 不应能通过 access 校验")
	}
	if _, err := m.Parse(access, TokenTypeRefresh); err == nil {
		t.Fatal("access token 不应能通过 refresh 校验")
	}
}

func TestParseInvalid(t *testing.T) {
	m := NewManager("test-secret-1234567890", 30*time.Minute, 24*time.Hour, "go-admin")
	other := NewManager("another-secret-12345678", 30*time.Minute, 24*time.Hour, "go-admin")
	token, _, _ := other.IssuePair(1, "bob", 0)
	if _, err := m.Parse(token, TokenTypeAccess); err == nil {
		t.Fatal("不同密钥签发的 token 不应通过校验")
	}
	if _, err := m.Parse("garbage.token.value", TokenTypeAccess); err == nil {
		t.Fatal("非法 token 不应通过校验")
	}
}

func TestPasswordHash(t *testing.T) {
	hash, err := HashPassword("s3cret-password")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "s3cret-password" {
		t.Fatal("密码必须被哈希")
	}
	if !VerifyPassword(hash, "s3cret-password") {
		t.Fatal("正确密码应通过校验")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatal("错误密码不应通过校验")
	}
}
