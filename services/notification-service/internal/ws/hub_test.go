package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

func TestHub_HandleWebSocket_NoUserID(t *testing.T) {
	hub := NewHub(zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/ws/notifications", nil)
	rec := httptest.NewRecorder()

	hub.HandleWebSocket(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHub_ConnectAndPush(t *testing.T) {
	hub := NewHub(zap.NewNop())

	server := httptest.NewServer(middleware.UserContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	})))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/notifications"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"X-User-Id": {"user-1"}})
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	time.Sleep(50 * time.Millisecond)

	if hub.ActiveConnections("user-1") != 1 {
		t.Errorf("expected 1 connection, got %d", hub.ActiveConnections("user-1"))
	}

	hub.Push("user-1", notification.Notification{
		ID:     "notif-1",
		UserID: "user-1",
		Type:   notification.TypeSystem,
		Title:  "Test",
	})

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	if !strings.Contains(string(msg), "notif-1") {
		t.Errorf("expected message to contain 'notif-1', got %s", string(msg))
	}
}

func TestHub_PushNoConnections(t *testing.T) {
	hub := NewHub(zap.NewNop())

	hub.Push("nonexistent-user", notification.Notification{
		ID:     "notif-1",
		UserID: "nonexistent-user",
	})
}

func TestHub_MultipleConnections(t *testing.T) {
	hub := NewHub(zap.NewNop())

	server := httptest.NewServer(middleware.UserContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	})))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/notifications"

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"X-User-Id": {"user-1"}})
	if err != nil {
		t.Fatalf("failed to connect (1): %v", err)
	}
	defer func() { _ = conn1.Close() }()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"X-User-Id": {"user-1"}})
	if err != nil {
		t.Fatalf("failed to connect (2): %v", err)
	}
	defer func() { _ = conn2.Close() }()

	time.Sleep(50 * time.Millisecond)

	if hub.ActiveConnections("user-1") != 2 {
		t.Errorf("expected 2 connections, got %d", hub.ActiveConnections("user-1"))
	}

	hub.Push("user-1", notification.Notification{
		ID:    "notif-2",
		Title: "Both",
	})

	for i, conn := range []*websocket.Conn{conn1, conn2} {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("conn %d: failed to read: %v", i+1, err)
		}
		if !strings.Contains(string(msg), "notif-2") {
			t.Errorf("conn %d: expected 'notif-2' in message", i+1)
		}
	}
}
