// 登录失败限流:进程内存实现,固定窗口计数。
// 达到阈值后锁定一段时间;单实例部署足够,引入 Redis 前不伪装分布式能力。
package auth

import (
	"sync"
	"time"
)

type windowState struct {
	failures    int
	windowStart time.Time
	lockedUntil time.Time
}

// Limiter 按 key(如 username+IP)限制失败次数。
type Limiter struct {
	mu         sync.Mutex
	states     map[string]*windowState
	maxFails   int
	window     time.Duration
	lockPeriod time.Duration
}

func NewLimiter(maxFails int, window, lockPeriod time.Duration) *Limiter {
	return &Limiter{
		states:     make(map[string]*windowState),
		maxFails:   maxFails,
		window:     window,
		lockPeriod: lockPeriod,
	}
}

// Locked 判断 key 是否处于锁定状态,返回剩余锁定秒数。
func (l *Limiter) Locked(key string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, ok := l.states[key]
	if !ok {
		return false, 0
	}
	if s.lockedUntil.After(time.Now()) {
		return true, int(time.Until(s.lockedUntil).Seconds()) + 1
	}
	return false, 0
}

// Fail 记录一次失败,达到阈值时进入锁定。
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	s, ok := l.states[key]
	if !ok || now.Sub(s.windowStart) > l.window {
		s = &windowState{windowStart: now}
		l.states[key] = s
	}
	s.failures++
	if s.failures >= l.maxFails {
		s.lockedUntil = now.Add(l.lockPeriod)
	}
}

// Reset 成功登录后清空计数。
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.states, key)
}
