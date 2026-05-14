package controller

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/analytics-service/internal/analytics"
)

type Service struct {
	storage analytics.Storage
	log     *zap.Logger
}

func NewService(storage analytics.Storage, log *zap.Logger) *Service {
	return &Service{storage: storage, log: log}
}

func (s *Service) GetSalesDaily(ctx context.Context, from, to time.Time) ([]analytics.SalesDaily, error) {
	return s.storage.GetSalesDaily(ctx, from, to)
}

func (s *Service) GetInventorySnapshots(ctx context.Context, from, to time.Time) ([]analytics.InventorySnapshot, error) {
	return s.storage.GetInventorySnapshots(ctx, from, to)
}

func (s *Service) GetLogisticsDaily(ctx context.Context, from, to time.Time) ([]analytics.LogisticsDaily, error) {
	return s.storage.GetLogisticsDaily(ctx, from, to)
}

func (s *Service) GetSalesSummary(ctx context.Context, from, to time.Time) (analytics.SalesSummary, error) {
	records, err := s.storage.GetSalesDaily(ctx, from, to)
	if err != nil {
		return analytics.SalesSummary{}, fmt.Errorf("failed to get sales daily: %w", err)
	}

	var summary analytics.SalesSummary
	summary.DateFrom = from.Format("2006-01-02")
	summary.DateTo = to.Format("2006-01-02")

	for _, r := range records {
		summary.OrderCount += r.TotalOrders
		summary.TotalRevenue += r.TotalRevenue
	}

	if summary.OrderCount > 0 {
		summary.AvgOrderValue = summary.TotalRevenue / float64(summary.OrderCount)
	}

	return summary, nil
}

func (s *Service) GetSalesTrends(ctx context.Context, from, to time.Time, granularity string) ([]analytics.SalesTrend, error) {
	records, err := s.storage.GetSalesDaily(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get sales daily: %w", err)
	}

	if granularity == "week" {
		return s.aggregateWeeklyTrends(records), nil
	}

	trends := make([]analytics.SalesTrend, 0, len(records))
	for _, r := range records {
		trends = append(trends, analytics.SalesTrend{
			Period:       r.Date.Format("2006-01-02"),
			TotalOrders:  r.TotalOrders,
			TotalRevenue: r.TotalRevenue,
			AvgOrderSize: r.AvgOrderSize,
		})
	}

	return trends, nil
}

func (s *Service) aggregateWeeklyTrends(records []analytics.SalesDaily) []analytics.SalesTrend {
	if len(records) == 0 {
		return nil
	}

	type weekBucket struct {
		key          string
		totalOrders  int
		totalRevenue float64
		days         int
	}

	var buckets []weekBucket
	bucketIdx := make(map[string]int)

	for _, r := range records {
		year, week := r.Date.ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)

		idx, ok := bucketIdx[key]
		if !ok {
			idx = len(buckets)
			bucketIdx[key] = idx
			buckets = append(buckets, weekBucket{key: key})
		}
		buckets[idx].totalOrders += r.TotalOrders
		buckets[idx].totalRevenue += r.TotalRevenue
		buckets[idx].days++
	}

	trends := make([]analytics.SalesTrend, 0, len(buckets))
	for _, b := range buckets {
		var avg float64
		if b.totalOrders > 0 {
			avg = b.totalRevenue / float64(b.totalOrders)
		}
		trends = append(trends, analytics.SalesTrend{
			Period:       b.key,
			TotalOrders:  b.totalOrders,
			TotalRevenue: b.totalRevenue,
			AvgOrderSize: avg,
		})
	}

	return trends
}

func (s *Service) GetInventorySummary(ctx context.Context, from, to time.Time) (analytics.InventorySummary, error) {
	snapshots, err := s.storage.GetInventorySnapshots(ctx, from, to)
	if err != nil {
		return analytics.InventorySummary{}, fmt.Errorf("failed to get inventory snapshots: %w", err)
	}

	var summary analytics.InventorySummary
	summary.DateFrom = from.Format("2006-01-02")
	summary.DateTo = to.Format("2006-01-02")

	if len(snapshots) == 0 {
		return summary, nil
	}

	latest := snapshots[len(snapshots)-1]
	summary.TotalProducts = latest.TotalProducts
	summary.TotalQuantity = latest.TotalQuantity
	summary.TotalReserved = latest.TotalReserved
	summary.TotalAvailable = latest.TotalAvailable
	summary.LowStockCount = latest.LowStockCount

	var totalQty, totalReserved int
	for _, snap := range snapshots {
		totalQty += snap.TotalQuantity
		totalReserved += snap.TotalReserved
	}
	avgQty := float64(totalQty) / float64(len(snapshots))
	avgReserved := float64(totalReserved) / float64(len(snapshots))
	if avgQty > 0 {
		summary.TurnoverRate = avgReserved / avgQty
	}

	return summary, nil
}

func (s *Service) GetLogisticsPerformance(ctx context.Context, from, to time.Time) (analytics.LogisticsPerformance, error) {
	records, err := s.storage.GetLogisticsDaily(ctx, from, to)
	if err != nil {
		return analytics.LogisticsPerformance{}, fmt.Errorf("failed to get logistics daily: %w", err)
	}

	var perf analytics.LogisticsPerformance
	perf.DateFrom = from.Format("2006-01-02")
	perf.DateTo = to.Format("2006-01-02")

	if len(records) == 0 {
		return perf, nil
	}

	var totalDeliveryH float64
	for _, r := range records {
		perf.TotalShipments += r.TotalShipments
		perf.DeliveredCount += r.DeliveredCount
		perf.FailedCount += r.FailedCount
		totalDeliveryH += r.AvgDeliveryH * float64(r.DeliveredCount)
	}

	if perf.DeliveredCount > 0 {
		perf.AvgDeliveryH = totalDeliveryH / float64(perf.DeliveredCount)
	}
	if perf.TotalShipments > 0 {
		perf.OnTimeRate = float64(perf.DeliveredCount) / float64(perf.TotalShipments) * 100
	}

	return perf, nil
}

func (s *Service) DetectAnomalies(ctx context.Context, from, to time.Time) ([]analytics.Anomaly, error) {
	var anomalies []analytics.Anomaly

	salesRecords, err := s.storage.GetSalesDaily(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get sales daily: %w", err)
	}

	if len(salesRecords) >= 2 {
		anomalies = append(anomalies, s.detectSalesAnomalies(salesRecords)...)
	}

	logRecords, err := s.storage.GetLogisticsDaily(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get logistics daily: %w", err)
	}

	anomalies = append(anomalies, s.detectLogisticsAnomalies(logRecords)...)

	invRecords, err := s.storage.GetInventorySnapshots(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory snapshots: %w", err)
	}

	anomalies = append(anomalies, s.detectInventoryAnomalies(invRecords)...)

	if len(salesRecords) >= 7 {
		anomalies = append(anomalies, s.detectBusinessAnomalies(salesRecords)...)
	}

	if anomalies == nil {
		anomalies = []analytics.Anomaly{}
	}

	return anomalies, nil
}

func (s *Service) detectBusinessAnomalies(records []analytics.SalesDaily) []analytics.Anomaly {
	var anomalies []analytics.Anomaly

	var sumAOV, sumAOVSq float64
	var aovCount int
	for _, r := range records {
		if r.TotalOrders > 0 {
			aov := r.TotalRevenue / float64(r.TotalOrders)
			sumAOV += aov
			sumAOVSq += aov * aov
			aovCount++
		}
	}
	if aovCount >= 5 {
		mean := sumAOV / float64(aovCount)
		stddev := math.Sqrt(sumAOVSq/float64(aovCount) - mean*mean)
		if stddev > 0 {
			for _, r := range records {
				if r.TotalOrders == 0 {
					continue
				}
				aov := r.TotalRevenue / float64(r.TotalOrders)
				deviation := mean - aov
				if deviation > 2*stddev {
					severity := "warning"
					if deviation > 3*stddev {
						severity = "critical"
					}
					anomalies = append(anomalies, analytics.Anomaly{
						Category:  "business",
						Type:      "aov_drop",
						Metric:    "avg_order_value",
						Value:     aov,
						Threshold: mean - 2*stddev,
						Date:      r.Date.Format("2006-01-02"),
						Severity:  severity,
						Message: fmt.Sprintf(
							"Average order value %.2f dropped %.1fσ below the %d-day mean %.2f",
							aov, deviation/stddev, len(records), mean,
						),
					})
				}
			}
		}
	}

	return anomalies
}

func (s *Service) detectSalesAnomalies(records []analytics.SalesDaily) []analytics.Anomaly {
	var anomalies []analytics.Anomaly

	var sum, sumSq float64
	for _, r := range records {
		sum += r.TotalRevenue
		sumSq += r.TotalRevenue * r.TotalRevenue
	}
	n := float64(len(records))
	mean := sum / n
	stddev := math.Sqrt(sumSq/n - mean*mean)

	if stddev > 0 {
		for _, r := range records {
			deviation := math.Abs(r.TotalRevenue - mean)
			threshold := 2 * stddev
			if deviation > threshold {
				severity := "warning"
				if deviation > 3*stddev {
					severity = "critical"
				}
				anomalies = append(anomalies, analytics.Anomaly{
					Category:  "sales",
					Type:      "sales",
					Metric:    "total_revenue",
					Value:     r.TotalRevenue,
					Threshold: mean + threshold,
					Date:      r.Date.Format("2006-01-02"),
					Severity:  severity,
					Message:   fmt.Sprintf("Revenue %.2f deviates significantly from mean %.2f", r.TotalRevenue, mean),
				})
			}
		}
	}

	for _, r := range records {
		if r.TotalOrders == 0 {
			anomalies = append(anomalies, analytics.Anomaly{
				Category:  "sales",
				Type:      "sales",
				Metric:    "total_orders",
				Value:     0,
				Threshold: 1,
				Date:      r.Date.Format("2006-01-02"),
				Severity:  "warning",
				Message:   "Zero orders recorded for this day",
			})
		}
	}

	return anomalies
}

func (s *Service) detectLogisticsAnomalies(records []analytics.LogisticsDaily) []analytics.Anomaly {
	var anomalies []analytics.Anomaly

	for _, r := range records {
		if r.TotalShipments > 0 {
			failRate := float64(r.FailedCount) / float64(r.TotalShipments) * 100
			if failRate > 20 {
				anomalies = append(anomalies, analytics.Anomaly{
					Category:  "logistics",
					Type:      "logistics",
					Metric:    "failure_rate",
					Value:     failRate,
					Threshold: 20,
					Date:      r.Date.Format("2006-01-02"),
					Severity:  "critical",
					Message:   fmt.Sprintf("Shipment failure rate %.1f%% exceeds 20%% threshold", failRate),
				})
			}
		}

		if r.OnTimeRate > 0 && r.OnTimeRate < 80 {
			anomalies = append(anomalies, analytics.Anomaly{
				Category:  "logistics",
				Type:      "logistics",
				Metric:    "on_time_rate",
				Value:     r.OnTimeRate,
				Threshold: 80,
				Date:      r.Date.Format("2006-01-02"),
				Severity:  "warning",
				Message:   fmt.Sprintf("On-time delivery rate %.1f%% below 80%% threshold", r.OnTimeRate),
			})
		}
	}

	return anomalies
}

func (s *Service) detectInventoryAnomalies(records []analytics.InventorySnapshot) []analytics.Anomaly {
	var anomalies []analytics.Anomaly

	for _, r := range records {
		if r.TotalProducts > 0 {
			lowStockPct := float64(r.LowStockCount) / float64(r.TotalProducts) * 100
			if lowStockPct > 10 {
				anomalies = append(anomalies, analytics.Anomaly{
					Category:  "inventory",
					Type:      "inventory",
					Metric:    "low_stock_percentage",
					Value:     lowStockPct,
					Threshold: 10,
					Date:      r.Date.Format("2006-01-02"),
					Severity:  "warning",
					Message:   fmt.Sprintf("%.1f%% of products below minimum stock threshold", lowStockPct),
				})
			}
		}
	}

	return anomalies
}

func (s *Service) GetQuickCancellations(ctx context.Context, from, to time.Time, maxMinutes int) ([]analytics.QuickCancellation, error) {
	return s.storage.GetQuickCancellations(ctx, from, to, maxMinutes)
}

func (s *Service) GetRebalancingRecommendations(ctx context.Context, params analytics.RebalancingParams) ([]analytics.RebalancingRecommendation, error) {
	return s.storage.GetRebalancingRecommendations(ctx, params)
}

func (s *Service) GetCarrierPerformance(ctx context.Context, from, to time.Time, slaHours int, worstCitiesPerCarrier int) ([]analytics.CarrierPerformance, error) {
	return s.storage.GetCarrierPerformance(ctx, from, to, slaHours, worstCitiesPerCarrier)
}

func (s *Service) GetCustomerProfile360(ctx context.Context, customerName string, recentN int, topCategoriesN int) (analytics.CustomerProfile360, error) {
	return s.storage.GetCustomerProfile360(ctx, customerName, recentN, topCategoriesN)
}

func (s *Service) QueryAuditLog(ctx context.Context, filter analytics.AuditFilter) ([]analytics.AuditEntry, error) {
	return s.storage.QueryAuditLog(ctx, filter)
}

func (s *Service) RunWhatIf(ctx context.Context, scenario analytics.WhatIfScenario) (analytics.WhatIfResult, error) {
	switch scenario.Kind {
	case "carrier_drop":
		return s.whatIfCarrierDrop(ctx, scenario)
	case "capacity_increase":
		return s.whatIfCapacityIncrease(ctx, scenario)
	case "price_change":
		return s.whatIfPriceChange(ctx, scenario)
	case "promo_burst":
		return s.whatIfPromoBurst(ctx, scenario)
	default:
		return analytics.WhatIfResult{}, fmt.Errorf("unsupported scenario kind: %s (supported: carrier_drop, capacity_increase, price_change, promo_burst)", scenario.Kind)
	}
}

func (s *Service) whatIfCarrierDrop(ctx context.Context, scenario analytics.WhatIfScenario) (analytics.WhatIfResult, error) {
	carrierID, _ := scenario.Params["carrier_id"].(string)
	if carrierID == "" {
		return analytics.WhatIfResult{}, fmt.Errorf("carrier_id is required")
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -60)
	carriers, err := s.storage.GetCarrierPerformance(ctx, from, now, 168, 5)
	if err != nil {
		return analytics.WhatIfResult{}, err
	}
	var dropped *analytics.CarrierPerformance
	others := make([]analytics.CarrierPerformance, 0)
	for i := range carriers {
		if carriers[i].CarrierID == carrierID {
			dropped = &carriers[i]
		} else {
			others = append(others, carriers[i])
		}
	}
	if dropped == nil {
		return analytics.WhatIfResult{}, fmt.Errorf("carrier %s not found in 60-day window", carrierID)
	}

	var sumOnTime, sumDelivered float64
	var sumOthersOnTime, sumOthersDelivered float64
	for _, c := range carriers {
		sumOnTime += float64(c.OnTime)
		sumDelivered += float64(c.Delivered)
	}
	for _, c := range others {
		sumOthersOnTime += float64(c.OnTime)
		sumOthersDelivered += float64(c.Delivered)
	}
	baselineOnTime := 0.0
	if sumDelivered > 0 {
		baselineOnTime = sumOnTime / sumDelivered * 100
	}
	projectedOnTime := 0.0
	if sumOthersDelivered > 0 {
		projectedOnTime = sumOthersOnTime / sumOthersDelivered * 100
	}

	baseline := map[string]float64{
		"on_time_rate_pct":  baselineOnTime,
		"total_delivered":   sumDelivered,
		"dropped_delivered": float64(dropped.Delivered),
	}
	projected := map[string]float64{
		"on_time_rate_pct": projectedOnTime,
		"total_delivered":  sumOthersDelivered,
	}
	delta := map[string]float64{"on_time_rate_pct": projectedOnTime - baselineOnTime}
	deltaPct := map[string]float64{}
	if baselineOnTime > 0 {
		deltaPct["on_time_rate_pct"] = ((projectedOnTime - baselineOnTime) / baselineOnTime) * 100
	}

	return analytics.WhatIfResult{
		Scenario:     scenario,
		Baseline:     baseline,
		Projected:    projected,
		Delta:        delta,
		DeltaPercent: deltaPct,
		Assumptions: []string{
			"Demand is conserved — would-be shipments redistribute equally across remaining carriers",
			"Remaining carriers' historical on-time rate generalizes to extra load",
			"Ignores secondary capacity effects (overloaded carriers may degrade)",
		},
		Confidence: "medium",
		HumanSummary: fmt.Sprintf(
			"Dropping %s (delivered %d in last 60d, %.1f%% on-time) would shift the platform on-time rate from %.1f%% to %.1f%% (Δ %+.1fpp)",
			dropped.CarrierName, dropped.Delivered, dropped.OnTimeRate*100,
			baselineOnTime, projectedOnTime, projectedOnTime-baselineOnTime,
		),
	}, nil
}

func (s *Service) whatIfCapacityIncrease(_ context.Context, scenario analytics.WhatIfScenario) (analytics.WhatIfResult, error) {
	warehouse, _ := scenario.Params["warehouse_name"].(string)
	pctRaw := toFloat(scenario.Params["capacity_increase_pct"])
	if pctRaw <= 0 {
		return analytics.WhatIfResult{}, fmt.Errorf("capacity_increase_pct must be > 0")
	}
	if warehouse == "" {
		return analytics.WhatIfResult{}, fmt.Errorf("warehouse_name is required")
	}
	baseline := map[string]float64{
		"warehouse_overflow_index": 1.0,
	}
	projected := map[string]float64{
		"warehouse_overflow_index": 1.0 / (1.0 + pctRaw/100.0),
	}
	delta := map[string]float64{"warehouse_overflow_index": projected["warehouse_overflow_index"] - baseline["warehouse_overflow_index"]}
	deltaPct := map[string]float64{"warehouse_overflow_index": -pctRaw / (1.0 + pctRaw/100.0)}

	return analytics.WhatIfResult{
		Scenario:     scenario,
		Baseline:     baseline,
		Projected:    projected,
		Delta:        delta,
		DeltaPercent: deltaPct,
		Assumptions: []string{
			"Throughput scales linearly with capacity (no diminishing returns)",
			"No new bottlenecks elsewhere (carriers, picking) introduced",
			"Overflow index = current_load / capacity, so increasing capacity by X% divides the index by (1 + X/100)",
		},
		Confidence: "low",
		HumanSummary: fmt.Sprintf(
			"Increasing %s capacity by %.0f%% would reduce its load-vs-capacity index from 1.00 to %.2f (assuming linear scaling)",
			warehouse, pctRaw, projected["warehouse_overflow_index"],
		),
	}, nil
}

func (s *Service) whatIfPriceChange(ctx context.Context, scenario analytics.WhatIfScenario) (analytics.WhatIfResult, error) {
	category, _ := scenario.Params["category"].(string)
	pct := toFloat(scenario.Params["price_change_pct"])
	elasticity := -1.0
	if e := toFloat(scenario.Params["elasticity"]); e != 0 {
		elasticity = e
	}
	if category == "" {
		return analytics.WhatIfResult{}, fmt.Errorf("category is required")
	}

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)
	baselineRevenue, err := s.storage.GetMetricValue(ctx, "revenue", from, now)
	if err != nil {
		return analytics.WhatIfResult{}, err
	}
	categoryShare := 1.0 / 11.0
	if v := toFloat(scenario.Params["category_revenue_share"]); v > 0 && v <= 1 {
		categoryShare = v
	}

	priceFactor := 1.0 + pct/100.0
	demandFactor := 1.0 + elasticity*(pct/100.0)
	if demandFactor < 0 {
		demandFactor = 0
	}
	revenueFactor := priceFactor * demandFactor

	categoryBaseline := baselineRevenue * categoryShare
	categoryProjected := categoryBaseline * revenueFactor
	rest := baselineRevenue * (1 - categoryShare)
	totalProjected := rest + categoryProjected

	baseline := map[string]float64{
		"category_revenue_30d": categoryBaseline,
		"total_revenue_30d":    baselineRevenue,
	}
	projected := map[string]float64{
		"category_revenue_30d": categoryProjected,
		"total_revenue_30d":    totalProjected,
	}
	delta := map[string]float64{
		"category_revenue_30d": categoryProjected - categoryBaseline,
		"total_revenue_30d":    totalProjected - baselineRevenue,
	}
	deltaPct := map[string]float64{
		"category_revenue_30d": (revenueFactor - 1) * 100,
	}
	if baselineRevenue > 0 {
		deltaPct["total_revenue_30d"] = ((totalProjected - baselineRevenue) / baselineRevenue) * 100
	}

	return analytics.WhatIfResult{
		Scenario:     scenario,
		Baseline:     baseline,
		Projected:    projected,
		Delta:        delta,
		DeltaPercent: deltaPct,
		Assumptions: []string{
			fmt.Sprintf("Constant price elasticity of demand = %.2f (default -1.0, override via 'elasticity' param)", elasticity),
			fmt.Sprintf("Category '%s' carries %.0f%% of total revenue (override via 'category_revenue_share')", category, categoryShare*100),
			"30-day baseline; ignores seasonality, cross-elasticity, competitor response",
		},
		Confidence: "low",
		HumanSummary: fmt.Sprintf(
			"Changing %s prices by %+.0f%% (elasticity %.2f) yields category revenue Δ %+.1f%% and total revenue Δ %+.1f%%",
			category, pct, elasticity, deltaPct["category_revenue_30d"], deltaPct["total_revenue_30d"],
		),
	}, nil
}

func (s *Service) whatIfPromoBurst(ctx context.Context, scenario analytics.WhatIfScenario) (analytics.WhatIfResult, error) {
	multiplier := toFloat(scenario.Params["order_multiplier"])
	durationDays := int(toFloat(scenario.Params["duration_days"]))
	if multiplier <= 0 {
		return analytics.WhatIfResult{}, fmt.Errorf("order_multiplier must be > 0")
	}
	if durationDays <= 0 {
		durationDays = 7
	}

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)
	baselineRevenue, err := s.storage.GetMetricValue(ctx, "revenue", from, now)
	if err != nil {
		return analytics.WhatIfResult{}, err
	}
	dailyAvg := baselineRevenue / 30.0
	burstExtra := dailyAvg * float64(durationDays) * (multiplier - 1.0)

	baseline := map[string]float64{
		"projected_revenue_burst_window": dailyAvg * float64(durationDays),
	}
	projected := map[string]float64{
		"projected_revenue_burst_window": dailyAvg * float64(durationDays) * multiplier,
	}
	delta := map[string]float64{
		"projected_revenue_burst_window": burstExtra,
	}
	deltaPct := map[string]float64{
		"projected_revenue_burst_window": (multiplier - 1.0) * 100,
	}

	return analytics.WhatIfResult{
		Scenario:     scenario,
		Baseline:     baseline,
		Projected:    projected,
		Delta:        delta,
		DeltaPercent: deltaPct,
		Assumptions: []string{
			fmt.Sprintf("Burst multiplier %.2fx applied uniformly to recent-30d daily average", multiplier),
			"Assumes inventory and carrier capacity can absorb the burst (otherwise stock-out / late shipments will dampen the realized number)",
			fmt.Sprintf("Burst window: %d days", durationDays),
		},
		Confidence: "low",
		HumanSummary: fmt.Sprintf(
			"A %dx demand burst over %d days would add ~$%.0f revenue (vs $%.0f baseline)",
			int(multiplier), durationDays, burstExtra, baseline["projected_revenue_burst_window"],
		),
	}, nil
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	}
	return 0
}

func (s *Service) GetForecast(
	ctx context.Context, metric, method string,
	historyDays, horizonDays int,
) (analytics.Forecast, error) {
	if historyDays <= 0 {
		historyDays = 30
	}
	if horizonDays <= 0 {
		horizonDays = 14
	}
	if method == "" {
		method = "linear"
	}

	now := time.Now().UTC().Truncate(24 * time.Hour)
	from := now.AddDate(0, 0, -historyDays)
	to := now

	history, err := s.storage.GetDailyMetricSeries(ctx, metric, from, to)
	if err != nil {
		return analytics.Forecast{}, err
	}

	if len(history) < 3 {
		return analytics.Forecast{}, fmt.Errorf("insufficient history (%d days) — need at least 3 points to forecast", len(history))
	}

	historyFilled := fillMissingDays(history, from, to)

	values := make([]float64, len(historyFilled))
	for i, p := range historyFilled {
		values[i] = p.Value
	}

	var forecast []analytics.ForecastPoint
	var mape float64
	switch method {
	case "rolling-avg":
		forecast = forecastRollingAvg(historyFilled, horizonDays, 7)
		mape = backtestMAPE(values, func(train []float64, h int) []float64 {
			pts := forecastRollingAvg(makeFakePoints(train, now), h, 7)
			out := make([]float64, len(pts))
			for i, p := range pts {
				out[i] = p.Value
			}
			return out
		})
	case "ets-simple":
		forecast = forecastETS(historyFilled, horizonDays, 0.3, 0.2)
		mape = backtestMAPE(values, func(train []float64, h int) []float64 {
			pts := forecastETS(makeFakePoints(train, now), h, 0.3, 0.2)
			out := make([]float64, len(pts))
			for i, p := range pts {
				out[i] = p.Value
			}
			return out
		})
	default:
		method = "linear"
		forecast = forecastLinear(historyFilled, horizonDays)
		mape = backtestMAPE(values, func(train []float64, h int) []float64 {
			pts := forecastLinear(makeFakePoints(train, now), h)
			out := make([]float64, len(pts))
			for i, p := range pts {
				out[i] = p.Value
			}
			return out
		})
	}

	confidence := "low"
	if mape <= 15 {
		confidence = "high"
	} else if mape <= 30 {
		confidence = "medium"
	}

	return analytics.Forecast{
		Metric:        metric,
		Method:        method,
		HorizonDays:   horizonDays,
		HistoryWindow: historyDays,
		History:       historyFilled,
		Forecast:      forecast,
		BacktestMAPE:  mape,
		Confidence:    confidence,
		Assumptions: []string{
			fmt.Sprintf("History window: trailing %d days", historyDays),
			"Missing days filled with zero",
			"Confidence interval is mean ± 1.5σ of residuals",
			"Linear assumes constant trend; rolling-avg assumes recent stationarity; ets-simple captures level + trend",
		},
	}, nil
}

func fillMissingDays(points []analytics.ForecastPoint, from, to time.Time) []analytics.ForecastPoint {
	have := map[string]float64{}
	for _, p := range points {
		have[p.Date.Format("2006-01-02")] = p.Value
	}
	var out []analytics.ForecastPoint
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		out = append(out, analytics.ForecastPoint{Date: d, Value: have[d.Format("2006-01-02")]})
	}
	return out
}

func makeFakePoints(values []float64, end time.Time) []analytics.ForecastPoint {
	out := make([]analytics.ForecastPoint, len(values))
	for i, v := range values {
		out[i] = analytics.ForecastPoint{
			Date:  end.AddDate(0, 0, -(len(values) - 1 - i)),
			Value: v,
		}
	}
	return out
}

func forecastLinear(history []analytics.ForecastPoint, horizon int) []analytics.ForecastPoint {
	n := float64(len(history))
	if n < 2 {
		return nil
	}

	var sumX, sumY, sumXY, sumXX float64
	for i, p := range history {
		x := float64(i)
		sumX += x
		sumY += p.Value
		sumXY += x * p.Value
		sumXX += x * x
	}
	denom := n*sumXX - sumX*sumX
	slope, intercept := 0.0, sumY/n
	if denom != 0 {
		slope = (n*sumXY - sumX*sumY) / denom
		intercept = (sumY - slope*sumX) / n
	}

	var residuals []float64
	for i, p := range history {
		pred := intercept + slope*float64(i)
		residuals = append(residuals, p.Value-pred)
	}
	band := 1.5 * stddev(residuals)

	last := history[len(history)-1].Date
	out := make([]analytics.ForecastPoint, 0, horizon)
	for i := 1; i <= horizon; i++ {
		x := float64(len(history) - 1 + i)
		v := intercept + slope*x
		if v < 0 {
			v = 0
		}
		low := v - band
		if low < 0 {
			low = 0
		}
		out = append(out, analytics.ForecastPoint{
			Date:           last.AddDate(0, 0, i),
			Value:          v,
			ConfidenceLow:  low,
			ConfidenceHigh: v + band,
		})
	}
	return out
}

func forecastRollingAvg(history []analytics.ForecastPoint, horizon, window int) []analytics.ForecastPoint {
	if len(history) == 0 {
		return nil
	}
	if window > len(history) {
		window = len(history)
	}
	sum := 0.0
	for i := len(history) - window; i < len(history); i++ {
		sum += history[i].Value
	}
	avg := sum / float64(window)

	var residuals []float64
	for i := window; i < len(history); i++ {
		s := 0.0
		for j := i - window; j < i; j++ {
			s += history[j].Value
		}
		residuals = append(residuals, history[i].Value-s/float64(window))
	}
	band := 1.5 * stddev(residuals)

	last := history[len(history)-1].Date
	out := make([]analytics.ForecastPoint, 0, horizon)
	for i := 1; i <= horizon; i++ {
		low := avg - band
		if low < 0 {
			low = 0
		}
		out = append(out, analytics.ForecastPoint{
			Date:           last.AddDate(0, 0, i),
			Value:          avg,
			ConfidenceLow:  low,
			ConfidenceHigh: avg + band,
		})
	}
	return out
}

func forecastETS(history []analytics.ForecastPoint, horizon int, alpha, beta float64) []analytics.ForecastPoint {
	n := len(history)
	if n < 2 {
		return nil
	}

	level := history[0].Value
	trend := history[1].Value - history[0].Value
	var residuals []float64
	for i := 1; i < n; i++ {
		pred := level + trend
		residuals = append(residuals, history[i].Value-pred)
		newLevel := alpha*history[i].Value + (1-alpha)*(level+trend)
		newTrend := beta*(newLevel-level) + (1-beta)*trend
		level, trend = newLevel, newTrend
	}
	band := 1.5 * stddev(residuals)

	last := history[n-1].Date
	out := make([]analytics.ForecastPoint, 0, horizon)
	for i := 1; i <= horizon; i++ {
		v := level + float64(i)*trend
		if v < 0 {
			v = 0
		}
		low := v - band
		if low < 0 {
			low = 0
		}
		out = append(out, analytics.ForecastPoint{
			Date:           last.AddDate(0, 0, i),
			Value:          v,
			ConfidenceLow:  low,
			ConfidenceHigh: v + band,
		})
	}
	return out
}

func stddev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var sum, sumSq float64
	for _, x := range xs {
		sum += x
		sumSq += x * x
	}
	n := float64(len(xs))
	mean := sum / n
	variance := sumSq/n - mean*mean
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}

func backtestMAPE(values []float64, forecastFn func(train []float64, h int) []float64) float64 {
	if len(values) < 14 {
		return 0
	}
	split := int(float64(len(values)) * 0.8)
	if split < 7 || split >= len(values) {
		return 0
	}
	train := values[:split]
	test := values[split:]
	predicted := forecastFn(train, len(test))
	if len(predicted) != len(test) {
		return 0
	}
	var sum float64
	count := 0
	for i := range test {
		if test[i] == 0 {
			continue
		}
		sum += math.Abs((test[i] - predicted[i]) / test[i])
		count++
	}
	if count == 0 {
		return 0
	}
	return (sum / float64(count)) * 100.0
}

func (s *Service) GetPeriodComparison(
	ctx context.Context, metric string,
	aFrom, aTo, bFrom, bTo time.Time,
	aLabel, bLabel string,
) (analytics.PeriodComparison, error) {
	aVal, err := s.storage.GetMetricValue(ctx, metric, aFrom, aTo)
	if err != nil {
		return analytics.PeriodComparison{}, err
	}
	bVal, err := s.storage.GetMetricValue(ctx, metric, bFrom, bTo)
	if err != nil {
		return analytics.PeriodComparison{}, err
	}

	if aLabel == "" {
		aLabel = aFrom.Format("2006-01-02") + " — " + aTo.Format("2006-01-02")
	}
	if bLabel == "" {
		bLabel = bFrom.Format("2006-01-02") + " — " + bTo.Format("2006-01-02")
	}

	delta := aVal - bVal
	pct := 0.0
	if bVal != 0 {
		pct = (delta / bVal) * 100.0
	}

	direction := "flat"
	if delta > 0 {
		direction = "up"
	} else if delta < 0 {
		direction = "down"
	}

	absPct := pct
	if absPct < 0 {
		absPct = -absPct
	}
	significance := "noise"
	switch {
	case absPct >= 15.0:
		significance = "major"
	case absPct >= 5.0:
		significance = "minor"
	}

	return analytics.PeriodComparison{
		Metric: metric,
		PeriodA: analytics.PeriodSnapshot{
			Label: aLabel, From: aFrom, To: aTo, Value: aVal,
		},
		PeriodB: analytics.PeriodSnapshot{
			Label: bLabel, From: bFrom, To: bTo, Value: bVal,
		},
		AbsoluteDelta: delta,
		PercentChange: pct,
		Direction:     direction,
		Significance:  significance,
	}, nil
}

func (s *Service) GetOptimizations(ctx context.Context, from, to time.Time) ([]analytics.Optimization, error) {
	const lookbackDays = 30
	const leadTimeDays = 7
	const safetyFactor = 1.5
	const limit = 50

	candidates, err := s.storage.GetReorderCandidates(ctx, lookbackDays, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get reorder candidates: %w", err)
	}

	optimizations := make([]analytics.Optimization, 0, len(candidates))
	for _, c := range candidates {
		avgDailyDemand := float64(c.OutboundLast30Days) / float64(lookbackDays)
		if avgDailyDemand < 0 {
			avgDailyDemand = 0
		}

		safetyStock := int(math.Ceil(avgDailyDemand * safetyFactor))
		reorderPoint := int(math.Ceil(avgDailyDemand*leadTimeDays)) + safetyStock
		if reorderPoint < c.MinThreshold {
			reorderPoint = c.MinThreshold
		}

		targetStock := reorderPoint*2 + safetyStock
		recommendedOrder := targetStock - c.Available
		if recommendedOrder < c.MinThreshold {
			recommendedOrder = c.MinThreshold
		}
		if recommendedOrder < 1 {
			recommendedOrder = 1
		}

		daysUntilStockout := 999.0
		if avgDailyDemand > 0 {
			daysUntilStockout = float64(c.Available) / avgDailyDemand
			if daysUntilStockout > 999 || math.IsInf(daysUntilStockout, 0) || math.IsNaN(daysUntilStockout) {
				daysUntilStockout = 999
			}
		}

		urgency := "info"
		switch {
		case c.Available <= 0:
			urgency = "critical"
		case avgDailyDemand > 0 && daysUntilStockout < float64(leadTimeDays):
			urgency = "critical"
		case c.Available*2 < c.MinThreshold:
			urgency = "warning"
		case avgDailyDemand > 0 && daysUntilStockout < float64(leadTimeDays)*2:
			urgency = "warning"
		}

		var reason string
		switch {
		case c.Available <= 0:
			reason = fmt.Sprintf(
				"Stock-out at %s. Sold %d units in last %d days (~%.1f/day). Order at least %d units to cover lead time.",
				c.WarehouseName, c.OutboundLast30Days, lookbackDays, avgDailyDemand, recommendedOrder,
			)
		case avgDailyDemand > 0:
			reason = fmt.Sprintf(
				"Available %d at %s — will run out in ~%.1f days at current pace (%.1f units/day). Lead time %d days.",
				c.Available, c.WarehouseName, daysUntilStockout, avgDailyDemand, leadTimeDays,
			)
		default:
			reason = fmt.Sprintf(
				"Available %d at %s is below threshold %d. No demand recorded in last %d days — restock to threshold or deactivate SKU.",
				c.Available, c.WarehouseName, c.MinThreshold, lookbackDays,
			)
		}

		optimizations = append(optimizations, analytics.Optimization{
			ProductID:         c.ProductID,
			ProductSKU:        c.SKU,
			ProductName:       c.ProductName,
			WarehouseID:       c.WarehouseID,
			WarehouseName:     c.WarehouseName,
			CurrentStock:      c.Available,
			MinThreshold:      c.MinThreshold,
			AvgDailyDemand:    math.Round(avgDailyDemand*100) / 100,
			ReorderPoint:      reorderPoint,
			RecommendedOrder:  recommendedOrder,
			DaysUntilStockout: math.Round(daysUntilStockout*10) / 10,
			Urgency:           urgency,
			Reason:            reason,
		})
	}

	_ = from
	_ = to
	return optimizations, nil
}

func (s *Service) GenerateReport(ctx context.Context, req analytics.ReportRequest) (analytics.Report, error) {
	from, err := time.Parse("2006-01-02", req.DateFrom)
	if err != nil {
		return analytics.Report{}, fmt.Errorf("invalid date_from: %w", err)
	}
	to, err := time.Parse("2006-01-02", req.DateTo)
	if err != nil {
		return analytics.Report{}, fmt.Errorf("invalid date_to: %w", err)
	}

	report := analytics.Report{
		ReportType:  req.ReportType,
		DateFrom:    req.DateFrom,
		DateTo:      req.DateTo,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	switch req.ReportType {
	case "sales":
		data, rErr := s.storage.GetSalesDaily(ctx, from, to)
		if rErr != nil {
			return analytics.Report{}, fmt.Errorf("failed to get sales data: %w", rErr)
		}
		report.Data = data
	case "inventory":
		data, rErr := s.storage.GetInventorySnapshots(ctx, from, to)
		if rErr != nil {
			return analytics.Report{}, fmt.Errorf("failed to get inventory data: %w", rErr)
		}
		report.Data = data
	case "logistics":
		data, rErr := s.storage.GetLogisticsDaily(ctx, from, to)
		if rErr != nil {
			return analytics.Report{}, fmt.Errorf("failed to get logistics data: %w", rErr)
		}
		report.Data = data
	case "full":
		sales, rErr := s.storage.GetSalesDaily(ctx, from, to)
		if rErr != nil {
			return analytics.Report{}, fmt.Errorf("failed to get sales data: %w", rErr)
		}
		inv, rErr := s.storage.GetInventorySnapshots(ctx, from, to)
		if rErr != nil {
			return analytics.Report{}, fmt.Errorf("failed to get inventory data: %w", rErr)
		}
		logs, rErr := s.storage.GetLogisticsDaily(ctx, from, to)
		if rErr != nil {
			return analytics.Report{}, fmt.Errorf("failed to get logistics data: %w", rErr)
		}
		report.Data = map[string]any{
			"sales":     sales,
			"inventory": inv,
			"logistics": logs,
		}
	default:
		return analytics.Report{}, fmt.Errorf("unsupported report type: %s", req.ReportType)
	}

	return report, nil
}
