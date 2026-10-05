package core_middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAccessLogger_OmitsQueryString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var out bytes.Buffer
	r := gin.New()
	r.Use(AccessLogger(&out))
	r.GET("/api/v1/public/appointments/today", func(c *gin.Context) { c.Status(http.StatusOK) })

	const token = "Ab3_-xYz0123456789abcdefghijKLMN"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/appointments/today?token="+token, nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	logged := out.String()
	if !strings.Contains(logged, "/api/v1/public/appointments/today") {
		t.Fatalf("access log lacks the path: %q", logged)
	}
	if strings.Contains(logged, token) || strings.Contains(logged, "token=") {
		t.Fatalf("access log leaks the token: %q", logged)
	}
}
