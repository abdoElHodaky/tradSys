package metrics

import (
	"sync"
	"time"
)

// Metrics represents HFT performance metrics
type Metrics struct {
	mu                sync.RWMutex
	OrderLatency      time.Duration
	ProcessingLatency time.Duration
	ThroughputRPS     float64
	ErrorRate         float64
	MemoryUsage       uint64
	CPUUsage          float64
	NetworkLatency    time.Duration
	OrdersProcessed   uint64
	OrdersRejected    uint64
	LastUpdate        time.Time
}

// Manager manages HFT metrics collection
type Manager struct {
	mu      sync.RWMutex
	metrics *Metrics
	started bool
}

// NewManager creates a new metrics manager
func NewManager() *Manager {
	return &Manager{
		metrics: &Metrics{
			LastUpdate: time.Now(),
		},
	}
}

// Start starts metrics collection
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
}

// Stop stops metrics collection
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
}

// RecordOrderLatency records order processing latency
func (m *Manager) RecordOrderLatency(latency time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.OrderLatency = latency
	m.metrics.LastUpdate = time.Now()
}

// RecordProcessingLatency records general processing latency
func (m *Manager) RecordProcessingLatency(latency time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.ProcessingLatency = latency
	m.metrics.LastUpdate = time.Now()
}

// RecordThroughput records throughput in requests per second
func (m *Manager) RecordThroughput(rps float64) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.ThroughputRPS = rps
	m.metrics.LastUpdate = time.Now()
}

// RecordError records an error occurrence
func (m *Manager) RecordError() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.OrdersRejected++
	m.metrics.LastUpdate = time.Now()
}

// RecordOrderProcessed records a successfully processed order
func (m *Manager) RecordOrderProcessed() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.OrdersProcessed++
	m.metrics.LastUpdate = time.Now()
}

// GetMetrics returns current metrics snapshot
func (m *Manager) GetMetrics() *Metrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	return &Metrics{
		OrderLatency:      m.metrics.OrderLatency,
		ProcessingLatency: m.metrics.ProcessingLatency,
		ThroughputRPS:     m.metrics.ThroughputRPS,
		ErrorRate:         m.metrics.ErrorRate,
		MemoryUsage:       m.metrics.MemoryUsage,
		CPUUsage:          m.metrics.CPUUsage,
		NetworkLatency:    m.metrics.NetworkLatency,
		OrdersProcessed:   m.metrics.OrdersProcessed,
		OrdersRejected:    m.metrics.OrdersRejected,
		LastUpdate:        m.metrics.LastUpdate,
	}
}

// CalculateErrorRate calculates the current error rate
func (m *Manager) CalculateErrorRate() float64 {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()
	
	total := m.metrics.OrdersProcessed + m.metrics.OrdersRejected
	if total == 0 {
		return 0.0
	}
	
	return float64(m.metrics.OrdersRejected) / float64(total) * 100.0
}

// Reset resets all metrics
func (m *Manager) Reset() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	
	m.metrics.OrderLatency = 0
	m.metrics.ProcessingLatency = 0
	m.metrics.ThroughputRPS = 0
	m.metrics.ErrorRate = 0
	m.metrics.MemoryUsage = 0
	m.metrics.CPUUsage = 0
	m.metrics.NetworkLatency = 0
	m.metrics.OrdersProcessed = 0
	m.metrics.OrdersRejected = 0
	m.metrics.LastUpdate = time.Now()
}

// Global metrics manager instance
var globalManager = NewManager()

// GetGlobalManager returns the global metrics manager
func GetGlobalManager() *Manager {
	return globalManager
}

// RecordLatency records latency using the global manager
func RecordLatency(latency time.Duration) {
	globalManager.RecordOrderLatency(latency)
}

// RecordThroughput records throughput using the global manager
func RecordThroughput(rps float64) {
	globalManager.RecordThroughput(rps)
}

// RecordError records an error using the global manager
func RecordError() {
	globalManager.RecordError()
}

// RecordSuccess records a successful operation using the global manager
func RecordSuccess() {
	globalManager.RecordOrderProcessed()
}
