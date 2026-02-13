package warehouse

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

type Warehouse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Filter struct {
	Name     *string
	IsActive *bool
}

var (
	ErrWarehouseNotFound = errors.New("warehouse not found")
)

type Storage interface {
	CreateWarehouse(ctx context.Context, w Warehouse) (Warehouse, error)
	GetWarehouseByID(ctx context.Context, id string) (Warehouse, error)
	ListWarehouses(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Warehouse, int, error)
	UpdateWarehouse(ctx context.Context, w Warehouse) (Warehouse, error)
}
