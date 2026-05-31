package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arcana_http_requests_total",
		Help: "Total HTTP requests by method, path, and status",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "arcana_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arcana_active_connections",
		Help: "Number of active HTTP connections",
	})

	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "arcana_db_query_duration_seconds",
		Help:    "Database query duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	}, []string{"query"})

	AgentsRegistered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arcana_agents_registered_total",
		Help: "Total number of registered agents",
	})

	APIKeysActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arcana_api_keys_active",
		Help: "Number of active (non-revoked) API keys",
	})

	AuditEntriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arcana_audit_entries_total",
		Help: "Total audit log entries written",
	})

	AuthFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arcana_auth_failures_total",
		Help: "Authentication failures by type",
	}, []string{"type"})

	RateLimitHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arcana_rate_limit_hits_total",
		Help: "Total rate limit rejections",
	})
)

func Handler() http.Handler {
	return promhttp.Handler()
}

func InstrumentHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ActiveConnections.Inc()
		defer ActiveConnections.Dec()

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)
		duration := time.Since(start).Seconds()

		path := normalizePath(r.URL.Path)
		status := http.StatusText(rw.statusCode)
		HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func normalizePath(path string) string {
	parts := splitPath(path)
	if len(parts) >= 3 {
		return "/" + parts[0] + "/" + parts[1] + "/" + parts[2]
	}
	return path
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
