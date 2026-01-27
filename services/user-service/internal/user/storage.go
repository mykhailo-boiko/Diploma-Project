package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

var _ Storage = (*PostgresStorage)(nil)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

func (s *PostgresStorage) CreateUser(ctx context.Context, u User) (User, error) {
	u.ID = uuid.NewString()
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	query := `
		INSERT INTO users.users (id, email, password_hash, first_name, last_name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, email, first_name, last_name, role, created_at, updated_at`

	var created User
	err := s.pool.QueryRow(ctx, query,
		u.ID, u.Email, u.PasswordHash, u.FirstName, u.LastName, string(u.Role), u.CreatedAt, u.UpdatedAt,
	).Scan(
		&created.ID, &created.Email, &created.FirstName, &created.LastName,
		&created.Role, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return User{}, ErrUserAlreadyExists
		}
		return User{}, fmt.Errorf("failed to create user: %w", err)
	}

	return created, nil
}

func (s *PostgresStorage) GetUserByID(ctx context.Context, id string) (User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, created_at, updated_at
		FROM users.users
		WHERE id = $1 AND deleted_at IS NULL`

	var u User
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("failed to get user by id: %w", err)
	}

	return u, nil
}

func (s *PostgresStorage) GetUserByEmail(ctx context.Context, email string) (User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, created_at, updated_at
		FROM users.users
		WHERE email = $1 AND deleted_at IS NULL`

	var u User
	err := s.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("failed to get user by email: %w", err)
	}

	return u, nil
}

func (s *PostgresStorage) ListUsers(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]User, int, error) {
	where := "WHERE deleted_at IS NULL"
	var args []any
	argIdx := 1

	if filter.Role != nil {
		where += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, string(*filter.Role))
		argIdx++
	}
	if filter.Email != nil {
		where += fmt.Sprintf(" AND email ILIKE $%d", argIdx)
		args = append(args, "%"+*filter.Email+"%")
		argIdx++
	}
	if filter.Name != nil {
		where += fmt.Sprintf(" AND (first_name ILIKE $%d OR last_name ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+*filter.Name+"%")
		argIdx++
	}

	countQuery := "SELECT COUNT(*) FROM users.users " + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	sortField := "created_at"
	allowedSorts := map[string]string{
		"created_at": "created_at",
		"email":      "email",
		"first_name": "first_name",
		"last_name":  "last_name",
		"role":       "role",
	}
	if mapped, ok := allowedSorts[sort.Field]; ok {
		sortField = mapped
	}

	query := fmt.Sprintf(`
		SELECT id, email, first_name, last_name, role, created_at, updated_at
		FROM users.users %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		where, sortField, sort.Direction(), argIdx, argIdx+1)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

func (s *PostgresStorage) UpdateUser(ctx context.Context, u User) (User, error) {
	u.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users.users
		SET email = $2, first_name = $3, last_name = $4, role = $5, updated_at = $6
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, email, first_name, last_name, role, created_at, updated_at`

	var updated User
	err := s.pool.QueryRow(ctx, query,
		u.ID, u.Email, u.FirstName, u.LastName, string(u.Role), u.UpdatedAt,
	).Scan(
		&updated.ID, &updated.Email, &updated.FirstName, &updated.LastName,
		&updated.Role, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		if isDuplicateKeyError(err) {
			return User{}, ErrUserAlreadyExists
		}
		return User{}, fmt.Errorf("failed to update user: %w", err)
	}

	return updated, nil
}

func (s *PostgresStorage) SoftDeleteUser(ctx context.Context, id string) error {
	query := `
		UPDATE users.users
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	now := time.Now().UTC()
	result, err := s.pool.Exec(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *PostgresStorage) CreatePasswordResetToken(ctx context.Context, t PasswordResetToken) (PasswordResetToken, error) {
	t.ID = uuid.NewString()
	t.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO users.password_reset_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, token, expires_at, created_at`

	var created PasswordResetToken
	err := s.pool.QueryRow(ctx, query, t.ID, t.UserID, t.Token, t.ExpiresAt, t.CreatedAt).Scan(
		&created.ID, &created.UserID, &created.Token, &created.ExpiresAt, &created.CreatedAt,
	)
	if err != nil {
		return PasswordResetToken{}, fmt.Errorf("failed to create password reset token: %w", err)
	}
	return created, nil
}

func (s *PostgresStorage) GetPasswordResetToken(ctx context.Context, token string) (PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM users.password_reset_tokens
		WHERE token = $1 AND used_at IS NULL`

	var t PasswordResetToken
	err := s.pool.QueryRow(ctx, query, token).Scan(
		&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PasswordResetToken{}, ErrResetTokenNotFound
		}
		return PasswordResetToken{}, fmt.Errorf("failed to get password reset token: %w", err)
	}

	if time.Now().UTC().After(t.ExpiresAt) {
		return PasswordResetToken{}, ErrResetTokenExpired
	}

	return t, nil
}

func (s *PostgresStorage) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	query := `
		UPDATE users.password_reset_tokens
		SET used_at = $2
		WHERE token = $1 AND used_at IS NULL`

	now := time.Now().UTC()
	result, err := s.pool.Exec(ctx, query, token, now)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrResetTokenNotFound
	}
	return nil
}

func (s *PostgresStorage) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	query := `
		UPDATE users.users
		SET password_hash = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL`

	now := time.Now().UTC()
	result, err := s.pool.Exec(ctx, query, userID, hash, now)
	if err != nil {
		return fmt.Errorf("failed to update password hash: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func isDuplicateKeyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}
