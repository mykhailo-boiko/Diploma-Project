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

	if anomalies == nil {
		anomalies = []analytics.Anomaly{}
	}

	return anomalies, nil
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

	delta := bVal - aVal
	pct := 0.0
	if aVal != 0 {
		pct = (delta / aVal) * 100.0
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
	salesRecords, err := s.storage.GetSalesDaily(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get sales daily: %w", err)
	}

	snapshots, err := s.storage.GetInventorySnapshots(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory snapshots: %w", err)
	}

	var optimizations []analytics.Optimization

	if len(salesRecords) == 0 || len(snapshots) == 0 {
		return optimizations, nil
	}

	var totalOrders int
	for _, r := range salesRecords {
		totalOrders += r.TotalOrders
	}
	avgDailyDemand := float64(totalOrders) / float64(len(salesRecords))

	latest := snapshots[len(snapshots)-1]

	const leadTimeDays = 7
	const safetyFactor = 1.5

	safetyStock := int(math.Ceil(avgDailyDemand * safetyFactor))
	reorderPoint := int(math.Ceil(avgDailyDemand*leadTimeDays)) + safetyStock
	recommendedQty := int(math.Ceil(avgDailyDemand * leadTimeDays * 2))

	if latest.TotalAvailable < reorderPoint {
		optimizations = append(optimizations, analytics.Optimization{
			Type:           "reorder",
			ProductMetric:  "all_products",
			CurrentStock:   latest.TotalAvailable,
			AvgDemand:      avgDailyDemand,
			ReorderPoint:   reorderPoint,
			RecommendedQty: recommendedQty,
			SafetyStock:    safetyStock,
			Message: fmt.Sprintf(
				"Current available stock (%d) is below reorder point (%d). Recommended order quantity: %d units",
				latest.TotalAvailable, reorderPoint, recommendedQty,
			),
		})
	}

	if latest.LowStockCount > 0 {
		optimizations = append(optimizations, analytics.Optimization{
			Type:           "low_stock_alert",
			ProductMetric:  "low_stock_products",
			CurrentStock:   latest.LowStockCount,
			AvgDemand:      avgDailyDemand,
			ReorderPoint:   reorderPoint,
			RecommendedQty: recommendedQty,
			SafetyStock:    safetyStock,
			Message: fmt.Sprintf(
				"%d products are below their minimum stock threshold. Immediate restocking recommended",
				latest.LowStockCount,
			),
		})
	}

	if optimizations == nil {
		optimizations = []analytics.Optimization{}
	}

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
