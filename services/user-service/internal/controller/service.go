package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/user-service/internal/auth"
	"github.com/haradrim/chainorchestra/services/user-service/internal/user"
)

const passwordResetTokenTTL = 1 * time.Hour

type Service struct {
	storage user.Storage
	auth    *auth.Service
	log     *zap.Logger
}

func NewService(storage user.Storage, authSvc *auth.Service, log *zap.Logger) *Service {
	return &Service{storage: storage, auth: authSvc, log: log}
}

func (s *Service) Login(ctx context.Context, email, password string) (auth.TokenPair, error) {
	u, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return auth.TokenPair{}, user.ErrInvalidCredentials
	}

	if !s.auth.CheckPassword(u.PasswordHash, password) {
		return auth.TokenPair{}, user.ErrInvalidCredentials
	}

	tokens, err := s.auth.GenerateTokenPair(u)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (user.User, error) {
	hash, err := s.auth.HashPassword(req.Password)
	if err != nil {
		return user.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	u := user.User{
		Email:        req.Email,
		PasswordHash: hash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
	}

	created, err := s.storage.CreateUser(ctx, u)
	if err != nil {
		return user.User{}, err
	}

	return created, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (auth.TokenPair, error) {
	userID, err := s.auth.ValidateRefreshToken(refreshToken)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("invalid refresh token: %w", err)
	}

	u, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("failed to get user: %w", err)
	}

	tokens, err := s.auth.GenerateTokenPair(u)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

func (s *Service) GetProfile(ctx context.Context, userID string) (user.User, error) {
	return s.storage.GetUserByID(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (user.User, error) {
	u, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return user.User{}, err
	}

	if req.FirstName != "" {
		u.FirstName = req.FirstName
	}
	if req.LastName != "" {
		u.LastName = req.LastName
	}
	if req.Email != "" {
		u.Email = req.Email
	}

	return s.storage.UpdateUser(ctx, u)
}

func (s *Service) ListUsers(ctx context.Context, filter user.Filter, sort pagination.Sort, page pagination.Page) ([]user.User, int, error) {
	return s.storage.ListUsers(ctx, filter, sort, page)
}

func (s *Service) UpdateUser(ctx context.Context, id string, req AdminUpdateUserRequest) (user.User, error) {
	u, err := s.storage.GetUserByID(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	if req.FirstName != "" {
		u.FirstName = req.FirstName
	}
	if req.LastName != "" {
		u.LastName = req.LastName
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Role != "" {
		u.Role = req.Role
	}

	return s.storage.UpdateUser(ctx, u)
}

func (s *Service) DeleteUser(ctx context.Context, id string) error {
	return s.storage.SoftDeleteUser(ctx, id)
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	u, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		s.log.Info("Password reset requested for unknown email", zap.String("email", email))
		return nil
	}

	token, err := generateSecureToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	resetToken := user.PasswordResetToken{
		UserID:    u.ID,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(passwordResetTokenTTL),
	}

	_, err = s.storage.CreatePasswordResetToken(ctx, resetToken)
	if err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	s.log.Info("MOCK EMAIL: Password reset requested",
		zap.String("to", u.Email),
		zap.String("user_id", u.ID),
		zap.String("reset_token", token),
		zap.String("reset_url", fmt.Sprintf("http://localhost:3000/reset-password?token=%s", token)),
		zap.Time("expires_at", resetToken.ExpiresAt),
	)

	return nil
}

func (s *Service) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	resetToken, err := s.storage.GetPasswordResetToken(ctx, token)
	if err != nil {
		return err
	}

	hash, err := s.auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.storage.UpdatePasswordHash(ctx, resetToken.UserID, hash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.storage.MarkPasswordResetTokenUsed(ctx, token); err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	s.log.Info("Password reset completed", zap.String("user_id", resetToken.UserID))
	return nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
