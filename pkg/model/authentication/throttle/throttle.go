// Package throttle limits repeated authentication failures. Failures are
// counted per entry point, login and client address: once too many failures
// occurred, new attempts are refused for a while, without locking the account
// for the other addresses.
package throttle

import (
	"net"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

// maxEntries bounds the memory used by the limiter: beyond it, the entries
// which are neither locked nor recent are purged.
const maxEntries = 10000

type key struct {
	scope, login, host string
}

type entry struct {
	failures    uint32
	lastFailure time.Time
	lockedUntil time.Time
	locks       uint
}

// Limiter counts the authentication failures. The zero value is not usable,
// use New.
type Limiter struct {
	mu      sync.Mutex
	entries map[key]*entry
	now     func() time.Time
}

// New returns a new, empty limiter.
func New() *Limiter {
	return &Limiter{entries: map[key]*entry{}, now: time.Now}
}

//nolint:gochecknoglobals //the limiter is shared by all the entry points
var defaultLimiter = New()

// Default returns the limiter shared by the whole process.
func Default() *Limiter { return defaultLimiter }

// host returns the address without its port, so that all the connections
// from the same client share the same counter.
func host(addr string) string {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}

	return addr
}

// Locked reports whether new authentication attempts of the given login,
// from the given address, must currently be refused.
func (l *Limiter) Locked(cfg *conf.SecurityConfig, scope, login, addr string) bool {
	if cfg.MaxAuthFailures == 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[key{scope, login, host(addr)}]

	return ok && l.now().Before(e.lockedUntil)
}

// Failure records a failed authentication, and returns whether it caused a
// new lock.
func (l *Limiter) Failure(cfg *conf.SecurityConfig, scope, login, addr string) bool {
	if cfg.MaxAuthFailures == 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	k := key{scope, login, host(addr)}

	counter, ok := l.entries[k]
	if !ok {
		if len(l.entries) >= maxEntries {
			l.purge(cfg, now)
		}

		counter = &entry{}
		l.entries[k] = counter
	}

	if now.Sub(counter.lastFailure) > cfg.AuthFailureWindow {
		counter.failures = 0
	}

	counter.failures++
	counter.lastFailure = now

	if counter.failures < cfg.MaxAuthFailures {
		return false
	}

	lock := cfg.AuthLockDuration << min(counter.locks, 30) //nolint:mnd //prevents an overflow of the shift
	if cfg.AuthMaxLockDuration > 0 && (lock > cfg.AuthMaxLockDuration || lock <= 0) {
		lock = cfg.AuthMaxLockDuration
	}

	counter.locks++
	counter.failures = 0
	counter.lockedUntil = now.Add(lock)

	return true
}

// Success records a successful authentication, which resets the counter.
func (l *Limiter) Success(scope, login, addr string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries, key{scope, login, host(addr)})
}

// purge removes the entries which are neither locked nor recent. The caller
// must hold the lock.
func (l *Limiter) purge(cfg *conf.SecurityConfig, now time.Time) {
	for k, e := range l.entries {
		if now.After(e.lockedUntil) && now.Sub(e.lastFailure) > cfg.AuthFailureWindow {
			delete(l.entries, k)
		}
	}
}
