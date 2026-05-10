package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/order-service/internal/order"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (order.Order, error)
	GetOrderByID(ctx context.Context, id string) (order.Order, error)
	ListOrders(ctx context.Context, filter order.Filter, sort pagination.Sort, page pagination.Page) ([]order.Order, int, error)
	UpdateOrderStatus(ctx context.Context, id string, newStatus order.Status) (order.Order, error)
	CancelOrder(ctx context.Context, id string, reason string) (order.Order, error)
	SearchOrders(ctx context.Context, query string, page pagination.Page) ([]order.Order, int, error)
	GetOrderStats(ctx context.Context) (order.OrderStats, error)
	GetSalesByProduct(ctx context.Context, from, to time.Time, includeStatuses []order.Status) ([]order.ProductSales, error)
	GetCustomerSummary(ctx context.Context, filter order.CustomerFilter) ([]order.CustomerSummary, error)
	BulkUpdateStatus(ctx context.Context, orderIDs []string, newStatus order.Status, note string) (order.BulkStatusResult, error)
}

type OrderController struct {
	svc OrderService
	log *zap.Logger
}

func NewOrderController(svc OrderService, log *zap.Logger) *OrderController {
	return &OrderController{svc: svc, log: log}
}

var allowedSortFields = map[string]bool{
	"created_at":    true,
	"total_amount":  true,
	"status":        true,
	"customer_name": true,
}

func (c *OrderController) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.CustomerName == "" {
		httpresponse.BadRequest(w, "validation_error", "customer_name is required")
		return
	}
	if len(req.Items) == 0 {
		httpresponse.BadRequest(w, "validation_error", "at least one item is required")
		return
	}
	for _, item := range req.Items {
		if item.Name == "" || item.Quantity <= 0 || item.UnitPrice <= 0 {
			httpresponse.BadRequest(w, "validation_error", "each item must have a name, quantity > 0, and unit_price > 0")
			return
		}
	}

	created, err := c.svc.CreateOrder(r.Context(), req)
	if err != nil {
		c.log.Error("Failed to create order", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *OrderController) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "order id is required")
		return
	}

	o, err := c.svc.GetOrderByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, order.ErrOrderNotFound) {
			httpresponse.NotFound(w, "order_not_found", "order not found")
			return
		}
		c.log.Error("Failed to get order", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, o)
}

func (c *OrderController) List(w http.ResponseWriter, r *http.Request) {
	filter := parseFilter(r)
	sort := pagination.SortFromRequest(r, allowedSortFields, "created_at")
	page := pagination.PageFromRequest(r)

	orders, total, err := c.svc.ListOrders(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list orders", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if orders == nil {
		orders = []order.Order{}
	}

	httpresponse.List(w, orders, total, page.Limit, page.Offset)
}

func (c *OrderController) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "order id is required")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Status == "" {
		httpresponse.BadRequest(w, "validation_error", "status is required")
		return
	}

	updated, err := c.svc.UpdateOrderStatus(r.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, order.ErrOrderNotFound) {
			httpresponse.NotFound(w, "order_not_found", "order not found")
			return
		}
		if errors.Is(err, order.ErrInvalidTransition) {
			httpresponse.BadRequest(w, "invalid_transition", "invalid status transition")
			return
		}
		c.log.Error("Failed to update order status", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *OrderController) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "order id is required")
		return
	}

	var req CancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Reason == "" {
		httpresponse.BadRequest(w, "validation_error", "reason is required")
		return
	}

	cancelled, err := c.svc.CancelOrder(r.Context(), id, req.Reason)
	if err != nil {
		if errors.Is(err, order.ErrOrderNotFound) {
			httpresponse.NotFound(w, "order_not_found", "order not found")
			return
		}
		if errors.Is(err, order.ErrInvalidTransition) {
			httpresponse.BadRequest(w, "invalid_transition", "order cannot be cancelled in its current status")
			return
		}
		c.log.Error("Failed to cancel order", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, cancelled)
}

func (c *OrderController) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		httpresponse.BadRequest(w, "validation_error", "search query must be at least 2 characters")
		return
	}

	page := pagination.PageFromRequest(r)

	orders, total, err := c.svc.SearchOrders(r.Context(), q, page)
	if err != nil {
		c.log.Error("Failed to search orders", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if orders == nil {
		orders = []order.Order{}
	}

	httpresponse.List(w, orders, total, page.Limit, page.Offset)
}

func (c *OrderController) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := c.svc.GetOrderStats(r.Context())
	if err != nil {
		c.log.Error("Failed to get order stats", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, stats)
}

func (c *OrderController) SalesByProduct(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	dateFromStr := q.Get("date_from")
	dateToStr := q.Get("date_to")
	if dateFromStr == "" || dateToStr == "" {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD)")
		return
	}

	from, err := parseFlexibleDate(dateFromStr)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", "invalid date_from format")
		return
	}
	to, err := parseFlexibleDate(dateToStr)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", "invalid date_to format")
		return
	}
	if !strings.Contains(dateToStr, "T") {
		to = to.Add(24*time.Hour - time.Second)
	}

	var includeStatuses []order.Status
	if s := q.Get("statuses"); s != "" {
		for _, raw := range strings.Split(s, ",") {
			trimmed := strings.TrimSpace(raw)
			if trimmed != "" {
				includeStatuses = append(includeStatuses, order.Status(trimmed))
			}
		}
	}

	limit := 0
	if s := q.Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			limit = v
		}
	}

	results, err := c.svc.GetSalesByProduct(r.Context(), from, to, includeStatuses)
	if err != nil {
		c.log.Error("Failed to get sales by product", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if results == nil {
		results = []order.ProductSales{}
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	httpresponse.OK(w, results)
}

func parseFlexibleDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

func (c *OrderController) BulkUpdateStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderIDs []string `json:"order_ids"`
		Status   string   `json:"status"`
		Note     string   `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if len(req.OrderIDs) == 0 {
		httpresponse.BadRequest(w, "validation_error", "order_ids is required")
		return
	}
	if len(req.OrderIDs) > 500 {
		httpresponse.BadRequest(w, "validation_error", "max 500 order_ids per call")
		return
	}
	if req.Status == "" {
		httpresponse.BadRequest(w, "validation_error", "status is required")
		return
	}

	result, err := c.svc.BulkUpdateStatus(r.Context(), req.OrderIDs, order.Status(req.Status), req.Note)
	if err != nil {
		c.log.Error("Failed bulk status update", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, result)
}

func (c *OrderController) CustomerSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var filter order.CustomerFilter

	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")
	if dateFrom != "" && dateTo != "" {
		from, err := parseFlexibleDate(dateFrom)
		if err != nil {
			httpresponse.BadRequest(w, "validation_error", "invalid date_from format")
			return
		}
		to, err := parseFlexibleDate(dateTo)
		if err != nil {
			httpresponse.BadRequest(w, "validation_error", "invalid date_to format")
			return
		}
		if !strings.Contains(dateTo, "T") {
			to = to.Add(24*time.Hour - time.Second)
		}
		filter.WindowFrom = &from
		filter.WindowTo = &to
	}

	if q.Get("only_new") == "true" {
		filter.OnlyNew = true
	}

	switch order.CustomerSortField(q.Get("sort_by")) {
	case order.CustomerSortRevenue:
		filter.SortBy = order.CustomerSortRevenue
	case order.CustomerSortRevenueWindow:
		filter.SortBy = order.CustomerSortRevenueWindow
	case order.CustomerSortOrders:
		filter.SortBy = order.CustomerSortOrders
	case order.CustomerSortLastOrder:
		filter.SortBy = order.CustomerSortLastOrder
	case order.CustomerSortFirstOrder:
		filter.SortBy = order.CustomerSortFirstOrder
	default:
		if filter.WindowFrom != nil {
			filter.SortBy = order.CustomerSortRevenueWindow
		} else {
			filter.SortBy = order.CustomerSortRevenue
		}
	}

	filter.SortDesc = strings.ToLower(q.Get("sort_order")) != "asc"

	if s := q.Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			filter.Limit = v
		}
	}

	results, err := c.svc.GetCustomerSummary(r.Context(), filter)
	if err != nil {
		c.log.Error("Failed to get customer summary", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if results == nil {
		results = []order.CustomerSummary{}
	}

	httpresponse.OK(w, results)
}

func parseFilter(r *http.Request) order.Filter {
	var filter order.Filter

	if s := r.URL.Query().Get("status"); s != "" {
		status := order.Status(s)
		filter.Status = &status
	}
	if s := r.URL.Query().Get("date_from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.DateFrom = &t
		}
	}
	if s := r.URL.Query().Get("date_to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.DateTo = &t
		}
	}
	if s := r.URL.Query().Get("customer_name"); s != "" {
		filter.CustomerName = &s
	}

	return filter
}
