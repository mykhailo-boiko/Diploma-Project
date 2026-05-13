package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/product"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/stock"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/warehouse"
)

type InventoryService interface {
	CreateProduct(ctx context.Context, req CreateProductRequest) (product.Product, error)
	GetProductByID(ctx context.Context, id string) (product.Product, error)
	ListProducts(ctx context.Context, filter product.Filter, sort pagination.Sort, page pagination.Page) ([]product.Product, int, error)
	UpdateProduct(ctx context.Context, id string, req UpdateProductRequest) (product.Product, error)
	DeleteProduct(ctx context.Context, id string) error

	CreateWarehouse(ctx context.Context, req CreateWarehouseRequest) (warehouse.Warehouse, error)
	GetWarehouseByID(ctx context.Context, id string) (warehouse.Warehouse, error)
	ListWarehouses(ctx context.Context, filter warehouse.Filter, sort pagination.Sort, page pagination.Page) ([]warehouse.Warehouse, int, error)
	UpdateWarehouse(ctx context.Context, id string, req UpdateWarehouseRequest) (warehouse.Warehouse, error)

	ListStock(ctx context.Context, filter stock.Filter, sort pagination.Sort, page pagination.Page) ([]stock.Stock, int, error)
	ReserveStock(ctx context.Context, req ReserveStockRequest) (stock.Stock, error)
	ReleaseStock(ctx context.Context, req ReleaseStockRequest) (stock.Stock, error)
	AdjustStock(ctx context.Context, req AdjustStockRequest) (stock.Stock, error)
	ListMovements(ctx context.Context, filter stock.MovementFilter, sort pagination.Sort, page pagination.Page) ([]stock.Movement, int, error)
	ListLowStock(ctx context.Context, page pagination.Page) ([]stock.LowStockItem, int, error)
	GetInventoryReport(ctx context.Context) (stock.InventoryReport, error)
	UpdateMinThreshold(ctx context.Context, req UpdateMinThresholdRequest) (stock.Stock, error)
}

type InventoryController struct {
	svc InventoryService
	log *zap.Logger
}

func NewInventoryController(svc InventoryService, log *zap.Logger) *InventoryController {
	return &InventoryController{svc: svc, log: log}
}

var productSortFields = map[string]bool{
	"created_at": true,
	"name":       true,
	"sku":        true,
	"category":   true,
	"unit_price": true,
}

var warehouseSortFields = map[string]bool{
	"created_at": true,
	"name":       true,
}

var stockSortFields = map[string]bool{
	"updated_at":   true,
	"product_id":   true,
	"warehouse_id": true,
	"quantity":     true,
	"available":    true,
}

var movementSortFields = map[string]bool{
	"created_at": true,
	"type":       true,
	"quantity":   true,
}

func (c *InventoryController) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.SKU == "" {
		httpresponse.BadRequest(w, "validation_error", "sku is required")
		return
	}
	if req.Name == "" {
		httpresponse.BadRequest(w, "validation_error", "name is required")
		return
	}

	created, err := c.svc.CreateProduct(r.Context(), req)
	if err != nil {
		if errors.Is(err, product.ErrSKUExists) {
			httpresponse.BadRequest(w, "sku_exists", "product with this SKU already exists")
			return
		}
		c.log.Error("Failed to create product", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *InventoryController) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := httpresponse.ValidateUUIDPath(w, r, "id")
	if !ok {
		return
	}

	p, err := c.svc.GetProductByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, product.ErrProductNotFound) {
			httpresponse.NotFound(w, "product_not_found", "product not found")
			return
		}
		c.log.Error("Failed to get product", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, p)
}

func (c *InventoryController) ListProducts(w http.ResponseWriter, r *http.Request) {
	filter := parseProductFilter(r)
	sort := pagination.SortFromRequest(r, productSortFields, "created_at")
	page := pagination.PageFromRequest(r)

	products, total, err := c.svc.ListProducts(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list products", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if products == nil {
		products = []product.Product{}
	}

	httpresponse.List(w, products, total, page.Limit, page.Offset)
}

func (c *InventoryController) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "product id is required")
		return
	}

	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Name == "" {
		httpresponse.BadRequest(w, "validation_error", "name is required")
		return
	}

	updated, err := c.svc.UpdateProduct(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, product.ErrProductNotFound) {
			httpresponse.NotFound(w, "product_not_found", "product not found")
			return
		}
		c.log.Error("Failed to update product", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *InventoryController) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "product id is required")
		return
	}

	if err := c.svc.DeleteProduct(r.Context(), id); err != nil {
		if errors.Is(err, product.ErrProductNotFound) {
			httpresponse.NotFound(w, "product_not_found", "product not found")
			return
		}
		c.log.Error("Failed to delete product", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, map[string]string{"status": "deleted"})
}

func (c *InventoryController) CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Name == "" {
		httpresponse.BadRequest(w, "validation_error", "name is required")
		return
	}

	created, err := c.svc.CreateWarehouse(r.Context(), req)
	if err != nil {
		c.log.Error("Failed to create warehouse", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.Created(w, created)
}

func (c *InventoryController) GetWarehouse(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "warehouse id is required")
		return
	}

	wh, err := c.svc.GetWarehouseByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, warehouse.ErrWarehouseNotFound) {
			httpresponse.NotFound(w, "warehouse_not_found", "warehouse not found")
			return
		}
		c.log.Error("Failed to get warehouse", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, wh)
}

func (c *InventoryController) ListWarehouses(w http.ResponseWriter, r *http.Request) {
	filter := parseWarehouseFilter(r)
	sort := pagination.SortFromRequest(r, warehouseSortFields, "created_at")
	page := pagination.PageFromRequest(r)

	warehouses, total, err := c.svc.ListWarehouses(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list warehouses", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if warehouses == nil {
		warehouses = []warehouse.Warehouse{}
	}

	httpresponse.List(w, warehouses, total, page.Limit, page.Offset)
}

func (c *InventoryController) UpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.BadRequest(w, "validation_error", "warehouse id is required")
		return
	}

	var req UpdateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.Name == "" {
		httpresponse.BadRequest(w, "validation_error", "name is required")
		return
	}

	updated, err := c.svc.UpdateWarehouse(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, warehouse.ErrWarehouseNotFound) {
			httpresponse.NotFound(w, "warehouse_not_found", "warehouse not found")
			return
		}
		c.log.Error("Failed to update warehouse", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, updated)
}

func (c *InventoryController) ListStock(w http.ResponseWriter, r *http.Request) {
	filter := parseStockFilter(r)
	sort := pagination.SortFromRequest(r, stockSortFields, "updated_at")
	page := pagination.PageFromRequest(r)

	stocks, total, err := c.svc.ListStock(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list stock", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if stocks == nil {
		stocks = []stock.Stock{}
	}

	httpresponse.List(w, stocks, total, page.Limit, page.Offset)
}

func (c *InventoryController) ReserveStock(w http.ResponseWriter, r *http.Request) {
	var req ReserveStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.ProductID == "" || req.WarehouseID == "" {
		httpresponse.BadRequest(w, "validation_error", "product_id and warehouse_id are required")
		return
	}
	if req.Quantity <= 0 {
		httpresponse.BadRequest(w, "validation_error", "quantity must be greater than zero")
		return
	}

	result, err := c.svc.ReserveStock(r.Context(), req)
	if err != nil {
		if errors.Is(err, stock.ErrStockNotFound) {
			httpresponse.NotFound(w, "stock_not_found", "stock record not found")
			return
		}
		if errors.Is(err, stock.ErrInsufficientStock) {
			httpresponse.BadRequest(w, "insufficient_stock", "insufficient available stock for reservation")
			return
		}
		if errors.Is(err, stock.ErrInvalidQuantity) {
			httpresponse.BadRequest(w, "validation_error", "quantity must be greater than zero")
			return
		}
		c.log.Error("Failed to reserve stock", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, result)
}

func (c *InventoryController) ReleaseStock(w http.ResponseWriter, r *http.Request) {
	var req ReleaseStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}

	if req.ProductID == "" || req.WarehouseID == "" {
		httpresponse.BadRequest(w, "validation_error", "product_id and warehouse_id are required")
		return
	}
	if req.Quantity <= 0 {
		httpresponse.BadRequest(w, "validation_error", "quantity must be greater than zero")
		return
	}

	result, err := c.svc.ReleaseStock(r.Context(), req)
	if err != nil {
		if errors.Is(err, stock.ErrStockNotFound) {
			httpresponse.NotFound(w, "stock_not_found", "stock record not found")
			return
		}
		if errors.Is(err, stock.ErrNothingToRelease) {
			httpresponse.BadRequest(w, "nothing_to_release", "no reserved stock to release")
			return
		}
		if errors.Is(err, stock.ErrReleaseExceedsReserved) {
			httpresponse.BadRequest(w, "release_exceeds_reserved", "release quantity exceeds reserved amount")
			return
		}
		if errors.Is(err, stock.ErrInvalidQuantity) {
			httpresponse.BadRequest(w, "validation_error", "quantity must be greater than zero")
			return
		}
		c.log.Error("Failed to release stock", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, result)
}

func (c *InventoryController) AdjustStock(w http.ResponseWriter, r *http.Request) {
	var req AdjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.InvalidBody(w, err.Error())
		return
	}

	if req.ProductID == "" {
		httpresponse.MissingField(w, "product_id",
			"product UUID string",
			"List products via products_list to find a valid product_id.")
		return
	}
	if req.WarehouseID == "" {
		httpresponse.MissingField(w, "warehouse_id",
			"warehouse UUID string",
			"List warehouses via warehouses_list to find a valid warehouse_id.")
		return
	}
	if req.Quantity <= 0 {
		httpresponse.InvalidField(w, "quantity",
			"positive integer (> 0)", req.Quantity,
			"Quantity is always positive — the 'type' field determines if it increases (inbound) or decreases (outbound) stock. Use type='outbound' instead of negative quantity.",
			"1", "5", "100")
		return
	}

	validTypes := map[string]bool{
		stock.MovementTypeInbound:    true,
		stock.MovementTypeOutbound:   true,
		stock.MovementTypeAdjustment: true,
	}
	if !validTypes[req.Type] {
		httpresponse.InvalidField(w, "type",
			"one of: inbound, outbound, adjustment", req.Type,
			"Use 'inbound' for restock/supply (increases quantity), 'outbound' for sales/damage (decreases quantity), 'adjustment' for inventory corrections.",
			"inbound", "outbound", "adjustment")
		return
	}

	if req.Reason == "" && (req.Type == stock.MovementTypeOutbound || req.Type == stock.MovementTypeAdjustment) {
		httpresponse.MissingField(w, "reason",
			"non-empty string explaining the adjustment",
			"Outbound and adjustment movements must record a reason for the audit trail.",
			"customer order", "damage write-off", "physical count correction")
		return
	}

	result, err := c.svc.AdjustStock(r.Context(), req)
	if err != nil {
		if errors.Is(err, stock.ErrStockNotFound) {
			httpresponse.NotFoundError(w, httpresponse.LLMError{
				Code:       "stock_not_found",
				Message:    "stock record for this product+warehouse pair was not found",
				Suggestion: "Stock records exist only for combinations that have ever had stock. Check stock_list with both product_id and warehouse_id filters to verify the pair exists.",
			})
			return
		}
		if errors.Is(err, stock.ErrInsufficientStock) {
			httpresponse.ConflictError(w, httpresponse.LLMError{
				Code:       "insufficient_stock",
				Message:    "not enough available stock to perform this outbound adjustment",
				Field:      "quantity",
				Suggestion: "Check current available stock via stock_list. Available = quantity - reserved. Reduce the quantity or restock first.",
			})
			return
		}
		if errors.Is(err, stock.ErrInvalidQuantity) {
			httpresponse.InvalidField(w, "quantity",
				"positive integer (> 0)", req.Quantity,
				"Quantity must be greater than zero.")
			return
		}
		c.log.Error("Failed to adjust stock", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, result)
}

func (c *InventoryController) ListMovements(w http.ResponseWriter, r *http.Request) {
	filter := parseMovementFilter(r)
	sort := pagination.SortFromRequest(r, movementSortFields, "created_at")
	page := pagination.PageFromRequest(r)

	movements, total, err := c.svc.ListMovements(r.Context(), filter, sort, page)
	if err != nil {
		c.log.Error("Failed to list movements", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if movements == nil {
		movements = []stock.Movement{}
	}

	httpresponse.List(w, movements, total, page.Limit, page.Offset)
}

func (c *InventoryController) ListLowStock(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageFromRequest(r)

	items, total, err := c.svc.ListLowStock(r.Context(), page)
	if err != nil {
		c.log.Error("Failed to list low stock", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if items == nil {
		items = []stock.LowStockItem{}
	}

	httpresponse.List(w, items, total, page.Limit, page.Offset)
}

func (c *InventoryController) GetInventoryReport(w http.ResponseWriter, r *http.Request) {
	report, err := c.svc.GetInventoryReport(r.Context())
	if err != nil {
		c.log.Error("Failed to get inventory report", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, report)
}

func (c *InventoryController) UpdateMinThreshold(w http.ResponseWriter, r *http.Request) {
	var req UpdateMinThresholdRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body: "+err.Error())
		return
	}

	if req.ProductID == "" || req.WarehouseID == "" {
		httpresponse.BadRequest(w, "validation_error", "product_id and warehouse_id are required")
		return
	}
	value, ok := req.ResolvedThreshold()
	if !ok {
		httpresponse.MissingField(w, "min_threshold",
			"non-negative integer",
			"Provide 'min_threshold' (preferred) or legacy 'threshold' field.",
			"0", "10", "50")
		return
	}
	if value < 0 {
		httpresponse.InvalidField(w, "min_threshold",
			"non-negative integer (>= 0)", value,
			"min_threshold cannot be negative. Use 0 to disable the threshold.",
			"0", "10", "50")
		return
	}

	result, err := c.svc.UpdateMinThreshold(r.Context(), req)
	if err != nil {
		if errors.Is(err, stock.ErrStockNotFound) {
			httpresponse.NotFound(w, "stock_not_found", "stock record not found")
			return
		}
		c.log.Error("Failed to update min threshold", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, result)
}

func parseProductFilter(r *http.Request) product.Filter {
	var filter product.Filter

	if s := r.URL.Query().Get("sku"); s != "" {
		filter.SKU = &s
	}
	if s := r.URL.Query().Get("name"); s != "" {
		filter.Name = &s
	}
	if s := r.URL.Query().Get("category"); s != "" {
		filter.Category = &s
	}

	return filter
}

func parseWarehouseFilter(r *http.Request) warehouse.Filter {
	var filter warehouse.Filter

	if s := r.URL.Query().Get("name"); s != "" {
		filter.Name = &s
	}

	return filter
}

func parseStockFilter(r *http.Request) stock.Filter {
	var filter stock.Filter

	if s := r.URL.Query().Get("product_id"); s != "" {
		filter.ProductID = &s
	}
	if s := r.URL.Query().Get("warehouse_id"); s != "" {
		filter.WarehouseID = &s
	}

	return filter
}

func parseMovementFilter(r *http.Request) stock.MovementFilter {
	var filter stock.MovementFilter

	if s := r.URL.Query().Get("stock_id"); s != "" {
		filter.StockID = &s
	}
	if s := r.URL.Query().Get("product_id"); s != "" {
		filter.ProductID = &s
	}
	if s := r.URL.Query().Get("warehouse_id"); s != "" {
		filter.WarehouseID = &s
	}
	if s := r.URL.Query().Get("type"); s != "" {
		filter.Type = &s
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

	return filter
}
