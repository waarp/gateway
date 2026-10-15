package throttle

import (
	"fmt"
	"hash/maphash"
	"math"
	"strings"
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
		attempt := l.Attempt(cfg, scope, login, addr)

		assert.False(t, attempt.Fail())
		assert.False(t, attempt.Fail())
		assert.False(t, attempt.Locked())
		assert.True(t, attempt.Fail(), "third failure locks")

		assert.True(t, attempt.Locked())
		assert.True(t, l.Attempt(cfg, scope, login, "1.2.3.4:6666").Locked(), "same host, other port")
		assert.False(t, l.Attempt(cfg, scope, login, "5.6.7.8:5555").Locked(), "other address")
		assert.False(t, l.Attempt(cfg, scope, "other", addr).Locked(), "other login")
		assert.False(t, l.Attempt(cfg, "other", login, addr).Locked(), "other entry point")

		*now = now.Add(time.Minute + time.Second)
		assert.False(t, attempt.Locked(), "the lock expires")
	})

	t.Run("The lock duration doubles, up to the maximum", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, now := testLimiter()
		attempt := l.Attempt(cfg, scope, login, addr)

		lockFor := func() time.Duration {
			for range cfg.MaxAuthFailures {
				attempt.Fail()
			}

			start := *now
			for attempt.Locked() {
				*now = now.Add(time.Second)
			}

			return now.Sub(start)
		}

		assert.Equal(t, time.Minute, lockFor())
		assert.Equal(t, 2*time.Minute, lockFor())
		assert.Equal(t, 3*time.Minute, lockFor(), "capped")
	})

	t.Run("The escalation is forgotten a window after the end of the lock", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		cfg.MaxAuthFailures = 1
		cfg.AuthFailureWindow = time.Minute
		cfg.AuthLockDuration = 2 * time.Minute
		cfg.AuthMaxLockDuration = time.Hour
		l, now := testLimiter()
		attempt := l.Attempt(cfg, scope, login, addr)

		lockFor := func() time.Duration {
			attempt.Fail()

			return l.entries[attempt.key].lockedUntil.Sub(*now)
		}

		assert.Equal(t, 2*time.Minute, lockFor())

		// The lock is longer than the window: the last failure is older
		// than the window, but the lock ended less than a window ago.
		*now = now.Add(2*time.Minute + 30*time.Second)
		assert.Equal(t, 4*time.Minute, lockFor(), "the escalation goes on")

		// A purge, triggered by a new key, must not change anything.
		*now = now.Add(4*time.Minute + 30*time.Second)
		l.Attempt(cfg, scope, "other", addr).Fail()
		assert.Equal(t, 8*time.Minute, lockFor(), "the escalation survives the purge")

		*now = now.Add(8*time.Minute + cfg.AuthFailureWindow + time.Second)
		assert.Equal(t, 2*time.Minute, lockFor(), "the escalation is reset")
	})

	t.Run("A success resets the counter", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, _ := testLimiter()
		attempt := l.Attempt(cfg, scope, login, addr)

		attempt.Fail()
		attempt.Fail()
		attempt.Succeed()

		assert.Empty(t, l.entries)
		assert.False(t, attempt.Fail())
		assert.False(t, attempt.Fail())
		assert.False(t, attempt.Locked())
	})

	t.Run("A success without previous failure does not create an entry", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, _ := testLimiter()

		l.Attempt(cfg, scope, "other", addr).Fail()
		l.Attempt(cfg, scope, login, addr).Succeed()

		assert.Len(t, l.entries, 1)
		assert.Contains(t, l.entries, l.key(scope, "other", addr))
	})

	t.Run("Old failures are forgotten", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, now := testLimiter()
		attempt := l.Attempt(cfg, scope, login, addr)

		attempt.Fail()
		attempt.Fail()

		*now = now.Add(6 * time.Minute)

		assert.False(t, attempt.Fail())
		assert.False(t, attempt.Locked())
	})

	t.Run("Disabled when the maximum is 0", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		cfg.MaxAuthFailures = 0
		l, _ := testLimiter()
		attempt := l.Attempt(cfg, scope, login, addr)

		for range 100 {
			assert.False(t, attempt.Fail())
		}

		assert.False(t, attempt.Locked())
		assert.Empty(t, l.entries)
	})

	t.Run("The lock duration does not overflow", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		cfg.MaxAuthFailures = 1
		cfg.AuthMaxLockDuration = 0
		l, now := testLimiter()
		attempt := l.Attempt(cfg, scope, login, addr)

		for i := range 100 {
			start := *now

			assert.True(t, attempt.Fail())
			assert.True(t, attempt.Locked(), "lock #%d must be effective", i+1)

			end := l.entries[attempt.key].lockedUntil
			assert.True(t, end.After(start), "lock #%d must end in the future", i+1)
		}

		assert.Less(t, l.entries[attempt.key].locks, uint(64),
			"the lock counter must stop growing once the maximum is reached")
	})

	t.Run("Old entries are purged even when the limiter is not full", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, now := testLimiter()

		for i := range 10 {
			l.Attempt(cfg, scope, fmt.Sprint("user", i), addr).Fail()
		}

		*now = now.Add(cfg.AuthFailureWindow + time.Second)
		l.Attempt(cfg, scope, login, addr).Fail()

		assert.Len(t, l.entries, 1)
	})

	t.Run("The number of entries is bounded", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, _ := testLimiter()

		// All these entries are recent, so none of them can be purged.
		for i := range maxEntries + 100 {
			l.Attempt(cfg, scope, fmt.Sprint("user", i), addr).Fail()
		}

		assert.Len(t, l.entries, maxEntries)
		assert.Contains(t, l.entries, l.key(scope, fmt.Sprint("user", maxEntries+99), addr),
			"the newest entry must be tracked")
	})

	t.Run("Locked entries are evicted last", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, _ := testLimiter()
		attempt := l.Attempt(cfg, scope, login, addr)

		for range cfg.MaxAuthFailures {
			attempt.Fail()
		}

		for i := range maxEntries + 100 {
			l.Attempt(cfg, scope, fmt.Sprint("user", i), addr).Fail()
		}

		assert.Len(t, l.entries, maxEntries)
		assert.True(t, attempt.Locked())

		// When all the entries are locked, the lock ending first is evicted.
		l2, now2 := testLimiter()
		cfg.MaxAuthFailures = 1

		for i := range maxEntries {
			*now2 = now2.Add(time.Millisecond)
			l2.Attempt(cfg, scope, fmt.Sprint("user", i), addr).Fail()
		}

		l2.Attempt(cfg, scope, login, addr).Fail()

		assert.Len(t, l2.entries, maxEntries)
		assert.NotContains(t, l2.entries, l2.key(scope, "user0", addr))
		assert.Contains(t, l2.entries, l2.key(scope, "user1", addr))
		assert.Contains(t, l2.entries, l2.key(scope, login, addr))
	})

	t.Run("The login is stored as a hash", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig()
		l, _ := testLimiter()
		long := strings.Repeat("a", 1<<20)
		attempt := l.Attempt(cfg, scope, long, addr)

		for range cfg.MaxAuthFailures {
			attempt.Fail()
		}

		assert.True(t, l.Attempt(cfg, scope, long, addr).Locked(), "same login, same counter")
		assert.False(t, l.Attempt(cfg, scope, long+"b", addr).Locked(), "other login")
		assert.False(t, l.Attempt(cfg, scope, long[:len(long)-1], addr).Locked(), "other login")

		assert.Equal(t, maphash.String(l.seed, long), attempt.key.login)
		assert.NotEqual(t, New().key(scope, long, addr), attempt.key,
			"each limiter has its own seed")
	})
}

func TestPasswordSpraying(t *testing.T) {
	t.Parallel()

	const (
		scope = "test"
		addr  = "1.2.3.4:5555"
	)

	sprayConfig := func() *conf.SecurityConfig {
		cfg := testConfig()
		cfg.MaxAuthFailuresPerHost = 10

		return cfg
	}

	// spray makes one failure for each of n different logins from addr, and
	// returns whether the last failure caused a lock.
	spray := func(l *Limiter, cfg *conf.SecurityConfig, n int) bool {
		var locked bool
		for i := range n {
			locked = l.Attempt(cfg, scope, fmt.Sprint("user", i), addr).Fail()
		}

		return locked
	}

	t.Run("An address failing on many logins is locked", func(t *testing.T) {
		t.Parallel()

		cfg := sprayConfig()
		l, now := testLimiter()

		assert.False(t, spray(l, cfg, 9))
		assert.False(t, l.Attempt(cfg, scope, "fresh", addr).Locked())
		assert.True(t, l.Attempt(cfg, scope, "last", addr).Fail(), "10th failure locks")

		assert.True(t, l.Attempt(cfg, scope, "fresh", addr).Locked(), "any login")
		assert.True(t, l.Attempt(cfg, scope, "fresh", "1.2.3.4:6666").Locked(), "other port")
		assert.False(t, l.Attempt(cfg, scope, "fresh", "5.6.7.8:5555").Locked(), "other address")
		assert.False(t, l.Attempt(cfg, "other", "fresh", addr).Locked(), "other entry point")

		*now = now.Add(cfg.AuthLockDuration + time.Second)
		assert.False(t, l.Attempt(cfg, scope, "fresh", addr).Locked(), "the lock expires")
	})

	t.Run("A success does not reset the counter of the address", func(t *testing.T) {
		t.Parallel()

		cfg := sprayConfig()
		l, _ := testLimiter()

		assert.False(t, spray(l, cfg, 9))
		l.Attempt(cfg, scope, "valid", addr).Succeed()

		assert.True(t, l.Attempt(cfg, scope, "last", addr).Fail())
		assert.True(t, l.Attempt(cfg, scope, "valid", addr).Locked())
	})

	t.Run("IPv6 addresses are grouped by /64", func(t *testing.T) {
		t.Parallel()

		cfg := sprayConfig()
		l, _ := testLimiter()

		for i := range 10 {
			l.Attempt(cfg, scope, fmt.Sprint("user", i), fmt.Sprintf("[2001:db8::%x]:5555", i+1)).Fail()
		}

		assert.True(t, l.Attempt(cfg, scope, "fresh", "[2001:db8::ffff:1]:5555").Locked(), "same /64")
		assert.False(t, l.Attempt(cfg, scope, "fresh", "[2001:db8:0:1::1]:5555").Locked(), "other /64")
		assert.False(t, l.Attempt(cfg, scope, "fresh", "2.3.4.5:5555").Locked(), "IPv4")
	})

	t.Run("Each counter can be disabled separately", func(t *testing.T) {
		t.Parallel()

		cfg := sprayConfig()
		cfg.MaxAuthFailuresPerHost = 0
		l, _ := testLimiter()

		assert.False(t, spray(l, cfg, 100))
		assert.False(t, l.Attempt(cfg, scope, "fresh", addr).Locked())
		assert.NotContains(t, l.entries, l.Attempt(cfg, scope, "", addr).hostKey)

		cfg = sprayConfig()
		cfg.MaxAuthFailures = 0
		l, _ = testLimiter()
		attempt := l.Attempt(cfg, scope, "user", addr)

		for range 9 {
			assert.False(t, attempt.Fail(), "no lock per login")
		}

		assert.True(t, attempt.Fail(), "the counter of the address still applies")
		assert.True(t, attempt.Locked())
		assert.NotContains(t, l.entries, attempt.key)
	})
}

func TestLockDuration(t *testing.T) {
	t.Parallel()

	cfg := &conf.SecurityConfig{
		AuthLockDuration:    time.Minute,
		AuthMaxLockDuration: 30 * time.Minute,
	}

	for locks, expected := range []time.Duration{
		time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 16 * time.Minute,
	} {
		lock, capped := lockDuration(cfg, uint(locks))
		assert.Equal(t, expected, lock)
		assert.False(t, capped)
	}

	// With a 1 minute base, the old shift overflowed from 28 locks on.
	for _, locks := range []uint{5, 27, 28, 30, 63, 64, 1000, math.MaxUint} {
		lock, capped := lockDuration(cfg, locks)
		assert.Equal(t, 30*time.Minute, lock, "%d locks", locks)
		assert.True(t, capped)
	}

	cfg.AuthMaxLockDuration = 0

	for _, locks := range []uint{27, 28, 30, 63, 64, 1000, math.MaxUint} {
		lock, _ := lockDuration(cfg, locks)
		assert.Positive(t, lock, "%d locks", locks)
		assert.GreaterOrEqual(t, lock, time.Minute<<27, "%d locks", locks)
	}

	lock, capped := lockDuration(cfg, math.MaxUint)
	assert.Equal(t, time.Duration(math.MaxInt64), lock)
	assert.True(t, capped)
}
