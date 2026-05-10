package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/order-service/internal/order"
)

type mockStorage struct {
	mu     sync.Mutex
	orders map[string]order.Order
	nextID int
}

func newMockStorage() *mockStorage {
	return &mockStorage{orders: make(map[string]order.Order)}
}

func (m *mockStorage) CreateOrder(_ context.Context, o order.Order) (order.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	o.ID = fmt.Sprintf("order-%d", m.nextID)
	o.Status = order.StatusPending

	var total float64
	for i := range o.Items {
		o.Items[i].ID = fmt.Sprintf("item-%d-%d", m.nextID, i)
		o.Items[i].OrderID = o.ID
		o.Items[i].Subtotal = float64(o.Items[i].Quantity) * o.Items[i].UnitPrice
		total += o.Items[i].Subtotal
	}
	o.TotalAmount = total
	m.orders[o.ID] = o
	return o, nil
}

func (m *mockStorage) GetOrderByID(_ context.Context, id string) (order.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	o, ok := m.orders[id]
	if !ok {
		return order.Order{}, order.ErrOrderNotFound
	}
	return o, nil
}

func (m *mockStorage) ListOrders(_ context.Context, _ order.Filter, _ pagination.Sort, _ pagination.Page) ([]order.Order, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []order.Order
	for _, o := range m.orders {
		result = append(result, o)
	}
	return result, len(result), nil
}

func (m *mockStorage) UpdateOrderStatus(_ context.Context, id string, status order.Status) (order.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	o, ok := m.orders[id]
	if !ok {
		return order.Order{}, order.ErrOrderNotFound
	}
	o.Status = status
	m.orders[id] = o
	return o, nil
}

func (m *mockStorage) CancelOrder(_ context.Context, id string, reason string) (order.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	o, ok := m.orders[id]
	if !ok {
		return order.Order{}, order.ErrOrderNotFound
	}
	o.Status = order.StatusCancelled
	o.CancelReason = &reason
	m.orders[id] = o
	return o, nil
}

func (m *mockStorage) SearchOrders(_ context.Context, query string, _ pagination.Page) ([]order.Order, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []order.Order
	for _, o := range m.orders {
		if strings.Contains(strings.ToLower(o.CustomerName), strings.ToLower(query)) ||
			strings.Contains(o.ID, query) {
			result = append(result, o)
		}
	}
	return result, len(result), nil
}

func (m *mockStorage) GetOrderStats(_ context.Context) (order.OrderStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	statusMap := make(map[string]*order.StatusStats)
	var stats order.OrderStats

	for _, o := range m.orders {
		s := string(o.Status)
		if _, ok := statusMap[s]; !ok {
			statusMap[s] = &order.StatusStats{Status: s}
		}
		statusMap[s].Count++
		statusMap[s].Revenue += o.TotalAmount
		stats.TotalOrders++
		stats.TotalRevenue += o.TotalAmount
	}

	for _, ss := range statusMap {
		stats.ByStatus = append(stats.ByStatus, *ss)
	}
	if stats.ByStatus == nil {
		stats.ByStatus = []order.StatusStats{}
	}

	return stats, nil
}

func (m *mockStorage) GetSalesByProduct(_ context.Context, _, _ time.Time, _ []order.Status) ([]order.ProductSales, error) {
	return []order.ProductSales{}, nil
}

func (m *mockStorage) GetCustomerSummary(_ context.Context, _ order.CustomerFilter) ([]order.CustomerSummary, error) {
	return []order.CustomerSummary{}, nil
}

func (m *mockStorage) BulkUpdateStatus(_ context.Context, ids []string, _ order.Status, _ string, dryRun bool) (order.BulkStatusResult, error) {
	return order.BulkStatusResult{Total: len(ids), DryRun: dryRun}, nil
}

func newTestService() (*Service, *mockStorage) {
	storage := newMockStorage()
	return NewService(storage, nil, nil, zap.NewNop()), storage
}

func TestCreateOrder(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "John Doe",
		Items: []CreateItemInput{
			{ProductID: "prod-1", Name: "Widget", Quantity: 2, UnitPrice: 10.50},
			{ProductID: "prod-2", Name: "Gadget", Quantity: 1, UnitPrice: 25.00},
		},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if created.CustomerName != "John Doe" {
		t.Errorf("expected customer_name 'John Doe', got %q", created.CustomerName)
	}
	if created.Status != order.StatusPending {
		t.Errorf("expected status pending, got %s", created.Status)
	}
	if len(created.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(created.Items))
	}

	expectedTotal := 2*10.50 + 1*25.00
	if created.TotalAmount != expectedTotal {
		t.Errorf("expected total_amount %.2f, got %.2f", expectedTotal, created.TotalAmount)
	}
}

func TestGetOrderByID(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Jane Doe",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 5.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	fetched, err := svc.GetOrderByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetOrderByID failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, fetched.ID)
	}
}

func TestGetOrderByID_NotFound(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.GetOrderByID(t.Context(), "nonexistent")
	if !errors.Is(err, order.ErrOrderNotFound) {
		t.Errorf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestUpdateOrderStatus_ValidTransition(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	updated, err := svc.UpdateOrderStatus(t.Context(), created.ID, order.StatusConfirmed)
	if err != nil {
		t.Fatalf("UpdateOrderStatus failed: %v", err)
	}

	if updated.Status != order.StatusConfirmed {
		t.Errorf("expected status confirmed, got %s", updated.Status)
	}
}

func TestUpdateOrderStatus_InvalidTransition(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	_, err = svc.UpdateOrderStatus(t.Context(), created.ID, order.StatusDelivered)
	if !errors.Is(err, order.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestListOrders(t *testing.T) {
	svc, _ := newTestService()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
			CustomerName: fmt.Sprintf("Customer %d", i),
			Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
		})
		if err != nil {
			t.Fatalf("CreateOrder failed: %v", err)
		}
	}

	orders, total, err := svc.ListOrders(t.Context(), order.Filter{}, pagination.Sort{Field: "created_at"}, pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListOrders failed: %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(orders) != 3 {
		t.Errorf("expected 3 orders, got %d", len(orders))
	}
}

func TestCancelOrder(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test Cancel",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	cancelled, err := svc.CancelOrder(t.Context(), created.ID, "customer changed mind")
	if err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}

	if cancelled.Status != order.StatusCancelled {
		t.Errorf("expected status cancelled, got %s", cancelled.Status)
	}
	if cancelled.CancelReason == nil || *cancelled.CancelReason != "customer changed mind" {
		t.Errorf("expected cancel_reason 'customer changed mind', got %v", cancelled.CancelReason)
	}
}

func TestCancelOrder_InvalidTransition(t *testing.T) {
	svc, storage := newTestService()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	storage.mu.Lock()
	o := storage.orders[created.ID]
	o.Status = order.StatusCompleted
	storage.orders[created.ID] = o
	storage.mu.Unlock()

	_, err = svc.CancelOrder(t.Context(), created.ID, "too late")
	if !errors.Is(err, order.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestCancelOrder_NotFound(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.CancelOrder(t.Context(), "nonexistent", "reason")
	if !errors.Is(err, order.ErrOrderNotFound) {
		t.Errorf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestSearchOrders(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "John Smith",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	_, err = svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Jane Doe",
		Items:        []CreateItemInput{{ProductID: "p2", Name: "Widget", Quantity: 2, UnitPrice: 5.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	results, total, err := svc.SearchOrders(t.Context(), "John", pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("SearchOrders failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 result, got %d", total)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 order, got %d", len(results))
	}
	if len(results) > 0 && results[0].CustomerName != "John Smith" {
		t.Errorf("expected John Smith, got %s", results[0].CustomerName)
	}
}

func TestSearchOrders_NoResults(t *testing.T) {
	svc, _ := newTestService()

	results, total, err := svc.SearchOrders(t.Context(), "nonexistent", pagination.Page{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("SearchOrders failed: %v", err)
	}

	if total != 0 {
		t.Errorf("expected 0 results, got %d", total)
	}
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %d", len(results))
	}
}

func TestGetOrderStats(t *testing.T) {
	svc, storage := newTestService()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
			CustomerName: fmt.Sprintf("Customer %d", i),
			Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 10.00}},
		})
		if err != nil {
			t.Fatalf("CreateOrder failed: %v", err)
		}
	}

	storage.mu.Lock()
	for id, o := range storage.orders {
		o.Status = order.StatusConfirmed
		storage.orders[id] = o
		break
	}
	storage.mu.Unlock()

	stats, err := svc.GetOrderStats(t.Context())
	if err != nil {
		t.Fatalf("GetOrderStats failed: %v", err)
	}

	if stats.TotalOrders != 3 {
		t.Errorf("expected 3 total orders, got %d", stats.TotalOrders)
	}
	if stats.TotalRevenue != 30.00 {
		t.Errorf("expected total revenue 30.00, got %.2f", stats.TotalRevenue)
	}
	if len(stats.ByStatus) < 1 {
		t.Errorf("expected at least 1 status group, got %d", len(stats.ByStatus))
	}
}

func TestGetOrderStats_Empty(t *testing.T) {
	svc, _ := newTestService()

	stats, err := svc.GetOrderStats(t.Context())
	if err != nil {
		t.Fatalf("GetOrderStats failed: %v", err)
	}

	if stats.TotalOrders != 0 {
		t.Errorf("expected 0 total orders, got %d", stats.TotalOrders)
	}
	if len(stats.ByStatus) != 0 {
		t.Errorf("expected empty by_status, got %d", len(stats.ByStatus))
	}
}

func TestFullWorkflow(t *testing.T) {
	svc, _ := newTestService()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Workflow Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 100.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	transitions := []order.Status{
		order.StatusConfirmed,
		order.StatusProcessing,
		order.StatusShipped,
		order.StatusDelivered,
		order.StatusCompleted,
	}

	current := created
	for _, next := range transitions {
		current, err = svc.UpdateOrderStatus(t.Context(), current.ID, next)
		if err != nil {
			t.Fatalf("transition to %s failed: %v", next, err)
		}
		if current.Status != next {
			t.Errorf("expected status %s, got %s", next, current.Status)
		}
	}
}

func TestShippedToReturned(t *testing.T) {
	svc, storage := newTestService()

	created, err := svc.CreateOrder(t.Context(), CreateOrderRequest{
		CustomerName: "Return Test",
		Items:        []CreateItemInput{{ProductID: "p1", Name: "Item", Quantity: 1, UnitPrice: 50.00}},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	storage.mu.Lock()
	o := storage.orders[created.ID]
	o.Status = order.StatusShipped
	storage.orders[created.ID] = o
	storage.mu.Unlock()

	returned, err := svc.UpdateOrderStatus(t.Context(), created.ID, order.StatusReturned)
	if err != nil {
		t.Fatalf("transition to returned failed: %v", err)
	}
	if returned.Status != order.StatusReturned {
		t.Errorf("expected status returned, got %s", returned.Status)
	}
}
