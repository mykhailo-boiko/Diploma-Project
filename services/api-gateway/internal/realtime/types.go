package realtime

import (
	"encoding/json"
	"time"
)

type Event struct {
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Subject   string          `json:"subject"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func subjectToEventType(subject string) string {
	switch subject {
	case "order.created":
		return "order.created"
	case "order.status_changed":
		return "order.updated"
	case "order.cancelled":
		return "order.cancelled"
	case "logistics.shipment_created":
		return "shipment.created"
	case "logistics.shipment_status_changed":
		return "shipment.updated"
	case "logistics.shipment_out_for_delivery":
		return "shipment.out_for_delivery"
	case "logistics.shipment_delivered":
		return "shipment.delivered"
	case "logistics.shipment_attempted":
		return "shipment.attempted"
	case "logistics.shipment_returned":
		return "shipment.returned"
	case "logistics.shipment_redirected":
		return "shipment.redirected"
	case "inventory.stock_changed":
		return "stock.changed"
	case "inventory.low_stock":
		return "stock.low"
	case "notification.created":
		return "notification.new"
	case "analytics.aggregate_updated":
		return "analytics.updated"
	}
	return subject
}

func roleAllows(role, subject string) bool {
	if role == "admin" {
		return true
	}
	switch role {
	case "operator":
		return true
	case "warehouse_manager":
		switch subject {
		case "inventory.stock_changed", "inventory.low_stock",
			"order.created", "order.status_changed", "order.cancelled",
			"notification.created", "analytics.aggregate_updated":
			return true
		}
	case "logistics_manager":
		switch subject {
		case "logistics.shipment_created", "logistics.shipment_status_changed",
			"logistics.shipment_out_for_delivery", "logistics.shipment_delivered",
			"logistics.shipment_attempted", "logistics.shipment_returned",
			"logistics.shipment_redirected",
			"notification.created", "analytics.aggregate_updated":
			return true
		}
	case "analyst":
		switch subject {
		case "analytics.aggregate_updated", "notification.created":
			return true
		}
	}
	return false
}
