package actors

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/httpclient"
)

type Catalog struct {
	hc *httpclient.Client

	mu          sync.RWMutex
	products    []productLite
	warehouses  []warehouseLite
	carriers    []carrierLite
	customers   []customerLite
	lastFetched time.Time
}

func NewCatalog(hc *httpclient.Client) *Catalog {
	return &Catalog{hc: hc}
}

func (c *Catalog) Refresh(ctx context.Context) error {
	products, err := c.fetchProducts(ctx)
	if err != nil {
		return err
	}
	warehouses, err := c.fetchWarehouses(ctx)
	if err != nil {
		return err
	}
	carriers, err := c.fetchCarriers(ctx)
	if err != nil {
		return err
	}
	customers, err := c.fetchCustomers(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.products = products
	c.warehouses = warehouses
	c.carriers = carriers
	c.customers = customers
	c.lastFetched = time.Now()
	c.mu.Unlock()
	return nil
}

func (c *Catalog) MaybeRefresh(ctx context.Context, ttl time.Duration) error {
	c.mu.RLock()
	fresh := time.Since(c.lastFetched) < ttl && len(c.products) > 0
	c.mu.RUnlock()
	if fresh {
		return nil
	}
	return c.Refresh(ctx)
}

func (c *Catalog) Products() []productLite {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]productLite, len(c.products))
	copy(out, c.products)
	return out
}

func (c *Catalog) Warehouses() []warehouseLite {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]warehouseLite, len(c.warehouses))
	copy(out, c.warehouses)
	return out
}

func (c *Catalog) Carriers() []carrierLite {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]carrierLite, len(c.carriers))
	copy(out, c.carriers)
	return out
}

func (c *Catalog) Customers() []customerLite {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]customerLite, len(c.customers))
	copy(out, c.customers)
	return out
}

func (c *Catalog) fetchProducts(ctx context.Context) ([]productLite, error) {
	q := url.Values{}
	q.Set("limit", "500")
	var env listEnvelope[productLite]
	if err := c.hc.Get(ctx, "/api/v1/products", q, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Catalog) fetchWarehouses(ctx context.Context) ([]warehouseLite, error) {
	q := url.Values{}
	q.Set("limit", "50")
	var env listEnvelope[warehouseLite]
	if err := c.hc.Get(ctx, "/api/v1/warehouses", q, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Catalog) fetchCarriers(ctx context.Context) ([]carrierLite, error) {
	q := url.Values{}
	q.Set("limit", "50")
	var env listEnvelope[carrierLite]
	if err := c.hc.Get(ctx, "/api/v1/carriers", q, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Catalog) fetchCustomers(ctx context.Context) ([]customerLite, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(60))
	var env listEnvelope[customerLite]
	if err := c.hc.Get(ctx, "/api/v1/orders/customers", q, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}
