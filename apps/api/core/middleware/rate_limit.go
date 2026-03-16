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
}

func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		burst:    burst,
	}
	go rl.cleanup()
	return rl
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

// cleanup removes limiters for IPs not seen in the last 10 minutes.
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if time.Since(entry.lastSeen) > 10*time.Minute {
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
			c.JSON(resp.Code, resp)
			c.Abort()
			return
		}
		c.Next()
	}
}
