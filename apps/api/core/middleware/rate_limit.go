package core_middleware

import (
	"net/http"
	"sync"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	burst    int
	// idle is how long an IP's limiter is kept unused: at least until its
	// bucket would be full again, so dropping it never gives an IP more room
	// (a daily cap keeps its state for a day).
	idle time.Duration
}

// minIdle is the shortest time an unused IP limiter is kept.
const minIdle = 10 * time.Minute

func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		burst:    burst,
		idle:     idleFor(r, burst),
	}
	go rl.cleanup()
	return rl
}

// idleFor is the time a limiter of rate r and burst takes to refill from
// empty, and never under minIdle.
func idleFor(r rate.Limit, burst int) time.Duration {
	if r <= 0 || r == rate.Inf {
		return minIdle
	}
	refill := time.Duration(float64(burst) / float64(r) * float64(time.Second))
	if refill < minIdle {
		return minIdle
	}
	return refill
}

func (rl *RateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		l := rate.NewLimiter(rl.r, rl.burst)
		rl.limiters[ip] = &ipLimiter{limiter: l, lastSeen: time.Now()}
		return l
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanup removes limiters for IPs not seen within rl.idle.
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if time.Since(entry.lastSeen) > rl.idle {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns a Gin handler that enforces the rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.get(ip).Allow() {
			resp := envelope.ErrorResponse(http.StatusTooManyRequests, "error.rate_limit_exceeded", core_errors.ErrRateLimitExceeded)
			envelope.Abort(c, resp)
			return
		}
		c.Next()
	}
}
