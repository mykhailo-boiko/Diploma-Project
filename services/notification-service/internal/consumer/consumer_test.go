package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/controller"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

type mockNotificationService struct {
	created []controller.CreateNotificationRequest
}

func (m *mockNotificationService) CreateNotification(_ context.Context, req controller.CreateNotificationRequest) (notification.Notification, error) {
	m.created = append(m.created, req)
	return notification.Notification{
		ID:     "notif-1",
		UserID: req.UserID,
		Type:   req.Type,
		Title:  req.Title,
	}, nil
}

func newTestConsumer() (*Consumer, *mockNotificationService) {
	svc := &mockNotificationService{}
	return &Consumer{svc: svc, log: zap.NewNop()}, svc
}

func makeEvent(eventType string, data any) natspkg.Event {
	raw, _ := json.Marshal(data)
	return natspkg.Event{
		ID:        "test-id",
		Type:      eventType,
		Source:    "test",
		Timestamp: time.Now().UTC(),
		Data:      raw,
	}
}

func TestHandleOrderCreated(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("order.created", map[string]any{
		"order_id":      "o1",
		"customer_name": "John",
		"total_amount":  99.99,
	})

	if err := c.handleOrderCreated(ev); err != nil {
		t.Fatalf("handleOrderCreated failed: %v", err)
	}

	if len(svc.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.created))
	}
	if svc.created[0].Type != notification.TypeOrderCreated {
		t.Errorf("expected type %s, got %s", notification.TypeOrderCreated, svc.created[0].Type)
	}
}

func TestHandleOrderStatusChanged(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("order.status_changed", map[string]any{
		"order_id":        "o1",
		"previous_status": "pending",
		"new_status":      "confirmed",
	})

	if err := c.handleOrderStatusChanged(ev); err != nil {
		t.Fatalf("handleOrderStatusChanged failed: %v", err)
	}

	if len(svc.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.created))
	}
	if svc.created[0].Type != notification.TypeOrderUpdated {
		t.Errorf("expected type %s, got %s", notification.TypeOrderUpdated, svc.created[0].Type)
	}
}

func TestHandleOrderCancelled(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("order.cancelled", map[string]any{
		"order_id": "o1",
		"reason":   "out of stock",
	})

	if err := c.handleOrderCancelled(ev); err != nil {
		t.Fatalf("handleOrderCancelled failed: %v", err)
	}

	if len(svc.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.created))
	}
	if svc.created[0].Type != notification.TypeOrderCancelled {
		t.Errorf("expected type %s, got %s", notification.TypeOrderCancelled, svc.created[0].Type)
	}
}

func TestHandleInventoryLowStock(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("inventory.low_stock", map[string]any{
		"product_id":    "p1",
		"warehouse_id":  "w1",
		"available":     5,
		"min_threshold": 20,
	})

	if err := c.handleInventoryLowStock(ev); err != nil {
		t.Fatalf("handleInventoryLowStock failed: %v", err)
	}

	if len(svc.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.created))
	}
	if svc.created[0].Type != notification.TypeLowStock {
		t.Errorf("expected type %s, got %s", notification.TypeLowStock, svc.created[0].Type)
	}
}

func TestHandleLogisticsShipmentCreated(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("logistics.shipment_created", map[string]any{
		"shipment_id": "sh1",
		"order_id":    "o1",
	})

	if err := c.handleLogisticsShipmentCreated(ev); err != nil {
		t.Fatalf("handleLogisticsShipmentCreated failed: %v", err)
	}

	if len(svc.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.created))
	}
	if svc.created[0].Type != notification.TypeShipmentCreated {
		t.Errorf("expected type %s, got %s", notification.TypeShipmentCreated, svc.created[0].Type)
	}
}

func TestHandleLogisticsShipmentStatusChanged(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("logistics.shipment_status_changed", map[string]any{
		"shipment_id":     "sh1",
		"order_id":        "o1",
		"previous_status": "created",
		"new_status":      "in_transit",
	})

	if err := c.handleLogisticsShipmentStatusChanged(ev); err != nil {
		t.Fatalf("handleLogisticsShipmentStatusChanged failed: %v", err)
	}

	if len(svc.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.created))
	}
	if svc.created[0].Type != notification.TypeShipmentUpdated {
		t.Errorf("expected type %s, got %s", notification.TypeShipmentUpdated, svc.created[0].Type)
	}
}

func TestHandleNotificationSend(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("events.notifications.send", map[string]any{
		"user_id": "user-1",
		"type":    "system",
		"title":   "Test",
		"message": "Hello",
	})

	if err := c.handleNotificationSend(ev); err != nil {
		t.Fatalf("handleNotificationSend failed: %v", err)
	}

	if len(svc.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.created))
	}
	if svc.created[0].UserID != "user-1" {
		t.Errorf("expected user_id 'user-1', got %s", svc.created[0].UserID)
	}
}
