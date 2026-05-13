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
	StatusProcessing: {StatusConfirmed, StatusShipped, StatusCancelled},
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

type ProductSales struct {
	ProductID    string  `json:"product_id"`
	Name         string  `json:"name"`
	UnitsSold    int     `json:"units_sold"`
	Revenue      float64 `json:"revenue"`
	OrderCount   int     `json:"order_count"`
	DailyDemand  float64 `json:"daily_demand"`
}

type BulkStatusItem struct {
	OrderID   string `json:"order_id"`
	OldStatus Status `json:"old_status,omitempty"`
	NewStatus Status `json:"new_status,omitempty"`
	Error     string `json:"error,omitempty"`
}

type BulkStatusResult struct {
	Total      int              `json:"total"`
	DryRun     bool             `json:"dry_run"`
	UpdatedIDs []string         `json:"updated_ids"`
	Successes  []BulkStatusItem `json:"successes"`
	Failures   []BulkStatusItem `json:"failures"`
}

type CustomerSummary struct {
	CustomerName     string    `json:"customer_name"`
	FirstOrderDate   time.Time `json:"first_order_date"`
	LastOrderDate    time.Time `json:"last_order_date"`
	TotalOrders      int       `json:"total_orders"`
	TotalRevenue     float64   `json:"total_revenue"`
	AvgOrderValue    float64   `json:"avg_order_value"`
	OrdersInWindow   int       `json:"orders_in_window,omitempty"`
	RevenueInWindow  float64   `json:"revenue_in_window,omitempty"`
	NewInWindow      bool      `json:"new_in_window,omitempty"`
}

type CustomerSortField string

const (
	CustomerSortRevenue       CustomerSortField = "revenue"
	CustomerSortRevenueWindow CustomerSortField = "revenue_in_window"
	CustomerSortOrders        CustomerSortField = "orders"
	CustomerSortLastOrder     CustomerSortField = "last_order"
	CustomerSortFirstOrder    CustomerSortField = "first_order"
)

type CustomerFilter struct {
	WindowFrom *time.Time
	WindowTo   *time.Time
	OnlyNew    bool
	SortBy     CustomerSortField
	SortDesc   bool
	Limit      int
}

var (
	ErrOrderNotFound          = errors.New("order not found")
	ErrInvalidTransition      = errors.New("invalid status transition")
	ErrProductNotFound        = errors.New("product not found")
	ErrConcurrentStatusUpdate = errors.New("order status was modified concurrently; refresh and retry")
)

type InvalidTransitionError struct {
	Current   string
	Requested string
}

func (e *InvalidTransitionError) Error() string {
	return "cannot transition order from '" + e.Current + "' to '" + e.Requested + "'"
}

func (e *InvalidTransitionError) Is(target error) bool {
	return target == ErrInvalidTransition
}

func IsValidStatus(s Status) bool {
	switch s {
	case StatusPending, StatusConfirmed, StatusProcessing, StatusShipped,
		StatusDelivered, StatusCompleted, StatusCancelled, StatusReturned:
		return true
	}
	return false
}

func AllStatuses() []string {
	return []string{
		string(StatusPending), string(StatusConfirmed), string(StatusProcessing),
		string(StatusShipped), string(StatusDelivered), string(StatusCompleted),
		string(StatusCancelled), string(StatusReturned),
	}
}

type ProductValidator interface {
	ValidateProduct(ctx context.Context, productID string) error
}

type Storage interface {
	CreateOrder(ctx context.Context, o Order) (Order, error)
	GetOrderByID(ctx context.Context, id string) (Order, error)
	ListOrders(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Order, int, error)
	UpdateOrderStatus(ctx context.Context, id string, expected, next Status) (Order, error)
	CancelOrder(ctx context.Context, id string, expected Status, reason string) (Order, error)
	SearchOrders(ctx context.Context, query string, page pagination.Page) ([]Order, int, error)
	GetOrderStats(ctx context.Context) (OrderStats, error)
	GetSalesByProduct(ctx context.Context, from, to time.Time, includeStatuses []Status) ([]ProductSales, error)
	GetCustomerSummary(ctx context.Context, filter CustomerFilter) ([]CustomerSummary, error)
	BulkUpdateStatus(ctx context.Context, orderIDs []string, newStatus Status, note string, dryRun bool) (BulkStatusResult, error)
}
