package consumer

import (
	"context"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/controller"
	"github.com/haradrim/chainorchestra/services/logistics-service/internal/shipment"
)

const queueGroup = "logistics"

type ShipmentCreator interface {
	CreateShipment(ctx context.Context, req controller.CreateShipmentRequest) (shipment.Shipment, error)
}

type ShipmentLookup interface {
	FindByOrderID(ctx context.Context, orderID string) ([]shipment.Shipment, error)
}

type Consumer struct {
	svc    ShipmentCreator
	lookup ShipmentLookup
	nc     *natspkg.Client
	log    *zap.Logger
}

func NewConsumer(svc ShipmentCreator, lookup ShipmentLookup, nc *natspkg.Client, log *zap.Logger) *Consumer {
	return &Consumer{svc: svc, lookup: lookup, nc: nc, log: log}
}

func (c *Consumer) Start() error {
	if _, err := c.nc.Subscribe("order.status_changed", queueGroup, c.handleOrderStatusChanged); err != nil {
		return err
	}

	c.log.Info("Logistics consumer started")
	return nil
}

func (c *Consumer) handleOrderStatusChanged(ev natspkg.Event) error {
	var data struct {
		OrderID      string `json:"order_id"`
		NewStatus    string `json:"new_status"`
		CustomerName string `json:"customer_name"`
	}
	if err := ev.DecodeData(&data); err != nil {
		c.log.Error("Failed to decode order.status_changed event", zap.Error(err))
		return nil
	}

	if data.NewStatus != "confirmed" {
		return nil
	}

	ctx := context.Background()

	if c.lookup != nil {
		existing, err := c.lookup.FindByOrderID(ctx, data.OrderID)
		if err != nil {
			c.log.Warn("Idempotency check failed; proceeding with create",
				zap.String("order_id", data.OrderID), zap.Error(err))
		} else if len(existing) > 0 {
			c.log.Info("Shipment already exists for order; skipping duplicate auto-create",
				zap.String("order_id", data.OrderID),
				zap.Int("existing_count", len(existing)))
			return nil
		}
	}

	c.log.Info("Order confirmed, auto-creating shipment",
		zap.String("order_id", data.OrderID),
	)

	req := controller.CreateShipmentRequest{OrderID: data.OrderID}
	if data.CustomerName != "" {
		req.RecipientName = data.CustomerName
	}

	_, err := c.svc.CreateShipment(ctx, req)
	if err != nil {
		c.log.Error("Failed to auto-create shipment for confirmed order",
			zap.String("order_id", data.OrderID),
			zap.Error(err),
		)
		return nil
	}

	return nil
}
