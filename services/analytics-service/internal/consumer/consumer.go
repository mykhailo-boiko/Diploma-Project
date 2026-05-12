package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
)

type Consumer struct {
	nc   *natspkg.Client
	pool *pgxpool.Pool
	log  *zap.Logger
}

func NewConsumer(nc *natspkg.Client, pool *pgxpool.Pool, log *zap.Logger) *Consumer {
	return &Consumer{nc: nc, pool: pool, log: log}
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
		{"logistics.shipment_delivered", "analytics", c.handleLogisticsShipmentDelivered},
		{"logistics.shipment_attempted", "analytics", c.handleLogisticsShipmentAttempted},
		{"logistics.shipment_returned", "analytics", c.handleLogisticsShipmentReturned},
	}

	for _, s := range subjects {
		if _, err := c.nc.Subscribe(s.subject, s.queue, s.handler); err != nil {
			return err
		}
	}

	return nil
}

type orderCreatedData struct {
	OrderID     string  `json:"order_id"`
	TotalAmount float64 `json:"total_amount"`
}

func (c *Consumer) handleOrderCreated(ev natspkg.Event) error {
	var data orderCreatedData
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.incrSalesDaily(ctx, time.Now().UTC(), 1, data.TotalAmount); err != nil {
		c.log.Warn("incr sales_daily failed", zap.Error(err))
		return err
	}
	c.publishAggregate("revenue", "order.created", data.TotalAmount)
	return nil
}

func (c *Consumer) handleOrderStatusChanged(ev natspkg.Event) error {
	c.publishAggregate("orders", "order.status_changed", 0)
	return nil
}

func (c *Consumer) handleOrderCancelled(ev natspkg.Event) error {
	c.publishAggregate("orders", "order.cancelled", 0)
	return nil
}

func (c *Consumer) handleInventoryStockChanged(ev natspkg.Event) error {
	c.publishAggregate("inventory", "stock_changed", 0)
	return nil
}

func (c *Consumer) handleInventoryLowStock(ev natspkg.Event) error {
	c.publishAggregate("inventory", "low_stock", 0)
	return nil
}

func (c *Consumer) handleLogisticsShipmentCreated(ev natspkg.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.incrLogisticsDaily(ctx, time.Now().UTC(), 1, 0, 0); err != nil {
		c.log.Warn("incr logistics_daily failed", zap.Error(err))
		return err
	}
	c.publishAggregate("shipments", "shipment.created", 0)
	return nil
}

func (c *Consumer) handleLogisticsShipmentStatusChanged(ev natspkg.Event) error {
	c.publishAggregate("shipments", "shipment.status_changed", 0)
	return nil
}

func (c *Consumer) handleLogisticsShipmentDelivered(ev natspkg.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.incrLogisticsDaily(ctx, time.Now().UTC(), 0, 1, 0); err != nil {
		c.log.Warn("incr logistics_daily failed", zap.Error(err))
		return err
	}
	c.publishAggregate("shipments", "shipment.delivered", 0)
	return nil
}

func (c *Consumer) handleLogisticsShipmentAttempted(ev natspkg.Event) error {
	c.publishAggregate("shipments", "shipment.attempted", 0)
	return nil
}

func (c *Consumer) handleLogisticsShipmentReturned(ev natspkg.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.incrLogisticsDaily(ctx, time.Now().UTC(), 0, 0, 1); err != nil {
		c.log.Warn("incr logistics_daily failed", zap.Error(err))
		return err
	}
	c.publishAggregate("shipments", "shipment.returned", 0)
	return nil
}

func (c *Consumer) incrSalesDaily(ctx context.Context, t time.Time, deltaOrders int, deltaRevenue float64) error {
	if c.pool == nil {
		return nil
	}
	date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	_, err := c.pool.Exec(ctx, `
		INSERT INTO analytics.sales_daily (id, date, total_orders, total_revenue, avg_order_size, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (date) DO UPDATE
		SET total_orders = analytics.sales_daily.total_orders + EXCLUDED.total_orders,
		    total_revenue = analytics.sales_daily.total_revenue + EXCLUDED.total_revenue,
		    avg_order_size = CASE
		      WHEN analytics.sales_daily.total_orders + EXCLUDED.total_orders > 0
		      THEN (analytics.sales_daily.total_revenue + EXCLUDED.total_revenue) / (analytics.sales_daily.total_orders + EXCLUDED.total_orders)
		      ELSE 0
		    END,
		    updated_at = NOW()
	`, uuid.NewString(), date, deltaOrders, deltaRevenue, 0)
	return err
}

func (c *Consumer) incrLogisticsDaily(ctx context.Context, t time.Time, deltaShipments, deltaDelivered, deltaFailed int) error {
	if c.pool == nil {
		return nil
	}
	date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	_, err := c.pool.Exec(ctx, `
		INSERT INTO analytics.logistics_daily (id, date, total_shipments, delivered_count, failed_count, avg_delivery_hours, on_time_rate, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 0, 0, NOW(), NOW())
		ON CONFLICT (date) DO UPDATE
		SET total_shipments = analytics.logistics_daily.total_shipments + EXCLUDED.total_shipments,
		    delivered_count = analytics.logistics_daily.delivered_count + EXCLUDED.delivered_count,
		    failed_count = analytics.logistics_daily.failed_count + EXCLUDED.failed_count,
		    on_time_rate = CASE
		      WHEN analytics.logistics_daily.total_shipments + EXCLUDED.total_shipments > 0
		      THEN ROUND(100.0 * (analytics.logistics_daily.delivered_count + EXCLUDED.delivered_count)::numeric / (analytics.logistics_daily.total_shipments + EXCLUDED.total_shipments)::numeric, 2)
		      ELSE 0
		    END,
		    updated_at = NOW()
	`, uuid.NewString(), date, deltaShipments, deltaDelivered, deltaFailed)
	return err
}

func (c *Consumer) publishAggregate(metric, source string, delta float64) {
	if c.nc == nil {
		return
	}
	payload := map[string]any{
		"metric":    metric,
		"source":    source,
		"delta":     delta,
		"date":      time.Now().UTC().Format("2006-01-02"),
		"timestamp": time.Now().UTC(),
	}
	if err := c.nc.Publish("analytics.aggregate_updated", "analytics.aggregate_updated", payload); err != nil {
		c.log.Debug("Failed to publish analytics.aggregate_updated", zap.Error(err))
	}
}
