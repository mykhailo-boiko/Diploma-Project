package user

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

type Role string

const (
	RoleAdmin            Role = "admin"
	RoleWarehouseManager Role = "warehouse_manager"
	RoleLogisticsManager Role = "logistics_manager"
	RoleAnalyst          Role = "analyst"
	RoleOperator         Role = "operator"
)

var ValidRoles = map[Role]bool{
	RoleAdmin:            true,
	RoleWarehouseManager: true,
	RoleLogisticsManager: true,
	RoleAnalyst:          true,
	RoleOperator:         true,
}

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Role         Role       `json:"role"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type Filter struct {
	Role  *Role
	Email *string
	Name  *string
}

type PasswordResetToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Token     string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrResetTokenNotFound = errors.New("reset token not found")
	ErrResetTokenExpired  = errors.New("reset token expired")
	ErrResetTokenUsed     = errors.New("reset token already used")
)

type Storage interface {
	CreateUser(ctx context.Context, u User) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	ListUsers(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]User, int, error)
	UpdateUser(ctx context.Context, u User) (User, error)
	SoftDeleteUser(ctx context.Context, id string) error
	CreatePasswordResetToken(ctx context.Context, t PasswordResetToken) (PasswordResetToken, error)
	GetPasswordResetToken(ctx context.Context, token string) (PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, token string) error
	UpdatePasswordHash(ctx context.Context, userID, hash string) error
}
