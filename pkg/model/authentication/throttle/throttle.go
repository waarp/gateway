// Package throttle limits repeated authentication failures, to slow down
// password guessing on the entry points of the gateway (REST API, web
// interface, transfer servers).
//
// Each failure is recorded on two counters:
//   - the counter of the login, which counts the failures of one login, from
//     one client address, on one entry point. Its threshold is
//     conf.SecurityConfig.MaxAuthFailures. Locking it does not prevent the
//     same account from authenticating from other addresses.
//   - the counter of the address, which counts the failures of all the logins
//     from one client address on one entry point, against an address trying a
//     few passwords on many logins (password spraying). Its threshold is
//     conf.SecurityConfig.MaxAuthFailuresPerHost. IPv6 addresses share this
//     counter by /64 prefix.
//
// The port of the address is ignored, so all the connections of a client share
// the same counters. A threshold of 0 disables the corresponding counter.
//
// Once a counter reaches its threshold, it is locked for
// conf.SecurityConfig.AuthLockDuration, and the attempts it applies to are
// refused until the lock ends, even with valid credentials. Each new lock of
// the same counter lasts twice as long as the previous one, up to
// conf.SecurityConfig.AuthMaxLockDuration (no maximum if it is 0 or less).
// The failures are forgotten when no new failure occurred during
// conf.SecurityConfig.AuthFailureWindow, and the doubling is forgotten once a
// whole window passed after both the last failure and the end of the last
// lock.
//
// The counters are kept in memory, in the process only: they are lost when the
// process restarts, and they are not shared between several instances of the
// gateway. Their number is bounded, so an attacker cannot exhaust the memory.
//
// A typical use, around the check of the credentials, is:
//
//	attempt := throttle.Default().Attempt(cfg, scope, login, remoteAddr)
//	if attempt.Locked() {
//		return refuse() // same answer as invalid credentials
//	}
//
//	if !checkCredentials(login, password) {
//		attempt.Fail()
//
//		return refuse()
//	}
//
//	attempt.Succeed()
package throttle

import (
	"hash/maphash"
	"math"
	"net"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

// maxEntries bounds the memory used by the limiter: once it is reached, an
// existing entry is evicted for each new one.
const maxEntries = 10000

// key identifies a counter. The login is given by the client, and its size is
// not bounded: it is therefore stored as a hash, so that all the entries have
// the same small size. The hash is seeded randomly for each limiter, so the
// rare collisions (which only make two logins share a counter) cannot be
// provoked on purpose.
type key struct {
	scope     string
	login     uint64
	host      string
	allLogins bool // the counter of all the logins from the host
}

type entry struct {
	failures    uint32
	lastFailure time.Time
	lockedUntil time.Time
	locks       uint
}

// expired reports whether the entry can be forgotten: neither its last failure
// nor the end of its last lock happened within the window. A locked entry is
// therefore never expired.
func (e *entry) expired(window time.Duration, now time.Time) bool {
	last := e.lastFailure
	if e.lockedUntil.After(last) {
		last = e.lockedUntil
	}

	return now.Sub(last) > window
}

// Limiter holds the failure counters of the authentication attempts, and
// decides which attempts must be refused. The expired counters are removed at
// most once per AuthFailureWindow, when a failure creates a new counter. It
// holds at most 10 000 counters: when it is full, a counter which is not
// locked, or failing that the counter whose lock ends first, is evicted to
// make room for the new one.
//
// A Limiter is safe for concurrent use by multiple goroutines. The zero value
// is not usable: use New, or the shared limiter returned by Default.
type Limiter struct {
	mu        sync.RWMutex
	entries   map[key]entry
	seed      maphash.Seed
	now       func() time.Time
	lastPurge time.Time
}

// New returns a new, empty limiter, independent of all the others. Its login
// hashes use a new random seed. The entry points of the gateway should share
// the limiter returned by Default instead, so that New is mostly useful in
// tests.
func New() *Limiter {
	return &Limiter{entries: map[key]entry{}, seed: maphash.MakeSeed(), now: time.Now}
}

//nolint:gochecknoglobals //the limiter is shared by all the entry points
var defaultLimiter = New()

// Default returns the limiter shared by the whole process. It is created when
// the package is initialized, and always returns the same limiter. The entry
// points tell their counters apart with the scope given to Limiter.Attempt.
func Default() *Limiter { return defaultLimiter }

// host returns the address without its port, so that all the connections
// from the same client share the same counter.
func host(addr string) string {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}

	return addr
}

// ipv6Network is the mask of the /64 prefix of an IPv6 address.
//
//nolint:gochecknoglobals //constant mask
var ipv6Network = net.CIDRMask(64, 8*net.IPv6len)

// network returns the host of the address, or its /64 prefix for an IPv6
// address, since a single client can easily use a whole /64.
func network(addr string) string {
	h := host(addr)

	if ip := net.ParseIP(h); ip != nil && ip.To4() == nil {
		return ip.Mask(ipv6Network).String()
	}

	return h
}

// Attempt is an authentication attempt of a login, from an address, on an
// entry point. It is obtained with Limiter.Attempt, before checking the
// credentials: Locked tells whether the attempt must be refused, then Fail or
// Succeed records its outcome.
//
// An Attempt identifies the counters it applies to, it holds no state of its
// own: it is cheap to copy, and calling Fail several times records several
// failures. The settings are read from the conf.SecurityConfig given to
// Limiter.Attempt at each call, so a change of the settings applies to the
// attempts already started. The zero value is not usable.
type Attempt struct {
	limiter *Limiter
	cfg     *conf.SecurityConfig
	key     key // the counter of the login from the host
	hostKey key // the counter of all the logins from the host
}

// Attempt starts an authentication attempt of the given login, from the given
// address, on the given entry point. It records nothing by itself.
//
// The scope identifies the entry point: attempts with different scopes never
// share a counter. The login is the one given by the client, whether the
// account exists or not, of any length: only a 64-bit hash of it is kept,
// so two logins can, rarely, share a counter. The address is the client
// address, usually in the "host:port" form of net.Conn.RemoteAddr or
// http.Request.RemoteAddr: the port is ignored. An address without a port is
// used as is.
//
// The settings are read from cfg, which must not be nil, each time a method
// of the returned Attempt is called.
func (l *Limiter) Attempt(cfg *conf.SecurityConfig, scope, login, addr string) Attempt {
	return Attempt{
		limiter: l, cfg: cfg,
		key:     l.key(scope, login, addr),
		hostKey: key{scope: scope, host: network(addr), allLogins: true},
	}
}

func (l *Limiter) key(scope, login, addr string) key {
	return key{scope: scope, login: maphash.String(l.seed, login), host: host(addr)}
}

func (a Attempt) enabled() bool {
	return a.cfg.MaxAuthFailures != 0 || a.cfg.MaxAuthFailuresPerHost != 0
}

// Locked reports whether the attempt must be refused, because the counter of
// the login or the counter of the address is currently locked. A disabled
// counter (threshold of 0) is never locked. Locked records nothing.
//
// A locked attempt should be refused without checking the credentials, with
// the same answer as invalid credentials, so that the client cannot tell
// whether the credentials were valid. Neither Fail nor Succeed should then be
// called: Fail would count one more failure, and Succeed would remove the
// lock of the login.
func (a Attempt) Locked() bool {
	if !a.enabled() {
		return false
	}

	l := a.limiter

	l.mu.RLock()
	defer l.mu.RUnlock()

	now := l.now()

	return l.locked(a.key, a.cfg.MaxAuthFailures, now) ||
		l.locked(a.hostKey, a.cfg.MaxAuthFailuresPerHost, now)
}

// locked reports whether the given counter is locked. The caller must hold
// the lock.
func (l *Limiter) locked(k key, maxFailures uint32, now time.Time) bool {
	if maxFailures == 0 {
		return false
	}

	e, ok := l.entries[k]

	return ok && now.Before(e.lockedUntil)
}

// Fail records that the authentication failed, on each counter of the attempt
// that is enabled:
//   - if the counter has not been used during the last window, after both its
//     last failure and the end of its last lock, it starts again from zero,
//     including the doubling of the lock duration;
//   - otherwise, if its last failure is older than the window, its failures
//     are reset, but the doubling of the lock duration is kept;
//   - the failure is then counted. If the counter reaches its threshold, it is
//     locked from now on, for AuthLockDuration doubled at each previous lock
//     (up to AuthMaxLockDuration), and its failures are reset to 0.
//
// Fail returns true if this failure locked at least one of the counters,
// which callers usually log. With an AuthLockDuration of 0 or less, a lock
// lasts 0, so Fail can return true while Locked stays false. If both
// counters are disabled, Fail does nothing and returns false.
func (a Attempt) Fail() bool {
	if !a.enabled() {
		return false
	}

	l, cfg := a.limiter, a.cfg

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	locked := l.fail(cfg, a.key, cfg.MaxAuthFailures, now)
	hostLocked := l.fail(cfg, a.hostKey, cfg.MaxAuthFailuresPerHost, now)

	return locked || hostLocked
}

// fail records a failure on the given counter, and returns whether it caused
// a new lock. The caller must hold the lock.
func (l *Limiter) fail(cfg *conf.SecurityConfig, k key, maxFailures uint32, now time.Time) bool {
	if maxFailures == 0 {
		return false
	}

	counter, ok := l.entries[k]

	switch {
	case !ok:
		// The purge is done at most once per window, so that its cost does
		// not add up to every failure when many entries cannot be purged.
		if now.Sub(l.lastPurge) >= cfg.AuthFailureWindow {
			l.purge(cfg, now)
			l.lastPurge = now
		}

		if len(l.entries) >= maxEntries {
			l.evictOne(now)
		}
	case counter.expired(cfg.AuthFailureWindow, now):
		// Same rule as the purge, so that the lock escalation does not depend
		// on whether the purge already ran.
		counter = entry{}
	case now.Sub(counter.lastFailure) > cfg.AuthFailureWindow:
		counter.failures = 0
	}

	counter.failures++
	counter.lastFailure = now

	locked := counter.failures >= maxFailures
	if locked {
		lock, capped := lockDuration(cfg, counter.locks)
		if !capped {
			counter.locks++
		}

		counter.failures = 0
		counter.lockedUntil = now.Add(lock)
	}

	l.entries[k] = counter

	return locked
}

// Succeed records that the authentication succeeded: it removes the counter
// of the login, with its failures, its lock and the doubling of the lock
// duration. It does so even if the counters are disabled.
//
// The counter of the address is kept, otherwise an attacker owning a valid
// account could reset it between two guesses on other logins.
func (a Attempt) Succeed() {
	l := a.limiter

	// Most of the time there is no entry: checking it under the read lock
	// avoids serializing all the successful authentications.
	l.mu.RLock()
	_, ok := l.entries[a.key]
	l.mu.RUnlock()

	if !ok {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries, a.key)
}

// lockDuration returns the duration of a lock, given the number of previous
// locks: the base duration is doubled at each lock, up to the maximum (or up
// to the largest possible duration if there is no maximum). It also reports
// whether the maximum was reached, in which case the number of locks no
// longer needs to grow.
func lockDuration(cfg *conf.SecurityConfig, locks uint) (time.Duration, bool) {
	maxLock := cfg.AuthMaxLockDuration
	if maxLock <= 0 {
		maxLock = math.MaxInt64
	}

	lock := cfg.AuthLockDuration
	if lock <= 0 {
		return 0, true
	}

	for range locks {
		if lock > maxLock/2 { // doubling would exceed the maximum (or overflow)
			return maxLock, true
		}

		lock *= 2
	}

	if lock >= maxLock {
		return maxLock, true
	}

	return lock, false
}

// evictOne removes an entry to make room for a new one, preferably one which
// is not locked. Since the iteration order of a map is random, an unlocked
// entry is usually found right away. If all the entries are locked, the one
// whose lock ends first is removed. The caller must hold the lock.
func (l *Limiter) evictOne(now time.Time) {
	var (
		victim key
		found  bool
		end    time.Time
	)

	for k, e := range l.entries {
		if !now.Before(e.lockedUntil) {
			delete(l.entries, k)

			return
		}

		if !found || e.lockedUntil.Before(end) {
			victim, end, found = k, e.lockedUntil, true
		}
	}

	if found {
		delete(l.entries, victim)
	}
}

// purge removes the expired entries. The caller must hold the lock.
func (l *Limiter) purge(cfg *conf.SecurityConfig, now time.Time) {
	for k, e := range l.entries {
		if e.expired(cfg.AuthFailureWindow, now) {
			delete(l.entries, k)
		}
	}
}
