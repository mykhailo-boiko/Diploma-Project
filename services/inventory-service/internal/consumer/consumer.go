package consumer

import (
	"context"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/controller"
	"github.com/haradrim/chainorchestra/services/inventory-service/internal/stock"
)

const queueGroup = "inventory"

type StockReleaser interface {
	ReleaseStock(ctx context.Context, req controller.ReleaseStockRequest) (stock.Stock, error)
}

type Consumer struct {
	svc StockReleaser
	nc  *natspkg.Client
	log *zap.Logger
}

func NewConsumer(svc StockReleaser, nc *natspkg.Client, log *zap.Logger) *Consumer {
	return &Consumer{svc: svc, nc: nc, log: log}
}

func (c *Consumer) Start() error {
	if _, err := c.nc.Subscribe("order.cancelled", queueGroup, c.handleOrderCancelled); err != nil {
		return err
	}

	c.log.Info("Inventory consumer started")
	return nil
}

func (c *Consumer) handleOrderCancelled(ev natspkg.Event) error {
	var data struct {
		OrderID string `json:"order_id"`
		Reason  string `json:"reason"`
	}
	if err := ev.DecodeData(&data); err != nil {
		c.log.Error("Failed to decode order.cancelled event", zap.Error(err))
		return nil
	}

	c.log.Info("Order cancelled, releasing reserved stock",
		zap.String("order_id", data.OrderID),
		zap.String("reason", data.Reason),
	)


	return nil
}
