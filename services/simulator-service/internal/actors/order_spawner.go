package actors

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/httpclient"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/metrics"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

type OrderSpawner struct {
	hc      *httpclient.Client
	catalog *Catalog
	state   *state.State
	log     *zap.Logger
	rng     *rand.Rand
	mu      sync.Mutex
}

func NewOrderSpawner(hc *httpclient.Client, catalog *Catalog, st *state.State, log *zap.Logger) *OrderSpawner {
	return &OrderSpawner{
		hc:      hc,
		catalog: catalog,
		state:   st,
		log:     log,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (a *OrderSpawner) Name() string { return "order_spawner" }

func (a *OrderSpawner) Tick(ctx context.Context) error {
	if err := a.catalog.MaybeRefresh(ctx, 5*time.Minute); err != nil {
		return fmt.Errorf("catalog refresh: %w", err)
	}

	products := a.catalog.Products()
	customers := a.catalog.Customers()
	if len(products) == 0 {
		return errors.New("no products available")
	}

	tuning := TuningFor(a.state.Scenario())
	count := tuning.OrdersPerSpawn
	if count < 1 {
		count = 1
	}

	for i := 0; i < count; i++ {
		if err := a.spawnOne(ctx, products, customers); err != nil {
			a.log.Warn("Failed to spawn order", zap.Error(err))
		}
	}
	return nil
}

func (a *OrderSpawner) spawnOne(ctx context.Context, products []productLite, customers []customerLite) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	customerName := a.pickCustomerName(customers)
	itemCount := 1 + a.rng.Intn(4)
	items := make([]map[string]any, 0, itemCount)
	used := map[string]bool{}
	for i := 0; i < itemCount; i++ {
		p := a.pickProduct(products)
		if used[p.ID] {
			continue
		}
		used[p.ID] = true
		qty := 1 + a.rng.Intn(5)
		items = append(items, map[string]any{
			"product_id": p.ID,
			"name":       p.Name,
			"quantity":   qty,
			"unit_price": p.UnitPrice,
		})
	}
	if len(items) == 0 {
		return errors.New("no items")
	}

	body := map[string]any{
		"customer_name": customerName,
		"items":         items,
	}

	var resp singleEnvelope[orderLite]
	if err := a.hc.Post(ctx, "/api/v1/orders", body, &resp); err != nil {
		return err
	}

	a.state.Counters.OrdersCreated.Add(1)
	metrics.OrdersCreated.Inc()
	a.log.Debug("Order created",
		zap.String("order_id", resp.Data.ID),
		zap.String("customer", customerName),
		zap.Float64("total", resp.Data.Total),
	)
	return nil
}

func (a *OrderSpawner) pickCustomerName(customers []customerLite) string {
	if len(customers) > 0 {
		weights := 0
		for i := range customers {
			w := customers[i].OrderCount
			if w < 1 {
				w = 1
			}
			weights += w
		}
		pick := a.rng.Intn(weights)
		for i := range customers {
			w := customers[i].OrderCount
			if w < 1 {
				w = 1
			}
			pick -= w
			if pick < 0 {
				return customers[i].CustomerName
			}
		}
	}
	first := []string{"Anton", "Bohdan", "Daryna", "Halyna", "Ivanna", "Kateryna", "Liliia", "Marko", "Nazar", "Olha", "Petro", "Roman", "Sofiia", "Taras", "Vira", "Yaroslav", "Zoryana"}
	last := []string{"Boiko", "Hrytsenko", "Koval", "Lysenko", "Marchenko", "Pavlenko", "Romanenko", "Sydorenko", "Tkachenko", "Voloshyn", "Yermak"}
	return fmt.Sprintf("%s %s", first[a.rng.Intn(len(first))], last[a.rng.Intn(len(last))])
}

func (a *OrderSpawner) pickProduct(products []productLite) productLite {
	weights := map[string]int{
		"Electronics": 18, "Clothing": 14, "Office": 12, "Food": 10, "Health": 9,
		"Furniture": 8, "Sports": 7, "Tools": 7, "Automotive": 6, "Industrial": 5, "Other": 4,
	}
	total := 0
	scored := make([]int, len(products))
	for i, p := range products {
		w, ok := weights[p.Category]
		if !ok {
			w = 4
		}
		scored[i] = w
		total += w
	}
	pick := a.rng.Intn(total)
	for i, w := range scored {
		pick -= w
		if pick < 0 {
			return products[i]
		}
	}
	return products[len(products)-1]
}
