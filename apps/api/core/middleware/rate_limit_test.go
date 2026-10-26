package core_middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func limitedRouter(rl *RateLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", rl.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func hit(r *gin.Engine, ip string) int {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = ip + ":1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRateLimiter_LimitsPerIP(t *testing.T) {
	r := limitedRouter(NewRateLimiter(rate.Every(time.Hour), 3))

	for i := 0; i < 3; i++ {
		if code := hit(r, "10.0.0.1"); code != http.StatusOK {
			t.Fatalf("request %d: code = %d, want 200", i+1, code)
		}
	}
	if code := hit(r, "10.0.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("over the burst: code = %d, want 429", code)
	}
	if code := hit(r, "10.0.0.2"); code != http.StatusOK {
		t.Fatalf("another IP: code = %d, want 200", code)
	}
}

func TestRateLimiter_KeepsIdleIPsUntilRefilled(t *testing.T) {
	for _, tc := range []struct {
		r     rate.Limit
		burst int
		want  time.Duration
	}{
		{rate.Every(time.Minute / 5), 5, minIdle},
		{rate.Every(24 * time.Hour / 20), 20, 24 * time.Hour},
		{rate.Inf, 1, minIdle},
	} {
		if got := idleFor(tc.r, tc.burst); got.Round(time.Second) != tc.want {
			t.Fatalf("idleFor(%v, %d) = %v, want %v", tc.r, tc.burst, got, tc.want)
		}
	}
}
