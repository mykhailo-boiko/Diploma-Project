package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/user-service/internal/auth"
	"github.com/haradrim/chainorchestra/services/user-service/internal/user"
)

type mockStorage struct {
	users       map[string]user.User
	resetTokens map[string]user.PasswordResetToken
	idCounter   int
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		users:       make(map[string]user.User),
		resetTokens: make(map[string]user.PasswordResetToken),
	}
}

func (m *mockStorage) CreateUser(_ context.Context, u user.User) (user.User, error) {
	for _, existing := range m.users {
		if existing.Email == u.Email && existing.DeletedAt == nil {
			return user.User{}, user.ErrUserAlreadyExists
		}
	}
	m.idCounter++
	u.ID = fmt.Sprintf("generated-id-%d", m.idCounter)
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now
	m.users[u.ID] = u
	return u, nil
}

func (m *mockStorage) GetUserByID(_ context.Context, id string) (user.User, error) {
	u, ok := m.users[id]
	if !ok || u.DeletedAt != nil {
		return user.User{}, user.ErrUserNotFound
	}
	return u, nil
}

func (m *mockStorage) GetUserByEmail(_ context.Context, email string) (user.User, error) {
	for _, u := range m.users {
		if u.Email == email && u.DeletedAt == nil {
			return u, nil
		}
	}
	return user.User{}, user.ErrUserNotFound
}

func (m *mockStorage) ListUsers(_ context.Context, filter user.Filter, _ pagination.Sort, page pagination.Page) ([]user.User, int, error) {
	var result []user.User
	for _, u := range m.users {
		if u.DeletedAt != nil {
			continue
		}
		if filter.Role != nil && u.Role != *filter.Role {
			continue
		}
		result = append(result, u)
	}
	total := len(result)
	if page.Offset >= len(result) {
		return nil, total, nil
	}
	end := page.Offset + page.Limit
	if end > len(result) {
		end = len(result)
	}
	return result[page.Offset:end], total, nil
}

func (m *mockStorage) UpdateUser(_ context.Context, u user.User) (user.User, error) {
	existing, ok := m.users[u.ID]
	if !ok || existing.DeletedAt != nil {
		return user.User{}, user.ErrUserNotFound
	}
	for _, other := range m.users {
		if other.ID != u.ID && other.Email == u.Email && other.DeletedAt == nil {
			return user.User{}, user.ErrUserAlreadyExists
		}
	}
	u.UpdatedAt = time.Now().UTC()
	u.CreatedAt = existing.CreatedAt
	m.users[u.ID] = u
	return u, nil
}

func (m *mockStorage) SoftDeleteUser(_ context.Context, id string) error {
	u, ok := m.users[id]
	if !ok || u.DeletedAt != nil {
		return user.ErrUserNotFound
	}
	now := time.Now().UTC()
	u.DeletedAt = &now
	m.users[id] = u
	return nil
}

func (m *mockStorage) CreatePasswordResetToken(_ context.Context, t user.PasswordResetToken) (user.PasswordResetToken, error) {
	t.ID = fmt.Sprintf("token-id-%d", len(m.resetTokens)+1)
	t.CreatedAt = time.Now().UTC()
	m.resetTokens[t.Token] = t
	return t, nil
}

func (m *mockStorage) GetPasswordResetToken(_ context.Context, token string) (user.PasswordResetToken, error) {
	t, ok := m.resetTokens[token]
	if !ok || t.UsedAt != nil {
		return user.PasswordResetToken{}, user.ErrResetTokenNotFound
	}
	if time.Now().UTC().After(t.ExpiresAt) {
		return user.PasswordResetToken{}, user.ErrResetTokenExpired
	}
	return t, nil
}

func (m *mockStorage) MarkPasswordResetTokenUsed(_ context.Context, token string) error {
	t, ok := m.resetTokens[token]
	if !ok || t.UsedAt != nil {
		return user.ErrResetTokenNotFound
	}
	now := time.Now().UTC()
	t.UsedAt = &now
	m.resetTokens[token] = t
	return nil
}

func (m *mockStorage) UpdatePasswordHash(_ context.Context, userID, hash string) error {
	u, ok := m.users[userID]
	if !ok || u.DeletedAt != nil {
		return user.ErrUserNotFound
	}
	u.PasswordHash = hash
	u.UpdatedAt = time.Now().UTC()
	m.users[userID] = u
	return nil
}

func newTestService() (*Service, *mockStorage) {
	storage := newMockStorage()
	authSvc := auth.NewService(auth.Config{
		Secret:     "test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 168 * time.Hour,
	})
	return NewService(storage, authSvc, zap.NewNop()), storage
}

func TestRegister(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, err := svc.Register(ctx, RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		Role:      user.RoleOperator,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if created.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", created.Email)
	}
	if created.FirstName != "John" {
		t.Errorf("expected first_name John, got %s", created.FirstName)
	}
	if created.Role != user.RoleOperator {
		t.Errorf("expected role operator, got %s", created.Role)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := RegisterRequest{
		Email:     "dup@example.com",
		Password:  "password123",
		FirstName: "Jane",
		LastName:  "Doe",
		Role:      user.RoleOperator,
	}

	_, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	_, err = svc.Register(ctx, req)
	if err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestLogin(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterRequest{
		Email:     "login@example.com",
		Password:  "correctpassword",
		FirstName: "User",
		LastName:  "Test",
		Role:      user.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	tokens, err := svc.Login(ctx, "login@example.com", "correctpassword")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("access token is empty")
	}
	if tokens.RefreshToken == "" {
		t.Error("refresh token is empty")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterRequest{
		Email:     "wrong@example.com",
		Password:  "correctpassword",
		FirstName: "User",
		LastName:  "Test",
		Role:      user.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, err = svc.Login(ctx, "wrong@example.com", "wrongpassword")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Login(ctx, "nonexistent@example.com", "password")
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestRefreshToken(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterRequest{
		Email:     "refresh@example.com",
		Password:  "password123",
		FirstName: "User",
		LastName:  "Test",
		Role:      user.RoleOperator,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	tokens, err := svc.Login(ctx, "refresh@example.com", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	newTokens, err := svc.RefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if newTokens.AccessToken == "" {
		t.Error("new access token is empty")
	}
	if newTokens.RefreshToken == "" {
		t.Error("new refresh token is empty")
	}
}

func TestRefreshToken_Invalid(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.RefreshToken(ctx, "invalid-refresh-token")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}
