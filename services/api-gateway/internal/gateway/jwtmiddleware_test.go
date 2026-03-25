package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/haradrim/chainorchestra/internal/pkg/auth"
)

const testSecret = "test-secret-key"

func generateTestToken(t *testing.T, secret string, claims auth.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return str
}

func validClaims() auth.Claims {
	now := time.Now()
	return auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: "user-1",
		Email:  "test@example.com",
		Role:   "admin",
	}
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	validator := auth.NewValidator(testSecret)
	mw := NewJWTMiddleware(validator)
	token := generateTestToken(t, testSecret, validClaims())

	var capturedUserID, capturedRole string
	handler := mw.Middleware(nil)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Header.Get("X-User-ID")
		capturedRole = r.Header.Get("X-User-Role")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if capturedUserID != "user-1" {
		t.Errorf("X-User-ID = %q, want %q", capturedUserID, "user-1")
	}
	if capturedRole != "admin" {
		t.Errorf("X-User-Role = %q, want %q", capturedRole, "admin")
	}
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	validator := auth.NewValidator(testSecret)
	mw := NewJWTMiddleware(validator)

	handler := mw.Middleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	validator := auth.NewValidator(testSecret)
	mw := NewJWTMiddleware(validator)

	handler := mw.Middleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	validator := auth.NewValidator(testSecret)
	mw := NewJWTMiddleware(validator)

	now := time.Now()
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
		UserID: "user-1",
		Email:  "test@example.com",
		Role:   "admin",
	}
	token := generateTestToken(t, testSecret, claims)

	handler := mw.Middleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_SkipPrefix(t *testing.T) {
	validator := auth.NewValidator(testSecret)
	mw := NewJWTMiddleware(validator)

	called := false
	handler := mw.Middleware([]string{"/api/v1/auth/"})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !called {
		t.Error("handler should have been called for skipped prefix")
	}
}

func TestJWTMiddleware_HealthSkip(t *testing.T) {
	validator := auth.NewValidator(testSecret)
	mw := NewJWTMiddleware(validator)

	called := false
	handler := mw.Middleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should have been called for /health")
	}
}
