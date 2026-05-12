package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
}

func NewMetrics(serviceName string) *Metrics {
	return newMetrics(serviceName, promauto.With(prometheus.DefaultRegisterer))
}

func NewMetricsWithRegistry(serviceName string, reg prometheus.Registerer) *Metrics {
	return newMetrics(serviceName, promauto.With(reg))
}

func newMetrics(serviceName string, factory promauto.Factory) *Metrics {
	return &Metrics{
		requestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "chainorchestra",
			Subsystem: serviceName,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		}, []string{"method", "path", "status"}),
		requestDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "chainorchestra",
			Subsystem: serviceName,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "path"}),
		requestsInFlight: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "chainorchestra",
			Subsystem: serviceName,
			Name:      "http_requests_in_flight",
			Help:      "Number of HTTP requests currently being processed.",
		}),
	}
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		start := time.Now()
		wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)
		path := normalizePath(r.URL.Path)

		m.requestsTotal.WithLabelValues(r.Method, path, status).Inc()
		m.requestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *metricsResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func normalizePath(path string) string {
	if path == "/health" || path == "/health/nats" || path == "/metrics" {
		return path
	}

	result := make([]byte, 0, len(path))
	i := 0
	for i < len(path) {
		if path[i] == '/' {
			result = append(result, '/')
			i++
			j := i
			for j < len(path) && path[j] != '/' {
				j++
			}
			segment := path[i:j]
			if isIDSegment(segment) {
				result = append(result, ":id"...)
				i = j
			}
			continue
		}
		result = append(result, path[i])
		i++
	}
	return string(result)
}

func isIDSegment(s string) bool {
	if len(s) == 0 {
		return false
	}
	if len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' {
		return true
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
