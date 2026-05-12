package analytics

import (
	"context"
	"errors"
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
	Category  string  `json:"category"`
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

type WhatIfScenario struct {
	Kind   string         `json:"kind"`
	Params map[string]any `json:"params"`
}

type WhatIfResult struct {
	Scenario       WhatIfScenario     `json:"scenario"`
	Baseline       map[string]float64 `json:"baseline"`
	Projected      map[string]float64 `json:"projected"`
	Delta          map[string]float64 `json:"delta"`
	DeltaPercent   map[string]float64 `json:"delta_percent"`
	Assumptions    []string           `json:"assumptions"`
	Confidence     string             `json:"confidence_qualitative"`
	HumanSummary   string             `json:"human_summary"`
}

type ForecastPoint struct {
	Date           time.Time `json:"date"`
	Value          float64   `json:"value"`
	ConfidenceLow  float64   `json:"confidence_low,omitempty"`
	ConfidenceHigh float64   `json:"confidence_high,omitempty"`
}

type Forecast struct {
	Metric          string          `json:"metric"`
	Method          string          `json:"method"`
	HorizonDays     int             `json:"horizon_days"`
	HistoryWindow   int             `json:"history_window_days"`
	History         []ForecastPoint `json:"history"`
	Forecast        []ForecastPoint `json:"forecast"`
	BacktestMAPE    float64         `json:"backtest_mape,omitempty"`
	Confidence      string          `json:"confidence_qualitative"`
	Assumptions     []string        `json:"assumptions"`
}

type AuditEntry struct {
	ID           string    `json:"id"`
	ActorUserID  string    `json:"actor_user_id"`
	ActorEmail   string    `json:"actor_email"`
	ActorRole    string    `json:"actor_role"`
	ServiceName  string    `json:"service_name"`
	Action       string    `json:"action"`
	EntityType   string    `json:"entity_type,omitempty"`
	EntityIDs    []string  `json:"entity_ids,omitempty"`
	ParamsSnip   string    `json:"params_snip,omitempty"`
	ResultStatus string    `json:"result_status"`
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
	ErrorMessage string    `json:"error_message,omitempty"`
	TraceID      string    `json:"trace_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditFilter struct {
	ActorEmail string
	Action     string
	EntityID   string
	From       *time.Time
	To         *time.Time
	Limit      int
}

type PeriodSnapshot struct {
	Label string    `json:"label"`
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	Value float64   `json:"value"`
}

type PeriodComparison struct {
	Metric        string         `json:"metric"`
	PeriodA       PeriodSnapshot `json:"period_a"`
	PeriodB       PeriodSnapshot `json:"period_b"`
	AbsoluteDelta float64        `json:"absolute_delta"`
	PercentChange float64        `json:"percent_change"`
	Direction     string         `json:"direction"`
	Significance  string         `json:"significance"`
}

type CategorySpend struct {
	Category  string  `json:"category"`
	Revenue   float64 `json:"revenue"`
	UnitsSold int     `json:"units_sold"`
}

type OrderHeader struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}

type CustomerProfile360 struct {
	CustomerName        string          `json:"customer_name"`
	FirstOrderDate      time.Time       `json:"first_order_date"`
	LastOrderDate       time.Time       `json:"last_order_date"`
	LifetimeValue       float64         `json:"lifetime_value"`
	OrderCount          int             `json:"order_count"`
	AvgOrderValue       float64         `json:"avg_order_value"`
	DaysSinceLastOrder  int             `json:"days_since_last_order"`
	MedianInterOrderD   float64         `json:"median_inter_order_days"`
	ChurnRiskScore      float64         `json:"churn_risk_score"`
	StatusBreakdown     map[string]int  `json:"status_breakdown"`
	TopCategories       []CategorySpend `json:"top_categories"`
	RecentOrders        []OrderHeader   `json:"recent_orders"`
	IsNewCustomer90Days bool            `json:"is_new_customer_90_days"`
}

type CarrierCityStat struct {
	City       string  `json:"city"`
	Delivered  int     `json:"delivered"`
	OnTime     int     `json:"on_time"`
	Late       int     `json:"late"`
	OnTimeRate float64 `json:"on_time_rate"`
	AvgHours   float64 `json:"avg_delivery_hours"`
}

type CarrierPerformance struct {
	CarrierID         string            `json:"carrier_id"`
	CarrierName       string            `json:"carrier_name"`
	IsActive          bool              `json:"is_active"`
	TotalShipments    int               `json:"total_shipments"`
	Delivered         int               `json:"delivered"`
	OnTime            int               `json:"on_time"`
	Late              int               `json:"late"`
	Cancelled         int               `json:"cancelled"`
	Returned          int               `json:"returned"`
	OnTimeRate        float64           `json:"on_time_rate"`
	AvgDeliveryHours  float64           `json:"avg_delivery_hours"`
	WorstCities       []CarrierCityStat `json:"worst_cities"`
}

type RebalancingRecommendation struct {
	ProductID         string  `json:"product_id"`
	SKU               string  `json:"sku"`
	ProductName       string  `json:"product_name"`
	Category          string  `json:"category"`
	UnitPrice         float64 `json:"unit_price"`
	DonorWarehouse    string  `json:"donor_warehouse"`
	DonorQuantity     int     `json:"donor_quantity"`
	DonorThreshold    int     `json:"donor_min_threshold"`
	AcceptorWarehouse string  `json:"acceptor_warehouse"`
	AcceptorQuantity  int     `json:"acceptor_quantity"`
	AcceptorThreshold int     `json:"acceptor_min_threshold"`
	TransferQty       int     `json:"transfer_qty"`
	HoldingSavings    float64 `json:"holding_savings"`
	TransferCost      float64 `json:"transfer_cost"`
	NetBenefit        float64 `json:"net_benefit"`
	ROIPct            float64 `json:"roi_pct"`
}

type QuickCancellation struct {
	CarrierName       string   `json:"carrier_name"`
	City              string   `json:"city"`
	Count             int      `json:"count"`
	AvgMinutes        float64  `json:"avg_minutes_between"`
	MinMinutes        float64  `json:"min_minutes_between"`
	MaxMinutes        float64  `json:"max_minutes_between"`
	LostRevenue       float64  `json:"lost_revenue"`
	SampleOrderIDs    []string `json:"sample_order_ids"`
	SampleCancelReasons []string `json:"sample_cancel_reasons,omitempty"`
}

type Storage interface {
	UpsertSalesDaily(ctx context.Context, record SalesDaily) (SalesDaily, error)
	UpsertInventorySnapshot(ctx context.Context, record InventorySnapshot) (InventorySnapshot, error)
	UpsertLogisticsDaily(ctx context.Context, record LogisticsDaily) (LogisticsDaily, error)

	GetSalesDaily(ctx context.Context, from, to time.Time) ([]SalesDaily, error)
	GetInventorySnapshots(ctx context.Context, from, to time.Time) ([]InventorySnapshot, error)
	GetLogisticsDaily(ctx context.Context, from, to time.Time) ([]LogisticsDaily, error)

	GetQuickCancellations(ctx context.Context, from, to time.Time, maxMinutes int) ([]QuickCancellation, error)
	GetRebalancingRecommendations(ctx context.Context, params RebalancingParams) ([]RebalancingRecommendation, error)
	GetCarrierPerformance(ctx context.Context, from, to time.Time, slaHours int, worstCitiesPerCarrier int) ([]CarrierPerformance, error)
	GetCustomerProfile360(ctx context.Context, customerName string, recentN int, topCategoriesN int) (CustomerProfile360, error)
	GetMetricValue(ctx context.Context, metric string, from, to time.Time) (float64, error)
	QueryAuditLog(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)
	GetDailyMetricSeries(ctx context.Context, metric string, from, to time.Time) ([]ForecastPoint, error)
}

var ErrCustomerNotFound = errors.New("customer not found")

type RebalancingParams struct {
	OverstockMultiplier float64
	HoldingDailyRate    float64
	HoldingHorizonDays  int
	TransferBaseFee     float64
	TransferPerUnit     float64
	OnlyPositiveROI     bool
	Limit               int
}
