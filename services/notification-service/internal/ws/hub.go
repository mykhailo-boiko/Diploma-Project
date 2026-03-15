package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/middleware"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 54 * time.Second
)

type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[*websocket.Conn]struct{}
	log   *zap.Logger
}

func NewHub(log *zap.Logger) *Hub {
	return &Hub{
		conns: make(map[string]map[*websocket.Conn]struct{}),
		log:   log,
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("Failed to upgrade WebSocket", zap.Error(err))
		return
	}

	h.addConn(userID, conn)
	h.log.Info("WebSocket connected", zap.String("user_id", userID))

	go h.readPump(userID, conn)
	go h.pingPump(conn)
}

func (h *Hub) Push(userID string, n notification.Notification) {
	h.mu.RLock()
	userConns, ok := h.conns[userID]
	if !ok || len(userConns) == 0 {
		h.mu.RUnlock()
		return
	}

	conns := make([]*websocket.Conn, 0, len(userConns))
	for c := range userConns {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	data, err := json.Marshal(n)
	if err != nil {
		h.log.Error("Failed to marshal notification for WebSocket", zap.Error(err))
		return
	}

	for _, c := range conns {
		_ = c.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			h.log.Debug("Failed to write to WebSocket, removing", zap.Error(err))
			h.removeConn(userID, c)
			_ = c.Close()
		}
	}
}

func (h *Hub) ActiveConnections(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns[userID])
}

func (h *Hub) addConn(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*websocket.Conn]struct{})
	}
	h.conns[userID][conn] = struct{}{}
}

func (h *Hub) removeConn(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if userConns, ok := h.conns[userID]; ok {
		delete(userConns, conn)
		if len(userConns) == 0 {
			delete(h.conns, userID)
		}
	}
}

func (h *Hub) readPump(userID string, conn *websocket.Conn) {
	defer func() {
		h.removeConn(userID, conn)
		_ = conn.Close()
		h.log.Info("WebSocket disconnected", zap.String("user_id", userID))
	}()

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

func (h *Hub) pingPump(conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for range ticker.C {
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}
