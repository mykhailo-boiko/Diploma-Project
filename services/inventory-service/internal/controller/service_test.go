package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/product"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/stock"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/warehouse"
)

type mockProductStorage struct {
	mu       sync.Mutex
	products map[string]product.Product
	nextID   int
}

func newMockProductStorage() *mockProductStorage {
	return &mockProductStorage{products: make(map[string]product.Product)}
}

func (m *mockProductStorage) CreateProduct(_ context.Context, p product.Product) (product.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.products {
		if existing.SKU == p.SKU {
			return product.Product{}, product.ErrSKUExists
		}
	}

	m.nextID++
	p.ID = fmt.Sprintf("prod-%d", m.nextID)
	m.products[p.ID] = p
	return p, nil
}

func (m *mockProductStorage) GetProductByID(_ context.Context, id string) (product.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.products[id]
	if !ok {
		return product.Product{}, product.ErrProductNotFound
	}
	return p, nil
}

func (m *mockProductStorage) ListProducts(_ context.Context, _ product.Filter, _ pagination.Sort, _ pagination.Page) ([]product.Product, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []product.Product
	for _, p := range m.products {
		result = append(result, p)
	}
	return result, len(result), nil
}

func (m *mockProductStorage) UpdateProduct(_ context.Context, p product.Product) (product.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.products[p.ID]
	if !ok {
		return product.Product{}, product.ErrProductNotFound
	}
	existing.Name = p.Name
	existing.Description = p.Description
	existing.Category = p.Category
	existing.UnitPrice = p.UnitPrice
	m.products[p.ID] = existing
	return existing, nil
}

func (m *mockProductStorage) DeleteProduct(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.products[id]; !ok {
		return product.ErrProductNotFound
	}
	delete(m.products, id)
	return nil
}

type mockWarehouseStorage struct {
	mu         sync.Mutex
	warehouses map[string]warehouse.Warehouse
	nextID     int
}

func newMockWarehouseStorage() *mockWarehouseStorage {
	return &mockWarehouseStorage{warehouses: make(map[string]warehouse.Warehouse)}
}

func (m *mockWarehouseStorage) CreateWarehouse(_ context.Context, w warehouse.Warehouse) (warehouse.Warehouse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	w.ID = fmt.Sprintf("wh-%d", m.nextID)
	w.IsActive = true
	m.warehouses[w.ID] = w
	return w, nil
}

func (m *mockWarehouseStorage) GetWarehouseByID(_ context.Context, id string) (warehouse.Warehouse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w, ok := m.warehouses[id]
	if !ok {
		return warehouse.Warehouse{}, warehouse.ErrWarehouseNotFound
	}
	return w, nil
}

func (m *mockWarehouseStorage) ListWarehouses(_ context.Context, _ warehouse.Filter, _ pagination.Sort, _ pagination.Page) ([]warehouse.Warehouse, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []warehouse.Warehouse
	for _, w := range m.warehouses {
		result = append(result, w)
	}
	return result, len(result), nil
}

func (m *mockWarehouseStorage) UpdateWarehouse(_ context.Context, w warehouse.Warehouse) (warehouse.Warehouse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.warehouses[w.ID]
	if !ok {
		return warehouse.Warehouse{}, warehouse.ErrWarehouseNotFound
	}
	existing.Name = w.Name
	existing.Address = w.Address
	existing.IsActive = w.IsActive
	m.warehouses[w.ID] = existing
	return existing, nil
}

type mockStockStorage struct {
	mu        sync.Mutex
	stocks    map[string]*stock.Stock
	movements []stock.Movement
	nextID    int
}

func newMockStockStorage() *mockStockStorage {
	return &mockStockStorage{stocks: make(map[string]*stock.Stock)}
}

func stockKey(productID, warehouseID string) string {
	return productID + ":" + warehouseID
}

func (m *mockStockStorage) ListStock(_ context.Context, _ stock.Filter, _ pagination.Sort, _ pagination.Page) ([]stock.Stock, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []stock.Stock
	for _, s := range m.stocks {
		result = append(result, *s)
	}
	return result, len(result), nil
}

func (m *mockStockStorage) GetStockByProductAndWarehouse(_ context.Context, productID, warehouseID string) (stock.Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.stocks[stockKey(productID, warehouseID)]
	if !ok {
		return stock.Stock{}, stock.ErrStockNotFound
	}
	return *s, nil
}

func (m *mockStockStorage) GetOrCreateStock(_ context.Context, productID, warehouseID string) (stock.Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := stockKey(productID, warehouseID)
	if s, ok := m.stocks[key]; ok {
		return *s, nil
	}

	m.nextID++
	s := &stock.Stock{
		ID:          fmt.Sprintf("stock-%d", m.nextID),
		ProductID:   productID,
		WarehouseID: warehouseID,
	}
	m.stocks[key] = s
	return *s, nil
}

func (m *mockStockStorage) ReserveStock(_ context.Context, productID, warehouseID string, quantity int, _ string) (stock.Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if quantity <= 0 {
		return stock.Stock{}, stock.ErrInvalidQuantity
	}

	key := stockKey(productID, warehouseID)
	s, ok := m.stocks[key]
	if !ok {
		return stock.Stock{}, stock.ErrStockNotFound
	}

	available := s.Quantity - s.Reserved
	if available < quantity {
		return stock.Stock{}, stock.ErrInsufficientStock
	}

	s.Reserved += quantity
	s.Available = s.Quantity - s.Reserved
	return *s, nil
}

func (m *mockStockStorage) ReleaseStock(_ context.Context, productID, warehouseID string, quantity int, _ string) (stock.Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if quantity <= 0 {
		return stock.Stock{}, stock.ErrInvalidQuantity
	}

	key := stockKey(productID, warehouseID)
	s, ok := m.stocks[key]
	if !ok {
		return stock.Stock{}, stock.ErrStockNotFound
	}

	if s.Reserved == 0 {
		return stock.Stock{}, stock.ErrNothingToRelease
	}
	if quantity > s.Reserved {
		return stock.Stock{}, stock.ErrReleaseExceedsReserved
	}

	s.Reserved -= quantity
	s.Available = s.Quantity - s.Reserved
	return *s, nil
}

func (m *mockStockStorage) AdjustStock(_ context.Context, productID, warehouseID string, quantity int, movementType, _ string) (stock.Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if quantity <= 0 {
		return stock.Stock{}, stock.ErrInvalidQuantity
	}

	key := stockKey(productID, warehouseID)
	s, ok := m.stocks[key]
	if !ok {
		return stock.Stock{}, stock.ErrStockNotFound
	}

	switch movementType {
	case stock.MovementTypeOutbound:
		available := s.Quantity - s.Reserved
		if available < quantity {
			return stock.Stock{}, stock.ErrInsufficientStock
		}
		s.Quantity -= quantity
	case stock.MovementTypeInbound, stock.MovementTypeAdjustment:
		s.Quantity += quantity
	default:
		return stock.Stock{}, fmt.Errorf("invalid movement type: %s", movementType)
	}

	s.Available = s.Quantity - s.Reserved
	return *s, nil
}

func (m *mockStockStorage) ListMovements(_ context.Context, _ stock.MovementFilter, _ pagination.Sort, _ pagination.Page) ([]stock.Movement, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.movements, len(m.movements), nil
}

func (m *mockStockStorage) ListLowStock(_ context.Context, _ pagination.Page) ([]stock.LowStockItem, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var items []stock.LowStockItem
	for _, s := range m.stocks {
		if s.MinThreshold > 0 && s.Available < s.MinThreshold {
			items = append(items, stock.LowStockItem{Stock: *s})
		}
	}
	return items, len(items), nil
}

func (m *mockStockStorage) GetInventoryReport(_ context.Context) (stock.InventoryReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var report stock.InventoryReport
	for _, s := range m.stocks {
		report.TotalQuantity += s.Quantity
		report.TotalReserved += s.Reserved
		report.TotalAvailable += s.Available
	}
	report.ByWarehouse = []stock.WarehouseSummary{}
	report.ByCategory = []stock.CategorySummary{}
	return report, nil
}

func (m *mockStockStorage) UpdateMinThreshold(_ context.Context, productID, warehouseID string, threshold int) (stock.Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := stockKey(productID, warehouseID)
	s, ok := m.stocks[key]
	if !ok {
		return stock.Stock{}, stock.ErrStockNotFound
	}
	s.MinThreshold = threshold
	return *s, nil
}

func (m *mockStockStorage) seedStock(productID, warehouseID string, quantity, reserved, minThreshold int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	key := stockKey(productID, warehouseID)
	m.stocks[key] = &stock.Stock{
		ID:           fmt.Sprintf("stock-%d", m.nextID),
		ProductID:    productID,
		WarehouseID:  warehouseID,
		Quantity:     quantity,
		Reserved:     reserved,
		Available:    quantity - reserved,
		MinThreshold: minThreshold,
	}
}

func newTestService() (*Service, *mockProductStorage, *mockWarehouseStorage, *mockStockStorage) {
	ps := newMockProductStorage()
	ws := newMockWarehouseStorage()
	ss := newMockStockStorage()
	log := zap.NewNop()
	return NewService(ps, ws, ss, nil, log), ps, ws, ss
}

func TestCreateProduct(t *testing.T) {
	svc, _, _, _ := newTestService()

	created, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:       "WIDGET-001",
		Name:      "Widget",
		Category:  "Electronics",
		UnitPrice: 29.99,
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	if created.SKU != "WIDGET-001" {
		t.Errorf("expected SKU 'WIDGET-001', got %q", created.SKU)
	}
	if created.Name != "Widget" {
		t.Errorf("expected name 'Widget', got %q", created.Name)
	}
}

func TestCreateProduct_DuplicateSKU(t *testing.T) {
	svc, _, _, _ := newTestService()

	_, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "WIDGET-001",
		Name: "Widget",
	})
	if err != nil {
		t.Fatalf("first CreateProduct failed: %v", err)
	}

	_, err = svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "WIDGET-001",
		Name: "Widget Duplicate",
	})
	if !errors.Is(err, product.ErrSKUExists) {
		t.Errorf("expected ErrSKUExists, got %v", err)
	}
}

func TestGetProductByID(t *testing.T) {
	svc, _, _, _ := newTestService()

	created, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "TEST-001",
		Name: "Test Product",
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	fetched, err := svc.GetProductByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetProductByID failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, fetched.ID)
	}
}

func TestGetProductByID_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	_, err := svc.GetProductByID(t.Context(), "nonexistent")
	if !errors.Is(err, product.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestUpdateProduct(t *testing.T) {
	svc, _, _, _ := newTestService()

	created, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "UPD-001",
		Name: "Original",
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	updated, err := svc.UpdateProduct(t.Context(), created.ID, UpdateProductRequest{
		Name:      "Updated",
		Category:  "New Category",
		UnitPrice: 49.99,
	})
	if err != nil {
		t.Fatalf("UpdateProduct failed: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}
}

func TestDeleteProduct(t *testing.T) {
	svc, _, _, _ := newTestService()

	created, err := svc.CreateProduct(t.Context(), CreateProductRequest{
		SKU:  "DEL-001",
		Name: "To Delete",
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	if err := svc.DeleteProduct(t.Context(), created.ID); err != nil {
		t.Fatalf("DeleteProduct failed: %v", err)
	}

	_, err = svc.GetProductByID(t.Context(), created.ID)
	if !errors.Is(err, product.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound after delete, got %v", err)
	}
}

func TestDeleteProduct_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	err := svc.DeleteProduct(t.Context(), "nonexistent")
	if !errors.Is(err, product.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestListProducts(t *testing.T) {
	svc, _, _, _ := newTestService()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateProduct(t.Context(), CreateProductRequest{
			SKU:  fmt.Sprintf("LIST-%03d", i),
			Name: fmt.Sprintf("Product %d", i),
		})
		if err != nil {
			t.Fatalf("CreateProduct failed: %v", err)
		}
	}

	products, total, err := svc.ListProducts(t.Context(), product.Filter{}, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListProducts failed: %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(products) != 3 {
		t.Errorf("expected 3 products, got %d", len(products))
	}
}

func TestCreateWarehouse(t *testing.T) {
	svc, _, _, _ := newTestService()

	created, err := svc.CreateWarehouse(t.Context(), CreateWarehouseRequest{
		Name:    "Main Warehouse",
		Address: "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateWarehouse failed: %v", err)
	}

	if created.Name != "Main Warehouse" {
		t.Errorf("expected name 'Main Warehouse', got %q", created.Name)
	}
	if !created.IsActive {
		t.Error("expected new warehouse to be active")
	}
}

func TestGetWarehouseByID(t *testing.T) {
	svc, _, _, _ := newTestService()

	created, err := svc.CreateWarehouse(t.Context(), CreateWarehouseRequest{
		Name: "Test Warehouse",
	})
	if err != nil {
		t.Fatalf("CreateWarehouse failed: %v", err)
	}

	fetched, err := svc.GetWarehouseByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetWarehouseByID failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, fetched.ID)
	}
}

func TestGetWarehouseByID_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	_, err := svc.GetWarehouseByID(t.Context(), "nonexistent")
	if !errors.Is(err, warehouse.ErrWarehouseNotFound) {
		t.Errorf("expected ErrWarehouseNotFound, got %v", err)
	}
}

func TestUpdateWarehouse(t *testing.T) {
	svc, _, _, _ := newTestService()

	created, err := svc.CreateWarehouse(t.Context(), CreateWarehouseRequest{
		Name: "Original",
	})
	if err != nil {
		t.Fatalf("CreateWarehouse failed: %v", err)
	}

	updated, err := svc.UpdateWarehouse(t.Context(), created.ID, UpdateWarehouseRequest{
		Name:     "Updated",
		Address:  "New Address",
		IsActive: false,
	})
	if err != nil {
		t.Fatalf("UpdateWarehouse failed: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}
}

func TestListWarehouses(t *testing.T) {
	svc, _, _, _ := newTestService()

	for i := 0; i < 2; i++ {
		_, err := svc.CreateWarehouse(t.Context(), CreateWarehouseRequest{
			Name: fmt.Sprintf("Warehouse %d", i),
		})
		if err != nil {
			t.Fatalf("CreateWarehouse failed: %v", err)
		}
	}

	warehouses, total, err := svc.ListWarehouses(t.Context(), warehouse.Filter{}, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListWarehouses failed: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(warehouses) != 2 {
		t.Errorf("expected 2 warehouses, got %d", len(warehouses))
	}
}

func TestListStock_Empty(t *testing.T) {
	svc, _, _, _ := newTestService()

	stocks, total, err := svc.ListStock(t.Context(), stock.Filter{}, pagination.Sort{Field: "updated_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListStock failed: %v", err)
	}

	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(stocks) != 0 {
		t.Errorf("expected 0 stocks, got %d", len(stocks))
	}
}

func TestReserveStock_Success(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	result, err := svc.ReserveStock(t.Context(), ReserveStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    30,
		Reference:   "order-123",
	})
	if err != nil {
		t.Fatalf("ReserveStock failed: %v", err)
	}

	if result.Reserved != 30 {
		t.Errorf("expected reserved=30, got %d", result.Reserved)
	}
	if result.Available != 70 {
		t.Errorf("expected available=70, got %d", result.Available)
	}
}

func TestReserveStock_InsufficientStock(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	_, err := svc.ReserveStock(t.Context(), ReserveStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    150,
	})
	if !errors.Is(err, stock.ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestReserveStock_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	_, err := svc.ReserveStock(t.Context(), ReserveStockRequest{
		ProductID:   "nonexistent",
		WarehouseID: "wh-1",
		Quantity:    10,
	})
	if !errors.Is(err, stock.ErrStockNotFound) {
		t.Errorf("expected ErrStockNotFound, got %v", err)
	}
}

func TestReleaseStock_Success(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 30, 0)

	result, err := svc.ReleaseStock(t.Context(), ReleaseStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    30,
		Reference:   "order-123-cancel",
	})
	if err != nil {
		t.Fatalf("ReleaseStock failed: %v", err)
	}

	if result.Reserved != 0 {
		t.Errorf("expected reserved=0, got %d", result.Reserved)
	}
	if result.Available != 100 {
		t.Errorf("expected available=100, got %d", result.Available)
	}
}

func TestReleaseStock_ExceedsReserved(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 10, 0)

	_, err := svc.ReleaseStock(t.Context(), ReleaseStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    20,
	})
	if !errors.Is(err, stock.ErrReleaseExceedsReserved) {
		t.Errorf("expected ErrReleaseExceedsReserved, got %v", err)
	}
}

func TestReleaseStock_NothingToRelease(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	_, err := svc.ReleaseStock(t.Context(), ReleaseStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    10,
	})
	if !errors.Is(err, stock.ErrNothingToRelease) {
		t.Errorf("expected ErrNothingToRelease, got %v", err)
	}
}

func TestAdjustStock_Inbound(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	result, err := svc.AdjustStock(t.Context(), AdjustStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    50,
		Type:        stock.MovementTypeInbound,
		Reference:   "shipment-456",
	})
	if err != nil {
		t.Fatalf("AdjustStock inbound failed: %v", err)
	}

	if result.Quantity != 150 {
		t.Errorf("expected quantity=150, got %d", result.Quantity)
	}
}

func TestAdjustStock_Outbound(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	result, err := svc.AdjustStock(t.Context(), AdjustStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    30,
		Type:        stock.MovementTypeOutbound,
		Reference:   "shipment-789",
	})
	if err != nil {
		t.Fatalf("AdjustStock outbound failed: %v", err)
	}

	if result.Quantity != 70 {
		t.Errorf("expected quantity=70, got %d", result.Quantity)
	}
}

func TestAdjustStock_Outbound_Insufficient(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 50, 30, 0)

	_, err := svc.AdjustStock(t.Context(), AdjustStockRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Quantity:    30,
		Type:        stock.MovementTypeOutbound,
	})
	if !errors.Is(err, stock.ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestListLowStock(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 10, 0, 200)
	ss.seedStock("prod-2", "wh-1", 500, 0, 100)

	items, total, err := svc.ListLowStock(t.Context(), pagination.Page{Limit: 20})
	if err != nil {
		t.Fatalf("ListLowStock failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 low stock item, got %d", total)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestGetInventoryReport(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 20, 0)
	ss.seedStock("prod-2", "wh-1", 50, 10, 0)

	report, err := svc.GetInventoryReport(t.Context())
	if err != nil {
		t.Fatalf("GetInventoryReport failed: %v", err)
	}

	if report.TotalQuantity != 150 {
		t.Errorf("expected total_quantity=150, got %d", report.TotalQuantity)
	}
	if report.TotalReserved != 30 {
		t.Errorf("expected total_reserved=30, got %d", report.TotalReserved)
	}
	if report.TotalAvailable != 120 {
		t.Errorf("expected total_available=120, got %d", report.TotalAvailable)
	}
}

func TestUpdateMinThreshold(t *testing.T) {
	svc, _, _, ss := newTestService()
	ss.seedStock("prod-1", "wh-1", 100, 0, 0)

	result, err := svc.UpdateMinThreshold(t.Context(), UpdateMinThresholdRequest{
		ProductID:   "prod-1",
		WarehouseID: "wh-1",
		Threshold:   50,
	})
	if err != nil {
		t.Fatalf("UpdateMinThreshold failed: %v", err)
	}

	if result.MinThreshold != 50 {
		t.Errorf("expected min_threshold=50, got %d", result.MinThreshold)
	}
}

func TestUpdateMinThreshold_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	_, err := svc.UpdateMinThreshold(t.Context(), UpdateMinThresholdRequest{
		ProductID:   "nonexistent",
		WarehouseID: "wh-1",
		Threshold:   50,
	})
	if !errors.Is(err, stock.ErrStockNotFound) {
		t.Errorf("expected ErrStockNotFound, got %v", err)
	}
}
