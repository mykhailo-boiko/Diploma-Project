package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
	"github.com/haradrim/chainorchestra/services/analytics-service/internal/analytics"
)

type AnalyticsService interface {
	GetSalesDaily(ctx context.Context, from, to time.Time) ([]analytics.SalesDaily, error)
	GetInventorySnapshots(ctx context.Context, from, to time.Time) ([]analytics.InventorySnapshot, error)
	GetLogisticsDaily(ctx context.Context, from, to time.Time) ([]analytics.LogisticsDaily, error)
	GetSalesSummary(ctx context.Context, from, to time.Time) (analytics.SalesSummary, error)
	GetSalesTrends(ctx context.Context, from, to time.Time, granularity string) ([]analytics.SalesTrend, error)
	GetInventorySummary(ctx context.Context, from, to time.Time) (analytics.InventorySummary, error)
	GetLogisticsPerformance(ctx context.Context, from, to time.Time) (analytics.LogisticsPerformance, error)
	DetectAnomalies(ctx context.Context, from, to time.Time) ([]analytics.Anomaly, error)
	GetOptimizations(ctx context.Context, from, to time.Time) ([]analytics.Optimization, error)
	GenerateReport(ctx context.Context, req analytics.ReportRequest) (analytics.Report, error)
	GetQuickCancellations(ctx context.Context, from, to time.Time, maxMinutes int) ([]analytics.QuickCancellation, error)
	GetRebalancingRecommendations(ctx context.Context, params analytics.RebalancingParams) ([]analytics.RebalancingRecommendation, error)
	GetCarrierPerformance(ctx context.Context, from, to time.Time, slaHours int, worstCitiesPerCarrier int) ([]analytics.CarrierPerformance, error)
	GetCustomerProfile360(ctx context.Context, customerName string, recentN int, topCategoriesN int) (analytics.CustomerProfile360, error)
	GetPeriodComparison(ctx context.Context, metric string, aFrom, aTo, bFrom, bTo time.Time, aLabel, bLabel string) (analytics.PeriodComparison, error)
	QueryAuditLog(ctx context.Context, filter analytics.AuditFilter) ([]analytics.AuditEntry, error)
	GetForecast(ctx context.Context, metric, method string, historyDays, horizonDays int) (analytics.Forecast, error)
	RunWhatIf(ctx context.Context, scenario analytics.WhatIfScenario) (analytics.WhatIfResult, error)
}

type AnalyticsController struct {
	svc AnalyticsService
	log *zap.Logger
}

func NewAnalyticsController(svc AnalyticsService, log *zap.Logger) *AnalyticsController {
	return &AnalyticsController{svc: svc, log: log}
}

func (c *AnalyticsController) GetSalesDaily(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	records, err := c.svc.GetSalesDaily(r.Context(), from, to)
	if err != nil {
		c.log.Error("Failed to get sales daily", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if records == nil {
		records = []analytics.SalesDaily{}
	}

	httpresponse.OK(w, records)
}

func (c *AnalyticsController) GetInventorySnapshots(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	records, err := c.svc.GetInventorySnapshots(r.Context(), from, to)
	if err != nil {
		c.log.Error("Failed to get inventory snapshots", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if records == nil {
		records = []analytics.InventorySnapshot{}
	}

	httpresponse.OK(w, records)
}

func (c *AnalyticsController) GetLogisticsDaily(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	records, err := c.svc.GetLogisticsDaily(r.Context(), from, to)
	if err != nil {
		c.log.Error("Failed to get logistics daily", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if records == nil {
		records = []analytics.LogisticsDaily{}
	}

	httpresponse.OK(w, records)
}

func (c *AnalyticsController) GetSalesSummary(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	summary, err := c.svc.GetSalesSummary(r.Context(), from, to)
	if err != nil {
		c.log.Error("Failed to get sales summary", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, summary)
}

func (c *AnalyticsController) GetSalesTrends(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "day"
	}

	if granularity != "day" && granularity != "week" {
		httpresponse.BadRequest(w, "validation_error", "granularity must be 'day' or 'week'")
		return
	}

	trends, err := c.svc.GetSalesTrends(r.Context(), from, to, granularity)
	if err != nil {
		c.log.Error("Failed to get sales trends", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if trends == nil {
		trends = []analytics.SalesTrend{}
	}

	httpresponse.OK(w, trends)
}

func (c *AnalyticsController) GetInventorySummary(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	summary, err := c.svc.GetInventorySummary(r.Context(), from, to)
	if err != nil {
		c.log.Error("Failed to get inventory summary", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, summary)
}

func (c *AnalyticsController) GetLogisticsPerformance(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	perf, err := c.svc.GetLogisticsPerformance(r.Context(), from, to)
	if err != nil {
		c.log.Error("Failed to get logistics performance", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, perf)
}

func (c *AnalyticsController) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	anomalies, err := c.svc.DetectAnomalies(r.Context(), from, to)
	if err != nil {
		c.log.Error("Failed to detect anomalies", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, anomalies)
}

func (c *AnalyticsController) GetOptimizations(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}

	opts, err := c.svc.GetOptimizations(r.Context(), from, to)
	if err != nil {
		c.log.Error("Failed to get optimizations", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, opts)
}

func (c *AnalyticsController) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var req analytics.ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "validation_error", "invalid request body")
		return
	}

	if req.ReportType == "" {
		httpresponse.BadRequest(w, "validation_error", "report_type is required")
		return
	}
	if req.DateFrom == "" || req.DateTo == "" {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required")
		return
	}

	report, err := c.svc.GenerateReport(r.Context(), req)
	if err != nil {
		c.log.Error("Failed to generate report", zap.Error(err))
		httpresponse.BadRequest(w, "report_error", err.Error())
		return
	}

	httpresponse.OK(w, report)
}

func (c *AnalyticsController) RunWhatIf(w http.ResponseWriter, r *http.Request) {
	var req analytics.WhatIfScenario
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.BadRequest(w, "invalid_request", "invalid request body")
		return
	}
	if req.Kind == "" {
		httpresponse.BadRequest(w, "validation_error", "kind is required")
		return
	}
	result, err := c.svc.RunWhatIf(r.Context(), req)
	if err != nil {
		c.log.Error("what-if failed", zap.Error(err))
		httpresponse.BadRequest(w, "internal_error", err.Error())
		return
	}
	httpresponse.OK(w, result)
}

func (c *AnalyticsController) GetForecast(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	metric := q.Get("metric")
	if metric == "" {
		httpresponse.BadRequest(w, "validation_error", "metric is required")
		return
	}
	method := q.Get("method")
	if method == "" {
		method = "linear"
	}

	horizonDays := 14
	if s := q.Get("horizon_days"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			horizonDays = v
		}
	}
	historyDays := 30
	if s := q.Get("history_days"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			historyDays = v
		}
	}

	result, err := c.svc.GetForecast(r.Context(), metric, method, historyDays, horizonDays)
	if err != nil {
		c.log.Error("Failed to forecast", zap.Error(err))
		httpresponse.BadRequest(w, "internal_error", err.Error())
		return
	}
	httpresponse.OK(w, result)
}

func (c *AnalyticsController) QueryAuditLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter analytics.AuditFilter
	filter.ActorEmail = q.Get("actor_email")
	filter.Action = q.Get("action")
	filter.EntityID = q.Get("entity_id")
	if s := q.Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.From = &t
		}
	}
	if s := q.Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Second)
			filter.To = &endOfDay
		}
	}
	if s := q.Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			filter.Limit = v
		}
	}

	results, err := c.svc.QueryAuditLog(r.Context(), filter)
	if err != nil {
		c.log.Error("Failed to query audit log", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}
	if results == nil {
		results = []analytics.AuditEntry{}
	}
	httpresponse.OK(w, results)
}

func (c *AnalyticsController) GetPeriodComparison(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	metric := q.Get("metric")
	if metric == "" {
		httpresponse.BadRequest(w, "validation_error", "metric is required")
		return
	}

	aFromStr := q.Get("a_from")
	aToStr := q.Get("a_to")
	bFromStr := q.Get("b_from")
	bToStr := q.Get("b_to")
	if aFromStr == "" || aToStr == "" || bFromStr == "" || bToStr == "" {
		httpresponse.BadRequest(w, "validation_error", "a_from, a_to, b_from, b_to are required (YYYY-MM-DD)")
		return
	}

	aFrom, err := time.Parse("2006-01-02", aFromStr)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", "invalid a_from")
		return
	}
	aTo, err := time.Parse("2006-01-02", aToStr)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", "invalid a_to")
		return
	}
	bFrom, err := time.Parse("2006-01-02", bFromStr)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", "invalid b_from")
		return
	}
	bTo, err := time.Parse("2006-01-02", bToStr)
	if err != nil {
		httpresponse.BadRequest(w, "validation_error", "invalid b_to")
		return
	}
	aTo = aTo.Add(24*time.Hour - time.Second)
	bTo = bTo.Add(24*time.Hour - time.Second)

	aLabel := q.Get("a_label")
	bLabel := q.Get("b_label")

	result, err := c.svc.GetPeriodComparison(r.Context(), metric, aFrom, aTo, bFrom, bTo, aLabel, bLabel)
	if err != nil {
		c.log.Error("Failed to compute period comparison", zap.Error(err))
		httpresponse.BadRequest(w, "internal_error", err.Error())
		return
	}

	httpresponse.OK(w, result)
}

func (c *AnalyticsController) GetCustomerProfile360(w http.ResponseWriter, r *http.Request) {
	customerName := r.URL.Query().Get("customer_name")
	if customerName == "" {
		httpresponse.BadRequest(w, "validation_error", "customer_name is required")
		return
	}

	recentN := 5
	if s := r.URL.Query().Get("recent_n"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			recentN = v
		}
	}

	topCategoriesN := 5
	if s := r.URL.Query().Get("top_categories_n"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			topCategoriesN = v
		}
	}

	profile, err := c.svc.GetCustomerProfile360(r.Context(), customerName, recentN, topCategoriesN)
	if err != nil {
		if errors.Is(err, analytics.ErrCustomerNotFound) {
			httpresponse.NotFound(w, "customer_not_found", "customer not found")
			return
		}
		c.log.Error("Failed to get customer profile 360", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	httpresponse.OK(w, profile)
}

func (c *AnalyticsController) GetCarrierPerformance(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}
	to = to.Add(24*time.Hour - time.Second)

	slaHours := 168
	if s := r.URL.Query().Get("sla_hours"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			slaHours = v
		}
	}

	worstCities := 5
	if s := r.URL.Query().Get("worst_cities"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			worstCities = v
		}
	}

	results, err := c.svc.GetCarrierPerformance(r.Context(), from, to, slaHours, worstCities)
	if err != nil {
		c.log.Error("Failed to get carrier performance", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if results == nil {
		results = []analytics.CarrierPerformance{}
	}

	httpresponse.OK(w, results)
}

func (c *AnalyticsController) GetRebalancing(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := analytics.RebalancingParams{
		OverstockMultiplier: 3.0,
		HoldingDailyRate:    0.0005,
		HoldingHorizonDays:  30,
		TransferBaseFee:     50.0,
		TransferPerUnit:     1.5,
		OnlyPositiveROI:     true,
		Limit:               50,
	}

	if s := q.Get("overstock_multiplier"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			params.OverstockMultiplier = v
		}
	}
	if s := q.Get("holding_daily_rate"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			params.HoldingDailyRate = v
		}
	}
	if s := q.Get("holding_horizon_days"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			params.HoldingHorizonDays = v
		}
	}
	if s := q.Get("transfer_base_fee"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v >= 0 {
			params.TransferBaseFee = v
		}
	}
	if s := q.Get("transfer_per_unit"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v >= 0 {
			params.TransferPerUnit = v
		}
	}
	if q.Get("include_unprofitable") == "true" {
		params.OnlyPositiveROI = false
	}
	if s := q.Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			params.Limit = v
		}
	}

	results, err := c.svc.GetRebalancingRecommendations(r.Context(), params)
	if err != nil {
		c.log.Error("Failed to get rebalancing recommendations", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if results == nil {
		results = []analytics.RebalancingRecommendation{}
	}

	httpresponse.OK(w, results)
}

func (c *AnalyticsController) GetQuickCancellations(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseDateRange(r)
	if !ok {
		httpresponse.BadRequest(w, "validation_error", "date_from and date_to are required (YYYY-MM-DD format)")
		return
	}
	to = to.Add(24*time.Hour - time.Second)

	maxMinutes := 60
	if s := r.URL.Query().Get("max_minutes"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			maxMinutes = v
		}
	}

	results, err := c.svc.GetQuickCancellations(r.Context(), from, to, maxMinutes)
	if err != nil {
		c.log.Error("Failed to get quick cancellations", zap.Error(err))
		httpresponse.InternalError(w, "internal_error", "internal server error")
		return
	}

	if results == nil {
		results = []analytics.QuickCancellation{}
	}

	httpresponse.OK(w, results)
}

func parseDateRange(r *http.Request) (time.Time, time.Time, bool) {
	fromStr := r.URL.Query().Get("date_from")
	toStr := r.URL.Query().Get("date_to")

	if fromStr == "" || toStr == "" {
		return time.Time{}, time.Time{}, false
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}

	return from, to, true
}
