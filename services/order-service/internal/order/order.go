package order

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusConfirmed  Status = "confirmed"
	StatusProcessing Status = "processing"
	StatusShipped    Status = "shipped"
	StatusDelivered  Status = "delivered"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
	StatusReturned   Status = "returned"
)

var validTransitions = map[Status][]Status{
	StatusPending:    {StatusConfirmed, StatusCancelled},
	StatusConfirmed:  {StatusProcessing, StatusCancelled},
	StatusProcessing: {StatusShipped, StatusCancelled},
	StatusShipped:    {StatusDelivered, StatusReturned, StatusCancelled},
	StatusDelivered:  {StatusCompleted, StatusCancelled},
}

func CanTransition(from, to Status) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

type Order struct {
	ID           string     `json:"id"`
	CustomerName string     `json:"customer_name"`
	Status       Status     `json:"status"`
	TotalAmount  float64    `json:"total_amount"`
	CancelReason *string    `json:"cancel_reason,omitempty"`
	Items        []Item     `json:"items,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type Item struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"order_id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
}

type Filter struct {
	Status       *Status
	DateFrom     *time.Time
	DateTo       *time.Time
	CustomerName *string
}

type StatusStats struct {
	Status string  `json:"status"`
	Count  int     `json:"count"`
	Revenue float64 `json:"revenue"`
}

type OrderStats struct {
	TotalOrders  int           `json:"total_orders"`
	TotalRevenue float64       `json:"total_revenue"`
	ByStatus     []StatusStats `json:"by_status"`
}

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type Storage interface {
	CreateOrder(ctx context.Context, o Order) (Order, error)
	GetOrderByID(ctx context.Context, id string) (Order, error)
	ListOrders(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Order, int, error)
	UpdateOrderStatus(ctx context.Context, id string, status Status) (Order, error)
	CancelOrder(ctx context.Context, id string, reason string) (Order, error)
	SearchOrders(ctx context.Context, query string, page pagination.Page) ([]Order, int, error)
	GetOrderStats(ctx context.Context) (OrderStats, error)
}
