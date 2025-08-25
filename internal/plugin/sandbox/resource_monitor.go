package sandbox

import (
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ResourceSnapshot represents a point-in-time snapshot of resource usage
type ResourceSnapshot struct {
	Timestamp      time.Time
	MemoryUsageBytes int64
	CPUUsagePercent float64
	GoroutineCount int
}

// ResourceThresholds defines thresholds for resource usage alerts
type ResourceThresholds struct {
	MemoryThresholdBytes int64
	CPUThresholdPercent  float64
	GoroutineThreshold   int
}

// AlertHandler is a function that handles resource usage alerts
type AlertHandler func(pluginID string, snapshot ResourceSnapshot, thresholds ResourceThresholds)

// ResourceMonitor tracks resource usage of plugins
type ResourceMonitor struct {
	// Current usage
	CurrentMemoryBytes int64
	CurrentCPUPercent  float64
	CurrentGoroutines  int
	
	// Historical data
	UsageHistory       []ResourceSnapshot
	MaxHistorySize     int
	
	// Alerts
	AlertThresholds    ResourceThresholds
	AlertHandlers      []AlertHandler
	
	// Monitoring state
	monitoringInterval time.Duration
	monitoredPlugins   map[string]chan struct{}
	logger             *zap.Logger
	mu                 sync.RWMutex
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(logger *zap.Logger) *ResourceMonitor {
	return &ResourceMonitor{
		CurrentMemoryBytes: 0,
		CurrentCPUPercent:  0,
		CurrentGoroutines:  0,
		UsageHistory:       []ResourceSnapshot{},
		MaxHistorySize:     100,
		AlertThresholds: ResourceThresholds{
			MemoryThresholdBytes: 500 * 1024 * 1024, // 500MB
			CPUThresholdPercent:  80.0,              // 80%
			GoroutineThreshold:   1000,              // 1000 goroutines
		},
		AlertHandlers:      []AlertHandler{},
		monitoringInterval: 5 * time.Second,
		monitoredPlugins:   make(map[string]chan struct{}),
		logger:             logger,
	}
}

// WithMonitoringInterval sets the monitoring interval
func (m *ResourceMonitor) WithMonitoringInterval(interval time.Duration) *ResourceMonitor {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.monitoringInterval = interval
	return m
}

// WithAlertThresholds sets the alert thresholds
func (m *ResourceMonitor) WithAlertThresholds(thresholds ResourceThresholds) *ResourceMonitor {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.AlertThresholds = thresholds
	return m
}

// AddAlertHandler adds an alert handler
func (m *ResourceMonitor) AddAlertHandler(handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.AlertHandlers = append(m.AlertHandlers, handler)
}

// StartMonitoring begins resource monitoring for a plugin
func (m *ResourceMonitor) StartMonitoring(pluginID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if already monitoring
	if _, exists := m.monitoredPlugins[pluginID]; exists {
		return
	}
	
	// Create stop channel
	stopCh := make(chan struct{})
	m.monitoredPlugins[pluginID] = stopCh
	
	// Start monitoring goroutine
	go func() {
		ticker := time.NewTicker(m.monitoringInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				m.collectMetrics(pluginID)
				m.checkThresholds(pluginID)
			case <-stopCh:
				return
			}
		}
	}()
	
	m.logger.Debug("Started resource monitoring",
		zap.String("plugin_id", pluginID),
		zap.Duration("interval", m.monitoringInterval))
}

// StopMonitoring stops resource monitoring for a plugin
func (m *ResourceMonitor) StopMonitoring(pluginID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if monitoring
	stopCh, exists := m.monitoredPlugins[pluginID]
	if !exists {
		return
	}
	
	// Signal stop
	close(stopCh)
	delete(m.monitoredPlugins, pluginID)
	
	m.logger.Debug("Stopped resource monitoring",
		zap.String("plugin_id", pluginID))
}

// collectMetrics collects resource usage metrics for a plugin
func (m *ResourceMonitor) collectMetrics(pluginID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Collect memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// In a real implementation, we would collect plugin-specific metrics
	// For now, we'll just use system-wide metrics as a placeholder
	m.CurrentMemoryBytes = int64(memStats.Alloc)
	m.CurrentCPUPercent = 0.0 // Placeholder
	m.CurrentGoroutines = runtime.NumGoroutine()
	
	// Create snapshot
	snapshot := ResourceSnapshot{
		Timestamp:        time.Now(),
		MemoryUsageBytes: m.CurrentMemoryBytes,
		CPUUsagePercent:  m.CurrentCPUPercent,
		GoroutineCount:   m.CurrentGoroutines,
	}
	
	// Add to history
	m.UsageHistory = append(m.UsageHistory, snapshot)
	
	// Trim history if needed
	if len(m.UsageHistory) > m.MaxHistorySize {
		m.UsageHistory = m.UsageHistory[len(m.UsageHistory)-m.MaxHistorySize:]
	}
	
	m.logger.Debug("Collected resource metrics",
		zap.String("plugin_id", pluginID),
		zap.Int64("memory_bytes", snapshot.MemoryUsageBytes),
		zap.Float64("cpu_percent", snapshot.CPUUsagePercent),
		zap.Int("goroutines", snapshot.GoroutineCount))
}

// checkThresholds checks if resource usage exceeds thresholds
func (m *ResourceMonitor) checkThresholds(pluginID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Skip if no history
	if len(m.UsageHistory) == 0 {
		return
	}
	
	// Get latest snapshot
	snapshot := m.UsageHistory[len(m.UsageHistory)-1]
	
	// Check thresholds
	memoryExceeded := snapshot.MemoryUsageBytes > m.AlertThresholds.MemoryThresholdBytes
	cpuExceeded := snapshot.CPUUsagePercent > m.AlertThresholds.CPUThresholdPercent
	goroutinesExceeded := snapshot.GoroutineCount > m.AlertThresholds.GoroutineThreshold
	
	// Trigger alerts if needed
	if memoryExceeded || cpuExceeded || goroutinesExceeded {
		m.logger.Warn("Resource threshold exceeded",
			zap.String("plugin_id", pluginID),
			zap.Bool("memory_exceeded", memoryExceeded),
			zap.Bool("cpu_exceeded", cpuExceeded),
			zap.Bool("goroutines_exceeded", goroutinesExceeded))
		
		// Call alert handlers
		for _, handler := range m.AlertHandlers {
			go handler(pluginID, snapshot, m.AlertThresholds)
		}
	}
}

// GetResourceHistory returns the resource usage history
func (m *ResourceMonitor) GetResourceHistory() []ResourceSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	history := make([]ResourceSnapshot, len(m.UsageHistory))
	copy(history, m.UsageHistory)
	
	return history
}

// GetCurrentUsage returns the current resource usage
func (m *ResourceMonitor) GetCurrentUsage() ResourceSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return ResourceSnapshot{
		Timestamp:        time.Now(),
		MemoryUsageBytes: m.CurrentMemoryBytes,
		CPUUsagePercent:  m.CurrentCPUPercent,
		GoroutineCount:   m.CurrentGoroutines,
	}
}
