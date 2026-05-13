package carrier

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

type Type string

const (
	TypeGround Type = "ground"
	TypeAir    Type = "air"
	TypeSea    Type = "sea"
)

func ValidType(t Type) bool {
	switch t {
	case TypeGround, TypeAir, TypeSea:
		return true
	}
	return false
}

type Carrier struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       Type      `json:"type"`
	CostPerKm  float64   `json:"cost_per_km"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Filter struct {
	Type     *Type
	IsActive *bool
	Name     *string
}

var (
	ErrCarrierNotFound      = errors.New("carrier not found")
	ErrNoActiveCarrierFound = errors.New("no active carrier available")
)

type Storage interface {
	CreateCarrier(ctx context.Context, c Carrier) (Carrier, error)
	GetCarrierByID(ctx context.Context, id string) (Carrier, error)
	ListCarriers(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Carrier, int, error)
	UpdateCarrier(ctx context.Context, c Carrier) (Carrier, error)
	PickDefaultActive(ctx context.Context) (Carrier, error)
}
