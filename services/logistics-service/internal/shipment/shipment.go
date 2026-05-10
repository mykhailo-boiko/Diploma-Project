package shipment

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

type Status string

const (
	StatusCreated   Status = "created"
	StatusPickedUp  Status = "picked_up"
	StatusInTransit Status = "in_transit"
	StatusDelivered Status = "delivered"
	StatusFailed    Status = "failed"
	StatusReturned  Status = "returned"
)

var validTransitions = map[Status][]Status{
	StatusCreated:   {StatusPickedUp},
	StatusPickedUp:  {StatusInTransit},
	StatusInTransit: {StatusDelivered, StatusFailed},
	StatusDelivered: {StatusReturned},
	StatusFailed:    {StatusReturned},
}

func CanTransition(from, to Status) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

type Shipment struct {
	ID          string     `json:"id"`
	OrderID     string     `json:"order_id"`
	WarehouseID string     `json:"warehouse_id"`
	CarrierID   string     `json:"carrier_id"`
	Status      Status     `json:"status"`
	Address     string     `json:"address"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type Filter struct {
	Status      *Status
	CarrierID   *string
	OrderID     *string
	WarehouseID *string
	DateFrom    *time.Time
	DateTo      *time.Time
}

var (
	ErrShipmentNotFound  = errors.New("shipment not found")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type ReassignResult struct {
	Total           int      `json:"total"`
	DryRun          bool     `json:"dry_run"`
	ReassignedIDs   []string `json:"reassigned_ids"`
	FromCarrierID   string   `json:"from_carrier_id"`
	ToCarrierID     string   `json:"to_carrier_id"`
	FromCarrierName string   `json:"from_carrier_name,omitempty"`
	ToCarrierName   string   `json:"to_carrier_name,omitempty"`
	City            string   `json:"city,omitempty"`
}

type Storage interface {
	CreateShipment(ctx context.Context, s Shipment) (Shipment, error)
	GetShipmentByID(ctx context.Context, id string) (Shipment, error)
	ListShipments(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Shipment, int, error)
	UpdateShipmentStatus(ctx context.Context, id string, status Status) (Shipment, error)
	ReassignCarrierByCity(ctx context.Context, fromCarrierID, toCarrierID, city string, statuses []Status, dryRun bool) (ReassignResult, error)
}
