package consumer

import (
	"encoding/json"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
)

type Consumer struct {
	nc  *natspkg.Client
	log *zap.Logger
}

func NewConsumer(nc *natspkg.Client, log *zap.Logger) *Consumer {
	return &Consumer{nc: nc, log: log}
}

func (c *Consumer) Subscribe() error {
	subjects := []struct {
		subject string
		queue   string
		handler natspkg.Handler
	}{
		{"order.created", "analytics", c.handleOrderCreated},
		{"order.status_changed", "analytics", c.handleOrderStatusChanged},
		{"order.cancelled", "analytics", c.handleOrderCancelled},
		{"inventory.stock_changed", "analytics", c.handleInventoryStockChanged},
		{"inventory.low_stock", "analytics", c.handleInventoryLowStock},
		{"logistics.shipment_created", "analytics", c.handleLogisticsShipmentCreated},
		{"logistics.shipment_status_changed", "analytics", c.handleLogisticsShipmentStatusChanged},
	}

	for _, s := range subjects {
		if _, err := c.nc.Subscribe(s.subject, s.queue, s.handler); err != nil {
			return err
		}
	}

	return nil
}

func (c *Consumer) handleOrderCreated(ev natspkg.Event) error {
	c.log.Info("Received event",
		zap.String("type", ev.Type),
		zap.String("id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("data", string(ev.Data)),
	)
	return nil
}

func (c *Consumer) handleOrderStatusChanged(ev natspkg.Event) error {
	c.log.Info("Received event",
		zap.String("type", ev.Type),
		zap.String("id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("data", string(ev.Data)),
	)
	return nil
}

func (c *Consumer) handleOrderCancelled(ev natspkg.Event) error {
	c.log.Info("Received event",
		zap.String("type", ev.Type),
		zap.String("id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("data", string(ev.Data)),
	)
	return nil
}

func (c *Consumer) handleInventoryStockChanged(ev natspkg.Event) error {
	c.log.Info("Received event",
		zap.String("type", ev.Type),
		zap.String("id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("data", truncateJSON(ev.Data)),
	)
	return nil
}

func (c *Consumer) handleInventoryLowStock(ev natspkg.Event) error {
	c.log.Info("Received event",
		zap.String("type", ev.Type),
		zap.String("id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("data", truncateJSON(ev.Data)),
	)
	return nil
}

func (c *Consumer) handleLogisticsShipmentCreated(ev natspkg.Event) error {
	c.log.Info("Received event",
		zap.String("type", ev.Type),
		zap.String("id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("data", string(ev.Data)),
	)
	return nil
}

func (c *Consumer) handleLogisticsShipmentStatusChanged(ev natspkg.Event) error {
	c.log.Info("Received event",
		zap.String("type", ev.Type),
		zap.String("id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("data", string(ev.Data)),
	)
	return nil
}

func truncateJSON(data json.RawMessage) string {
	s := string(data)
	if len(s) > 512 {
		return s[:512] + "..."
	}
	return s
}
