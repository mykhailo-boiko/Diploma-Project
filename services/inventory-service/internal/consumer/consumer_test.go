package consumer

import (
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
)

func newTestConsumer() *Consumer {
	return &Consumer{log: zap.NewNop()}
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

func TestHandleOrderCancelled(t *testing.T) {
	c := newTestConsumer()

	ev := makeEvent("order.cancelled", map[string]any{
		"order_id": "o1",
		"reason":   "customer changed mind",
	})

	if err := c.handleOrderCancelled(ev); err != nil {
		t.Fatalf("handleOrderCancelled failed: %v", err)
	}
}

func TestHandleOrderCancelled_InvalidData(t *testing.T) {
	c := newTestConsumer()

	ev := natspkg.Event{
		ID:        "test-id",
		Type:      "order.cancelled",
		Source:    "test",
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`invalid json`),
	}

	if err := c.handleOrderCancelled(ev); err != nil {
		t.Fatalf("handleOrderCancelled should not return error for invalid data: %v", err)
	}
}
