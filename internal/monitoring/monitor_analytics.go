package monitoring

import (
	"fmt"
	"math"
	"sort"
	"time"

	"go.uber.org/zap"
)

// GetPerformanceReport generates a comprehensive performance report
func (pt *PerformanceTracker) GetPerformanceReport(period string) (*PerformanceReport, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if len(pt.metricsHistory) == 0 {
		return nil, fmt.Errorf("no metrics data available")
	}

	startTime := pt.metricsHistory[0].Timestamp
	endTime := pt.metricsHistory[len(pt.metricsHistory)-1].Timestamp

	summary := pt.calculatePerformanceSummary()
	trends := pt.calculateTrends()
	anomalies := pt.detectAnomalies()
	recommendations := pt.generateRecommendations(summary, anomalies)

	return &PerformanceReport{
		Period:          period,
		StartTime:       startTime,
		EndTime:         endTime,
		Summary:         summary,
		Trends:          trends,
		Anomalies:       anomalies,
		Recommendations: recommendations,
	}, nil
}

// calculatePerformanceSummary calculates performance summary statistics
func (pt *PerformanceTracker) calculatePerformanceSummary() *PerformanceSummary {
	if len(pt.metricsHistory) == 0 {
		return &PerformanceSummary{}
	}

	var (
		totalOrdersPerSecond float64
		totalTradesPerSecond float64
		totalMatchingLatency float64
		totalCPUUsage        float64
		totalMemoryUsage     float64
		totalErrorRate       float64
		totalResponseTime    float64

		maxOrdersPerSecond float64
		maxTradesPerSecond float64
		maxMatchingLatency float64
		maxCPUUsage        float64
		maxMemoryUsage     float64
		maxErrorRate       float64
		maxResponseTime    float64
	)

	count := float64(len(pt.metricsHistory))

	for _, metrics := range pt.metricsHistory {
		totalOrdersPerSecond += metrics.OrdersPerSecond
		totalTradesPerSecond += metrics.TradesPerSecond
		totalMatchingLatency += metrics.MatchingLatency
		totalCPUUsage += metrics.CPUUsage
		totalMemoryUsage += metrics.MemoryUsage
		totalErrorRate += metrics.ErrorRate
		totalResponseTime += metrics.ResponseTime

		if metrics.OrdersPerSecond > maxOrdersPerSecond {
			maxOrdersPerSecond = metrics.OrdersPerSecond
		}
		if metrics.TradesPerSecond > maxTradesPerSecond {
			maxTradesPerSecond = metrics.TradesPerSecond
		}
		if metrics.MatchingLatency > maxMatchingLatency {
			maxMatchingLatency = metrics.MatchingLatency
		}
		if metrics.CPUUsage > maxCPUUsage {
			maxCPUUsage = metrics.CPUUsage
		}
		if metrics.MemoryUsage > maxMemoryUsage {
			maxMemoryUsage = metrics.MemoryUsage
		}
		if metrics.ErrorRate > maxErrorRate {
			maxErrorRate = metrics.ErrorRate
		}
		if metrics.ResponseTime > maxResponseTime {
			maxResponseTime = metrics.ResponseTime
		}
	}

	return &PerformanceSummary{
		AvgOrdersPerSecond:  totalOrdersPerSecond / count,
		AvgTradesPerSecond:  totalTradesPerSecond / count,
		AvgMatchingLatency:  totalMatchingLatency / count,
		AvgCPUUsage:         totalCPUUsage / count,
		AvgMemoryUsage:      totalMemoryUsage / count,
		AvgErrorRate:        totalErrorRate / count,
		AvgResponseTime:     totalResponseTime / count,
		PeakOrdersPerSecond: maxOrdersPerSecond,
		PeakTradesPerSecond: maxTradesPerSecond,
		MaxMatchingLatency:  maxMatchingLatency,
		MaxCPUUsage:         maxCPUUsage,
		MaxMemoryUsage:      maxMemoryUsage,
		MaxErrorRate:        maxErrorRate,
		MaxResponseTime:     maxResponseTime,
	}
}

// calculateTrends calculates performance trends
func (pt *PerformanceTracker) calculateTrends() map[string]float64 {
	trends := make(map[string]float64)

	if len(pt.metricsHistory) < 2 {
		return trends
	}

	// Calculate simple linear trends
	trends["orders_per_second"] = pt.calculateLinearTrend("orders_per_second")
	trends["trades_per_second"] = pt.calculateLinearTrend("trades_per_second")
	trends["matching_latency"] = pt.calculateLinearTrend("matching_latency")
	trends["cpu_usage"] = pt.calculateLinearTrend("cpu_usage")
	trends["memory_usage"] = pt.calculateLinearTrend("memory_usage")
	trends["error_rate"] = pt.calculateLinearTrend("error_rate")
	trends["response_time"] = pt.calculateLinearTrend("response_time")

	return trends
}

// calculateLinearTrend calculates linear trend for a metric
func (pt *PerformanceTracker) calculateLinearTrend(metric string) float64 {
	if len(pt.metricsHistory) < 2 {
		return 0
	}

	n := len(pt.metricsHistory)
	var sumX, sumY, sumXY, sumX2 float64

	for i, metrics := range pt.metricsHistory {
		x := float64(i)
		var y float64

		switch metric {
		case "orders_per_second":
			y = metrics.OrdersPerSecond
		case "trades_per_second":
			y = metrics.TradesPerSecond
		case "matching_latency":
			y = metrics.MatchingLatency
		case "cpu_usage":
			y = metrics.CPUUsage
		case "memory_usage":
			y = metrics.MemoryUsage
		case "error_rate":
			y = metrics.ErrorRate
		case "response_time":
			y = metrics.ResponseTime
		}

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope (trend)
	nf := float64(n)
	slope := (nf*sumXY - sumX*sumY) / (nf*sumX2 - sumX*sumX)

	return slope
}

// detectAnomalies detects performance anomalies
func (pt *PerformanceTracker) detectAnomalies() []*PerformanceAnomaly {
	var anomalies []*PerformanceAnomaly

	if len(pt.metricsHistory) < 10 {
		return anomalies
	}

	// Calculate baseline statistics
	baseline := pt.calculateBaseline()

	// Check recent metrics for anomalies
	recentCount := int(math.Min(10, float64(len(pt.metricsHistory))))
	recentMetrics := pt.metricsHistory[len(pt.metricsHistory)-recentCount:]

	for _, metrics := range recentMetrics {
		anomalies = append(anomalies, pt.checkMetricAnomalies(metrics, baseline)...)
	}

	return anomalies
}

// calculateBaseline calculates baseline statistics for anomaly detection
func (pt *PerformanceTracker) calculateBaseline() map[string]map[string]float64 {
	baseline := make(map[string]map[string]float64)

	metrics := []string{
		"orders_per_second", "trades_per_second", "matching_latency",
		"cpu_usage", "memory_usage", "error_rate", "response_time",
	}

	for _, metric := range metrics {
		values := pt.getMetricValues(metric)
		baseline[metric] = pt.calculateStatistics(values)
	}

	return baseline
}

// getMetricValues extracts values for a specific metric
func (pt *PerformanceTracker) getMetricValues(metric string) []float64 {
	var values []float64

	for _, m := range pt.metricsHistory {
		var value float64
		switch metric {
		case "orders_per_second":
			value = m.OrdersPerSecond
		case "trades_per_second":
			value = m.TradesPerSecond
		case "matching_latency":
			value = m.MatchingLatency
		case "cpu_usage":
			value = m.CPUUsage
		case "memory_usage":
			value = m.MemoryUsage
		case "error_rate":
			value = m.ErrorRate
		case "response_time":
			value = m.ResponseTime
		}
		values = append(values, value)
	}

	return values
}

// calculateStatistics calculates mean, std dev, and percentiles
func (pt *PerformanceTracker) calculateStatistics(values []float64) map[string]float64 {
	if len(values) == 0 {
		return map[string]float64{}
	}

	// Sort values for percentile calculation
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate standard deviation
	var sumSquaredDiff float64
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(values)))

	// Calculate percentiles
	p50 := sorted[len(sorted)/2]
	p95 := sorted[int(float64(len(sorted))*0.95)]
	p99 := sorted[int(float64(len(sorted))*0.99)]

	return map[string]float64{
		"mean":   mean,
		"stddev": stdDev,
		"p50":    p50,
		"p95":    p95,
		"p99":    p99,
	}
}

// checkMetricAnomalies checks for anomalies in a single metrics snapshot
func (pt *PerformanceTracker) checkMetricAnomalies(metrics *SystemMetrics, baseline map[string]map[string]float64) []*PerformanceAnomaly {
	var anomalies []*PerformanceAnomaly

	// Check each metric for anomalies
	anomalies = append(anomalies, pt.checkSingleMetricAnomaly("orders_per_second", metrics.OrdersPerSecond, baseline["orders_per_second"], metrics.Timestamp)...)
	anomalies = append(anomalies, pt.checkSingleMetricAnomaly("trades_per_second", metrics.TradesPerSecond, baseline["trades_per_second"], metrics.Timestamp)...)
	anomalies = append(anomalies, pt.checkSingleMetricAnomaly("matching_latency", metrics.MatchingLatency, baseline["matching_latency"], metrics.Timestamp)...)
	anomalies = append(anomalies, pt.checkSingleMetricAnomaly("cpu_usage", metrics.CPUUsage, baseline["cpu_usage"], metrics.Timestamp)...)
	anomalies = append(anomalies, pt.checkSingleMetricAnomaly("memory_usage", metrics.MemoryUsage, baseline["memory_usage"], metrics.Timestamp)...)
	anomalies = append(anomalies, pt.checkSingleMetricAnomaly("error_rate", metrics.ErrorRate, baseline["error_rate"], metrics.Timestamp)...)
	anomalies = append(anomalies, pt.checkSingleMetricAnomaly("response_time", metrics.ResponseTime, baseline["response_time"], metrics.Timestamp)...)

	return anomalies
}

// checkSingleMetricAnomaly checks for anomalies in a single metric
func (pt *PerformanceTracker) checkSingleMetricAnomaly(metricName string, value float64, stats map[string]float64, timestamp time.Time) []*PerformanceAnomaly {
	var anomalies []*PerformanceAnomaly

	if len(stats) == 0 {
		return anomalies
	}

	mean := stats["mean"]
	stdDev := stats["stddev"]
	p95 := stats["p95"]

	// Check for statistical anomalies (beyond 3 standard deviations)
	if math.Abs(value-mean) > 3*stdDev {
		severity := "warning"
		if math.Abs(value-mean) > 4*stdDev {
			severity = "critical"
		}

		anomalies = append(anomalies, &PerformanceAnomaly{
			Type:        "statistical",
			Metric:      metricName,
			Value:       value,
			Expected:    mean,
			Deviation:   math.Abs(value-mean) / stdDev,
			Timestamp:   timestamp,
			Severity:    severity,
			Description: fmt.Sprintf("%s value %.2f is %.1f standard deviations from mean %.2f", metricName, value, math.Abs(value-mean)/stdDev, mean),
		})
	}

	// Check for threshold anomalies (beyond 95th percentile)
	if value > p95*1.5 {
		anomalies = append(anomalies, &PerformanceAnomaly{
			Type:        "threshold",
			Metric:      metricName,
			Value:       value,
			Expected:    p95,
			Deviation:   (value - p95) / p95 * 100,
			Timestamp:   timestamp,
			Severity:    "warning",
			Description: fmt.Sprintf("%s value %.2f exceeds 95th percentile threshold %.2f by %.1f%%", metricName, value, p95, (value-p95)/p95*100),
		})
	}

	return anomalies
}

// generateRecommendations generates performance recommendations
func (pt *PerformanceTracker) generateRecommendations(summary *PerformanceSummary, anomalies []*PerformanceAnomaly) []string {
	var recommendations []string

	// CPU usage recommendations
	if summary.AvgCPUUsage > 80 {
		recommendations = append(recommendations, "High CPU usage detected. Consider scaling horizontally or optimizing CPU-intensive operations.")
	}

	// Memory usage recommendations
	if summary.AvgMemoryUsage > 85 {
		recommendations = append(recommendations, "High memory usage detected. Consider increasing memory allocation or optimizing memory usage.")
	}

	// Latency recommendations
	if summary.AvgMatchingLatency > 10 {
		recommendations = append(recommendations, "High matching latency detected. Consider optimizing matching algorithms or increasing processing capacity.")
	}

	// Error rate recommendations
	if summary.AvgErrorRate > 1 {
		recommendations = append(recommendations, "Elevated error rate detected. Review error logs and implement additional error handling.")
	}

	// Response time recommendations
	if summary.AvgResponseTime > 500 {
		recommendations = append(recommendations, "High response times detected. Consider optimizing database queries and API endpoints.")
	}

	// Anomaly-based recommendations
	criticalAnomalies := 0
	for _, anomaly := range anomalies {
		if anomaly.Severity == "critical" {
			criticalAnomalies++
		}
	}

	if criticalAnomalies > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Found %d critical performance anomalies. Immediate investigation recommended.", criticalAnomalies))
	}

	// Throughput recommendations
	if summary.PeakOrdersPerSecond > summary.AvgOrdersPerSecond*3 {
		recommendations = append(recommendations, "High throughput variance detected. Consider implementing load balancing and auto-scaling.")
	}

	return recommendations
}

// GetMetricsAggregation returns aggregated metrics for a time period
func (pt *PerformanceTracker) GetMetricsAggregation(period string, startTime, endTime time.Time) (*MetricsAggregation, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	// Filter metrics by time range
	var filteredMetrics []*SystemMetrics
	for _, metrics := range pt.metricsHistory {
		if metrics.Timestamp.After(startTime) && metrics.Timestamp.Before(endTime) {
			filteredMetrics = append(filteredMetrics, metrics)
		}
	}

	if len(filteredMetrics) == 0 {
		return nil, fmt.Errorf("no metrics found in specified time range")
	}

	aggregation := &MetricsAggregation{
		Period:      period,
		StartTime:   startTime,
		EndTime:     endTime,
		Count:       len(filteredMetrics),
		Min:         make(map[string]float64),
		Max:         make(map[string]float64),
		Avg:         make(map[string]float64),
		Sum:         make(map[string]float64),
		Percentiles: make(map[string]map[string]float64),
	}

	// Calculate aggregations for each metric
	metrics := []string{
		"orders_per_second", "trades_per_second", "matching_latency",
		"cpu_usage", "memory_usage", "error_rate", "response_time",
	}

	for _, metric := range metrics {
		values := pt.getFilteredMetricValues(metric, filteredMetrics)
		aggregation.Min[metric] = pt.getMin(values)
		aggregation.Max[metric] = pt.getMax(values)
		aggregation.Avg[metric] = pt.getAverage(values)
		aggregation.Sum[metric] = pt.getSum(values)
		aggregation.Percentiles[metric] = pt.getPercentiles(values)
	}

	return aggregation, nil
}

// getFilteredMetricValues extracts values for a specific metric from filtered data
func (pt *PerformanceTracker) getFilteredMetricValues(metric string, filteredMetrics []*SystemMetrics) []float64 {
	var values []float64

	for _, m := range filteredMetrics {
		var value float64
		switch metric {
		case "orders_per_second":
			value = m.OrdersPerSecond
		case "trades_per_second":
			value = m.TradesPerSecond
		case "matching_latency":
			value = m.MatchingLatency
		case "cpu_usage":
			value = m.CPUUsage
		case "memory_usage":
			value = m.MemoryUsage
		case "error_rate":
			value = m.ErrorRate
		case "response_time":
			value = m.ResponseTime
		}
		values = append(values, value)
	}

	return values
}

// Helper functions for aggregation calculations
func (pt *PerformanceTracker) getMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (pt *PerformanceTracker) getMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (pt *PerformanceTracker) getAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := pt.getSum(values)
	return sum / float64(len(values))
}

func (pt *PerformanceTracker) getSum(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum
}

func (pt *PerformanceTracker) getPercentiles(values []float64) map[string]float64 {
	if len(values) == 0 {
		return map[string]float64{}
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	return map[string]float64{
		"p50": sorted[len(sorted)/2],
		"p90": sorted[int(float64(len(sorted))*0.90)],
		"p95": sorted[int(float64(len(sorted))*0.95)],
		"p99": sorted[int(float64(len(sorted))*0.99)],
	}
}

// LogPerformanceSummary logs a comprehensive performance summary
func (m *UnifiedMonitor) LogPerformanceSummary() {
	metrics, err := m.GetMetrics()
	if err != nil {
		m.logger.Error("Failed to get metrics for summary", zap.Error(err))
		return
	}

	health, err := m.GetHealth()
	if err != nil {
		m.logger.Error("Failed to get health for summary", zap.Error(err))
		return
	}

	alerts, err := m.GetAlerts()
	if err != nil {
		m.logger.Error("Failed to get alerts for summary", zap.Error(err))
		return
	}

	m.logger.Info("System Performance Summary",
		zap.Float64("orders_per_second", metrics.OrdersPerSecond),
		zap.Float64("trades_per_second", metrics.TradesPerSecond),
		zap.Float64("matching_latency_ms", metrics.MatchingLatency),
		zap.Float64("cpu_usage_percent", metrics.CPUUsage),
		zap.Float64("memory_usage_percent", metrics.MemoryUsage),
		zap.Float64("error_rate_percent", metrics.ErrorRate),
		zap.Float64("response_time_ms", metrics.ResponseTime),
		zap.String("overall_health", string(health.Overall)),
		zap.Int("active_alerts", len(alerts)),
		zap.Duration("uptime", m.GetUptime()))
}

// GetComponentStatus returns status of all monitored components
func (m *UnifiedMonitor) GetComponentStatus() (map[string]*ComponentStatus, error) {
	health, err := m.GetHealth()
	if err != nil {
		return nil, err
	}

	components := make(map[string]*ComponentStatus)

	for name, status := range health.Components {
		var message string
		var metadata map[string]interface{}

		if health.Details != nil {
			if detail, exists := health.Details[name]; exists {
				if detailMap, ok := detail.(map[string]interface{}); ok {
					metadata = detailMap
					if msg, exists := detailMap["message"]; exists {
						if msgStr, ok := msg.(string); ok {
							message = msgStr
						}
					}
				}
			}
		}

		components[name] = &ComponentStatus{
			Name:      name,
			Status:    status,
			LastCheck: health.Timestamp,
			Message:   message,
			Metadata:  metadata,
		}
	}

	return components, nil
}
