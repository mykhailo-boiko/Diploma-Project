package controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/product"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/stock"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/warehouse"
)

type Service struct {
	products   product.Storage
	warehouses warehouse.Storage
	stocks     stock.Storage
	nc         *natspkg.Client
	log        *zap.Logger
}

func NewService(products product.Storage, warehouses warehouse.Storage, stocks stock.Storage, nc *natspkg.Client, log *zap.Logger) *Service {
	return &Service{products: products, warehouses: warehouses, stocks: stocks, nc: nc, log: log}
}


type CreateProductRequest struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	UnitPrice   float64 `json:"unit_price"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	UnitPrice   float64 `json:"unit_price"`
}


type CreateWarehouseRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type UpdateWarehouseRequest struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	IsActive bool   `json:"is_active"`
}


func (s *Service) CreateProduct(ctx context.Context, req CreateProductRequest) (product.Product, error) {
	p := product.Product{
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		UnitPrice:   req.UnitPrice,
	}

	created, err := s.products.CreateProduct(ctx, p)
	if err != nil {
		return product.Product{}, fmt.Errorf("failed to create product: %w", err)
	}

	return created, nil
}

func (s *Service) GetProductByID(ctx context.Context, id string) (product.Product, error) {
	return s.products.GetProductByID(ctx, id)
}

func (s *Service) ListProducts(ctx context.Context, filter product.Filter, sort pagination.Sort, page pagination.Page) ([]product.Product, int, error) {
	return s.products.ListProducts(ctx, filter, sort, page)
}

func (s *Service) UpdateProduct(ctx context.Context, id string, req UpdateProductRequest) (product.Product, error) {
	p := product.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		UnitPrice:   req.UnitPrice,
	}

	updated, err := s.products.UpdateProduct(ctx, p)
	if err != nil {
		return product.Product{}, fmt.Errorf("failed to update product: %w", err)
	}

	return updated, nil
}

func (s *Service) DeleteProduct(ctx context.Context, id string) error {
	return s.products.DeleteProduct(ctx, id)
}


func (s *Service) CreateWarehouse(ctx context.Context, req CreateWarehouseRequest) (warehouse.Warehouse, error) {
	w := warehouse.Warehouse{
		Name:    req.Name,
		Address: req.Address,
	}

	created, err := s.warehouses.CreateWarehouse(ctx, w)
	if err != nil {
		return warehouse.Warehouse{}, fmt.Errorf("failed to create warehouse: %w", err)
	}

	return created, nil
}

func (s *Service) GetWarehouseByID(ctx context.Context, id string) (warehouse.Warehouse, error) {
	return s.warehouses.GetWarehouseByID(ctx, id)
}

func (s *Service) ListWarehouses(ctx context.Context, filter warehouse.Filter, sort pagination.Sort, page pagination.Page) ([]warehouse.Warehouse, int, error) {
	return s.warehouses.ListWarehouses(ctx, filter, sort, page)
}

func (s *Service) UpdateWarehouse(ctx context.Context, id string, req UpdateWarehouseRequest) (warehouse.Warehouse, error) {
	w := warehouse.Warehouse{
		ID:       id,
		Name:     req.Name,
		Address:  req.Address,
		IsActive: req.IsActive,
	}

	updated, err := s.warehouses.UpdateWarehouse(ctx, w)
	if err != nil {
		return warehouse.Warehouse{}, fmt.Errorf("failed to update warehouse: %w", err)
	}

	return updated, nil
}


type ReserveStockRequest struct {
	ProductID   string `json:"product_id"`
	WarehouseID string `json:"warehouse_id"`
	Quantity    int    `json:"quantity"`
	Reference   string `json:"reference"`
}

type ReleaseStockRequest struct {
	ProductID   string `json:"product_id"`
	WarehouseID string `json:"warehouse_id"`
	Quantity    int    `json:"quantity"`
	Reference   string `json:"reference"`
}

type AdjustStockRequest struct {
	ProductID   string `json:"product_id"`
	WarehouseID string `json:"warehouse_id"`
	Quantity    int    `json:"quantity"`
	Type        string `json:"type"`
	Reference   string `json:"reference"`
}

type UpdateMinThresholdRequest struct {
	ProductID   string `json:"product_id"`
	WarehouseID string `json:"warehouse_id"`
	Threshold   int    `json:"threshold"`
}

func (s *Service) ListStock(ctx context.Context, filter stock.Filter, sort pagination.Sort, page pagination.Page) ([]stock.Stock, int, error) {
	return s.stocks.ListStock(ctx, filter, sort, page)
}

func (s *Service) ReserveStock(ctx context.Context, req ReserveStockRequest) (stock.Stock, error) {
	st, err := s.stocks.ReserveStock(ctx, req.ProductID, req.WarehouseID, req.Quantity, req.Reference)
	if err != nil {
		return stock.Stock{}, fmt.Errorf("failed to reserve stock: %w", err)
	}

	s.publishStockChanged("reserve", st)
	s.checkLowStock(st)

	return st, nil
}

func (s *Service) ReleaseStock(ctx context.Context, req ReleaseStockRequest) (stock.Stock, error) {
	st, err := s.stocks.ReleaseStock(ctx, req.ProductID, req.WarehouseID, req.Quantity, req.Reference)
	if err != nil {
		return stock.Stock{}, fmt.Errorf("failed to release stock: %w", err)
	}

	s.publishStockChanged("release", st)

	return st, nil
}

func (s *Service) AdjustStock(ctx context.Context, req AdjustStockRequest) (stock.Stock, error) {
	st, err := s.stocks.AdjustStock(ctx, req.ProductID, req.WarehouseID, req.Quantity, req.Type, req.Reference)
	if err != nil {
		return stock.Stock{}, fmt.Errorf("failed to adjust stock: %w", err)
	}

	s.publishStockChanged(req.Type, st)
	s.checkLowStock(st)

	return st, nil
}

func (s *Service) ListMovements(ctx context.Context, filter stock.MovementFilter, sort pagination.Sort, page pagination.Page) ([]stock.Movement, int, error) {
	return s.stocks.ListMovements(ctx, filter, sort, page)
}

func (s *Service) ListLowStock(ctx context.Context, page pagination.Page) ([]stock.LowStockItem, int, error) {
	return s.stocks.ListLowStock(ctx, page)
}

func (s *Service) GetInventoryReport(ctx context.Context) (stock.InventoryReport, error) {
	return s.stocks.GetInventoryReport(ctx)
}

func (s *Service) UpdateMinThreshold(ctx context.Context, req UpdateMinThresholdRequest) (stock.Stock, error) {
	st, err := s.stocks.UpdateMinThreshold(ctx, req.ProductID, req.WarehouseID, req.Threshold)
	if err != nil {
		return stock.Stock{}, fmt.Errorf("failed to update min_threshold: %w", err)
	}

	s.checkLowStock(st)

	return st, nil
}

func (s *Service) publishStockChanged(operation string, st stock.Stock) {
	if s.nc == nil {
		return
	}

	data := map[string]any{
		"operation":    operation,
		"product_id":   st.ProductID,
		"warehouse_id": st.WarehouseID,
		"quantity":     st.Quantity,
		"reserved":     st.Reserved,
		"available":    st.Available,
	}

	if err := s.nc.Publish("inventory.stock_changed", "inventory.stock_changed", data); err != nil {
		s.log.Error("Failed to publish inventory.stock_changed", zap.String("product_id", st.ProductID), zap.Error(err))
	}
}

func (s *Service) checkLowStock(st stock.Stock) {
	if st.MinThreshold > 0 && st.Available < st.MinThreshold {
		if s.nc == nil {
			return
		}

		data := map[string]any{
			"product_id":   st.ProductID,
			"warehouse_id": st.WarehouseID,
			"available":    st.Available,
			"min_threshold": st.MinThreshold,
		}

		if err := s.nc.Publish("inventory.low_stock", "inventory.low_stock", data); err != nil {
			s.log.Error("Failed to publish inventory.low_stock", zap.String("product_id", st.ProductID), zap.Error(err))
		}
	}
}
