// 密码强度策略:所有设置密码的入口(创建用户、重置密码、自助改密)统一走这里。
// 放在 Service 层而非 HTTP binding:CLI/定时任务等非 HTTP 入口复用 Service 时同样受约束。
// 策略刻意保持最低共识(长度 + 字符类别),不引入弱口令字典等重型手段;
// 收紧策略会让存量账号的密码失效,属于破坏性变更,需先说明兼容性影响。
package service

import (
	"unicode"
	"unicode/utf8"

	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 128
)

// ValidatePassword 校验密码强度:至少 8 位,且同时包含字母与数字。
// 与 handler binding 的 min=8 保持一致;binding 只挡明显乱传,裁决在这里。
func ValidatePassword(password string) error {
	if n := utf8.RuneCountInString(password); n < minPasswordLen {
		return errs.InvalidParam("密码长度至少 8 位")
	} else if n > maxPasswordLen {
		return errs.InvalidParam("密码最长 128 位")
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errs.InvalidParam("密码必须同时包含字母和数字")
	}
	return nil
}
