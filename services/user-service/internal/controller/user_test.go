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

func withUserCtx(r *http.Request, userID, role string) *http.Request {
	return r.WithContext(middleware.WithUserContext(r.Context(), userID, role))
}

func setupUserController() (*UserController, *Service, *mockStorage) {
	svc, storage := newTestService()
	log := zap.NewNop()
	ctrl := NewUserController(svc, log)
	return ctrl, svc, storage
}

func registerTestUser(t *testing.T, svc *Service, email string, role user.Role) user.User {
	t.Helper()
	u, err := svc.Register(t.Context(), RegisterRequest{
		Email: email, Password: "password123", FirstName: "Test", LastName: "User", Role: role,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	return u
}

func TestUserController_GetProfile_Success(t *testing.T) {
	ctrl, svc, _ := setupUserController()
	u := registerTestUser(t, svc, "profile@example.com", user.RoleOperator)

	req := withUserCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil), u.ID, "")
	rec := httptest.NewRecorder()

	ctrl.GetProfile(rec, req)

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

func TestUserController_GetProfile_MissingUserID(t *testing.T) {
	ctrl, _, _ := setupUserController()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()

	ctrl.GetProfile(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestUserController_UpdateProfile_Success(t *testing.T) {
	ctrl, svc, _ := setupUserController()
	u := registerTestUser(t, svc, "update-profile@example.com", user.RoleOperator)

	body, _ := json.Marshal(UpdateProfileRequest{FirstName: "Updated", LastName: "Name"})
	req := withUserCtx(httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader(body)), u.ID, "")
	rec := httptest.NewRecorder()

	ctrl.UpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUserController_UpdateProfile_CannotChangeRole(t *testing.T) {
	ctrl, svc, storage := setupUserController()
	u := registerTestUser(t, svc, "role-test@example.com", user.RoleOperator)

	body, _ := json.Marshal(UpdateProfileRequest{FirstName: "Hacker"})
	req := withUserCtx(httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader(body)), u.ID, "")
	rec := httptest.NewRecorder()

	ctrl.UpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated := storage.users[u.ID]
	if updated.Role != user.RoleOperator {
		t.Errorf("role should remain operator, got %s", updated.Role)
	}
}

func TestUserController_ListUsers_AdminOnly(t *testing.T) {
	ctrl, svc, _ := setupUserController()
	registerTestUser(t, svc, "admin-list@example.com", user.RoleAdmin)
	registerTestUser(t, svc, "operator-list@example.com", user.RoleOperator)

	req := withUserCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=10", nil), "", "admin")
	rec := httptest.NewRecorder()

	ctrl.ListUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp httpresponse.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta == nil || resp.Meta.Total != 2 {
		t.Errorf("expected 2 total users, got %+v", resp.Meta)
	}
}

func TestUserController_ListUsers_ForbiddenForNonAdmin(t *testing.T) {
	ctrl, _, _ := setupUserController()

	req := withUserCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users", nil), "", "operator")
	rec := httptest.NewRecorder()

	ctrl.ListUsers(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestUserController_CreateUser_AdminOnly(t *testing.T) {
	ctrl, _, _ := setupUserController()

	body, _ := json.Marshal(RegisterRequest{
		Email: "new-user@example.com", Password: "pass123",
		FirstName: "New", LastName: "User", Role: user.RoleOperator,
	})

	req := withUserCtx(httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body)), "", "admin")
	rec := httptest.NewRecorder()

	ctrl.CreateUser(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUserController_UpdateUser_AdminOnly(t *testing.T) {
	ctrl, svc, _ := setupUserController()
	u := registerTestUser(t, svc, "to-update@example.com", user.RoleOperator)

	body, _ := json.Marshal(AdminUpdateUserRequest{Role: user.RoleAnalyst})

	req := withUserCtx(httptest.NewRequest(http.MethodPut, "/api/v1/users/"+u.ID, bytes.NewReader(body)), "", "admin")
	req.SetPathValue("id", u.ID)
	rec := httptest.NewRecorder()

	ctrl.UpdateUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUserController_DeleteUser_AdminOnly(t *testing.T) {
	ctrl, svc, _ := setupUserController()
	u := registerTestUser(t, svc, "to-delete@example.com", user.RoleOperator)

	req := withUserCtx(httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+u.ID, nil), "", "admin")
	req.SetPathValue("id", u.ID)
	rec := httptest.NewRecorder()

	ctrl.DeleteUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	getReq := withUserCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil), u.ID, "")
	getRec := httptest.NewRecorder()
	ctrl.GetProfile(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Errorf("expected 404 after deletion, got %d", getRec.Code)
	}
}

func TestUserController_DeleteUser_ForbiddenForNonAdmin(t *testing.T) {
	ctrl, _, _ := setupUserController()

	req := withUserCtx(httptest.NewRequest(http.MethodDelete, "/api/v1/users/some-id", nil), "", "operator")
	req.SetPathValue("id", "some-id")
	rec := httptest.NewRecorder()

	ctrl.DeleteUser(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestUserController_PasswordReset_Flow(t *testing.T) {
	ctrl, svc, storage := setupUserController()
	registerTestUser(t, svc, "reset@example.com", user.RoleOperator)

	body, _ := json.Marshal(PasswordResetRequest{Email: "reset@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.RequestPasswordReset(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resetToken string
	for token := range storage.resetTokens {
		resetToken = token
		break
	}
	if resetToken == "" {
		t.Fatal("no reset token was created")
	}

	confirmBody, _ := json.Marshal(PasswordResetConfirmRequest{
		Token:       resetToken,
		NewPassword: "newpassword123",
	})
	confirmReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/confirm", bytes.NewReader(confirmBody))
	confirmRec := httptest.NewRecorder()

	ctrl.ConfirmPasswordReset(confirmRec, confirmReq)

	if confirmRec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", confirmRec.Code, confirmRec.Body.String())
	}

	tokens, err := svc.Login(t.Context(), "reset@example.com", "newpassword123")
	if err != nil {
		t.Errorf("login with new password failed: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected non-empty access token after password reset")
	}
}

func TestUserController_PasswordReset_UnknownEmail(t *testing.T) {
	ctrl, _, _ := setupUserController()

	body, _ := json.Marshal(PasswordResetRequest{Email: "unknown@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.RequestPasswordReset(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (should not reveal email existence), got %d", rec.Code)
	}
}

func TestUserController_PasswordResetConfirm_InvalidToken(t *testing.T) {
	ctrl, _, _ := setupUserController()

	body, _ := json.Marshal(PasswordResetConfirmRequest{Token: "invalid", NewPassword: "new123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/confirm", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ctrl.ConfirmPasswordReset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
