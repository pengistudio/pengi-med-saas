package core_middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func requestLogRouter(slow time.Duration) (*gin.Engine, *observer.ObservedLogs) {
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zapcore.DebugLevel)
	r := gin.New()
	r.Use(RequestLogger(zap.New(core), slow), gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/v1/patients/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/slow", func(c *gin.Context) { time.Sleep(20 * time.Millisecond); c.Status(http.StatusOK) })
	r.GET("/panic", func(c *gin.Context) { panic("boom") })
	return r, logs
}

func serveRequest(r *gin.Engine, target string, header http.Header) {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	r.ServeHTTP(httptest.NewRecorder(), req)
}

func TestRequestLogger_LogsRouteTemplateStatusAndTenant(t *testing.T) {
	r, logs := requestLogRouter(time.Second)
	serveRequest(r, "/api/v1/patients/42?token=SECRET", http.Header{"X-Tenant-Slug": {"clinica"}})

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.Level != zapcore.InfoLevel {
		t.Errorf("level = %s, want info", e.Level)
	}
	fields := e.ContextMap()
	if fields["route"] != "/api/v1/patients/:id" || fields["method"] != "GET" || fields["status"] != int64(200) || fields["tenant"] != "clinica" {
		t.Errorf("fields = %v", fields)
	}
	if _, ok := fields["duration"]; !ok {
		t.Error("duration missing")
	}
	for _, v := range fields {
		if s, ok := v.(string); ok && (strings.Contains(s, "SECRET") || strings.Contains(s, "42")) {
			t.Errorf("log carries the raw path or query: %v", fields)
		}
	}
}

func TestRequestLogger_FlagsSlowRequestsAtWarn(t *testing.T) {
	r, logs := requestLogRouter(5 * time.Millisecond)
	serveRequest(r, "/slow", nil)
	entries := logs.FilterMessage("slow request").All()
	if len(entries) != 1 || entries[0].Level != zapcore.WarnLevel {
		t.Fatalf("want one warn 'slow request', got %v", logs.All())
	}
}

func TestRequestLogger_SkipsHealth(t *testing.T) {
	r, logs := requestLogRouter(time.Second)
	serveRequest(r, "/health", nil)
	if logs.Len() != 0 {
		t.Fatalf("/health was logged: %v", logs.All())
	}
}

func TestRequestLogger_UnmatchedAndPanics(t *testing.T) {
	r, logs := requestLogRouter(time.Second)
	serveRequest(r, "/wp-admin/setup.php", nil)
	serveRequest(r, "/panic", nil)

	entries := logs.All()
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if f := entries[0].ContextMap(); f["route"] != UnmatchedRoute || f["status"] != int64(404) {
		t.Errorf("unmatched fields = %v", f)
	}
	if f := entries[1].ContextMap(); f["route"] != "/panic" || f["status"] != int64(500) {
		t.Errorf("panic fields = %v", f)
	}
}
