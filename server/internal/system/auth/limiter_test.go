package auth

import (
	"testing"
	"time"
)

func TestLimiterLocksAfterMaxFails(t *testing.T) {
	l := NewLimiter(3, time.Minute, time.Minute)
	key := "alice|1.2.3.4"
	for i := 0; i < 3; i++ {
		l.Fail(key)
	}
	locked, _ := l.Locked(key)
	if !locked {
		t.Fatal("达到最大失败次数后应锁定")
	}
	if locked, _ := l.Locked("bob|1.2.3.4"); locked {
		t.Fatal("其他 key 不应被锁定")
	}
	l.Reset(key)
	if locked, _ := l.Locked(key); locked {
		t.Fatal("重置后不应锁定")
	}
}

func TestLimiterWindowExpires(t *testing.T) {
	l := NewLimiter(2, time.Minute, time.Minute)
	key := "carol|1.2.3.4"
	l.Fail(key)
	l.Fail(key)
	if locked, _ := l.Locked(key); !locked {
		t.Fatal("达到阈值应锁定")
	}
	// 手动过期窗口
	l.mu.Lock()
	l.states[key].lockedUntil = time.Now().Add(-time.Second)
	l.mu.Unlock()
	if locked, _ := l.Locked(key); locked {
		t.Fatal("锁定过期后应解锁")
	}
}
