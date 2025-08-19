package latency

import (
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

// Critical latency thresholds in nanoseconds
const (
	StrategyLatencyThresholdNs  = 1000000  // 1ms
	OrderLatencyThresholdNs     = 500000   // 500μs
	MarketDataLatencyThresholdNs = 100000  // 100μs
)

// LatencyTracker provides high-precision latency tracking for HFT operations
type LatencyTracker struct {
	strategyLatencies   map[string]metrics.Histogram
	orderLatencies      metrics.Histogram
	marketDataLatencies metrics.Histogram
	mu                  sync.RWMutex
	logger              *zap.Logger
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker(logger *zap.Logger) *LatencyTracker {
	return &LatencyTracker{
		strategyLatencies:   make(map[string]metrics.Histogram),
		orderLatencies:      metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)),
		marketDataLatencies: metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015)),
		logger:              logger,
	}
}

// TrackStrategyExecution tracks the execution time of a strategy
func (t *LatencyTracker) TrackStrategyExecution(strategyName string, start time.Time) {
	t.mu.RLock()
	histogram, exists := t.strategyLatencies[strategyName]
	t.mu.RUnlock()
	
	if !exists {
		t.mu.Lock()
		histogram = metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))
		t.strategyLatencies[strategyName] = histogram
		t.mu.Unlock()
	}
	
	latencyNs := time.Since(start).Nanoseconds()
	histogram.Update(latencyNs)
	
	// Alert on excessive latency
	if latencyNs > StrategyLatencyThresholdNs {
		t.logger.Warn("Strategy execution exceeded critical latency threshold",
			zap.String("strategy", strategyName),
			zap.Int64("latency_ns", latencyNs),
			zap.Int64("threshold_ns", StrategyLatencyThresholdNs))
	}
}

// TrackOrderProcessing tracks the processing time of an order
func (t *LatencyTracker) TrackOrderProcessing(orderID string, start time.Time) {
	latencyNs := time.Since(start).Nanoseconds()
	t.orderLatencies.Update(latencyNs)
	
	// Alert on excessive latency
	if latencyNs > OrderLatencyThresholdNs {
		t.logger.Warn("Order processing exceeded critical latency threshold",
			zap.String("order_id", orderID),
			zap.Int64("latency_ns", latencyNs),
			zap.Int64("threshold_ns", OrderLatencyThresholdNs))
	}
}

// TrackMarketDataProcessing tracks the processing time of market data
func (t *LatencyTracker) TrackMarketDataProcessing(symbol string, start time.Time) {
	latencyNs := time.Since(start).Nanoseconds()
	t.marketDataLatencies.Update(latencyNs)
	
	// Alert on excessive latency
	if latencyNs > MarketDataLatencyThresholdNs {
		t.logger.Warn("Market data processing exceeded critical latency threshold",
			zap.String("symbol", symbol),
			zap.Int64("latency_ns", latencyNs),
			zap.Int64("threshold_ns", MarketDataLatencyThresholdNs))
	}
}

// GetStrategyLatencyStats returns latency statistics for a strategy
func (t *LatencyTracker) GetStrategyLatencyStats(strategyName string) (min, max, mean, p95, p99 int64, err error) {
	t.mu.RLock()
	histogram, exists := t.strategyLatencies[strategyName]
	t.mu.RUnlock()
	
	if !exists {
		return 0, 0, 0, 0, 0, ErrStrategyNotFound
	}
	
	snapshot := histogram.Snapshot()
	return snapshot.Min(), snapshot.Max(), int64(snapshot.Mean()), 
		int64(snapshot.Percentile(0.95)), int64(snapshot.Percentile(0.99)), nil
}

// GetOrderLatencyStats returns latency statistics for order processing
func (t *LatencyTracker) GetOrderLatencyStats() (min, max, mean, p95, p99 int64) {
	snapshot := t.orderLatencies.Snapshot()
	return snapshot.Min(), snapshot.Max(), int64(snapshot.Mean()), 
		int64(snapshot.Percentile(0.95)), int64(snapshot.Percentile(0.99))
}

// GetMarketDataLatencyStats returns latency statistics for market data processing
func (t *LatencyTracker) GetMarketDataLatencyStats() (min, max, mean, p95, p99 int64) {
	snapshot := t.marketDataLatencies.Snapshot()
	return snapshot.Min(), snapshot.Max(), int64(snapshot.Mean()), 
		int64(snapshot.Percentile(0.95)), int64(snapshot.Percentile(0.99))
}

// Errors
var (
	ErrStrategyNotFound = &LatencyError{"strategy not found"}
)

// LatencyError represents a latency tracking error
type LatencyError struct {
	message string
}

// Error returns the error message
func (e *LatencyError) Error() string {
	return e.message
}

