package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

type HttpMetrics struct {
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
}

func NewHttp(reg prometheus.Registerer) *HttpMetrics {
	m := &HttpMetrics{
		// http
		httpRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests processed.",
			},
			[]string{"method", "route", "status"},
		),
		httpDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		),
	}
	reg.MustRegister(m.httpRequests, m.httpDuration)
	return m
}

func (m *HttpMetrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		statusStr := strconv.Itoa(status)

		m.httpRequests.WithLabelValues(r.Method, route, statusStr).Inc()
		m.httpDuration.WithLabelValues(r.Method, route, statusStr).Observe(float64(time.Since(start).Seconds()))
	})
}
