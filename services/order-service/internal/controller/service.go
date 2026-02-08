package controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/order-service/internal/order"
)

type Service struct {
	storage order.Storage
	nc      *natspkg.Client
	log     *zap.Logger
}

func NewService(storage order.Storage, nc *natspkg.Client, log *zap.Logger) *Service {
	return &Service{storage: storage, nc: nc, log: log}
}

type CreateOrderRequest struct {
	CustomerName string            `json:"customer_name"`
	Items        []CreateItemInput `json:"items"`
}

type CreateItemInput struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type UpdateStatusRequest struct {
	Status order.Status `json:"status"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest) (order.Order, error) {
	items := make([]order.Item, 0, len(req.Items))
	for _, input := range req.Items {
		items = append(items, order.Item{
			ProductID: input.ProductID,
			Name:      input.Name,
			Quantity:  input.Quantity,
			UnitPrice: input.UnitPrice,
		})
	}

	o := order.Order{
		CustomerName: req.CustomerName,
		Items:        items,
	}

	created, err := s.storage.CreateOrder(ctx, o)
	if err != nil {
		return order.Order{}, fmt.Errorf("failed to create order: %w", err)
	}

	s.publishOrderCreated(created)

	return created, nil
}

func (s *Service) GetOrderByID(ctx context.Context, id string) (order.Order, error) {
	return s.storage.GetOrderByID(ctx, id)
}

func (s *Service) ListOrders(ctx context.Context, filter order.Filter, sort pagination.Sort, page pagination.Page) ([]order.Order, int, error) {
	return s.storage.ListOrders(ctx, filter, sort, page)
}

func (s *Service) UpdateOrderStatus(ctx context.Context, id string, newStatus order.Status) (order.Order, error) {
	current, err := s.storage.GetOrderByID(ctx, id)
	if err != nil {
		return order.Order{}, err
	}

	if !order.CanTransition(current.Status, newStatus) {
		return order.Order{}, order.ErrInvalidTransition
	}

	updated, err := s.storage.UpdateOrderStatus(ctx, id, newStatus)
	if err != nil {
		return order.Order{}, fmt.Errorf("failed to update order status: %w", err)
	}

	s.publishOrderStatusChanged(updated, current.Status)

	return updated, nil
}

func (s *Service) CancelOrder(ctx context.Context, id string, reason string) (order.Order, error) {
	current, err := s.storage.GetOrderByID(ctx, id)
	if err != nil {
		return order.Order{}, err
	}

	if !order.CanTransition(current.Status, order.StatusCancelled) {
		return order.Order{}, order.ErrInvalidTransition
	}

	cancelled, err := s.storage.CancelOrder(ctx, id, reason)
	if err != nil {
		return order.Order{}, fmt.Errorf("failed to cancel order: %w", err)
	}

	s.publishOrderCancelled(cancelled, reason)

	return cancelled, nil
}

func (s *Service) SearchOrders(ctx context.Context, query string, page pagination.Page) ([]order.Order, int, error) {
	return s.storage.SearchOrders(ctx, query, page)
}

func (s *Service) GetOrderStats(ctx context.Context) (order.OrderStats, error) {
	return s.storage.GetOrderStats(ctx)
}


func (s *Service) publishOrderCreated(o order.Order) {
	if s.nc == nil {
		return
	}

	data := map[string]any{
		"order_id":      o.ID,
		"customer_name": o.CustomerName,
		"status":        string(o.Status),
		"total_amount":  o.TotalAmount,
		"item_count":    len(o.Items),
	}

	if err := s.nc.Publish("order.created", "order.created", data); err != nil {
		s.log.Error("Failed to publish order.created", zap.String("order_id", o.ID), zap.Error(err))
	}
}

func (s *Service) publishOrderStatusChanged(o order.Order, previousStatus order.Status) {
	if s.nc == nil {
		return
	}

	data := map[string]any{
		"order_id":        o.ID,
		"previous_status": string(previousStatus),
		"new_status":      string(o.Status),
	}

	if err := s.nc.Publish("order.status_changed", "order.status_changed", data); err != nil {
		s.log.Error("Failed to publish order.status_changed", zap.String("order_id", o.ID), zap.Error(err))
	}
}

func (s *Service) publishOrderCancelled(o order.Order, reason string) {
	if s.nc == nil {
		return
	}

	data := map[string]any{
		"order_id":     o.ID,
		"reason":       reason,
		"total_amount": o.TotalAmount,
	}

	if err := s.nc.Publish("order.cancelled", "order.cancelled", data); err != nil {
		s.log.Error("Failed to publish order.cancelled", zap.String("order_id", o.ID), zap.Error(err))
	}
}
