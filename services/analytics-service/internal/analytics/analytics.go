package analytics

import (
	"context"
	"time"
)

type SalesDaily struct {
	ID           string    `json:"id"`
	Date         time.Time `json:"date"`
	TotalOrders  int       `json:"total_orders"`
	TotalRevenue float64   `json:"total_revenue"`
	AvgOrderSize float64   `json:"avg_order_size"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type InventorySnapshot struct {
	ID              string    `json:"id"`
	Date            time.Time `json:"date"`
	TotalProducts   int       `json:"total_products"`
	TotalQuantity   int       `json:"total_quantity"`
	TotalReserved   int       `json:"total_reserved"`
	TotalAvailable  int       `json:"total_available"`
	LowStockCount   int       `json:"low_stock_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LogisticsDaily struct {
	ID              string    `json:"id"`
	Date            time.Time `json:"date"`
	TotalShipments  int       `json:"total_shipments"`
	DeliveredCount  int       `json:"delivered_count"`
	FailedCount     int       `json:"failed_count"`
	AvgDeliveryH    float64   `json:"avg_delivery_hours"`
	OnTimeRate      float64   `json:"on_time_rate"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SalesSummary struct {
	TotalRevenue  float64 `json:"total_revenue"`
	OrderCount    int     `json:"order_count"`
	AvgOrderValue float64 `json:"avg_order_value"`
	DateFrom      string  `json:"date_from"`
	DateTo        string  `json:"date_to"`
}

type SalesTrend struct {
	Period       string  `json:"period"`
	TotalOrders  int     `json:"total_orders"`
	TotalRevenue float64 `json:"total_revenue"`
	AvgOrderSize float64 `json:"avg_order_size"`
}

type InventorySummary struct {
	TotalProducts  int     `json:"total_products"`
	TotalQuantity  int     `json:"total_quantity"`
	TotalReserved  int     `json:"total_reserved"`
	TotalAvailable int     `json:"total_available"`
	LowStockCount  int     `json:"low_stock_count"`
	TurnoverRate   float64 `json:"turnover_rate"`
	DateFrom       string  `json:"date_from"`
	DateTo         string  `json:"date_to"`
}

type LogisticsPerformance struct {
	TotalShipments int     `json:"total_shipments"`
	DeliveredCount int     `json:"delivered_count"`
	FailedCount    int     `json:"failed_count"`
	OnTimeRate     float64 `json:"on_time_rate"`
	AvgDeliveryH   float64 `json:"avg_delivery_hours"`
	DateFrom       string  `json:"date_from"`
	DateTo         string  `json:"date_to"`
}

type Anomaly struct {
	Type      string  `json:"type"`
	Metric    string  `json:"metric"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Date      string  `json:"date"`
	Severity  string  `json:"severity"`
	Message   string  `json:"message"`
}

type Optimization struct {
	Type            string  `json:"type"`
	ProductMetric   string  `json:"product_metric"`
	CurrentStock    int     `json:"current_stock"`
	AvgDemand       float64 `json:"avg_demand"`
	ReorderPoint    int     `json:"reorder_point"`
	RecommendedQty  int     `json:"recommended_qty"`
	SafetyStock     int     `json:"safety_stock"`
	Message         string  `json:"message"`
}

type ReportRequest struct {
	ReportType string `json:"report_type"`
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
}

type Report struct {
	ReportType string `json:"report_type"`
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
	GeneratedAt string `json:"generated_at"`
	Data       any    `json:"data"`
}

type Storage interface {
	UpsertSalesDaily(ctx context.Context, record SalesDaily) (SalesDaily, error)
	UpsertInventorySnapshot(ctx context.Context, record InventorySnapshot) (InventorySnapshot, error)
	UpsertLogisticsDaily(ctx context.Context, record LogisticsDaily) (LogisticsDaily, error)

	GetSalesDaily(ctx context.Context, from, to time.Time) ([]SalesDaily, error)
	GetInventorySnapshots(ctx context.Context, from, to time.Time) ([]InventorySnapshot, error)
	GetLogisticsDaily(ctx context.Context, from, to time.Time) ([]LogisticsDaily, error)
}
