package actors

import "github.com/haradrim/chainorchestra/services/simulator-service/internal/state"

type ScenarioTuning struct {
	OrderSpawnIntervalSec   float64
	OrderProgressIntervalSec float64
	ShipmentTickIntervalSec  float64
	InventoryTickIntervalSec float64
	NotificationIntervalSec  float64

	OrdersPerSpawn    int
	OrderCancelRate   float64
	ShipmentBatchSize int
	StockBatchSize    int

	FailedDeliveryRate    float64
	CarrierFailureCarrier string
}

func TuningFor(s state.Scenario) ScenarioTuning {
	base := ScenarioTuning{
		OrderSpawnIntervalSec:    45,
		OrderProgressIntervalSec: 20,
		ShipmentTickIntervalSec:  25,
		InventoryTickIntervalSec: 60,
		NotificationIntervalSec:  90,
		OrdersPerSpawn:           1,
		OrderCancelRate:          0.06,
		ShipmentBatchSize:        8,
		StockBatchSize:           3,
		FailedDeliveryRate:       0.20,
	}
	switch s {
	case state.ScenarioIdle:
		base.OrderSpawnIntervalSec = 0
		base.OrderProgressIntervalSec = 0
		base.ShipmentTickIntervalSec = 0
		base.InventoryTickIntervalSec = 0
		base.NotificationIntervalSec = 0
	case state.ScenarioHolidaySpike:
		base.OrderSpawnIntervalSec = 15
		base.OrdersPerSpawn = 2
		base.OrderProgressIntervalSec = 12
		base.ShipmentTickIntervalSec = 12
		base.ShipmentBatchSize = 16
		base.StockBatchSize = 6
	case state.ScenarioCarrierFailure:
		base.FailedDeliveryRate = 0.60
	case state.ScenarioDemandSurge:
		base.OrderSpawnIntervalSec = 9
		base.OrdersPerSpawn = 3
		base.OrderProgressIntervalSec = 9
		base.ShipmentTickIntervalSec = 10
		base.ShipmentBatchSize = 20
		base.InventoryTickIntervalSec = 40
		base.StockBatchSize = 8
	}
	return base
}
