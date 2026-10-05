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
