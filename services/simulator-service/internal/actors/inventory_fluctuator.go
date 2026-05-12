package actors

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/httpclient"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/metrics"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

type InventoryFluctuator struct {
	hc      *httpclient.Client
	catalog *Catalog
	state   *state.State
	log     *zap.Logger
	rng     *rand.Rand
	mu      sync.Mutex
}

func NewInventoryFluctuator(hc *httpclient.Client, catalog *Catalog, st *state.State, log *zap.Logger) *InventoryFluctuator {
	return &InventoryFluctuator{
		hc:      hc,
		catalog: catalog,
		state:   st,
		log:     log,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano() + 3)),
	}
}

func (a *InventoryFluctuator) Name() string { return "inventory_fluctuator" }

func (a *InventoryFluctuator) Tick(ctx context.Context) error {
	if err := a.catalog.MaybeRefresh(ctx, 5*time.Minute); err != nil {
		return fmt.Errorf("catalog refresh: %w", err)
	}
	products := a.catalog.Products()
	warehouses := a.catalog.Warehouses()
	if len(products) == 0 || len(warehouses) == 0 {
		return errors.New("catalog empty")
	}

	tuning := TuningFor(a.state.Scenario())
	batch := tuning.StockBatchSize
	if batch <= 0 {
		batch = 3
	}

	for i := 0; i < batch; i++ {
		if err := a.adjustOne(ctx, products, warehouses); err != nil {
			a.log.Debug("Adjust failed", zap.Error(err))
		}
	}
	return nil
}

func (a *InventoryFluctuator) adjustOne(ctx context.Context, products []productLite, warehouses []warehouseLite) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	roll := a.rng.Float64()
	var kind, label string
	var qty int

	switch {
	case roll < 0.55:
		kind = "inbound"
		label = "restock"
		qty = 5 + a.rng.Intn(46)
	case roll < 0.85:
		kind = "outbound"
		label = "damage"
		qty = 1 + a.rng.Intn(10)
	default:
		kind = "inbound"
		label = "supply"
		qty = 200 + a.rng.Intn(801)
	}

	var sk stockLite
	for attempt := 0; attempt < 5; attempt++ {
		p := products[a.rng.Intn(len(products))]
		w := warehouses[a.rng.Intn(len(warehouses))]
		s, err := a.fetchStock(ctx, p.ID, w.ID)
		if err == nil {
			sk = s
			break
		}
	}
	if sk.ProductID == "" {
		return errors.New("no stock record")
	}

	if kind == "outbound" {
		available := sk.Quantity - sk.Reserved
		if qty > available {
			qty = available
		}
	}
	if qty <= 0 {
		return nil
	}

	body := map[string]any{
		"product_id":   sk.ProductID,
		"warehouse_id": sk.WarehouseID,
		"quantity":     qty,
		"type":         kind,
		"reference":    fmt.Sprintf("simulator-%s-%d", label, time.Now().UnixNano()),
	}
	if err := a.hc.Post(ctx, "/api/v1/stock/adjust", body, nil); err != nil {
		return err
	}
	a.state.Counters.StockAdjustments.Add(1)
	metrics.StockAdjusted.WithLabelValues(label).Inc()
	return nil
}

func (a *InventoryFluctuator) fetchStock(ctx context.Context, productID, warehouseID string) (stockLite, error) {
	q := url.Values{}
	q.Set("product_id", productID)
	q.Set("warehouse_id", warehouseID)
	q.Set("limit", "1")
	var env listEnvelope[stockLite]
	if err := a.hc.Get(ctx, "/api/v1/stock", q, &env); err != nil {
		return stockLite{}, err
	}
	if len(env.Data) == 0 {
		return stockLite{}, errors.New("no stock found")
	}
	return env.Data[0], nil
}
