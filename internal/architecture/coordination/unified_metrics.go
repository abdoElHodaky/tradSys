package coordination

import (
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"go.uber.org/zap"
)

// UnifiedMetricsCollector provides a unified approach to metrics collection
// to reduce overhead and prevent duplication across different systems.
type UnifiedMetricsCollector struct {
	// Underlying metrics collector
	collector *metrics.Collector
	
	// Component metrics
	componentMetrics map[string]*ComponentMetrics
	
	// Mutex for protecting component metrics
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
	
	// Configuration
	config MetricsConfig
	
	// Aggregation
	aggregator *MetricsAggregator
}

// ComponentMetrics contains metrics for a component
type ComponentMetrics struct {
	// Component identity
	Name string
	
	// Initialization metrics
	InitCount     int64
	InitErrors    int64
	TotalInitTime time.Duration
	LastInitTime  time.Duration
	
	// Usage metrics
	AccessCount   int64
	ErrorCount    int64
	
	// Performance metrics
	MemoryUsage   int64
	CPUUsage      float64
	
	// Custom metrics
	CustomMetrics map[string]interface{}
}

// MetricsConfig contains configuration for metrics collection
type MetricsConfig struct {
	// Whether metrics are enabled
	Enabled bool
	
	// Sample rate for metrics (0.0-1.0)
	SampleRate float64
	
	// Aggregation interval
	AggregationInterval time.Duration
	
	// Whether to collect detailed metrics
	DetailedMetrics bool
	
	// Maximum number of custom metrics per component
	MaxCustomMetrics int
}

// DefaultMetricsConfig returns the default metrics configuration
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled:             true,
		SampleRate:          0.1, // 10% sampling
		AggregationInterval: 10 * time.Second,
		DetailedMetrics:     false,
		MaxCustomMetrics:    10,
	}
}

// NewUnifiedMetricsCollector creates a new unified metrics collector
func NewUnifiedMetricsCollector(config MetricsConfig, logger *zap.Logger) *UnifiedMetricsCollector {
	collector := metrics.NewCollector("unified_metrics", metrics.WithSampleRate(config.SampleRate))
	
	m := &UnifiedMetricsCollector{
		collector:        collector,
		componentMetrics: make(map[string]*ComponentMetrics),
		logger:           logger,
		config:           config,
	}
	
	// Create aggregator
	m.aggregator = NewMetricsAggregator(m, config.AggregationInterval, logger)
	
	// Start aggregator if enabled
	if config.Enabled {
		m.aggregator.Start()
	}
	
	return m
}

// RegisterComponent registers a component with the metrics collector
func (m *UnifiedMetricsCollector) RegisterComponent(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.componentMetrics[name]; !exists {
		m.componentMetrics[name] = &ComponentMetrics{
			Name:          name,
			CustomMetrics: make(map[string]interface{}),
		}
	}
}

// RecordInitialization records a component initialization
func (m *UnifiedMetricsCollector) RecordInitialization(component string, duration time.Duration) {
	if !m.config.Enabled {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metrics, exists := m.componentMetrics[component]
	if !exists {
		metrics = &ComponentMetrics{
			Name:          component,
			CustomMetrics: make(map[string]interface{}),
		}
		m.componentMetrics[component] = metrics
	}
	
	metrics.InitCount++
	metrics.TotalInitTime += duration
	metrics.LastInitTime = duration
	
	// Record in underlying collector
	m.collector.RecordLatency("component_init", duration, map[string]string{
		"component": component,
	})
}

// RecordInitializationError records a component initialization error
func (m *UnifiedMetricsCollector) RecordInitializationError(component string, duration time.Duration) {
	if !m.config.Enabled {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metrics, exists := m.componentMetrics[component]
	if !exists {
		metrics = &ComponentMetrics{
			Name:          component,
			CustomMetrics: make(map[string]interface{}),
		}
		m.componentMetrics[component] = metrics
	}
	
	metrics.InitErrors++
	
	// Record in underlying collector
	m.collector.RecordError("component_init_error", map[string]string{
		"component": component,
	})
}

// RecordAccess records a component access
func (m *UnifiedMetricsCollector) RecordAccess(component string) {
	if !m.config.Enabled {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metrics, exists := m.componentMetrics[component]
	if !exists {
		metrics = &ComponentMetrics{
			Name:          component,
			CustomMetrics: make(map[string]interface{}),
		}
		m.componentMetrics[component] = metrics
	}
	
	metrics.AccessCount++
	
	// Record in underlying collector
	m.collector.Increment("component_access", map[string]string{
		"component": component,
	})
}

// RecordError records a component error
func (m *UnifiedMetricsCollector) RecordError(component string, errorType string) {
	if !m.config.Enabled {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metrics, exists := m.componentMetrics[component]
	if !exists {
		metrics = &ComponentMetrics{
			Name:          component,
			CustomMetrics: make(map[string]interface{}),
		}
		m.componentMetrics[component] = metrics
	}
	
	metrics.ErrorCount++
	
	// Record in underlying collector
	m.collector.RecordError("component_error", map[string]string{
		"component":  component,
		"error_type": errorType,
	})
}

// RecordMemoryUsage records component memory usage
func (m *UnifiedMetricsCollector) RecordMemoryUsage(component string, bytes int64) {
	if !m.config.Enabled {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metrics, exists := m.componentMetrics[component]
	if !exists {
		metrics = &ComponentMetrics{
			Name:          component,
			CustomMetrics: make(map[string]interface{}),
		}
		m.componentMetrics[component] = metrics
	}
	
	metrics.MemoryUsage = bytes
	
	// Record in underlying collector if detailed metrics are enabled
	if m.config.DetailedMetrics {
		m.collector.RecordValue("component_memory", float64(bytes), map[string]string{
			"component": component,
		})
	}
}

// RecordCPUUsage records component CPU usage
func (m *UnifiedMetricsCollector) RecordCPUUsage(component string, percent float64) {
	if !m.config.Enabled {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metrics, exists := m.componentMetrics[component]
	if !exists {
		metrics = &ComponentMetrics{
			Name:          component,
			CustomMetrics: make(map[string]interface{}),
		}
		m.componentMetrics[component] = metrics
	}
	
	metrics.CPUUsage = percent
	
	// Record in underlying collector if detailed metrics are enabled
	if m.config.DetailedMetrics {
		m.collector.RecordValue("component_cpu", percent, map[string]string{
			"component": component,
		})
	}
}

// RecordCustomMetric records a custom metric for a component
func (m *UnifiedMetricsCollector) RecordCustomMetric(component string, name string, value interface{}) {
	if !m.config.Enabled {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	metrics, exists := m.componentMetrics[component]
	if !exists {
		metrics = &ComponentMetrics{
			Name:          component,
			CustomMetrics: make(map[string]interface{}),
		}
		m.componentMetrics[component] = metrics
	}
	
	// Check if we've reached the maximum number of custom metrics
	if len(metrics.CustomMetrics) >= m.config.MaxCustomMetrics && metrics.CustomMetrics[name] == nil {
		m.logger.Warn("Maximum number of custom metrics reached for component",
			zap.String("component", component),
			zap.String("metric", name),
		)
		return
	}
	
	metrics.CustomMetrics[name] = value
	
	// Record in underlying collector if it's a numeric value
	if m.config.DetailedMetrics {
		switch v := value.(type) {
		case int:
			m.collector.RecordValue("component_custom", float64(v), map[string]string{
				"component": component,
				"metric":    name,
			})
		case int64:
			m.collector.RecordValue("component_custom", float64(v), map[string]string{
				"component": component,
				"metric":    name,
			})
		case float32:
			m.collector.RecordValue("component_custom", float64(v), map[string]string{
				"component": component,
				"metric":    name,
			})
		case float64:
			m.collector.RecordValue("component_custom", v, map[string]string{
				"component": component,
				"metric":    name,
			})
		}
	}
}

// GetComponentMetrics gets metrics for a component
func (m *UnifiedMetricsCollector) GetComponentMetrics(component string) (*ComponentMetrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metrics, exists := m.componentMetrics[component]
	return metrics, exists
}

// GetAllComponentMetrics gets metrics for all components
func (m *UnifiedMetricsCollector) GetAllComponentMetrics() map[string]*ComponentMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy to avoid concurrent access issues
	result := make(map[string]*ComponentMetrics, len(m.componentMetrics))
	
	for name, metrics := range m.componentMetrics {
		// Create a deep copy of the metrics
		metricsCopy := &ComponentMetrics{
			Name:          metrics.Name,
			InitCount:     metrics.InitCount,
			InitErrors:    metrics.InitErrors,
			TotalInitTime: metrics.TotalInitTime,
			LastInitTime:  metrics.LastInitTime,
			AccessCount:   metrics.AccessCount,
			ErrorCount:    metrics.ErrorCount,
			MemoryUsage:   metrics.MemoryUsage,
			CPUUsage:      metrics.CPUUsage,
			CustomMetrics: make(map[string]interface{}),
		}
		
		// Copy custom metrics
		for k, v := range metrics.CustomMetrics {
			metricsCopy.CustomMetrics[k] = v
		}
		
		result[name] = metricsCopy
	}
	
	return result
}

// Shutdown shuts down the metrics collector
func (m *UnifiedMetricsCollector) Shutdown() {
	if m.config.Enabled {
		m.aggregator.Stop()
	}
}

// MetricsAggregator aggregates metrics at regular intervals
type MetricsAggregator struct {
	// Metrics collector
	collector *UnifiedMetricsCollector
	
	// Aggregation interval
	interval time.Duration
	
	// Logger
	logger *zap.Logger
	
	// Stop channel
	stopCh chan struct{}
	
	// Whether the aggregator is running
	running bool
	
	// Mutex for protecting running state
	mu sync.Mutex
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(collector *UnifiedMetricsCollector, interval time.Duration, logger *zap.Logger) *MetricsAggregator {
	return &MetricsAggregator{
		collector: collector,
		interval:  interval,
		logger:    logger,
		stopCh:    make(chan struct{}),
		running:   false,
	}
}

// Start starts the metrics aggregator
func (a *MetricsAggregator) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.running {
		return
	}
	
	a.running = true
	a.stopCh = make(chan struct{})
	
	go a.run()
}

// Stop stops the metrics aggregator
func (a *MetricsAggregator) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if !a.running {
		return
	}
	
	close(a.stopCh)
	a.running = false
}

// run runs the metrics aggregator
func (a *MetricsAggregator) run() {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			a.aggregate()
		case <-a.stopCh:
			return
		}
	}
}

// aggregate aggregates metrics
func (a *MetricsAggregator) aggregate() {
	metrics := a.collector.GetAllComponentMetrics()
	
	// Calculate system-wide metrics
	var totalMemory int64
	var totalInitCount int64
	var totalAccessCount int64
	var totalErrorCount int64
	
	for _, m := range metrics {
		totalMemory += m.MemoryUsage
		totalInitCount += m.InitCount
		totalAccessCount += m.AccessCount
		totalErrorCount += m.ErrorCount
	}
	
	// Record system-wide metrics
	a.collector.collector.RecordValue("system_memory_usage", float64(totalMemory), nil)
	a.collector.collector.RecordValue("system_init_count", float64(totalInitCount), nil)
	a.collector.collector.RecordValue("system_access_count", float64(totalAccessCount), nil)
	a.collector.collector.RecordValue("system_error_count", float64(totalErrorCount), nil)
	
	a.logger.Debug("Metrics aggregated",
		zap.Int64("total_memory", totalMemory),
		zap.Int64("total_init_count", totalInitCount),
		zap.Int64("total_access_count", totalAccessCount),
		zap.Int64("total_error_count", totalErrorCount),
	)
}

