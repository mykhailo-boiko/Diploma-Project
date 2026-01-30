package auth

import (
	"testing"
	"time"

	"github.com/haradrim/chainorchestra/services/user-service/internal/user"
)

func newTestService() *Service {
	return NewService(Config{
		Secret:     "test-secret-key-for-unit-tests",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 168 * time.Hour,
	})
}

func TestHashAndCheckPassword(t *testing.T) {
	svc := newTestService()

	hash, err := svc.HashPassword("mysecretpassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("hash is empty")
	}

	if hash == "mysecretpassword" {
		t.Fatal("hash should not equal plaintext")
	}

	if !svc.CheckPassword(hash, "mysecretpassword") {
		t.Error("CheckPassword should return true for correct password")
	}

	if svc.CheckPassword(hash, "wrongpassword") {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestGenerateTokenPair(t *testing.T) {
	svc := newTestService()

	u := user.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  user.RoleAdmin,
	}

	pair, err := svc.GenerateTokenPair(u)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("access token is empty")
	}

	if pair.RefreshToken == "" {
		t.Error("refresh token is empty")
	}

	if pair.AccessToken == pair.RefreshToken {
		t.Error("access and refresh tokens should be different")
	}
}

func TestValidateAccessToken(t *testing.T) {
	svc := newTestService()

	u := user.User{
		ID:    "user-456",
		Email: "admin@example.com",
		Role:  user.RoleOperator,
	}

	pair, err := svc.GenerateTokenPair(u)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if claims.UserID != "user-456" {
		t.Errorf("expected user_id user-456, got %s", claims.UserID)
	}
	if claims.Email != "admin@example.com" {
		t.Errorf("expected email admin@example.com, got %s", claims.Email)
	}
	if claims.Role != user.RoleOperator {
		t.Errorf("expected role operator, got %s", claims.Role)
	}
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	svc := newTestService()

	_, err := svc.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	svc1 := NewService(Config{Secret: "secret-1", AccessTTL: 15 * time.Minute, RefreshTTL: 168 * time.Hour})
	svc2 := NewService(Config{Secret: "secret-2", AccessTTL: 15 * time.Minute, RefreshTTL: 168 * time.Hour})

	u := user.User{ID: "user-789", Email: "test@example.com", Role: user.RoleAdmin}
	pair, err := svc1.GenerateTokenPair(u)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	_, err = svc2.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Error("expected error when validating with different secret")
	}
}

func TestValidateRefreshToken(t *testing.T) {
	svc := newTestService()

	u := user.User{ID: "user-refresh-1", Email: "test@example.com", Role: user.RoleAdmin}
	pair, err := svc.GenerateTokenPair(u)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	userID, err := svc.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}

	if userID != "user-refresh-1" {
		t.Errorf("expected user_id user-refresh-1, got %s", userID)
	}
}

func TestValidateRefreshToken_Invalid(t *testing.T) {
	svc := newTestService()

	_, err := svc.ValidateRefreshToken("not-a-valid-token")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}
