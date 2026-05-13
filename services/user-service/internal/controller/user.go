package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/user-service/internal/user"
)

var userSortFields = map[string]bool{
	"created_at": true,
	"email":      true,
	"first_name": true,
	"last_name":  true,
	"role":       true,
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type AdminUpdateUserRequest struct {
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Role      user.Role `json:"role"`
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type UserController struct {
	svc *Service
	log *zap.Logger
}

func NewUserController(svc *Service, log *zap.Logger) *UserController {
	return &UserController{svc: svc, log: log}
}

func (c *UserController) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httpresponse.Unauthorized(w, "missing_user_id", "user id not found in request")
		return
	}

	u, err := c.svc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			httpresponse.NotFound(w, "user_not_found", "user not found")
			return
		}
		c.log.Error("Failed to get profile", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, u)
}

func (c *UserController) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httpresponse.Unauthorized(w, "missing_user_id", "user id not found in request")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	updated, err := c.svc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			httpresponse.NotFound(w, "user_not_found", "user not found")
			return
		}
		if errors.Is(err, user.ErrUserAlreadyExists) {
			httpresponse.Err(w, http.StatusConflict, "email_taken", "email already in use")
			return
		}
		c.log.Error("Failed to update profile", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *UserController) GetUserByID(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")
	if targetID == "" {
		httpresponse.BadRequest(w, "validation_error", "user id is required")
		return
	}
	callerID := middleware.GetUserID(r.Context())
	callerRole := middleware.GetUserRole(r.Context())
	if callerRole != string(user.RoleAdmin) && callerID != targetID {
		httpresponse.Forbidden(w, "forbidden", "cannot view another user's profile")
		return
	}
	u, err := c.svc.GetProfile(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			httpresponse.NotFound(w, "user_not_found", "user not found")
			return
		}
		c.log.Error("Failed to get user", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}
	httpresponse.OK(w, u)
}

func (c *UserController) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !c.requireAdmin(w, r) {
		return
	}

	filter := parseUserFilter(r)
	page := pagination.PageFromRequest(r)
	sort := pagination.SortFromRequest(r, userSortFields, "created_at")

	users, total, err := c.svc.ListUsers(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list users", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if users == nil {
		users = []user.User{}
	}
	httpresponse.List(w, users, total, page.Limit, page.Offset)
}

func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
	if !c.requireAdmin(w, r) {
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
		c.log.Error("Failed to create user", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *UserController) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if !c.requireAdmin(w, r) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "missing_id", "user id is required")
		return
	}

	var req AdminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Role != "" && !user.ValidRoles[req.Role] {
		httpresponse.BadRequest(w, "invalid_role", "invalid role")
		return
	}

	updated, err := c.svc.UpdateUser(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			httpresponse.NotFound(w, "user_not_found", "user not found")
			return
		}
		if errors.Is(err, user.ErrUserAlreadyExists) {
			httpresponse.Err(w, http.StatusConflict, "email_taken", "email already in use")
			return
		}
		c.log.Error("Failed to update user", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *UserController) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if !c.requireAdmin(w, r) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "missing_id", "user id is required")
		return
	}

	if err := c.svc.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			httpresponse.NotFound(w, "user_not_found", "user not found")
			return
		}
		c.log.Error("Failed to delete user", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, map[string]string{"status": "deleted"})
}

func (c *UserController) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Email == "" {
		httpresponse.BadRequest(w, "validation_error", "email is required")
		return
	}

	if err := c.svc.RequestPasswordReset(r.Context(), req.Email); err != nil {
		c.log.Error("Failed to request password reset", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, map[string]string{"message": "if the email exists, a reset link has been sent"})
}

func (c *UserController) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		httpresponse.BadRequest(w, "validation_error", "token and new_password are required")
		return
	}

	if err := c.svc.ConfirmPasswordReset(r.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, user.ErrResetTokenNotFound) {
			httpresponse.BadRequest(w, "invalid_token", "invalid or expired reset token")
			return
		}
		if errors.Is(err, user.ErrResetTokenExpired) {
			httpresponse.BadRequest(w, "token_expired", "reset token has expired")
			return
		}
		if errors.Is(err, user.ErrResetTokenUsed) {
			httpresponse.BadRequest(w, "token_used", "reset token has already been used")
			return
		}
		c.log.Error("Failed to confirm password reset", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, map[string]string{"message": "password has been reset successfully"})
}

func (c *UserController) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	role := middleware.GetUserRole(r.Context())
	if role != string(user.RoleAdmin) {
		httpresponse.Forbidden(w, "forbidden", "admin access required")
		return false
	}
	return true
}

func parseUserFilter(r *http.Request) user.Filter {
	var filter user.Filter

	if roleStr := r.URL.Query().Get("role"); roleStr != "" {
		role := user.Role(roleStr)
		filter.Role = &role
	}
	if email := r.URL.Query().Get("email"); email != "" {
		filter.Email = &email
	}
	if name := r.URL.Query().Get("name"); name != "" {
		filter.Name = &name
	}

	return filter
}
