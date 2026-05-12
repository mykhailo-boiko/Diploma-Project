package shipment

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

type Status string

const (
	StatusCreated            Status = "created"
	StatusLabelCreated       Status = "label_created"
	StatusAwaitingPickup     Status = "awaiting_pickup"
	StatusPickedUp           Status = "picked_up"
	StatusInTransit          Status = "in_transit"
	StatusAtHub              Status = "at_hub"
	StatusOutForDelivery     Status = "out_for_delivery"
	StatusDeliveryAttempted  Status = "delivery_attempted"
	StatusHeldAtOffice       Status = "held_at_office"
	StatusDelivered          Status = "delivered"
	StatusFailed             Status = "failed"
	StatusReturnedToSender   Status = "returned_to_sender"
	StatusReturned           Status = "returned"
	StatusCancelled          Status = "cancelled"
	StatusRedirected         Status = "redirected"
)

var validTransitions = map[Status][]Status{
	StatusCreated:           {StatusLabelCreated, StatusAwaitingPickup, StatusPickedUp, StatusCancelled},
	StatusLabelCreated:      {StatusAwaitingPickup, StatusPickedUp, StatusCancelled},
	StatusAwaitingPickup:    {StatusPickedUp, StatusCancelled},
	StatusPickedUp:          {StatusInTransit, StatusAtHub, StatusCancelled},
	StatusInTransit:         {StatusAtHub, StatusOutForDelivery, StatusDelivered, StatusFailed, StatusRedirected},
	StatusAtHub:             {StatusInTransit, StatusOutForDelivery, StatusRedirected, StatusReturnedToSender},
	StatusOutForDelivery:    {StatusDelivered, StatusDeliveryAttempted, StatusHeldAtOffice, StatusFailed},
	StatusDeliveryAttempted: {StatusOutForDelivery, StatusHeldAtOffice, StatusReturnedToSender, StatusDelivered},
	StatusHeldAtOffice:      {StatusDelivered, StatusReturnedToSender},
	StatusDelivered:         {StatusReturned},
	StatusFailed:            {StatusReturned, StatusReturnedToSender},
	StatusRedirected:        {StatusInTransit, StatusAtHub, StatusOutForDelivery},
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

type Address struct {
	FullName       string `json:"full_name,omitempty"`
	Phone          string `json:"phone,omitempty"`
	Email          string `json:"email,omitempty"`
	Company        string `json:"company,omitempty"`
	Street         string `json:"street,omitempty"`
	City           string `json:"city,omitempty"`
	Region         string `json:"region,omitempty"`
	Postcode       string `json:"postcode,omitempty"`
	Country        string `json:"country,omitempty"`
	DeliveryNotes  string `json:"delivery_notes,omitempty"`
}

type Shipment struct {
	ID                  string     `json:"id"`
	OrderID             string     `json:"order_id"`
	WarehouseID         string     `json:"warehouse_id"`
	CarrierID           string     `json:"carrier_id"`
	Status              Status     `json:"status"`
	TrackingNumber      string     `json:"tracking_number"`
	Address             string     `json:"address"`
	Recipient           Address    `json:"recipient"`
	Sender              Address    `json:"sender"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at,omitempty"`
	DeliveredAt         *time.Time `json:"delivered_at,omitempty"`
	DeliveryAttempts    int        `json:"delivery_attempts"`
	DeliverySignature   string     `json:"delivery_signature,omitempty"`
	DeliveryPhotoURL    string     `json:"delivery_photo_url,omitempty"`
	CurrentLocationCity string     `json:"current_location_city,omitempty"`
	CurrentLocationHub  string     `json:"current_location_hub,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
}

type ShipmentEvent struct {
	ID           string         `json:"id"`
	ShipmentID   string         `json:"shipment_id"`
	Type         string         `json:"event_type"`
	LocationCity string         `json:"location_city,omitempty"`
	LocationHub  string         `json:"location_hub,omitempty"`
	Notes        string         `json:"notes,omitempty"`
	OccurredAt   time.Time      `json:"occurred_at"`
	RecordedBy   string         `json:"recorded_by"`
	Payload      map[string]any `json:"payload,omitempty"`
}

type DeliveryAttempt struct {
	ID            string     `json:"id"`
	ShipmentID    string     `json:"shipment_id"`
	AttemptNumber int        `json:"attempt_number"`
	Reason        string     `json:"reason"`
	Notes         string     `json:"notes,omitempty"`
	NextAttemptAt *time.Time `json:"next_attempt_at,omitempty"`
	OccurredAt    time.Time  `json:"occurred_at"`
}

type Timeline struct {
	Shipment         Shipment          `json:"shipment"`
	Events           []ShipmentEvent   `json:"events"`
	DeliveryAttempts []DeliveryAttempt `json:"delivery_attempts"`
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

type RecipientPatch struct {
	FullName      *string `json:"full_name,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Email         *string `json:"email,omitempty"`
	Company       *string `json:"company,omitempty"`
	Street        *string `json:"street,omitempty"`
	City          *string `json:"city,omitempty"`
	Region        *string `json:"region,omitempty"`
	Postcode      *string `json:"postcode,omitempty"`
	Country       *string `json:"country,omitempty"`
	DeliveryNotes *string `json:"delivery_notes,omitempty"`
}

type Storage interface {
	CreateShipment(ctx context.Context, s Shipment) (Shipment, error)
	GetShipmentByID(ctx context.Context, id string) (Shipment, error)
	GetShipmentByTracking(ctx context.Context, trackingNumber string) (Shipment, error)
	ListShipments(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Shipment, int, error)
	UpdateShipmentStatus(ctx context.Context, id string, status Status) (Shipment, error)
	ReassignCarrierByCity(ctx context.Context, fromCarrierID, toCarrierID, city string, statuses []Status, dryRun bool) (ReassignResult, error)

	RecordEvent(ctx context.Context, e ShipmentEvent) (ShipmentEvent, error)
	GetTimeline(ctx context.Context, shipmentID string) ([]ShipmentEvent, error)
	UpdateRecipient(ctx context.Context, id string, patch RecipientPatch) (Shipment, error)
	UpdateSender(ctx context.Context, id string, patch RecipientPatch) (Shipment, error)
	UpdateEstimatedDelivery(ctx context.Context, id string, eta time.Time) (Shipment, error)
	UpdateCurrentLocation(ctx context.Context, id, city, hub string) (Shipment, error)
	RecordDelivery(ctx context.Context, id, signature, photoURL string) (Shipment, error)
	RecordDeliveryAttempt(ctx context.Context, attempt DeliveryAttempt) (DeliveryAttempt, error)
	GetDeliveryAttempts(ctx context.Context, shipmentID string) ([]DeliveryAttempt, error)
	RedirectAddress(ctx context.Context, id string, newAddress Address, reason string) (Shipment, error)
}
