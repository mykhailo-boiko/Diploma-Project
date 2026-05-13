package actors

import (
	"context"
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

type OrderProgressor struct {
	hc      *httpclient.Client
	catalog *Catalog
	state   *state.State
	log     *zap.Logger
	rng     *rand.Rand
	mu      sync.Mutex
}

func NewOrderProgressor(hc *httpclient.Client, catalog *Catalog, st *state.State, log *zap.Logger) *OrderProgressor {
	return &OrderProgressor{
		hc:      hc,
		catalog: catalog,
		state:   st,
		log:     log,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano() + 1)),
	}
}

func (a *OrderProgressor) Name() string { return "order_progressor" }

type orderProgressTransition struct {
	from         string
	to           string
	minAgeSec    int
	cancelChance float64
}

func (a *OrderProgressor) Tick(ctx context.Context) error {
	tuning := TuningFor(a.state.Scenario())
	speed := a.state.Speed()
	if speed <= 0 {
		speed = 1
	}

	mins := func(m int) int {
		v := int(float64(m*60) / speed)
		if v < 10 {
			v = 10
		}
		return v
	}

	transitions := []orderProgressTransition{
		{from: "pending", to: "confirmed", minAgeSec: mins(1), cancelChance: tuning.OrderCancelRate / 2},
		{from: "confirmed", to: "processing", minAgeSec: mins(2), cancelChance: tuning.OrderCancelRate / 3},
		{from: "processing", to: "shipped", minAgeSec: mins(5), cancelChance: tuning.OrderCancelRate / 4},
		{from: "shipped", to: "delivered", minAgeSec: mins(30), cancelChance: 0.01},
	}

	for _, t := range transitions {
		if err := a.processBatch(ctx, t); err != nil {
			a.log.Warn("Order batch failed", zap.String("from", t.from), zap.Error(err))
		}
	}
	return nil
}

func (a *OrderProgressor) processBatch(ctx context.Context, t orderProgressTransition) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	q := url.Values{}
	q.Set("status", t.from)
	q.Set("limit", "20")
	q.Set("sort_by", "updated_at")
	q.Set("sort_order", "asc")

	var env listEnvelope[orderLite]
	if err := a.hc.Get(ctx, "/api/v1/orders", q, &env); err != nil {
		return fmt.Errorf("list orders: %w", err)
	}

	if len(env.Data) == 0 {
		return nil
	}

	now := time.Now()
	maxBatch := 5
	advanced := 0

	for _, o := range env.Data {
		if advanced >= maxBatch {
			break
		}
		age := now.Sub(o.UpdatedAt)
		if age.Seconds() < float64(t.minAgeSec) {
			continue
		}
		if a.rng.Float64() < t.cancelChance && t.to != "delivered" {
			if err := a.cancelOrder(ctx, o.ID); err != nil {
				a.log.Debug("Cancel failed", zap.String("order_id", o.ID), zap.Error(err))
				continue
			}
			a.state.Counters.OrdersCancelled.Add(1)
			metrics.OrdersProgressed.WithLabelValues(t.from, "cancelled").Inc()
			advanced++
			continue
		}

		if err := a.updateStatus(ctx, o.ID, t.to); err != nil {
			a.log.Debug("Status update failed", zap.String("order_id", o.ID), zap.String("to", t.to), zap.Error(err))
			continue
		}

		if t.to == "shipped" {
			if err := a.maybeCreateShipment(ctx, o.ID, o.CustomerName); err != nil {
				a.log.Debug("Shipment create failed", zap.String("order_id", o.ID), zap.Error(err))
			}
		}

		a.state.Counters.OrdersProgressed.Add(1)
		metrics.OrdersProgressed.WithLabelValues(t.from, t.to).Inc()
		advanced++
	}
	return nil
}

func (a *OrderProgressor) updateStatus(ctx context.Context, id, status string) error {
	body := map[string]string{"status": status}
	return a.hc.Put(ctx, "/api/v1/orders/"+id+"/status", body, nil)
}

func (a *OrderProgressor) cancelOrder(ctx context.Context, id string) error {
	body := map[string]string{"reason": "simulator: stalled in pipeline"}
	return a.hc.Post(ctx, "/api/v1/orders/"+id+"/cancel", body, nil)
}

func (a *OrderProgressor) maybeCreateShipment(ctx context.Context, orderID, customerName string) error {
	warehouses := a.catalog.Warehouses()
	carriers := a.catalog.Carriers()
	if len(warehouses) == 0 || len(carriers) == 0 {
		return fmt.Errorf("catalog empty")
	}
	var activeCarriers []carrierLite
	for _, c := range carriers {
		if c.IsActive {
			activeCarriers = append(activeCarriers, c)
		}
	}
	if len(activeCarriers) == 0 {
		activeCarriers = carriers
	}
	w := warehouses[a.rng.Intn(len(warehouses))]
	c := activeCarriers[a.rng.Intn(len(activeCarriers))]
	cities := []string{"Kyiv", "Lviv", "Odesa", "Dnipro", "Kharkiv", "Vinnytsia", "Poltava", "Zaporizhzhia", "Cherkasy", "Chernihiv"}
	streets := []string{"Khreshchatyk", "Sahaidachnoho", "Shevchenka", "Franka", "Lesi Ukrainky", "Bandery", "Hrushevskoho"}
	address := fmt.Sprintf("%s str. %d, %s",
		streets[a.rng.Intn(len(streets))],
		1+a.rng.Intn(180),
		cities[a.rng.Intn(len(cities))],
	)
	body := map[string]any{
		"order_id":     orderID,
		"warehouse_id": w.ID,
		"carrier_id":   c.ID,
		"address":      address,
	}
	if customerName != "" {
		body["recipient_name"] = customerName
	}
	return a.hc.Post(ctx, "/api/v1/shipments", body, nil)
}
