package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestReverseProxy_ForwardsRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer backend.Close()

	p, err := New(backend.URL, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body, _ := io.ReadAll(rec.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", string(body), `{"status":"ok"}`)
	}
}

func TestReverseProxy_ForwardsIdentityHeaders(t *testing.T) {
	var capturedUserID, capturedRole string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Header.Get("X-User-ID")
		capturedRole = r.Header.Get("X-User-Role")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p, err := New(backend.URL, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("X-User-Role", "admin")
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if capturedUserID != "user-123" {
		t.Errorf("X-User-ID = %q, want %q", capturedUserID, "user-123")
	}
	if capturedRole != "admin" {
		t.Errorf("X-User-Role = %q, want %q", capturedRole, "admin")
	}
}

func TestReverseProxy_BackendDown(t *testing.T) {
	p, err := New("http://127.0.0.1:1", zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}
