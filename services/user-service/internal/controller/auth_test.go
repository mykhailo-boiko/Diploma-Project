package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	"github.com/haradrim/chainorchestra/services/user-service/internal/user"
)

func setupAuthController() (*AuthController, *Service) {
	svc, _ := newTestService()
	log := zap.NewNop()
	ctrl := NewAuthController(svc, log)
	return ctrl, svc
}

func TestAuthController_Login_Success(t *testing.T) {
	ctrl, svc := setupAuthController()

	_, err := svc.Register(t.Context(), RegisterRequest{
		Email:     "ctrl-login@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      user.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	body, _ := json.Marshal(LoginRequest{Email: "ctrl-login@example.com", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}
}

func TestAuthController_Login_InvalidCredentials(t *testing.T) {
	ctrl, _ := setupAuthController()

	body, _ := json.Marshal(LoginRequest{Email: "nobody@example.com", Password: "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthController_Login_EmptyBody(t *testing.T) {
	ctrl, _ := setupAuthController()

	body, _ := json.Marshal(LoginRequest{Email: "", Password: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAuthController_Register_Forbidden(t *testing.T) {
	ctrl, _ := setupAuthController()

	body, _ := json.Marshal(RegisterRequest{
		Email: "new@example.com", Password: "pass", FirstName: "A", LastName: "B", Role: user.RoleOperator,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("X-User-Role", "operator")
	rec := httptest.NewRecorder()

	ctrl.Register(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAuthController_Register_Success(t *testing.T) {
	ctrl, _ := setupAuthController()

	body, _ := json.Marshal(RegisterRequest{
		Email: "new@example.com", Password: "Pass1234", FirstName: "New", LastName: "User", Role: user.RoleOperator,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req = req.WithContext(middleware.WithUserContext(req.Context(), "", "admin"))
	rec := httptest.NewRecorder()

	ctrl.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthController_Refresh_Success(t *testing.T) {
	ctrl, svc := setupAuthController()

	_, err := svc.Register(t.Context(), RegisterRequest{
		Email: "refresh-ctrl@example.com", Password: "pass123", FirstName: "R", LastName: "U", Role: user.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	tokens, err := svc.Login(t.Context(), "refresh-ctrl@example.com", "pass123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	body, _ := json.Marshal(RefreshRequest{RefreshToken: tokens.RefreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthController_Refresh_InvalidToken(t *testing.T) {
	ctrl, _ := setupAuthController()

	body, _ := json.Marshal(RefreshRequest{RefreshToken: "bad-token"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
