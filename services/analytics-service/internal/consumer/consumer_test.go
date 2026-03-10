package consumer

import (
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
)

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func TestHandleOrderCreated(t *testing.T) {
	c := &Consumer{log: testLogger()}

	ev := natspkg.Event{
		ID:        "test-id",
		Type:      "order.created",
		Source:    "order-service",
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"order_id":"o1","customer_name":"John"}`),
	}

	if err := c.handleOrderCreated(ev); err != nil {
		t.Errorf("handleOrderCreated failed: %v", err)
	}
}

func TestHandleOrderStatusChanged(t *testing.T) {
	c := &Consumer{log: testLogger()}

	ev := natspkg.Event{
		ID:        "test-id",
		Type:      "order.status_changed",
		Source:    "order-service",
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"order_id":"o1","previous_status":"pending","new_status":"confirmed"}`),
	}

	if err := c.handleOrderStatusChanged(ev); err != nil {
		t.Errorf("handleOrderStatusChanged failed: %v", err)
	}
}

func TestHandleOrderCancelled(t *testing.T) {
	c := &Consumer{log: testLogger()}

	ev := natspkg.Event{
		ID:        "test-id",
		Type:      "order.cancelled",
		Source:    "order-service",
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"order_id":"o1","reason":"out of stock"}`),
	}

	if err := c.handleOrderCancelled(ev); err != nil {
		t.Errorf("handleOrderCancelled failed: %v", err)
	}
}

func TestHandleInventoryStockChanged(t *testing.T) {
	c := &Consumer{log: testLogger()}

	ev := natspkg.Event{
		ID:        "test-id",
		Type:      "inventory.stock_changed",
		Source:    "inventory-service",
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"stock_id":"s1","product_id":"p1","type":"reserve"}`),
	}

	if err := c.handleInventoryStockChanged(ev); err != nil {
		t.Errorf("handleInventoryStockChanged failed: %v", err)
	}
}

func TestHandleInventoryLowStock(t *testing.T) {
	c := &Consumer{log: testLogger()}

	ev := natspkg.Event{
		ID:        "test-id",
		Type:      "inventory.low_stock",
		Source:    "inventory-service",
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"stock_id":"s1","product_id":"p1","available":3,"threshold":10}`),
	}

	if err := c.handleInventoryLowStock(ev); err != nil {
		t.Errorf("handleInventoryLowStock failed: %v", err)
	}
}

func TestHandleLogisticsShipmentCreated(t *testing.T) {
	c := &Consumer{log: testLogger()}

	ev := natspkg.Event{
		ID:        "test-id",
		Type:      "logistics.shipment_created",
		Source:    "logistics-service",
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"shipment_id":"sh1","order_id":"o1"}`),
	}

	if err := c.handleLogisticsShipmentCreated(ev); err != nil {
		t.Errorf("handleLogisticsShipmentCreated failed: %v", err)
	}
}

func TestHandleLogisticsShipmentStatusChanged(t *testing.T) {
	c := &Consumer{log: testLogger()}

	ev := natspkg.Event{
		ID:        "test-id",
		Type:      "logistics.shipment_status_changed",
		Source:    "logistics-service",
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"shipment_id":"sh1","previous_status":"created","new_status":"in_transit"}`),
	}

	if err := c.handleLogisticsShipmentStatusChanged(ev); err != nil {
		t.Errorf("handleLogisticsShipmentStatusChanged failed: %v", err)
	}
}

func TestTruncateJSON(t *testing.T) {
	short := json.RawMessage(`{"key":"value"}`)
	result := truncateJSON(short)
	if result != `{"key":"value"}` {
		t.Errorf("expected unchanged string, got %q", result)
	}

	long := make(json.RawMessage, 600)
	for i := range long {
		long[i] = 'a'
	}
	result = truncateJSON(long)
	if len(result) != 515 {
		t.Errorf("expected truncated to 515 chars, got %d", len(result))
	}
}
