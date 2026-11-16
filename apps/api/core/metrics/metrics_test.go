package metrics_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/lib/pq"

	"pengi-med-saas/core/metrics"

	"github.com/gin-gonic/gin"
)

func metricsRouter(t *testing.T, token string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	sqlDB, err := sql.Open("postgres", "host=127.0.0.1 port=1 user=u dbname=n sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	r := gin.New()
	endpoint := metrics.Install(r, sqlDB, token)
	r.Use(gin.Recovery())
	r.GET("/api/v1/patients/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	endpoint.Route(r)
	return r
}

func get(r *gin.Engine, target, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMetrics_DisabledWithoutToken(t *testing.T) {
	r := metricsRouter(t, "")
	if w := get(r, metrics.Path, "Bearer "); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 with no METRICS_TOKEN", w.Code)
	}
}

func TestMetrics_RejectsWrongOrMissingToken(t *testing.T) {
	r := metricsRouter(t, "s3cret")
	for _, auth := range []string{"", "Bearer wrong", "s3cret", "Basic s3cret"} {
		w := get(r, metrics.Path, auth)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("auth %q: status = %d, want 401", auth, w.Code)
		}
		if strings.Contains(w.Body.String(), "http_requests_total") {
			t.Errorf("auth %q: metrics leaked", auth)
		}
	}
}

func TestMetrics_ServesRequestAndPoolMetrics(t *testing.T) {
	r := metricsRouter(t, "s3cret")
	get(r, "/api/v1/patients/42", "")
	get(r, "/nope", "")

	w := get(r, metrics.Path, "Bearer s3cret")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`http_requests_total{method="GET",route="/api/v1/patients/:id",status="200"} 1`,
		`http_requests_total{method="GET",route="unmatched",status="404"} 1`,
		`http_request_duration_seconds_bucket{method="GET",route="/api/v1/patients/:id",status="200"`,
		`go_sql_max_open_connections{db_name="pengi"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics lack %q", want)
		}
	}
	if strings.Contains(body, "/api/v1/patients/42") {
		t.Error("metrics carry a raw path")
	}
}
