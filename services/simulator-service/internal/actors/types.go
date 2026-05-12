package actors

import "time"

type listEnvelope[T any] struct {
	Data []T `json:"data"`
}

type singleEnvelope[T any] struct {
	Data T `json:"data"`
}

type productLite struct {
	ID        string  `json:"id"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	UnitPrice float64 `json:"unit_price"`
}

type warehouseLite struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type carrierLite struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type stockLite struct {
	ID            string `json:"id"`
	ProductID     string `json:"product_id"`
	WarehouseID   string `json:"warehouse_id"`
	Quantity      int    `json:"quantity"`
	Reserved      int    `json:"reserved"`
	MinThreshold  int    `json:"min_threshold"`
}

type customerLite struct {
	CustomerName string  `json:"customer_name"`
	OrderCount   int     `json:"order_count"`
	TotalSpent   float64 `json:"total_spent"`
}

type orderLite struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	CustomerName string    `json:"customer_name"`
	Total        float64   `json:"total"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type shipmentLite struct {
	ID                  string     `json:"id"`
	OrderID             string     `json:"order_id"`
	WarehouseID         string     `json:"warehouse_id"`
	CarrierID           string     `json:"carrier_id"`
	Status              string     `json:"status"`
	Address             string     `json:"address"`
	TrackingNumber      string     `json:"tracking_number"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at,omitempty"`
	CurrentLocationCity string     `json:"current_location_city,omitempty"`
	CurrentLocationHub  string     `json:"current_location_hub,omitempty"`
	DeliveryAttempts    int        `json:"delivery_attempts"`
	Recipient           any        `json:"recipient,omitempty"`
	Sender              any        `json:"sender,omitempty"`
}
