package performance

import (
	"sync"
	"time"
)

// MarketDataPool provides a pool for market data objects
type MarketDataPool struct {
	pool sync.Pool
}

// NewMarketDataPool creates a new market data pool
func NewMarketDataPool() *MarketDataPool {
	return &MarketDataPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &MarketData{}
			},
		},
	}
}

// Get retrieves a market data object from the pool
func (p *MarketDataPool) Get() *MarketData {
	return p.pool.Get().(*MarketData)
}

// Put returns a market data object to the pool
func (p *MarketDataPool) Put(data *MarketData) {
	data.Reset()
	p.pool.Put(data)
}

// MarketData represents market data information
type MarketData struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp time.Time
	BidPrice  float64
	AskPrice  float64
	BidSize   float64
	AskSize   float64
}

// Reset resets the market data fields
func (m *MarketData) Reset() {
	m.Symbol = ""
	m.Price = 0
	m.Volume = 0
	m.Timestamp = time.Time{}
	m.BidPrice = 0
	m.AskPrice = 0
	m.BidSize = 0
	m.AskSize = 0
}

// PerformanceMetrics provides performance tracking
type PerformanceMetrics struct {
	mu                sync.RWMutex
	latencySum        int64
	latencyCount      int64
	throughputCounter int64
	startTime         time.Time
}

// NewPerformanceMetrics creates a new performance metrics tracker
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		startTime: time.Now(),
	}
}

// RecordLatency records a latency measurement
func (p *PerformanceMetrics) RecordLatency(latency time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.latencySum += latency.Nanoseconds()
	p.latencyCount++
}

// RecordThroughput increments the throughput counter
func (p *PerformanceMetrics) RecordThroughput() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.throughputCounter++
}

// GetAverageLatency returns the average latency
func (p *PerformanceMetrics) GetAverageLatency() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.latencyCount == 0 {
		return 0
	}
	return time.Duration(p.latencySum / p.latencyCount)
}

// GetThroughput returns the current throughput (operations per second)
func (p *PerformanceMetrics) GetThroughput() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	elapsed := time.Since(p.startTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(p.throughputCounter) / elapsed
}
