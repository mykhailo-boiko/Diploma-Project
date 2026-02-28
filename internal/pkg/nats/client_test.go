package nats

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"go.uber.org/zap"
)

func startTestServer(t *testing.T) *natsserver.Server {
	t.Helper()
	opts := &natsserver.Options{
		Host: "127.0.0.1",
		Port: -1,
	}
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("failed to create test NATS server: %v", err)
	}
	srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("NATS server not ready")
	}
	t.Cleanup(func() { srv.Shutdown() })
	return srv
}

func TestClient_ConnectAndClose(t *testing.T) {
	srv := startTestServer(t)
	log := zap.NewNop()

	cfg := DefaultConfig(srv.ClientURL(), "test-service")
	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}

	client.Close()
}

func TestClient_ConnectError(t *testing.T) {
	log := zap.NewNop()
	cfg := DefaultConfig("nats://127.0.0.1:1", "test-service")
	cfg.MaxReconnects = 0

	_, err := NewClient(cfg, log)
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestClient_PublishAndSubscribe(t *testing.T) {
	srv := startTestServer(t)
	log := zap.NewNop()

	cfg := DefaultConfig(srv.ClientURL(), "test-service")
	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	type orderData struct {
		OrderID string `json:"order_id"`
		Total   int    `json:"total"`
	}

	var (
		mu       sync.Mutex
		received []Event
	)

	sub, err := client.Subscribe("orders.>", "test-group", func(ev Event) error {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	payload := orderData{OrderID: "ord-123", Total: 500}
	if err := client.Publish("orders.created", "order.created", payload); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if err := client.Conn().Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("received %d events, want 1", len(received))
	}

	ev := received[0]
	if ev.Type != "order.created" {
		t.Errorf("Type = %q, want %q", ev.Type, "order.created")
	}
	if ev.Source != "test-service" {
		t.Errorf("Source = %q, want %q", ev.Source, "test-service")
	}

	var decoded orderData
	if err := ev.DecodeData(&decoded); err != nil {
		t.Fatalf("DecodeData() error = %v", err)
	}
	if decoded.OrderID != "ord-123" {
		t.Errorf("OrderID = %q, want %q", decoded.OrderID, "ord-123")
	}
	if decoded.Total != 500 {
		t.Errorf("Total = %d, want %d", decoded.Total, 500)
	}
}

func TestClient_QueueGroupDistribution(t *testing.T) {
	srv := startTestServer(t)
	log := zap.NewNop()

	cfg := DefaultConfig(srv.ClientURL(), "test-service")
	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	var (
		mu     sync.Mutex
		count1 int
		count2 int
	)

	sub1, err := client.Subscribe("events.test", "worker-group", func(_ Event) error {
		mu.Lock()
		defer mu.Unlock()
		count1++
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	defer func() { _ = sub1.Unsubscribe() }()

	sub2, err := client.Subscribe("events.test", "worker-group", func(_ Event) error {
		mu.Lock()
		defer mu.Unlock()
		count2++
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	defer func() { _ = sub2.Unsubscribe() }()

	const messageCount = 20
	for i := range messageCount {
		if err := client.Publish("events.test", "test.event", map[string]int{"i": i}); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	if err := client.Conn().Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	total := count1 + count2
	if total != messageCount {
		t.Errorf("total processed = %d, want %d", total, messageCount)
	}
}

func TestClient_HealthHandler_Connected(t *testing.T) {
	srv := startTestServer(t)
	log := zap.NewNop()

	cfg := DefaultConfig(srv.ClientURL(), "test-service")
	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/nats", nil)
	client.HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", rec.Body.String(), `{"status":"ok"}`)
	}
}

func TestClient_HealthHandler_Disconnected(t *testing.T) {
	srv := startTestServer(t)
	log := zap.NewNop()

	cfg := DefaultConfig(srv.ClientURL(), "test-service")
	cfg.MaxReconnects = 0
	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	srv.Shutdown()
	time.Sleep(200 * time.Millisecond)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/nats", nil)
	client.HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestClient_MultipleSubjects(t *testing.T) {
	srv := startTestServer(t)
	log := zap.NewNop()

	cfg := DefaultConfig(srv.ClientURL(), "test-service")
	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	var (
		mu     sync.Mutex
		events = make(map[string]int)
	)

	handler := func(ev Event) error {
		mu.Lock()
		defer mu.Unlock()
		events[ev.Type]++
		return nil
	}

	sub1, err := client.Subscribe("orders.*", "svc", handler)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	defer func() { _ = sub1.Unsubscribe() }()

	sub2, err := client.Subscribe("inventory.*", "svc", handler)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	defer func() { _ = sub2.Unsubscribe() }()

	if err := client.Publish("orders.created", "order.created", nil); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if err := client.Publish("inventory.changed", "inventory.stock_changed", nil); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if err := client.Conn().Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if events["order.created"] != 1 {
		t.Errorf("order.created count = %d, want 1", events["order.created"])
	}
	if events["inventory.stock_changed"] != 1 {
		t.Errorf("inventory.stock_changed count = %d, want 1", events["inventory.stock_changed"])
	}
}
