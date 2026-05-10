package stock

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

const (
	MovementTypeReserve    = "reserve"
	MovementTypeRelease    = "release"
	MovementTypeInbound    = "inbound"
	MovementTypeOutbound   = "outbound"
	MovementTypeAdjustment = "adjustment"
)

type Stock struct {
	ID           string    `json:"id"`
	ProductID    string    `json:"product_id"`
	WarehouseID  string    `json:"warehouse_id"`
	Quantity     int       `json:"quantity"`
	Reserved     int       `json:"reserved"`
	Available    int       `json:"available"`
	MinThreshold int       `json:"min_threshold"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Movement struct {
	ID          string    `json:"id"`
	StockID     string    `json:"stock_id"`
	ProductID   string    `json:"product_id"`
	WarehouseID string    `json:"warehouse_id"`
	Type        string    `json:"type"`
	Quantity    int       `json:"quantity"`
	Reference   string    `json:"reference"`
	CreatedAt   time.Time `json:"created_at"`
}

type Filter struct {
	ProductID   *string
	WarehouseID *string
}

type MovementFilter struct {
	StockID     *string
	ProductID   *string
	WarehouseID *string
	Type        *string
	DateFrom    *time.Time
	DateTo      *time.Time
}

type LowStockItem struct {
	Stock
	ProductName   string `json:"product_name"`
	ProductSKU    string `json:"product_sku"`
	WarehouseName string `json:"warehouse_name"`
}

type InventoryReport struct {
	TotalProducts   int               `json:"total_products"`
	TotalWarehouses int               `json:"total_warehouses"`
	TotalQuantity   int               `json:"total_quantity"`
	TotalReserved   int               `json:"total_reserved"`
	TotalAvailable  int               `json:"total_available"`
	LowStockCount   int               `json:"low_stock_count"`
	ByWarehouse     []WarehouseSummary `json:"by_warehouse"`
	ByCategory      []CategorySummary  `json:"by_category"`
}

type WarehouseSummary struct {
	WarehouseID   string `json:"warehouse_id"`
	WarehouseName string `json:"warehouse_name"`
	TotalQuantity int    `json:"total_quantity"`
	TotalReserved int    `json:"total_reserved"`
	TotalAvailable int   `json:"total_available"`
	ProductCount  int    `json:"product_count"`
}

type CategorySummary struct {
	Category       string `json:"category"`
	TotalQuantity  int    `json:"total_quantity"`
	TotalReserved  int    `json:"total_reserved"`
	TotalAvailable int    `json:"total_available"`
	ProductCount   int    `json:"product_count"`
}

var (
	ErrStockNotFound       = errors.New("stock record not found")
	ErrInsufficientStock   = errors.New("insufficient available stock")
	ErrInvalidQuantity     = errors.New("quantity must be greater than zero")
	ErrNothingToRelease    = errors.New("no reserved stock to release")
	ErrReleaseExceedsReserved = errors.New("release quantity exceeds reserved amount")
)

type Storage interface {
	ListStock(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Stock, int, error)
	GetStockByProductAndWarehouse(ctx context.Context, productID, warehouseID string) (Stock, error)
	GetOrCreateStock(ctx context.Context, productID, warehouseID string) (Stock, error)
	ReserveStock(ctx context.Context, productID, warehouseID string, quantity int, reference string) (Stock, error)
	ReleaseStock(ctx context.Context, productID, warehouseID string, quantity int, reference string) (Stock, error)
	AdjustStock(ctx context.Context, productID, warehouseID string, quantity int, movementType, reference string) (Stock, error)
	ListMovements(ctx context.Context, filter MovementFilter, sort pagination.Sort, page pagination.Page) ([]Movement, int, error)
	ListLowStock(ctx context.Context, page pagination.Page) ([]LowStockItem, int, error)
	GetInventoryReport(ctx context.Context) (InventoryReport, error)
	UpdateMinThreshold(ctx context.Context, productID, warehouseID string, threshold int) (Stock, error)
}
