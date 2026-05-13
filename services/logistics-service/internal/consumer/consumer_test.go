package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/controller"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/shipment"
)

type mockShipmentCreator struct {
	created []controller.CreateShipmentRequest
}

func (m *mockShipmentCreator) CreateShipment(_ context.Context, req controller.CreateShipmentRequest) (shipment.Shipment, error) {
	m.created = append(m.created, req)
	return shipment.Shipment{ID: "sh-1", OrderID: req.OrderID, Status: shipment.StatusCreated}, nil
}

type mockShipmentLookup struct {
	byOrder map[string][]shipment.Shipment
}

func (m *mockShipmentLookup) FindByOrderID(_ context.Context, orderID string) ([]shipment.Shipment, error) {
	if m.byOrder == nil {
		return nil, nil
	}
	return m.byOrder[orderID], nil
}

func newTestConsumer() (*Consumer, *mockShipmentCreator) {
	svc := &mockShipmentCreator{}
	lookup := &mockShipmentLookup{byOrder: map[string][]shipment.Shipment{}}
	return &Consumer{svc: svc, lookup: lookup, log: zap.NewNop()}, svc
}

func newTestConsumerWithExisting(orderID string) (*Consumer, *mockShipmentCreator) {
	svc := &mockShipmentCreator{}
	lookup := &mockShipmentLookup{byOrder: map[string][]shipment.Shipment{
		orderID: {{ID: "preexisting-1", OrderID: orderID}},
	}}
	return &Consumer{svc: svc, lookup: lookup, log: zap.NewNop()}, svc
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

func TestHandleOrderStatusChanged_Confirmed(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("order.status_changed", map[string]any{
		"order_id":   "o1",
		"new_status": "confirmed",
	})

	if err := c.handleOrderStatusChanged(ev); err != nil {
		t.Fatalf("handleOrderStatusChanged failed: %v", err)
	}

	if len(svc.created) != 1 {
		t.Fatalf("expected 1 shipment creation, got %d", len(svc.created))
	}
	if svc.created[0].OrderID != "o1" {
		t.Errorf("expected order_id 'o1', got %s", svc.created[0].OrderID)
	}
}

func TestHandleOrderStatusChanged_NotConfirmed(t *testing.T) {
	c, svc := newTestConsumer()

	ev := makeEvent("order.status_changed", map[string]any{
		"order_id":   "o1",
		"new_status": "processing",
	})

	if err := c.handleOrderStatusChanged(ev); err != nil {
		t.Fatalf("handleOrderStatusChanged failed: %v", err)
	}

	if len(svc.created) != 0 {
		t.Errorf("expected no shipment creation for non-confirmed status, got %d", len(svc.created))
	}
}

func TestHandleOrderStatusChanged_IdempotentSkipExisting(t *testing.T) {
	c, svc := newTestConsumerWithExisting("o1")

	ev := makeEvent("order.status_changed", map[string]any{
		"order_id":   "o1",
		"new_status": "confirmed",
	})

	if err := c.handleOrderStatusChanged(ev); err != nil {
		t.Fatalf("handleOrderStatusChanged failed: %v", err)
	}

	if len(svc.created) != 0 {
		t.Errorf("expected no duplicate shipment creation when one already exists, got %d", len(svc.created))
	}
}
