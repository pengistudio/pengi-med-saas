package auth

import (
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// LoginLimiter caps failed sign-ins per username, on top of the per-IP route
// limit: an attacker rotating IPs still gets only a few guesses a minute at
// one account. Only failures spend the budget, so the real owner signing in
// is never slowed down by their own successful logins. In memory: one API
// instance (a restart resets it).
type LoginLimiter struct {
	mu      sync.Mutex
	entries map[string]*loginEntry
	r       rate.Limit
	burst   int
}

type loginEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// loginIdle is how long an untouched username is remembered: longer than the
// bucket takes to refill, so forgetting it never grants extra attempts.
const loginIdle = 10 * time.Minute

// NewLoginLimiter allows burst failed attempts per username, refilled at r.
func NewLoginLimiter(r rate.Limit, burst int) *LoginLimiter {
	return &LoginLimiter{entries: map[string]*loginEntry{}, r: r, burst: burst}
}

// NewDefaultLoginLimiter allows 5 failed attempts per username per minute.
func NewDefaultLoginLimiter() *LoginLimiter {
	return NewLoginLimiter(rate.Every(time.Minute/5), 5)
}

func loginKey(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// get returns the username's limiter, sweeping idle entries on the way so the
// map can't grow without bound (no background goroutine).
func (l *LoginLimiter) get(username string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.entries) > 1024 {
		for k, e := range l.entries {
			if now.Sub(e.lastSeen) > loginIdle {
				delete(l.entries, k)
			}
		}
	}
	key := loginKey(username)
	e, ok := l.entries[key]
	if !ok {
		e = &loginEntry{limiter: rate.NewLimiter(l.r, l.burst)}
		l.entries[key] = e
	}
	e.lastSeen = now
	return e.limiter
}

// Blocked reports whether username has no failed attempts left right now.
func (l *LoginLimiter) Blocked(username string) bool {
	return l.get(username).Tokens() < 1
}

// Failed spends one attempt of username's budget.
func (l *LoginLimiter) Failed(username string) {
	l.get(username).Allow()
}

// Succeeded forgets username's failures.
func (l *LoginLimiter) Succeeded(username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, loginKey(username))
}
