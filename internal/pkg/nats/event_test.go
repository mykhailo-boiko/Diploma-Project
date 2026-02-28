package nats

import (
	"encoding/json"
	"testing"
	"time"
)

type testPayload struct {
	OrderID string `json:"order_id"`
	Amount  int    `json:"amount"`
}

func TestNewEvent(t *testing.T) {
	data := testPayload{OrderID: "ord-123", Amount: 500}
	ev, err := NewEvent("order.created", "order-service", data)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	if ev.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if ev.Type != "order.created" {
		t.Errorf("Type = %q, want %q", ev.Type, "order.created")
	}
	if ev.Source != "order-service" {
		t.Errorf("Source = %q, want %q", ev.Source, "order-service")
	}
	if ev.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if time.Since(ev.Timestamp) > time.Second {
		t.Error("timestamp should be recent")
	}
	if ev.Data == nil {
		t.Error("expected non-nil Data")
	}
}

func TestNewEvent_MarshalError(t *testing.T) {
	_, err := NewEvent("test", "src", make(chan int))
	if err == nil {
		t.Error("expected error for unmarshalable type")
	}
}

func TestEvent_DecodeData(t *testing.T) {
	data := testPayload{OrderID: "ord-456", Amount: 100}
	ev, err := NewEvent("order.created", "order-service", data)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	var decoded testPayload
	if err := ev.DecodeData(&decoded); err != nil {
		t.Fatalf("DecodeData() error = %v", err)
	}

	if decoded.OrderID != "ord-456" {
		t.Errorf("OrderID = %q, want %q", decoded.OrderID, "ord-456")
	}
	if decoded.Amount != 100 {
		t.Errorf("Amount = %d, want %d", decoded.Amount, 100)
	}
}

func TestEvent_DecodeData_InvalidJSON(t *testing.T) {
	ev := Event{Data: json.RawMessage(`{invalid`)}
	var decoded testPayload
	if err := ev.DecodeData(&decoded); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestEvent_EncodeAndDecode(t *testing.T) {
	data := testPayload{OrderID: "ord-789", Amount: 999}
	ev, err := NewEvent("inventory.stock_changed", "inventory-service", data)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	raw, err := ev.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	decoded, err := DecodeEvent(raw)
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}

	if decoded.ID != ev.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, ev.ID)
	}
	if decoded.Type != ev.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, ev.Type)
	}
	if decoded.Source != ev.Source {
		t.Errorf("Source = %q, want %q", decoded.Source, ev.Source)
	}
	if !decoded.Timestamp.Equal(ev.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, ev.Timestamp)
	}

	var payload testPayload
	if err := decoded.DecodeData(&payload); err != nil {
		t.Fatalf("DecodeData() error = %v", err)
	}
	if payload.OrderID != "ord-789" {
		t.Errorf("OrderID = %q, want %q", payload.OrderID, "ord-789")
	}
	if payload.Amount != 999 {
		t.Errorf("Amount = %d, want %d", payload.Amount, 999)
	}
}

func TestDecodeEvent_InvalidJSON(t *testing.T) {
	_, err := DecodeEvent([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("nats://localhost:4222", "test-service")
	if cfg.URL != "nats://localhost:4222" {
		t.Errorf("URL = %q, want %q", cfg.URL, "nats://localhost:4222")
	}
	if cfg.Name != "test-service" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test-service")
	}
	if cfg.MaxReconnects != -1 {
		t.Errorf("MaxReconnects = %d, want -1", cfg.MaxReconnects)
	}
	if cfg.ReconnectWait != 2*time.Second {
		t.Errorf("ReconnectWait = %v, want 2s", cfg.ReconnectWait)
	}
	if cfg.PingInterval != 20*time.Second {
		t.Errorf("PingInterval = %v, want 20s", cfg.PingInterval)
	}
	if cfg.MaxPingsOut != 5 {
		t.Errorf("MaxPingsOut = %d, want 5", cfg.MaxPingsOut)
	}
}

func TestEvent_JSONFieldNames(t *testing.T) {
	data := testPayload{OrderID: "o1", Amount: 1}
	ev, err := NewEvent("test.event", "src", data)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	raw, err := ev.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	expectedKeys := []string{"id", "type", "source", "timestamp", "data"}
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("missing expected JSON key %q", key)
		}
	}
}
