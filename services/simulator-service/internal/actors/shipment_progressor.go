package actors

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/httpclient"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/metrics"
	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

type ShipmentProgressor struct {
	hc    *httpclient.Client
	state *state.State
	log   *zap.Logger
	rng   *rand.Rand
	mu    sync.Mutex
}

func NewShipmentProgressor(hc *httpclient.Client, st *state.State, log *zap.Logger) *ShipmentProgressor {
	return &ShipmentProgressor{
		hc:    hc,
		state: st,
		log:   log,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano() + 2)),
	}
}

func (a *ShipmentProgressor) Name() string { return "shipment_progressor" }

var transitStatuses = []string{
	"created", "label_created", "awaiting_pickup", "picked_up",
	"in_transit", "at_hub", "out_for_delivery", "delivery_attempted", "held_at_office",
}

var transitHubs = []string{
	"Kyiv North Hub", "Kyiv South Hub", "Lviv West Hub", "Odesa Port Hub",
	"Dnipro Central Hub", "Kharkiv East Hub", "Vinnytsia Hub",
}

var transitCities = []string{
	"Kyiv", "Lviv", "Odesa", "Dnipro", "Kharkiv", "Vinnytsia", "Poltava",
	"Zaporizhzhia", "Cherkasy", "Chernihiv", "Rivne", "Ternopil", "Mykolaiv",
}

func (a *ShipmentProgressor) Tick(ctx context.Context) error {
	tuning := TuningFor(a.state.Scenario())
	batch := tuning.ShipmentBatchSize
	if batch <= 0 {
		batch = 6
	}

	for _, status := range transitStatuses {
		if err := a.processStatus(ctx, status, batch, tuning); err != nil {
			a.log.Warn("Shipment batch failed", zap.String("status", status), zap.Error(err))
		}
	}
	return nil
}

func (a *ShipmentProgressor) processStatus(ctx context.Context, status string, batch int, tuning ScenarioTuning) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	q := url.Values{}
	q.Set("status", status)
	q.Set("limit", fmt.Sprintf("%d", batch*2))
	q.Set("sort_by", "updated_at")
	q.Set("sort_order", "asc")

	var env listEnvelope[shipmentLite]
	if err := a.hc.Get(ctx, "/api/v1/shipments", q, &env); err != nil {
		return fmt.Errorf("list shipments: %w", err)
	}

	speed := a.state.Speed()
	if speed <= 0 {
		speed = 1
	}
	minAge := time.Duration(float64(60*time.Second) / speed)
	if minAge < 5*time.Second {
		minAge = 5 * time.Second
	}

	advanced := 0
	for _, sh := range env.Data {
		if advanced >= batch {
			break
		}
		if time.Since(sh.UpdatedAt) < minAge {
			continue
		}
		if err := a.advance(ctx, sh, tuning); err != nil {
			a.log.Debug("Advance failed", zap.String("shipment_id", sh.ID), zap.String("status", sh.Status), zap.Error(err))
			continue
		}
		advanced++
	}
	return nil
}

func (a *ShipmentProgressor) advance(ctx context.Context, sh shipmentLite, tuning ScenarioTuning) error {
	switch sh.Status {
	case "created", "label_created":
		return a.statusTo(ctx, sh, "awaiting_pickup", "Awaiting pickup at warehouse", a.warehouseCity(sh), "")

	case "awaiting_pickup":
		return a.statusTo(ctx, sh, "picked_up", "Picked up by courier", a.warehouseCity(sh), "")

	case "picked_up":
		hub := transitHubs[a.rng.Intn(len(transitHubs))]
		return a.addEventThenStatus(ctx, sh, "in_transit", "In transit", a.transitCityFor(sh), hub)

	case "in_transit":
		if a.rng.Float64() < 0.4 {
			hub := transitHubs[a.rng.Intn(len(transitHubs))]
			return a.statusTo(ctx, sh, "at_hub", "Arrived at sorting hub", a.transitCityFor(sh), hub)
		}
		return a.statusTo(ctx, sh, "out_for_delivery", "Out for delivery", a.destinationCity(sh), "")

	case "at_hub":
		return a.statusTo(ctx, sh, "out_for_delivery", "Out for delivery from hub", a.destinationCity(sh), "")

	case "out_for_delivery":
		roll := a.rng.Float64()
		if roll < tuning.FailedDeliveryRate {
			return a.recordAttempt(ctx, sh)
		}
		return a.recordDelivery(ctx, sh)

	case "delivery_attempted":
		if sh.DeliveryAttempts >= 3 {
			return a.statusTo(ctx, sh, "returned_to_sender", "Returned to sender after 3 failed attempts", a.destinationCity(sh), "")
		}
		if a.rng.Float64() < 0.5 {
			return a.statusTo(ctx, sh, "out_for_delivery", "Retrying delivery", a.destinationCity(sh), "")
		}
		return a.statusTo(ctx, sh, "held_at_office", "Held at post office for pickup", a.destinationCity(sh), "")

	case "held_at_office":
		if a.rng.Float64() < 0.6 {
			return a.recordDelivery(ctx, sh)
		}
		return a.statusTo(ctx, sh, "returned_to_sender", "Not picked up — returning to sender", a.destinationCity(sh), "")
	}
	return nil
}

func (a *ShipmentProgressor) statusTo(ctx context.Context, sh shipmentLite, status, note, city, hub string) error {
	body := map[string]any{"status": status}
	if err := a.hc.Put(ctx, "/api/v1/shipments/"+sh.ID+"/status", body, nil); err != nil {
		return err
	}
	metrics.ShipmentsProgressed.WithLabelValues(sh.Status, status).Inc()
	a.state.Counters.ShipmentsAdvanced.Add(1)
	if city != "" || hub != "" {
		_ = a.addEvent(ctx, sh.ID, "status_change", city, hub, note)
	}
	return nil
}

func (a *ShipmentProgressor) addEventThenStatus(ctx context.Context, sh shipmentLite, status, eventType, city, hub string) error {
	if err := a.addEvent(ctx, sh.ID, eventType, city, hub, eventType); err != nil {
		a.log.Debug("Add event failed", zap.Error(err))
	}
	return a.statusTo(ctx, sh, status, eventType, city, hub)
}

func (a *ShipmentProgressor) addEvent(ctx context.Context, shipmentID, eventType, city, hub, notes string) error {
	body := map[string]any{
		"type":          eventType,
		"location_city": city,
		"location_hub":  hub,
		"notes":         notes,
	}
	if err := a.hc.Post(ctx, "/api/v1/shipments/"+shipmentID+"/events", body, nil); err != nil {
		return err
	}
	a.state.Counters.ShipmentEvents.Add(1)
	return nil
}

func (a *ShipmentProgressor) recordAttempt(ctx context.Context, sh shipmentLite) error {
	reasons := []string{
		"recipient not at home",
		"no answer at door",
		"access to address restricted",
		"recipient asked to reschedule",
	}
	body := map[string]any{
		"reason": reasons[a.rng.Intn(len(reasons))],
		"notes":  "auto-recorded by simulator",
	}
	if err := a.hc.Post(ctx, "/api/v1/shipments/"+sh.ID+"/record-attempt", body, nil); err != nil {
		return err
	}
	metrics.ShipmentsProgressed.WithLabelValues(sh.Status, "delivery_attempted").Inc()
	a.state.Counters.ShipmentsAdvanced.Add(1)
	return nil
}

func (a *ShipmentProgressor) recordDelivery(ctx context.Context, sh shipmentLite) error {
	signers := []string{
		"Andrii Koval", "Mariia Petrenko", "Oleksii Bondar", "Iryna Lysenko",
		"Vitalii Marchenko", "Olha Pavlenko", "Roman Sydorenko", "Yana Romanenko",
	}
	body := map[string]any{
		"signature_name": signers[a.rng.Intn(len(signers))],
		"photo_url":      "",
	}
	if err := a.hc.Post(ctx, "/api/v1/shipments/"+sh.ID+"/record-delivery", body, nil); err != nil {
		return err
	}
	metrics.ShipmentsProgressed.WithLabelValues(sh.Status, "delivered").Inc()
	a.state.Counters.ShipmentsAdvanced.Add(1)
	return nil
}

func (a *ShipmentProgressor) warehouseCity(sh shipmentLite) string {
	if sh.CurrentLocationCity != "" {
		return sh.CurrentLocationCity
	}
	return "Kyiv"
}

func (a *ShipmentProgressor) transitCityFor(sh shipmentLite) string {
	for _, c := range transitCities {
		if strings.Contains(sh.Address, c) {
			candidates := make([]string, 0)
			for _, t := range transitCities {
				if t != c {
					candidates = append(candidates, t)
				}
			}
			return candidates[a.rng.Intn(len(candidates))]
		}
	}
	return transitCities[a.rng.Intn(len(transitCities))]
}

func (a *ShipmentProgressor) destinationCity(sh shipmentLite) string {
	for _, c := range transitCities {
		if strings.Contains(sh.Address, c) {
			return c
		}
	}
	return transitCities[a.rng.Intn(len(transitCities))]
}
