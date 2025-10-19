package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BaselineMetrics provides performance monitoring for HFT operations
type BaselineMetrics struct {
	// Latency metrics (microseconds)
	OrderLatency    prometheus.Histogram
	WSLatency       prometheus.Histogram
	DBLatency       prometheus.Histogram
	
	// Throughput metrics
	OrdersPerSecond   prometheus.Gauge
	MessagesPerSecond prometheus.Gauge
	
	// Resource metrics
	MemoryAllocations prometheus.Counter
	GCPauses         prometheus.Histogram
	ActiveConnections prometheus.Gauge
	
	// Error metrics
	ErrorRate   prometheus.Counter
	TimeoutRate prometheus.Counter
	
	// Internal tracking
	mu            sync.RWMutex
	orderCount    int64
	messageCount  int64
	lastReset     time.Time
}

// NewBaselineMetrics creates a new baseline metrics collector
func NewBaselineMetrics() *BaselineMetrics {
	return &BaselineMetrics{
		OrderLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "hft_order_latency_microseconds",
			Help: "Order processing latency in microseconds",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		}),
		WSLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "hft_websocket_latency_microseconds",
			Help: "WebSocket message latency in microseconds",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		}),
		DBLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "hft_database_latency_microseconds",
			Help: "Database query latency in microseconds",
			Buckets: []float64{100, 250, 500, 1000, 2500, 5000, 10000, 25000, 50000},
		}),
		OrdersPerSecond: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "hft_orders_per_second",
			Help: "Current orders processed per second",
		}),
		MessagesPerSecond: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "hft_messages_per_second",
			Help: "Current WebSocket messages per second",
		}),
		MemoryAllocations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "hft_memory_allocations_total",
			Help: "Total memory allocations by component",
		}),
		GCPauses: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "hft_gc_pause_microseconds",
			Help: "Garbage collection pause time in microseconds",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000, 50000},
		}),
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "hft_active_connections",
			Help: "Number of active WebSocket connections",
		}),
		ErrorRate: promauto.NewCounter(prometheus.CounterOpts{
			Name: "hft_errors_total",
			Help: "Total number of errors by type",
		}),
		TimeoutRate: promauto.NewCounter(prometheus.CounterOpts{
			Name: "hft_timeouts_total",
			Help: "Total number of timeouts by operation",
		}),
		lastReset: time.Now(),
	}
}

// RecordOrderLatency records order processing latency
func (m *BaselineMetrics) RecordOrderLatency(duration time.Duration) {
	microseconds := float64(duration.Nanoseconds()) / 1000.0
	m.OrderLatency.Observe(microseconds)
	
	m.mu.Lock()
	m.orderCount++
	m.mu.Unlock()
}

// RecordWSLatency records WebSocket message latency
func (m *BaselineMetrics) RecordWSLatency(duration time.Duration) {
	microseconds := float64(duration.Nanoseconds()) / 1000.0
	m.WSLatency.Observe(microseconds)
	
	m.mu.Lock()
	m.messageCount++
	m.mu.Unlock()
}

// RecordDBLatency records database query latency
func (m *BaselineMetrics) RecordDBLatency(duration time.Duration) {
	microseconds := float64(duration.Nanoseconds()) / 1000.0
	m.DBLatency.Observe(microseconds)
}

// RecordError records an error occurrence
func (m *BaselineMetrics) RecordError() {
	m.ErrorRate.Inc()
}

// RecordTimeout records a timeout occurrence
func (m *BaselineMetrics) RecordTimeout() {
	m.TimeoutRate.Inc()
}

// RecordMemoryAllocation records a memory allocation
func (m *BaselineMetrics) RecordMemoryAllocation() {
	m.MemoryAllocations.Inc()
}

// RecordGCPause records a garbage collection pause
func (m *BaselineMetrics) RecordGCPause(duration time.Duration) {
	microseconds := float64(duration.Nanoseconds()) / 1000.0
	m.GCPauses.Observe(microseconds)
}

// UpdateActiveConnections updates the active connection count
func (m *BaselineMetrics) UpdateActiveConnections(count int) {
	m.ActiveConnections.Set(float64(count))
}

// RecordHTTPRequest records HTTP request metrics
func (m *BaselineMetrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	// For now, just record the latency as order latency
	// This can be expanded to have dedicated HTTP metrics
	m.RecordOrderLatency(duration)
	
	// Record errors for non-2xx status codes
	if statusCode >= 400 {
		m.RecordError()
	}
}

// UpdateThroughputMetrics updates throughput metrics
func (m *BaselineMetrics) UpdateThroughputMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(m.lastReset).Seconds()
	
	if elapsed >= 1.0 { // Update every second
		ordersPerSec := float64(m.orderCount) / elapsed
		messagesPerSec := float64(m.messageCount) / elapsed
		
		m.OrdersPerSecond.Set(ordersPerSec)
		m.MessagesPerSecond.Set(messagesPerSec)
		
		// Reset counters
		m.orderCount = 0
		m.messageCount = 0
		m.lastReset = now
	}
}

// LatencyTracker provides high-precision latency tracking
type LatencyTracker struct {
	startTime time.Time
	metrics   *BaselineMetrics
	operation string
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker(metrics *BaselineMetrics, operation string) *LatencyTracker {
	return &LatencyTracker{
		startTime: time.Now(),
		metrics:   metrics,
		operation: operation,
	}
}

// Finish records the latency measurement
func (lt *LatencyTracker) Finish() {
	duration := time.Since(lt.startTime)
	
	switch lt.operation {
	case "order":
		lt.metrics.RecordOrderLatency(duration)
	case "websocket":
		lt.metrics.RecordWSLatency(duration)
	case "database":
		lt.metrics.RecordDBLatency(duration)
	}
}

// Global metrics instance
var GlobalMetrics *BaselineMetrics

// InitMetrics initializes the global metrics instance
func InitMetrics() {
	GlobalMetrics = NewBaselineMetrics()
	
	// Start throughput update goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			GlobalMetrics.UpdateThroughputMetrics()
		}
	}()
}

// TrackOrderLatency is a convenience function for tracking order latency
func TrackOrderLatency() *LatencyTracker {
	if GlobalMetrics == nil {
		InitMetrics()
	}
	return NewLatencyTracker(GlobalMetrics, "order")
}

// TrackWSLatency is a convenience function for tracking WebSocket latency
func TrackWSLatency() *LatencyTracker {
	if GlobalMetrics == nil {
		InitMetrics()
	}
	return NewLatencyTracker(GlobalMetrics, "websocket")
}

// TrackDBLatency is a convenience function for tracking database latency
func TrackDBLatency() *LatencyTracker {
	if GlobalMetrics == nil {
		InitMetrics()
	}
	return NewLatencyTracker(GlobalMetrics, "database")
}
