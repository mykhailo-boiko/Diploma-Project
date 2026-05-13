package notification

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

func (s *PostgresStorage) CreateNotification(ctx context.Context, n Notification) (Notification, error) {
	n.ID = uuid.NewString()
	n.Status = StatusPending
	n.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO notifications.notification (id, user_id, type, title, message, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err := s.pool.Exec(ctx, query,
		n.ID, n.UserID, string(n.Type), n.Title, n.Message, string(n.Status), n.CreatedAt,
	); err != nil {
		return Notification{}, fmt.Errorf("failed to insert notification: %w", err)
	}

	return n, nil
}

func (s *PostgresStorage) GetNotificationByID(ctx context.Context, id string) (Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, status, read_at, created_at
		FROM notifications.notification
		WHERE id = $1`

	var n Notification
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Status, &n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotificationNotFound
		}
		return Notification{}, fmt.Errorf("failed to get notification: %w", err)
	}

	return n, nil
}

func (s *PostgresStorage) ListNotifications(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Notification, int, error) {
	where, args := buildWhereClause(filter)

	countQuery := "SELECT COUNT(*) FROM notifications.notification" + where
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	sortColumn := mapSortField(sort.Field)
	query := fmt.Sprintf(
		"SELECT id, user_id, type, title, message, status, read_at, created_at FROM notifications.notification%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortColumn, sort.Direction(), len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Status, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate notifications: %w", err)
	}

	return notifications, total, nil
}

func (s *PostgresStorage) MarkAsRead(ctx context.Context, id string) (Notification, error) {
	query := `
		UPDATE notifications.notification
		SET status = $1, read_at = $2
		WHERE id = $3 AND status = $4
		RETURNING id, user_id, type, title, message, status, read_at, created_at`

	now := time.Now().UTC()
	var n Notification
	err := s.pool.QueryRow(ctx, query, string(StatusRead), now, id, string(StatusPending)).Scan(
		&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Status, &n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotificationNotFound
		}
		return Notification{}, fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return n, nil
}

func (s *PostgresStorage) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*) FROM notifications.notification
		WHERE user_id = $1 AND status = $2`

	var count int
	if err := s.pool.QueryRow(ctx, query, userID, string(StatusPending)).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

func (s *PostgresStorage) GetUnreadCountsAll(ctx context.Context) ([]UserUnreadCount, error) {
	query := `
		SELECT user_id, COUNT(*) AS unread_count
		FROM notifications.notification
		WHERE status = $1
		GROUP BY user_id
		ORDER BY unread_count DESC`

	rows, err := s.pool.Query(ctx, query, string(StatusPending))
	if err != nil {
		return nil, fmt.Errorf("failed to query unread counts: %w", err)
	}
	defer rows.Close()

	var counts []UserUnreadCount
	for rows.Next() {
		var c UserUnreadCount
		if err := rows.Scan(&c.UserID, &c.UnreadCount); err != nil {
			return nil, fmt.Errorf("failed to scan unread count: %w", err)
		}
		counts = append(counts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate unread counts: %w", err)
	}

	return counts, nil
}

func (s *PostgresStorage) GetPreferences(ctx context.Context, userID string) ([]Preference, error) {
	query := `
		SELECT id, user_id, type, in_app, email, sms, updated_at
		FROM notifications.notification_preference
		WHERE user_id = $1
		ORDER BY type`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list preferences: %w", err)
	}
	defer rows.Close()

	var prefs []Preference
	for rows.Next() {
		var p Preference
		if err := rows.Scan(&p.ID, &p.UserID, &p.Type, &p.InApp, &p.Email, &p.SMS, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan preference: %w", err)
		}
		prefs = append(prefs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate preferences: %w", err)
	}

	return prefs, nil
}

func (s *PostgresStorage) UpsertPreference(ctx context.Context, pref Preference) (Preference, error) {
	pref.ID = uuid.NewString()
	pref.UpdatedAt = time.Now().UTC()

	query := `
		INSERT INTO notifications.notification_preference (id, user_id, type, in_app, email, sms, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, type) DO UPDATE SET
			in_app = EXCLUDED.in_app,
			email = EXCLUDED.email,
			sms = EXCLUDED.sms,
			updated_at = EXCLUDED.updated_at
		RETURNING id, user_id, type, in_app, email, sms, updated_at`

	var result Preference
	err := s.pool.QueryRow(ctx, query,
		pref.ID, pref.UserID, string(pref.Type), pref.InApp, pref.Email, pref.SMS, pref.UpdatedAt,
	).Scan(&result.ID, &result.UserID, &result.Type, &result.InApp, &result.Email, &result.SMS, &result.UpdatedAt)
	if err != nil {
		return Preference{}, fmt.Errorf("failed to upsert preference: %w", err)
	}

	return result, nil
}

func buildWhereClause(filter Filter) (string, []any) {
	var conditions []string
	var args []any

	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if filter.Type != nil {
		args = append(args, string(*filter.Type))
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
	}
	if filter.Status != nil {
		switch string(*filter.Status) {
		case "unread":
			conditions = append(conditions, "status != 'read'")
		case "read":
			conditions = append(conditions, "status = 'read'")
		default:
			args = append(args, string(*filter.Status))
			conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
		}
	}
	if filter.IsRead != nil {
		if *filter.IsRead {
			conditions = append(conditions, "status = 'read'")
		} else {
			conditions = append(conditions, "status != 'read'")
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func mapSortField(field string) string {
	switch field {
	case "created_at":
		return "created_at"
	case "type":
		return "type"
	case "status":
		return "status"
	default:
		return "created_at"
	}
}
