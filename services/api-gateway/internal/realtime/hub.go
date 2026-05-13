package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type subscriber struct {
	id    string
	role  string
	ch    chan Event
	done  chan struct{}
	close sync.Once
}

func (s *subscriber) shutdown() {
	s.close.Do(func() {
		close(s.done)
		close(s.ch)
	})
}

type Hub struct {
	log *zap.Logger

	mu   sync.RWMutex
	subs map[string]*subscriber

	subjects []string

	ncMu     sync.RWMutex
	nc       *nats.Conn
	subs2    []*nats.Subscription
	attached bool
}

func NewHub(nc *nats.Conn, log *zap.Logger) *Hub {
	h := NewLazyHub(log)
	if nc != nil {
		_ = h.Attach(nc)
	}
	return h
}

func NewLazyHub(log *zap.Logger) *Hub {
	return &Hub{
		log:  log,
		subs: make(map[string]*subscriber),
		subjects: []string{
			"order.created",
			"order.status_changed",
			"order.cancelled",
			"logistics.shipment_created",
			"logistics.shipment_status_changed",
			"logistics.shipment_out_for_delivery",
			"logistics.shipment_delivered",
			"logistics.shipment_attempted",
			"logistics.shipment_returned",
			"logistics.shipment_redirected",
			"inventory.stock_changed",
			"inventory.low_stock",
			"notification.created",
			"analytics.aggregate_updated",
		},
	}
}

func (h *Hub) Attach(nc *nats.Conn) error {
	if nc == nil {
		return fmt.Errorf("nats connection is nil")
	}
	h.ncMu.Lock()
	defer h.ncMu.Unlock()
	if h.attached {
		return nil
	}
	h.nc = nc
	for _, subj := range h.subjects {
		s, err := nc.Subscribe(subj, h.onMessage(subj))
		if err != nil {
			h.log.Error("Failed to subscribe", zap.String("subject", subj), zap.Error(err))
			continue
		}
		h.subs2 = append(h.subs2, s)
	}
	h.attached = true
	h.log.Info("Realtime hub attached to NATS", zap.Int("subjects", len(h.subs2)))
	return nil
}

func (h *Hub) IsAttached() bool {
	h.ncMu.RLock()
	defer h.ncMu.RUnlock()
	return h.attached
}

func (h *Hub) Start() error {
	if h.IsAttached() {
		return nil
	}
	return fmt.Errorf("nats connection not configured")
}

func (h *Hub) Stop() {
	for _, s := range h.subs2 {
		_ = s.Unsubscribe()
	}
	h.mu.Lock()
	for _, s := range h.subs {
		s.shutdown()
	}
	h.subs = map[string]*subscriber{}
	h.mu.Unlock()
}

func (h *Hub) onMessage(subject string) nats.MsgHandler {
	return func(msg *nats.Msg) {
		var envelope struct {
			ID        string          `json:"id"`
			Type      string          `json:"type"`
			Source    string          `json:"source"`
			Timestamp time.Time       `json:"timestamp"`
			Data      json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg.Data, &envelope); err != nil {
			h.log.Warn("Failed to decode NATS event", zap.String("subject", subject), zap.Error(err))
			return
		}
		ev := Event{
			Type:      subjectToEventType(subject),
			Source:    envelope.Source,
			Timestamp: envelope.Timestamp,
			Subject:   subject,
			Data:      envelope.Data,
		}
		if ev.Timestamp.IsZero() {
			ev.Timestamp = time.Now().UTC()
		}
		h.broadcast(subject, ev)
	}
}

func (h *Hub) broadcast(subject string, ev Event) {
	h.mu.RLock()
	subs := make([]*subscriber, 0, len(h.subs))
	for _, s := range h.subs {
		if roleAllows(s.role, subject) {
			subs = append(subs, s)
		}
	}
	h.mu.RUnlock()
	for _, s := range subs {
		select {
		case s.ch <- ev:
		default:
		}
	}
}

func (h *Hub) Handle(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if !h.IsAttached() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"realtime initializing","retry_after_seconds":5}`))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub := &subscriber{
		id:   fmt.Sprintf("%s-%d", userID, time.Now().UnixNano()),
		role: role,
		ch:   make(chan Event, 64),
		done: make(chan struct{}),
	}

	h.mu.Lock()
	h.subs[sub.id] = sub
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.subs, sub.id)
		h.mu.Unlock()
		sub.shutdown()
	}()

	hello := Event{Type: "connection.established", Source: "api-gateway", Timestamp: time.Now().UTC()}
	writeSSE(w, hello)
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case <-sub.done:
			return
		case <-heartbeat.C:
			_, _ = fmt.Fprintf(w, ": keepalive %d\n\n", time.Now().Unix())
			flusher.Flush()
		case ev, ok := <-sub.ch:
			if !ok {
				return
			}
			writeSSE(w, ev)
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, ev Event) {
	raw, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", ev.Type)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
}
