package latency

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// Critical latency thresholds in nanoseconds
const (
	StrategyLatencyThresholdNs  = 1000000  // 1ms
	OrderLatencyThresholdNs     = 500000   // 500μs
	MarketDataLatencyThresholdNs = 100000  // 100μs
)

// Histogram represents a histogram of recorded values
type Histogram interface {
	// Update adds a value to the histogram
	Update(int64)
	// Snapshot returns a read-only copy of the histogram
	Snapshot() HistogramSnapshot
}

// HistogramSnapshot represents a read-only snapshot of a histogram
type HistogramSnapshot interface {
	// Min returns the minimum value in the histogram
	Min() int64
	// Max returns the maximum value in the histogram
	Max() int64
	// Mean returns the mean of the values in the histogram
	Mean() float64
	// Percentile returns the value at the given percentile
	Percentile(float64) float64
}

// SimpleHistogram is a simple histogram implementation
type SimpleHistogram struct {
	values []int64
	mu     sync.RWMutex
}

// NewSimpleHistogram creates a new simple histogram
func NewSimpleHistogram() *SimpleHistogram {
	return &SimpleHistogram{
		values: make([]int64, 0, 1000),
	}
}

// Update adds a value to the histogram
func (h *SimpleHistogram) Update(value int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = append(h.values, value)
	// Keep only the last 1000 values
	if len(h.values) > 1000 {
		h.values = h.values[len(h.values)-1000:]
	}
}

// Snapshot returns a read-only copy of the histogram
func (h *SimpleHistogram) Snapshot() HistogramSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	values := make([]int64, len(h.values))
	copy(values, h.values)
	return &SimpleHistogramSnapshot{values: values}
}

// SimpleHistogramSnapshot is a read-only snapshot of a simple histogram
type SimpleHistogramSnapshot struct {
	values []int64
}

// Min returns the minimum value in the histogram
func (s *SimpleHistogramSnapshot) Min() int64 {
	if len(s.values) == 0 {
		return 0
	}
	min := s.values[0]
	for _, v := range s.values {
		if v < min {
			min = v
		}
	}
	return min
}

// Max returns the maximum value in the histogram
func (s *SimpleHistogramSnapshot) Max() int64 {
	if len(s.values) == 0 {
		return 0
	}
	max := s.values[0]
	for _, v := range s.values {
		if v > max {
			max = v
		}
	}
	return max
}

// Mean returns the mean of the values in the histogram
func (s *SimpleHistogramSnapshot) Mean() float64 {
	if len(s.values) == 0 {
		return 0
	}
	var sum int64
	for _, v := range s.values {
		sum += v
	}
	return float64(sum) / float64(len(s.values))
}

// Percentile returns the value at the given percentile
func (s *SimpleHistogramSnapshot) Percentile(p float64) float64 {
	if len(s.values) == 0 {
		return 0
	}
	// Sort the values
	values := make([]int64, len(s.values))
	copy(values, s.values)
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
	// Calculate the index
	index := int(float64(len(values)) * p)
	if index >= len(values) {
		index = len(values) - 1
	}
	return float64(values[index])
}

// Tracker provides high-precision latency tracking for HFT operations
type Tracker struct {
	strategyLatencies   map[string]Histogram
	orderLatencies      Histogram
	marketDataLatencies Histogram
	mu                  sync.RWMutex
	logger              *zap.Logger
}

// NewTracker creates a new latency tracker
func NewTracker(logger *zap.Logger) *Tracker {
	return &Tracker{
		strategyLatencies:   make(map[string]Histogram),
		orderLatencies:      NewSimpleHistogram(),
		marketDataLatencies: NewSimpleHistogram(),
		logger:              logger,
	}
}

// TrackStrategyExecution tracks the execution time of a strategy
func (t *Tracker) TrackStrategyExecution(strategyName string, start time.Time) {
	t.mu.RLock()
	histogram, exists := t.strategyLatencies[strategyName]
	t.mu.RUnlock()
	
	if !exists {
		t.mu.Lock()
		histogram = NewSimpleHistogram()
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
func (t *Tracker) TrackOrderProcessing(orderID string, start time.Time) {
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
func (t *Tracker) TrackMarketDataProcessing(symbol string, start time.Time) {
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
func (t *Tracker) GetStrategyLatencyStats(strategyName string) (min, max, mean, p95, p99 int64, err error) {
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
func (t *Tracker) GetOrderLatencyStats() (min, max, mean, p95, p99 int64) {
	snapshot := t.orderLatencies.Snapshot()
	return snapshot.Min(), snapshot.Max(), int64(snapshot.Mean()), 
		int64(snapshot.Percentile(0.95)), int64(snapshot.Percentile(0.99))
}

// GetMarketDataLatencyStats returns latency statistics for market data processing
func (t *Tracker) GetMarketDataLatencyStats() (min, max, mean, p95, p99 int64) {
	snapshot := t.marketDataLatencies.Snapshot()
	return snapshot.Min(), snapshot.Max(), int64(snapshot.Mean()), 
		int64(snapshot.Percentile(0.95)), int64(snapshot.Percentile(0.99))
}

// Errors
var (
	ErrStrategyNotFound = &Error{"strategy not found"}
)

// Error represents a latency tracking error
type Error struct {
	message string
}

// Error returns the error message
func (e *Error) Error() string {
	return e.message
}
