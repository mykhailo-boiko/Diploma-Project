package notification

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

type Type string

const (
	TypeOrderCreated    Type = "order_created"
	TypeOrderUpdated    Type = "order_updated"
	TypeOrderCancelled  Type = "order_cancelled"
	TypeLowStock        Type = "low_stock"
	TypeStockChanged    Type = "stock_changed"
	TypeShipmentCreated Type = "shipment_created"
	TypeShipmentUpdated Type = "shipment_updated"
	TypeSystem          Type = "system"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusRead    Status = "read"
)

type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      Type       `json:"type"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Status    Status     `json:"status"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type Filter struct {
	UserID *string
	Type   *Type
	Status *Status
}

type Preference struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      Type      `json:"type"`
	InApp     bool      `json:"in_app"`
	Email     bool      `json:"email"`
	SMS       bool      `json:"sms"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrNotificationNotFound = errors.New("notification not found")
)

type Storage interface {
	CreateNotification(ctx context.Context, n Notification) (Notification, error)
	GetNotificationByID(ctx context.Context, id string) (Notification, error)
	ListNotifications(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Notification, int, error)
	MarkAsRead(ctx context.Context, id string) (Notification, error)
	GetUnreadCount(ctx context.Context, userID string) (int, error)
	GetPreferences(ctx context.Context, userID string) ([]Preference, error)
	UpsertPreference(ctx context.Context, pref Preference) (Preference, error)
}
