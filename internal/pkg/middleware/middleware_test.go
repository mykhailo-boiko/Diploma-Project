package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestRequestID_GeneratesNew(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("expected request ID to be set")
		}
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Header().Get(requestIDHeader) == "" {
		t.Error("expected X-Request-ID header in response")
	}
}

func TestRequestID_PreservesExisting(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != "existing-id" {
			t.Errorf("expected 'existing-id', got %q", id)
		}
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(requestIDHeader, "existing-id")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Header().Get(requestIDHeader) != "existing-id" {
		t.Errorf("expected 'existing-id' in response header, got %q", w.Header().Get(requestIDHeader))
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	id := GetRequestID(context.Background())
	if id != "" {
		t.Errorf("expected empty string, got %q", id)
	}
}

func TestLogging(t *testing.T) {
	log := zap.NewNop()
	handler := Logging(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestRecovery(t *testing.T) {
	log := zap.NewNop()
	handler := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:3000" {
		t.Errorf("expected origin 'http://localhost:3000', got %q", origin)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("expected empty origin for disallowed, got %q", origin)
	}
}

func TestUserContext_SetsValues(t *testing.T) {
	handler := UserContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetUserID(r.Context())
		if id != "user-123" {
			t.Errorf("expected 'user-123', got %q", id)
		}
		role := GetUserRole(r.Context())
		if role != "admin" {
			t.Errorf("expected 'admin', got %q", role)
		}
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-User-ID", "user-123")
	r.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
}

func TestUserContext_EmptyHeaders(t *testing.T) {
	handler := UserContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetUserID(r.Context())
		if id != "" {
			t.Errorf("expected empty, got %q", id)
		}
		role := GetUserRole(r.Context())
		if role != "" {
			t.Errorf("expected empty, got %q", role)
		}
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
}

func TestGetUserID_EmptyContext(t *testing.T) {
	id := GetUserID(context.Background())
	if id != "" {
		t.Errorf("expected empty string, got %q", id)
	}
}

func TestGetUserRole_EmptyContext(t *testing.T) {
	role := GetUserRole(context.Background())
	if role != "" {
		t.Errorf("expected empty string, got %q", role)
	}
}

func TestCORS_Preflight(t *testing.T) {
	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for OPTIONS preflight")
	}))

	r := httptest.NewRequest(http.MethodOptions, "/", nil)
	r.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for preflight, got %d", w.Code)
	}
}
