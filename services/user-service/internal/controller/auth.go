package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	"github.com/haradrim/chainorchestra/services/user-service/internal/auth"
	"github.com/haradrim/chainorchestra/services/user-service/internal/user"
)

type UserService interface {
	Login(ctx context.Context, email, password string) (auth.TokenPair, error)
	Register(ctx context.Context, req RegisterRequest) (user.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (auth.TokenPair, error)
}

type AuthController struct {
	svc UserService
	log *zap.Logger
}

func NewAuthController(svc UserService, log *zap.Logger) *AuthController {
	return &AuthController{svc: svc, log: log}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      user.Role `json:"role"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		httpresponse.BadRequest(w, "validation_error", "email and password are required")
		return
	}

	tokens, err := c.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) || errors.Is(err, user.ErrUserNotFound) {
			httpresponse.Unauthorized(w, "invalid_credentials", "invalid email or password")
			return
		}
		c.log.Error("Login failed", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, tokens)
}

func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	callerRole := middleware.GetUserRole(r.Context())
	if callerRole != string(user.RoleAdmin) {
		httpresponse.Forbidden(w, "forbidden", "only admin can register users")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" || req.Role == "" {
		httpresponse.BadRequest(w, "validation_error", "all fields are required")
		return
	}

	if err := ValidatePassword(req.Password); err != nil {
		httpresponse.BadRequest(w, "weak_password", err.Error())
		return
	}

	if !user.ValidRoles[req.Role] {
		httpresponse.BadRequest(w, "invalid_role", "invalid role")
		return
	}

	created, err := c.svc.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, user.ErrUserAlreadyExists) {
			httpresponse.Err(w, http.StatusConflict, "user_already_exists", "user with this email already exists")
			return
		}
		c.log.Error("Register failed", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *AuthController) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		httpresponse.BadRequest(w, "validation_error", "refresh_token is required")
		return
	}

	tokens, err := c.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		httpresponse.Unauthorized(w, "invalid_token", "invalid or expired refresh token")
		return
	}

	httpresponse.OK(w, tokens)
}
