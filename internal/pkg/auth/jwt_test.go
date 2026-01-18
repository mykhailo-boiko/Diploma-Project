package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key"

func generateTestToken(t *testing.T, secret string, claims Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return str
}

func TestValidateAccessToken(t *testing.T) {
	v := NewValidator(testSecret)
	now := time.Now()

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		wantID    string
		wantEmail string
		wantRole  string
	}{
		{
			name: "valid token",
			token: generateTestToken(t, testSecret, Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-1",
					ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
				UserID: "user-1",
				Email:  "test@example.com",
				Role:   "admin",
			}),
			wantID:    "user-1",
			wantEmail: "test@example.com",
			wantRole:  "admin",
		},
		{
			name: "expired token",
			token: generateTestToken(t, testSecret, Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-1",
					ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
				},
				UserID: "user-1",
				Email:  "test@example.com",
				Role:   "admin",
			}),
			wantErr: true,
		},
		{
			name:    "invalid token string",
			token:   "not-a-valid-token",
			wantErr: true,
		},
		{
			name: "wrong secret",
			token: generateTestToken(t, "wrong-secret", Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-1",
					ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
				UserID: "user-1",
				Email:  "test@example.com",
				Role:   "admin",
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := v.ValidateAccessToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if claims.UserID != tt.wantID {
				t.Errorf("UserID = %q, want %q", claims.UserID, tt.wantID)
			}
			if claims.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", claims.Email, tt.wantEmail)
			}
			if claims.Role != tt.wantRole {
				t.Errorf("Role = %q, want %q", claims.Role, tt.wantRole)
			}
		})
	}
}
