package product

import (
	"context"
	"errors"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/pagination"
)

type Product struct {
	ID          string     `json:"id"`
	SKU         string     `json:"sku"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Category    string     `json:"category,omitempty"`
	UnitPrice   float64    `json:"unit_price"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type Filter struct {
	SKU      *string
	Name     *string
	Category *string
}

var (
	ErrProductNotFound = errors.New("product not found")
	ErrSKUExists       = errors.New("product with this SKU already exists")
)

type Storage interface {
	CreateProduct(ctx context.Context, p Product) (Product, error)
	GetProductByID(ctx context.Context, id string) (Product, error)
	ListProducts(ctx context.Context, filter Filter, sort pagination.Sort, page pagination.Page) ([]Product, int, error)
	UpdateProduct(ctx context.Context, p Product) (Product, error)
	DeleteProduct(ctx context.Context, id string) error
}
