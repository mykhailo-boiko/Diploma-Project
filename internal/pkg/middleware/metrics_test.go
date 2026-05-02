package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func newTestMetrics(t *testing.T) (*Metrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	m := NewMetricsWithRegistry("test_svc", reg)
	return m, reg
}

func TestMetrics_IncrementsCounter(t *testing.T) {
	m, reg := newTestMetrics(t)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	found := findMetricFamily(families, "chainorchestra_test_svc_http_requests_total")
	if found == nil {
		t.Fatal("expected http_requests_total metric")
	}
	if len(found.Metric) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(found.Metric))
	}
	if found.Metric[0].Counter.GetValue() != 1 {
		t.Errorf("expected counter value 1, got %f", found.Metric[0].Counter.GetValue())
	}

	labels := labelMap(found.Metric[0].Label)
	if labels["method"] != "GET" {
		t.Errorf("expected method=GET, got %q", labels["method"])
	}
	if labels["path"] != "/api/v1/orders" {
		t.Errorf("expected path=/api/v1/orders, got %q", labels["path"])
	}
	if labels["status"] != "200" {
		t.Errorf("expected status=200, got %q", labels["status"])
	}
}

func TestMetrics_RecordsDuration(t *testing.T) {
	m, reg := newTestMetrics(t)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	found := findMetricFamily(families, "chainorchestra_test_svc_http_request_duration_seconds")
	if found == nil {
		t.Fatal("expected http_request_duration_seconds metric")
	}
	if found.Metric[0].Histogram.GetSampleCount() != 1 {
		t.Errorf("expected 1 sample, got %d", found.Metric[0].Histogram.GetSampleCount())
	}
}

func TestMetrics_SkipsMetricsPath(t *testing.T) {
	m, reg := newTestMetrics(t)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	found := findMetricFamily(families, "chainorchestra_test_svc_http_requests_total")
	if found != nil {
		t.Error("expected no metrics for /metrics path")
	}
}

func TestMetrics_CapturesStatusCode(t *testing.T) {
	m, reg := newTestMetrics(t)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	found := findMetricFamily(families, "chainorchestra_test_svc_http_requests_total")
	if found == nil {
		t.Fatal("expected http_requests_total metric")
	}

	labels := labelMap(found.Metric[0].Label)
	if labels["status"] != "404" {
		t.Errorf("expected status=404, got %q", labels["status"])
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "health", path: "/health", want: "/health"},
		{name: "health nats", path: "/health/nats", want: "/health/nats"},
		{name: "metrics", path: "/metrics", want: "/metrics"},
		{name: "no id", path: "/api/v1/orders", want: "/api/v1/orders"},
		{name: "uuid id", path: "/api/v1/orders/550e8400-e29b-41d4-a716-446655440000", want: "/api/v1/orders/:id"},
		{name: "numeric id", path: "/api/v1/orders/123", want: "/api/v1/orders/:id"},
		{name: "uuid then action", path: "/api/v1/orders/550e8400-e29b-41d4-a716-446655440000/status", want: "/api/v1/orders/:id/status"},
		{name: "text segment", path: "/api/v1/orders/search", want: "/api/v1/orders/search"},
		{name: "empty", path: "/", want: "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePath(tt.path)
			if got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIDSegment(t *testing.T) {
	tests := []struct {
		name    string
		segment string
		want    bool
	}{
		{name: "uuid", segment: "550e8400-e29b-41d4-a716-446655440000", want: true},
		{name: "numeric", segment: "42", want: true},
		{name: "text", segment: "orders", want: false},
		{name: "empty", segment: "", want: false},
		{name: "mixed", segment: "abc123", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIDSegment(tt.segment)
			if got != tt.want {
				t.Errorf("isIDSegment(%q) = %v, want %v", tt.segment, got, tt.want)
			}
		})
	}
}

func findMetricFamily(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func labelMap(labels []*dto.LabelPair) map[string]string {
	m := make(map[string]string, len(labels))
	for _, l := range labels {
		m[l.GetName()] = l.GetValue()
	}
	return m
}
