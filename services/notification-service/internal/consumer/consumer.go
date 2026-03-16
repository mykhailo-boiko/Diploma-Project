package consumer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/controller"
	"github.com/haradrim/chainorchestra/services/notification-service/internal/notification"
)

const queueGroup = "notifications"

type NotificationSendData struct {
	UserID  string `json:"user_id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type NotificationCreator interface {
	CreateNotification(ctx context.Context, req controller.CreateNotificationRequest) (notification.Notification, error)
}

type Consumer struct {
	svc NotificationCreator
	nc  *natspkg.Client
	log *zap.Logger
}

func NewConsumer(svc NotificationCreator, nc *natspkg.Client, log *zap.Logger) *Consumer {
	return &Consumer{svc: svc, nc: nc, log: log}
}

func (c *Consumer) Start() error {
	subjects := []struct {
		subject string
		handler natspkg.Handler
	}{
		{"events.notifications.send", c.handleNotificationSend},
		{"order.created", c.handleOrderCreated},
		{"order.status_changed", c.handleOrderStatusChanged},
		{"order.cancelled", c.handleOrderCancelled},
		{"inventory.low_stock", c.handleInventoryLowStock},
		{"logistics.shipment_created", c.handleLogisticsShipmentCreated},
		{"logistics.shipment_status_changed", c.handleLogisticsShipmentStatusChanged},
	}

	for _, s := range subjects {
		if _, err := c.nc.Subscribe(s.subject, queueGroup, s.handler); err != nil {
			return err
		}
	}

	c.log.Info("Notification consumer started")
	return nil
}

func (c *Consumer) broadcastNotification(notifType, title, message string) {
	_, err := c.svc.CreateNotification(context.Background(), controller.CreateNotificationRequest{
		UserID:  "system",
		Type:    notification.Type(notifType),
		Title:   title,
		Message: message,
	})
	if err != nil {
		c.log.Error("Failed to create notification from domain event",
			zap.String("type", notifType),
			zap.Error(err),
		)
	}
}

func (c *Consumer) handleOrderCreated(ev natspkg.Event) error {
	var data struct {
		OrderID      string  `json:"order_id"`
		CustomerName string  `json:"customer_name"`
		TotalAmount  float64 `json:"total_amount"`
	}
	if err := ev.DecodeData(&data); err != nil {
		c.log.Error("Failed to decode order.created event", zap.Error(err))
		return nil
	}

	c.broadcastNotification(
		string(notification.TypeOrderCreated),
		"New order created",
		fmt.Sprintf("Order %s from %s (total: %.2f)", data.OrderID, data.CustomerName, data.TotalAmount),
	)
	return nil
}

func (c *Consumer) handleOrderStatusChanged(ev natspkg.Event) error {
	var data struct {
		OrderID        string `json:"order_id"`
		PreviousStatus string `json:"previous_status"`
		NewStatus      string `json:"new_status"`
	}
	if err := ev.DecodeData(&data); err != nil {
		c.log.Error("Failed to decode order.status_changed event", zap.Error(err))
		return nil
	}

	c.broadcastNotification(
		string(notification.TypeOrderUpdated),
		"Order status changed",
		fmt.Sprintf("Order %s: %s → %s", data.OrderID, data.PreviousStatus, data.NewStatus),
	)
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

	c.broadcastNotification(
		string(notification.TypeOrderCancelled),
		"Order cancelled",
		fmt.Sprintf("Order %s cancelled: %s", data.OrderID, data.Reason),
	)
	return nil
}

func (c *Consumer) handleInventoryLowStock(ev natspkg.Event) error {
	var data struct {
		ProductID    string `json:"product_id"`
		WarehouseID  string `json:"warehouse_id"`
		Available    int    `json:"available"`
		MinThreshold int    `json:"min_threshold"`
	}
	if err := ev.DecodeData(&data); err != nil {
		c.log.Error("Failed to decode inventory.low_stock event", zap.Error(err))
		return nil
	}

	c.broadcastNotification(
		string(notification.TypeLowStock),
		"Low stock alert",
		fmt.Sprintf("Product %s in warehouse %s: %d available (threshold: %d)", data.ProductID, data.WarehouseID, data.Available, data.MinThreshold),
	)
	return nil
}

func (c *Consumer) handleLogisticsShipmentCreated(ev natspkg.Event) error {
	var data struct {
		ShipmentID string `json:"shipment_id"`
		OrderID    string `json:"order_id"`
	}
	if err := ev.DecodeData(&data); err != nil {
		c.log.Error("Failed to decode logistics.shipment_created event", zap.Error(err))
		return nil
	}

	c.broadcastNotification(
		string(notification.TypeShipmentCreated),
		"Shipment created",
		fmt.Sprintf("Shipment %s created for order %s", data.ShipmentID, data.OrderID),
	)
	return nil
}

func (c *Consumer) handleLogisticsShipmentStatusChanged(ev natspkg.Event) error {
	var data struct {
		ShipmentID     string `json:"shipment_id"`
		OrderID        string `json:"order_id"`
		PreviousStatus string `json:"previous_status"`
		NewStatus      string `json:"new_status"`
	}
	if err := ev.DecodeData(&data); err != nil {
		c.log.Error("Failed to decode logistics.shipment_status_changed event", zap.Error(err))
		return nil
	}

	c.broadcastNotification(
		string(notification.TypeShipmentUpdated),
		"Shipment status changed",
		fmt.Sprintf("Shipment %s (order %s): %s → %s", data.ShipmentID, data.OrderID, data.PreviousStatus, data.NewStatus),
	)
	return nil
}

func (c *Consumer) handleNotificationSend(ev natspkg.Event) error {
	var data NotificationSendData
	if err := ev.DecodeData(&data); err != nil {
		c.log.Error("Failed to decode notification send event", zap.Error(err))
		return err
	}

	c.log.Info("Received notification send event",
		zap.String("event_id", ev.ID),
		zap.String("user_id", data.UserID),
		zap.String("type", data.Type),
	)

	_, err := c.svc.CreateNotification(context.Background(), controller.CreateNotificationRequest{
		UserID:  data.UserID,
		Type:    notification.Type(data.Type),
		Title:   data.Title,
		Message: data.Message,
	})
	if err != nil {
		c.log.Error("Failed to create notification from event",
			zap.String("event_id", ev.ID),
			zap.Error(err),
		)
		return err
	}

	return nil
}
