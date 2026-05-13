package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/order-service/internal/order"
)

var customerNameAllowedRune = func(r rune) bool {
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 0x00C0 && r <= 0x00FF && r != 0x00D7 && r != 0x00F7 {
		return true
	}
	if r >= 0x0100 && r <= 0x017F {
		return true
	}
	if r >= 0x0180 && r <= 0x024F {
		return true
	}
	if r == ' ' || r == '-' || r == '\'' || r == '.' {
		return true
	}
	return false
}

func isValidCustomerName(s string) bool {
	if s == "" {
		return false
	}
	if len(s) > 200 {
		return false
	}
	runeCount := 0
	for _, r := range s {
		runeCount++
		if !customerNameAllowedRune(r) {
			return false
		}
	}
	if runeCount < 2 || runeCount > 100 {
		return false
	}
	first := []rune(s)[0]
	if first == ' ' || first == '-' || first == '\'' || first == '.' {
		return false
	}
	return true
}

func isLatinName(s string) bool {
	return isValidCustomerName(s)
}

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
	BulkUpdateStatus(ctx context.Context, orderIDs []string, newStatus order.Status, note string, dryRun bool) (order.BulkStatusResult, error)
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
		httpresponse.InvalidBody(w, err.Error())
		return
	}

	if req.CustomerName == "" {
		httpresponse.MissingField(w, "customer_name",
			"non-empty string (full customer name)",
			"Provide the full customer name (Latin transliteration). Example: 'Yuliia Morozenko'",
			"Yuliia Morozenko", "John Doe")
		return
	}
	if !isLatinName(req.CustomerName) {
		httpresponse.InvalidField(w, "customer_name",
			"Latin-script characters only (ASCII)", req.CustomerName,
			"Use Latin transliteration. Cyrillic, Hebrew, Arabic, CJK and other non-ASCII characters are not accepted.",
			"Yuliia Morozenko", "John Doe", "Ivan Petrenko")
		return
	}
	if len(req.Items) == 0 {
		httpresponse.MissingField(w, "items",
			"non-empty array of order items",
			"At least one item is required. Each item needs product_id, name, quantity > 0, unit_price > 0.")
		return
	}
	for i, item := range req.Items {
		field := "items[" + strconv.Itoa(i) + "]"
		if item.ProductID == "" {
			httpresponse.MissingField(w, field+".product_id",
				"product UUID v4",
				"Each item must reference an existing product. Use products_list/products_get to obtain a valid product_id.")
			return
		}
		if _, err := uuid.Parse(item.ProductID); err != nil {
			httpresponse.InvalidField(w, field+".product_id",
				"valid UUID v4", item.ProductID,
				"product_id must be a UUID. Use products_list/products_get to obtain a valid product_id.",
				"9ebcaf0f-c50d-4f36-b417-c3fa7477fc8c")
			return
		}
		if item.Name == "" {
			httpresponse.MissingField(w, field+".name",
				"non-empty product name string",
				"Each item must include the product name.")
			return
		}
		if item.Quantity <= 0 {
			httpresponse.InvalidField(w, field+".quantity",
				"positive integer (> 0)", item.Quantity,
				"Quantity must be greater than 0. Use 1 for a single unit.",
				"1", "5", "10")
			return
		}
		if item.UnitPrice <= 0 {
			httpresponse.InvalidField(w, field+".unit_price",
				"positive number (> 0)", item.UnitPrice,
				"Unit price must be greater than 0. Use the product's unit_price from products_list/products_get.",
				"19.99", "499.00")
			return
		}
	}

	created, err := c.svc.CreateOrder(r.Context(), req)
	if err != nil {
		if errors.Is(err, order.ErrProductNotFound) {
			httpresponse.Err(w, http.StatusNotFound, "product_not_found",
				"one or more product_id values do not exist in inventory")
			return
		}
		c.log.Error("Failed to create order", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *OrderController) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := httpresponse.ValidateUUIDPath(w, r, "id")
	if !ok {
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
	filter, ok := parseFilterStrict(w, r)
	if !ok {
		return
	}
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
		httpresponse.MissingField(w, "id (path)",
			"order UUID in path",
			"Specify the order ID in the URL path: PUT /api/v1/orders/{id}/status")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.InvalidBody(w, err.Error())
		return
	}

	if req.Status == "" {
		httpresponse.MissingField(w, "status",
			"one of: pending, confirmed, processing, shipped, delivered, completed, cancelled, returned",
			"Specify the target status. Check the current status with orders_get before transitioning.",
			"confirmed", "processing", "shipped", "delivered", "cancelled")
		return
	}

	updated, err := c.svc.UpdateOrderStatus(r.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, order.ErrOrderNotFound) {
			httpresponse.NotFoundError(w, httpresponse.LLMError{
				Message:    "order with id '" + id + "' was not found",
				Field:      "id",
				Received:   id,
				Suggestion: "Verify the order exists using orders_list or orders_search before updating its status. The ID may be wrong, deleted (soft-deleted), or belong to a different scope.",
			})
			return
		}
		var transitionErr *order.InvalidTransitionError
		if errors.As(err, &transitionErr) {
			httpresponse.InvalidTransition(w, "order", transitionErr.Current, transitionErr.Requested,
				[]string{"pending→confirmed", "confirmed→processing", "processing→shipped", "shipped→delivered", "delivered→completed", "* →cancelled (from non-terminal)"})
			return
		}
		if errors.Is(err, order.ErrInvalidTransition) {
			httpresponse.InvalidTransition(w, "order", "unknown", string(req.Status),
				[]string{"pending→confirmed", "confirmed→processing", "processing→shipped", "shipped→delivered", "delivered→completed", "* →cancelled (from non-terminal)"})
			return
		}
		if errors.Is(err, order.ErrConcurrentStatusUpdate) {
			httpresponse.ConflictError(w, httpresponse.LLMError{
				Code:       "concurrent_update",
				Message:    "order status was modified by another request; refresh and retry",
				Field:      "status",
				Suggestion: "Refetch order with orders_get and retry the transition based on its current status.",
			})
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
		httpresponse.MissingField(w, "id (path)",
			"order UUID in path",
			"Specify the order ID in the URL path: POST /api/v1/orders/{id}/cancel")
		return
	}

	var req CancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.InvalidBody(w, err.Error())
		return
	}

	if req.Reason == "" {
		httpresponse.MissingField(w, "reason",
			"non-empty string explaining the cancellation",
			"Cancellation must include a reason for audit purposes.",
			"customer requested cancellation", "out of stock", "fraud suspected", "wrong address")
		return
	}

	cancelled, err := c.svc.CancelOrder(r.Context(), id, req.Reason)
	if err != nil {
		if errors.Is(err, order.ErrOrderNotFound) {
			httpresponse.NotFoundError(w, httpresponse.LLMError{
				Message:    "order with id '" + id + "' was not found",
				Field:      "id",
				Received:   id,
				Suggestion: "Verify the order exists using orders_list or orders_search.",
			})
			return
		}
		if errors.Is(err, order.ErrInvalidTransition) {
			httpresponse.ConflictError(w, httpresponse.LLMError{
				Code:       "cannot_cancel",
				Message:    "order cannot be cancelled in its current status (likely already delivered/completed/returned/cancelled)",
				Field:      "status",
				Suggestion: "Fetch the order with orders_get to see its current status. Terminal statuses (completed, returned, cancelled) cannot be cancelled.",
			})
			return
		}
		if errors.Is(err, order.ErrConcurrentStatusUpdate) {
			httpresponse.ConflictError(w, httpresponse.LLMError{
				Code:       "concurrent_update",
				Message:    "order status was modified by another request; refresh and retry",
				Field:      "status",
				Suggestion: "Refetch order with orders_get and retry the cancel based on its current status.",
			})
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
		DryRun   bool     `json:"dry_run"`
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

	result, err := c.svc.BulkUpdateStatus(r.Context(), req.OrderIDs, order.Status(req.Status), req.Note, req.DryRun)
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

func parseFilterStrict(w http.ResponseWriter, r *http.Request) (order.Filter, bool) {
	var filter order.Filter

	if s := r.URL.Query().Get("status"); s != "" {
		status := order.Status(s)
		if !order.IsValidStatus(status) {
			httpresponse.InvalidField(w, "status",
				"one of: "+strings.Join(order.AllStatuses(), ", "), s,
				"Use one of the allowed order statuses.",
				order.AllStatuses()...)
			return order.Filter{}, false
		}
		filter.Status = &status
	}

	from, fromSet, fromErr := httpresponse.ParseDateFromQuery(r, "date_from")
	if fromErr != nil {
		httpresponse.WriteDateParseError(w, fromErr.(*httpresponse.DateParseError))
		return order.Filter{}, false
	}
	if fromSet {
		filter.DateFrom = &from
	}
	to, toSet, toErr := httpresponse.ParseDateFromQuery(r, "date_to")
	if toErr != nil {
		httpresponse.WriteDateParseError(w, toErr.(*httpresponse.DateParseError))
		return order.Filter{}, false
	}
	if toSet {
		filter.DateTo = &to
	}

	if s := r.URL.Query().Get("customer_name"); s != "" {
		filter.CustomerName = &s
	}

	return filter, true
}
