package realtime

import "testing"

func TestSubjectToEventType_Mappings(t *testing.T) {
	cases := map[string]string{
		"order.created":                         "order.created",
		"order.status_changed":                  "order.updated",
		"logistics.shipment_created":            "shipment.created",
		"logistics.shipment_out_for_delivery":   "shipment.out_for_delivery",
		"logistics.shipment_delivered":          "shipment.delivered",
		"inventory.stock_changed":               "stock.changed",
		"inventory.low_stock":                   "stock.low",
		"notification.created":                  "notification.new",
		"analytics.aggregate_updated":           "analytics.updated",
		"unknown.thing":                         "unknown.thing",
	}
	for subj, want := range cases {
		got := subjectToEventType(subj)
		if got != want {
			t.Errorf("subjectToEventType(%q) = %q, want %q", subj, got, want)
		}
	}
}

func TestRoleAllows_AdminGetsEverything(t *testing.T) {
	for _, subj := range []string{
		"order.created", "logistics.shipment_delivered", "inventory.low_stock",
		"notification.created", "analytics.aggregate_updated", "weird.unknown",
	} {
		if !roleAllows("admin", subj) {
			t.Errorf("admin must see %q", subj)
		}
	}
}

func TestRoleAllows_OperatorScopedToOrdersAndNotifications(t *testing.T) {
	if !roleAllows("operator", "order.created") {
		t.Errorf("operator must see order.created")
	}
}

func TestRoleAllows_WarehouseManagerScopedToInventoryAndOrders(t *testing.T) {
	if !roleAllows("warehouse_manager", "inventory.stock_changed") {
		t.Errorf("warehouse_manager must see stock changes")
	}
	if roleAllows("warehouse_manager", "logistics.shipment_delivered") {
		t.Errorf("warehouse_manager must NOT see shipment events")
	}
}

func TestRoleAllows_LogisticsManagerScopedToShipments(t *testing.T) {
	if !roleAllows("logistics_manager", "logistics.shipment_delivered") {
		t.Errorf("logistics_manager must see shipment events")
	}
	if roleAllows("logistics_manager", "inventory.stock_changed") {
		t.Errorf("logistics_manager must NOT see stock events")
	}
}

func TestRoleAllows_AnalystScopedToAnalyticsOnly(t *testing.T) {
	if !roleAllows("analyst", "analytics.aggregate_updated") {
		t.Errorf("analyst must see analytics events")
	}
	if roleAllows("analyst", "order.created") {
		t.Errorf("analyst must NOT see order events")
	}
}

func TestRoleAllows_UnknownRoleDeniedEverywhere(t *testing.T) {
	if roleAllows("hacker", "order.created") {
		t.Errorf("unknown role must be denied")
	}
}
