package nats

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type Config struct {
	URL            string
	Name           string
	MaxReconnects  int
	ReconnectWait  time.Duration
	PingInterval   time.Duration
	MaxPingsOut    int
}

func DefaultConfig(url, name string) Config {
	return Config{
		URL:            url,
		Name:           name,
		MaxReconnects:  -1,
		ReconnectWait:  2 * time.Second,
		PingInterval:   20 * time.Second,
		MaxPingsOut:    5,
	}
}

type Client struct {
	conn   *nats.Conn
	source string
	log    *zap.Logger
}

func NewClient(cfg Config, log *zap.Logger) (*Client, error) {
	opts := []nats.Option{
		nats.Name(cfg.Name),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.PingInterval(cfg.PingInterval),
		nats.MaxPingsOutstanding(cfg.MaxPingsOut),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Info("NATS reconnected", zap.String("url", c.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			log.Info("NATS connection closed")
		}),
		nats.ErrorHandler(func(_ *nats.Conn, sub *nats.Subscription, err error) {
			subj := ""
			if sub != nil {
				subj = sub.Subject
			}
			log.Error("NATS async error", zap.String("subject", subj), zap.Error(err))
		}),
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	log.Info("Connected to NATS", zap.String("url", conn.ConnectedUrl()))

	return &Client{
		conn:   conn,
		source: cfg.Name,
		log:    log,
	}, nil
}

func (c *Client) Publish(subject, eventType string, data any) error {
	ev, err := NewEvent(eventType, c.source, data)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	raw, err := ev.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	if err := c.conn.Publish(subject, raw); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}

	c.log.Debug("Published event",
		zap.String("subject", subject),
		zap.String("event_type", eventType),
		zap.String("event_id", ev.ID),
	)

	return nil
}

type Handler func(Event) error

func (c *Client) Subscribe(subject, queue string, handler Handler) (*nats.Subscription, error) {
	sub, err := c.conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		ev, err := DecodeEvent(msg.Data)
		if err != nil {
			c.log.Error("Failed to decode event",
				zap.String("subject", msg.Subject),
				zap.Error(err),
			)
			return
		}

		c.log.Debug("Received event",
			zap.String("subject", msg.Subject),
			zap.String("event_type", ev.Type),
			zap.String("event_id", ev.ID),
		)

		if err := handler(ev); err != nil {
			c.log.Error("Failed to handle event",
				zap.String("subject", msg.Subject),
				zap.String("event_type", ev.Type),
				zap.String("event_id", ev.ID),
				zap.Error(err),
			)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	c.log.Info("Subscribed", zap.String("subject", subject), zap.String("queue", queue))

	return sub, nil
}

func (c *Client) Conn() *nats.Conn {
	return c.conn
}

func (c *Client) IsConnected() bool {
	return c.conn.IsConnected()
}

func (c *Client) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if c.conn.IsConnected() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"disconnected"}`))
	}
}

func (c *Client) Close() {
	if err := c.conn.Drain(); err != nil {
		c.log.Error("Failed to drain NATS connection", zap.Error(err))
		c.conn.Close()
		return
	}
	c.log.Info("NATS connection drained and closed")
}
