package actors

import (
	"testing"

	"github.com/haradrim/chainorchestra/services/simulator-service/internal/state"
)

func TestTuningFor_IdleZeroesAllIntervals(t *testing.T) {
	tun := TuningFor(state.ScenarioIdle)
	if tun.OrderSpawnIntervalSec != 0 || tun.OrderProgressIntervalSec != 0 ||
		tun.ShipmentTickIntervalSec != 0 || tun.InventoryTickIntervalSec != 0 ||
		tun.NotificationIntervalSec != 0 {
		t.Errorf("idle scenario should have zero intervals: %+v", tun)
	}
}

func TestTuningFor_HolidaySpikeIncreasesThroughput(t *testing.T) {
	base := TuningFor(state.ScenarioSteady)
	spike := TuningFor(state.ScenarioHolidaySpike)
	if spike.OrderSpawnIntervalSec >= base.OrderSpawnIntervalSec {
		t.Errorf("holiday_spike must lower spawn interval, base=%v spike=%v",
			base.OrderSpawnIntervalSec, spike.OrderSpawnIntervalSec)
	}
	if spike.OrdersPerSpawn <= base.OrdersPerSpawn {
		t.Errorf("holiday_spike must increase OrdersPerSpawn")
	}
	if spike.ShipmentBatchSize <= base.ShipmentBatchSize {
		t.Errorf("holiday_spike must increase shipment batch size")
	}
}

func TestTuningFor_DemandSurgeIsFastestScenario(t *testing.T) {
	tun := TuningFor(state.ScenarioDemandSurge)
	if tun.OrdersPerSpawn < 3 {
		t.Errorf("demand_surge must spawn ≥3 orders per tick")
	}
	if tun.OrderSpawnIntervalSec > 20 {
		t.Errorf("demand_surge tick should be <= 20s base, got %v", tun.OrderSpawnIntervalSec)
	}
}

func TestTuningFor_CarrierFailureRaisesFailedRate(t *testing.T) {
	tun := TuningFor(state.ScenarioCarrierFailure)
	if tun.FailedDeliveryRate < 0.5 {
		t.Errorf("carrier_failure must increase failed delivery rate, got %v", tun.FailedDeliveryRate)
	}
}

func TestTuningFor_UnknownScenarioFallsBackToSteady(t *testing.T) {
	base := TuningFor(state.ScenarioSteady)
	fallback := TuningFor(state.Scenario("bogus"))
	if base != fallback {
		t.Errorf("unknown scenario should equal steady tuning")
	}
}
