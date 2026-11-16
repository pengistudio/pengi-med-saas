// Package metrics exposes Prometheus metrics: HTTP requests (count and
// duration by route, method and status), the database pool (sql.DBStats) and
// the Go runtime.
//
// GET /metrics needs `Authorization: Bearer <METRICS_TOKEN>`; with no token
// configured the route isn't registered (404). Caddy proxies every path of the
// API host, so the token is what keeps it private.
package metrics

import (
	"crypto/subtle"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	core_middleware "pengi-med-saas/core/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Path is where the metrics are served.
const Path = "/metrics"

// Metrics owns its own registry (not the global one) so every instance, e.g.
// one per test router, registers without conflicts.
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// New builds the collectors; sqlDB (may be nil) adds the pool stats as
// go_sql_* with db_name="pengi".
func New(sqlDB *sql.DB) *Metrics {
	labels := []string{"route", "method", "status"}
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by route template, method and status.",
		}, labels),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by route template, method and status.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		}, labels),
	}
	m.registry.MustRegister(
		m.requests, m.duration,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	if sqlDB != nil {
		m.registry.MustRegister(collectors.NewDBStatsCollector(sqlDB, "pengi"))
	}
	return m
}

// Middleware records every request except /metrics itself. Labels are only
// bounded values: the route template (unmatched paths share one value), the
// method and the status; never the tenant or the raw path.
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == Path {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = core_middleware.UnmatchedRoute
		}
		status := strconv.Itoa(c.Writer.Status())
		m.requests.WithLabelValues(route, c.Request.Method, status).Inc()
		m.duration.WithLabelValues(route, c.Request.Method, status).Observe(time.Since(start).Seconds())
	}
}

// Handler serves the registry in the Prometheus text format to requests
// carrying the bearer token (constant-time compare); others get 401.
func (m *Metrics) Handler(token string) gin.HandlerFunc {
	serve := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
	want := []byte(token)
	return func(c *gin.Context) {
		got, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" || !ok || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			envelope.Abort(c, envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrAuthInvalidRequest))
			return
		}
		serve.ServeHTTP(c.Writer, c.Request)
	}
}

// Endpoint is the metrics wiring of one router: the middleware, installed
// first, and GET /metrics, registered next to the other routes so it runs
// behind Recovery and the i18n middleware like any handler.
type Endpoint struct {
	metrics *Metrics
	token   string
}

// Install adds the middleware to r when token is set. Call it right after
// gin.New(), before gin.Recovery, so panics count as 500. With no token it
// returns nil: no collection, and Route registers nothing.
func Install(r *gin.Engine, sqlDB *sql.DB, token string) *Endpoint {
	if token == "" {
		return nil
	}
	m := New(sqlDB)
	r.Use(m.Middleware())
	return &Endpoint{metrics: m, token: token}
}

// Route registers GET /metrics; a nil Endpoint (no token) registers nothing,
// so the path is a plain 404.
func (e *Endpoint) Route(r gin.IRoutes) {
	if e == nil {
		return
	}
	r.GET(Path, e.metrics.Handler(e.token))
}
