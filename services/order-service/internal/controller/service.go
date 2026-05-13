package controller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/audit"
	natspkg "github.com/haradrim/chainorchestra/internal/pkg/nats"
	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
	"github.com/haradrim/chainorchestra/services/order-service/internal/order"
)

type Service struct {
	storage    order.Storage
	nc         *natspkg.Client
	audit      *audit.Logger
	log        *zap.Logger
	productVal order.ProductValidator
}

func NewService(storage order.Storage, nc *natspkg.Client, auditLog *audit.Logger, log *zap.Logger) *Service {
	return &Service{storage: storage, nc: nc, audit: auditLog, log: log}
}

func (s *Service) SetProductValidator(v order.ProductValidator) {
	s.productVal = v
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
	if s.productVal != nil {
		seen := make(map[string]bool, len(req.Items))
		for _, input := range req.Items {
			if seen[input.ProductID] {
				continue
			}
			seen[input.ProductID] = true
			if err := s.productVal.ValidateProduct(ctx, input.ProductID); err != nil {
				return order.Order{}, err
			}
		}
	}

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
	s.audit.Log(ctx, audit.Entry{
		Action:       "orders.create",
		EntityType:   "order",
		EntityIDs:    []string{created.ID},
		Params:       map[string]any{"customer_name": created.CustomerName, "total_amount": created.TotalAmount, "items_count": len(created.Items)},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})

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
		return order.Order{}, &order.InvalidTransitionError{Current: string(current.Status), Requested: string(newStatus)}
	}

	updated, err := s.storage.UpdateOrderStatus(ctx, id, newStatus)
	if err != nil {
		return order.Order{}, fmt.Errorf("failed to update order status: %w", err)
	}

	s.publishOrderStatusChanged(updated, current.Status)
	s.audit.Log(ctx, audit.Entry{
		Action:       "orders.update_status",
		EntityType:   "order",
		EntityIDs:    []string{id},
		Params:       map[string]any{"old_status": current.Status, "new_status": newStatus},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})

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
	s.audit.Log(ctx, audit.Entry{
		Action:       "orders.cancel",
		EntityType:   "order",
		EntityIDs:    []string{id},
		Params:       map[string]any{"reason": reason, "previous_status": current.Status},
		ResultStatus: audit.StatusSuccess,
		SuccessCount: 1,
	})

	return cancelled, nil
}

func (s *Service) SearchOrders(ctx context.Context, query string, page pagination.Page) ([]order.Order, int, error) {
	return s.storage.SearchOrders(ctx, query, page)
}

func (s *Service) GetOrderStats(ctx context.Context) (order.OrderStats, error) {
	return s.storage.GetOrderStats(ctx)
}

func (s *Service) GetSalesByProduct(ctx context.Context, from, to time.Time, includeStatuses []order.Status) ([]order.ProductSales, error) {
	return s.storage.GetSalesByProduct(ctx, from, to, includeStatuses)
}

func (s *Service) GetCustomerSummary(ctx context.Context, filter order.CustomerFilter) ([]order.CustomerSummary, error) {
	return s.storage.GetCustomerSummary(ctx, filter)
}

func (s *Service) BulkUpdateStatus(ctx context.Context, orderIDs []string, newStatus order.Status, note string, dryRun bool) (order.BulkStatusResult, error) {
	result, err := s.storage.BulkUpdateStatus(ctx, orderIDs, newStatus, note, dryRun)
	if err != nil {
		s.audit.Log(ctx, audit.Entry{
			Action:       "orders.bulk_update_status",
			EntityType:   "order",
			EntityIDs:    orderIDs,
			Params:       map[string]any{"status": newStatus, "note": note, "dry_run": dryRun},
			ResultStatus: audit.StatusFailed,
			ErrorMessage: err.Error(),
		})
		return order.BulkStatusResult{}, err
	}
	if !dryRun {
		for _, item := range result.Successes {
			ord, getErr := s.storage.GetOrderByID(ctx, item.OrderID)
			if getErr == nil {
				s.publishOrderStatusChanged(ord, item.OldStatus)
			}
		}
	}
	status := audit.StatusSuccess
	if len(result.Failures) > 0 && len(result.Successes) > 0 {
		status = audit.StatusPartial
	} else if len(result.Successes) == 0 {
		status = audit.StatusFailed
	}
	if !dryRun {
		s.audit.Log(ctx, audit.Entry{
			Action:       "orders.bulk_update_status",
			EntityType:   "order",
			EntityIDs:    result.UpdatedIDs,
			Params:       map[string]any{"status": newStatus, "note": note, "total": result.Total},
			ResultStatus: status,
			SuccessCount: len(result.Successes),
			FailureCount: len(result.Failures),
		})
	}
	return result, nil
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
