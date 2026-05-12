package state

import (
	"sync"
	"sync/atomic"
	"time"
)

type Scenario string

const (
	ScenarioIdle           Scenario = "idle"
	ScenarioSteady         Scenario = "steady"
	ScenarioHolidaySpike   Scenario = "holiday_spike"
	ScenarioCarrierFailure Scenario = "carrier_failure"
	ScenarioDemandSurge    Scenario = "demand_surge"
)

func ParseScenario(s string) Scenario {
	switch Scenario(s) {
	case ScenarioIdle, ScenarioSteady, ScenarioHolidaySpike, ScenarioCarrierFailure, ScenarioDemandSurge:
		return Scenario(s)
	default:
		return ScenarioSteady
	}
}

type ActorCounters struct {
	OrdersCreated     atomic.Int64
	OrdersProgressed  atomic.Int64
	OrdersCancelled   atomic.Int64
	ShipmentsAdvanced atomic.Int64
	ShipmentEvents    atomic.Int64
	StockAdjustments  atomic.Int64
	NotificationsSent atomic.Int64
	Errors            atomic.Int64
}

type State struct {
	mu        sync.RWMutex
	enabled   bool
	scenario  Scenario
	speed     float64
	startedAt time.Time

	Counters *ActorCounters
}

func New(initialScenario Scenario, initialSpeed float64) *State {
	return &State{
		scenario: initialScenario,
		speed:    initialSpeed,
		Counters: &ActorCounters{},
	}
}

func (s *State) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

func (s *State) Scenario() Scenario {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scenario
}

func (s *State) Speed() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.speed
}

func (s *State) StartedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startedAt
}

func (s *State) Start(scenario Scenario, speed float64) {
	s.mu.Lock()
	s.enabled = true
	s.scenario = scenario
	if speed > 0 {
		s.speed = speed
	}
	s.startedAt = time.Now().UTC()
	s.mu.Unlock()
}

func (s *State) Stop() {
	s.mu.Lock()
	s.enabled = false
	s.mu.Unlock()
}

func (s *State) SetSpeed(speed float64) {
	s.mu.Lock()
	if speed > 0 {
		s.speed = speed
	}
	s.mu.Unlock()
}

func (s *State) SetScenario(scenario Scenario) {
	s.mu.Lock()
	s.scenario = scenario
	s.mu.Unlock()
}

type Snapshot struct {
	Enabled    bool      `json:"enabled"`
	Scenario   Scenario  `json:"scenario"`
	Speed      float64   `json:"speed"`
	StartedAt  time.Time `json:"started_at"`
	UptimeSecs int64     `json:"uptime_secs"`
	Counters   struct {
		OrdersCreated     int64 `json:"orders_created"`
		OrdersProgressed  int64 `json:"orders_progressed"`
		OrdersCancelled   int64 `json:"orders_cancelled"`
		ShipmentsAdvanced int64 `json:"shipments_advanced"`
		ShipmentEvents    int64 `json:"shipment_events"`
		StockAdjustments  int64 `json:"stock_adjustments"`
		NotificationsSent int64 `json:"notifications_sent"`
		Errors            int64 `json:"errors"`
	} `json:"counters"`
}

func (s *State) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := Snapshot{
		Enabled:   s.enabled,
		Scenario:  s.scenario,
		Speed:     s.speed,
		StartedAt: s.startedAt,
	}
	if !s.startedAt.IsZero() && s.enabled {
		snap.UptimeSecs = int64(time.Since(s.startedAt).Seconds())
	}
	snap.Counters.OrdersCreated = s.Counters.OrdersCreated.Load()
	snap.Counters.OrdersProgressed = s.Counters.OrdersProgressed.Load()
	snap.Counters.OrdersCancelled = s.Counters.OrdersCancelled.Load()
	snap.Counters.ShipmentsAdvanced = s.Counters.ShipmentsAdvanced.Load()
	snap.Counters.ShipmentEvents = s.Counters.ShipmentEvents.Load()
	snap.Counters.StockAdjustments = s.Counters.StockAdjustments.Load()
	snap.Counters.NotificationsSent = s.Counters.NotificationsSent.Load()
	snap.Counters.Errors = s.Counters.Errors.Load()
	return snap
}
