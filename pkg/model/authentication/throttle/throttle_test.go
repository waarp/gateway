package throttle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

func testConfig() *conf.SecurityConfig {
	return &conf.SecurityConfig{
		MaxAuthFailures:     3,
		AuthFailureWindow:   5 * time.Minute,
		AuthLockDuration:    time.Minute,
		AuthMaxLockDuration: 3 * time.Minute,
	}
}

func testLimiter() (*Limiter, *time.Time) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	l := New()
	l.now = func() time.Time { return now }

	return l, &now
}

func TestLimiter(t *testing.T) {
	t.Parallel()

	const (
		scope = "test"
		login = "toto"
		addr  = "1.2.3.4:5555"
	)

	t.Run("Locks after the maximum number of failures", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, now := testLimiter()

		assert.False(t, l.Failure(cfg, scope, login, addr))
		assert.False(t, l.Failure(cfg, scope, login, addr))
		assert.False(t, l.Locked(cfg, scope, login, addr))
		assert.True(t, l.Failure(cfg, scope, login, addr), "third failure locks")

		assert.True(t, l.Locked(cfg, scope, login, addr))
		assert.True(t, l.Locked(cfg, scope, login, "1.2.3.4:6666"), "same host, other port")
		assert.False(t, l.Locked(cfg, scope, login, "5.6.7.8:5555"), "other address")
		assert.False(t, l.Locked(cfg, scope, "other", addr), "other login")
		assert.False(t, l.Locked(cfg, "other", login, addr), "other entry point")

		*now = now.Add(time.Minute + time.Second)
		assert.False(t, l.Locked(cfg, scope, login, addr), "the lock expires")
	})

	t.Run("The lock duration doubles, up to the maximum", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, now := testLimiter()

		lockFor := func() time.Duration {
			for range cfg.MaxAuthFailures {
				l.Failure(cfg, scope, login, addr)
			}

			start := *now
			for l.Locked(cfg, scope, login, addr) {
				*now = now.Add(time.Second)
			}

			return now.Sub(start)
		}

		assert.Equal(t, time.Minute, lockFor())
		assert.Equal(t, 2*time.Minute, lockFor())
		assert.Equal(t, 3*time.Minute, lockFor(), "capped")
	})

	t.Run("A success resets the counter", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, _ := testLimiter()

		l.Failure(cfg, scope, login, addr)
		l.Failure(cfg, scope, login, addr)
		l.Success(scope, login, addr)

		assert.False(t, l.Failure(cfg, scope, login, addr))
		assert.False(t, l.Failure(cfg, scope, login, addr))
		assert.False(t, l.Locked(cfg, scope, login, addr))
	})

	t.Run("Old failures are forgotten", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, now := testLimiter()

		l.Failure(cfg, scope, login, addr)
		l.Failure(cfg, scope, login, addr)

		*now = now.Add(6 * time.Minute)

		assert.False(t, l.Failure(cfg, scope, login, addr))
		assert.False(t, l.Locked(cfg, scope, login, addr))
	})

	t.Run("Disabled when the maximum is 0", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		cfg.MaxAuthFailures = 0
		l, _ := testLimiter()

		for range 100 {
			assert.False(t, l.Failure(cfg, scope, login, addr))
		}

		assert.False(t, l.Locked(cfg, scope, login, addr))
	})
}
