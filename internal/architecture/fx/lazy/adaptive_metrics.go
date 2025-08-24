package lazy

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// AdaptiveMetrics provides enhanced metrics collection for lazy loading
// with adaptive sampling to reduce overhead.
type AdaptiveMetrics struct {
	logger *zap.Logger
	
	// Component registration
	components     map[string]bool
	componentsMu   sync.RWMutex
	
	// Metrics
	initCount           *prometheus.CounterVec
	initErrorCount      *prometheus.CounterVec
	initDuration        *prometheus.HistogramVec
	memoryUsage         *prometheus.GaugeVec
	componentStatus     *prometheus.GaugeVec
	
	// Sampling
	samplingEnabled     bool
	samplingRate        float64
	samplingRateMu      sync.RWMutex
	samplingCounters    map[string]int
	samplingCountersMu  sync.RWMutex
	
	// Aggregation
	aggregationEnabled  bool
	aggregationInterval time.Duration
	aggregationBuffer   map[string][]time.Duration
	aggregationMu       sync.Mutex
	aggregationTimer    *time.Ticker
	aggregationStopCh   chan struct{}
}

// NewAdaptiveMetrics creates a new adaptive metrics collector
func NewAdaptiveMetrics(logger *zap.Logger) *AdaptiveMetrics {
	metrics := &AdaptiveMetrics{
		logger:              logger,
		components:          make(map[string]bool),
		samplingEnabled:     true,
		samplingRate:        0.1, // Sample 10% by default
		samplingCounters:    make(map[string]int),
		aggregationEnabled:  true,
		aggregationInterval: 10 * time.Second,
		aggregationBuffer:   make(map[string][]time.Duration),
		aggregationStopCh:   make(chan struct{}),
	}
	
	// Initialize Prometheus metrics
	metrics.initCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lazy_loading_initialization_count",
			Help: "Number of component initializations",
		},
		[]string{"component"},
	)
	
	metrics.initErrorCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lazy_loading_initialization_error_count",
			Help: "Number of component initialization errors",
		},
		[]string{"component"},
	)
	
	metrics.initDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lazy_loading_initialization_duration_seconds",
			Help:    "Duration of component initialization in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // From 1ms to ~16s
		},
		[]string{"component"},
	)
	
	metrics.memoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lazy_loading_memory_usage_bytes",
			Help: "Memory usage of lazy-loaded components in bytes",
		},
		[]string{"component"},
	)
	
	metrics.componentStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lazy_loading_component_status",
			Help: "Status of lazy-loaded components (1=initialized, 0=not initialized)",
		},
		[]string{"component"},
	)
	
	// Register metrics with Prometheus
	prometheus.MustRegister(
		metrics.initCount,
		metrics.initErrorCount,
		metrics.initDuration,
		metrics.memoryUsage,
		metrics.componentStatus,
	)
	
	// Start aggregation timer if enabled
	if metrics.aggregationEnabled {
		metrics.startAggregation()
	}
	
	return metrics
}

// RegisterComponent registers a component for metrics tracking
func (m *AdaptiveMetrics) RegisterComponent(name string) {
	m.componentsMu.Lock()
	defer m.componentsMu.Unlock()
	
	m.components[name] = true
	
	// Initialize status to 0 (not initialized)
	m.componentStatus.WithLabelValues(name).Set(0)
}

// RecordInitialization records a component initialization
func (m *AdaptiveMetrics) RecordInitialization(component string, duration time.Duration) {
	// Check if sampling is enabled and if we should sample this event
	if m.samplingEnabled && !m.shouldSample(component) {
		return
	}
	
	// Record metrics
	m.initCount.WithLabelValues(component).Inc()
	
	// If aggregation is enabled, buffer the duration
	if m.aggregationEnabled {
		m.bufferDuration(component, duration)
	} else {
		// Otherwise, record directly
		m.initDuration.WithLabelValues(component).Observe(duration.Seconds())
	}
	
	// Update status
	m.componentStatus.WithLabelValues(component).Set(1)
}

// RecordInitializationError records a component initialization error
func (m *AdaptiveMetrics) RecordInitializationError(component string, duration time.Duration) {
	// Check if sampling is enabled and if we should sample this event
	if m.samplingEnabled && !m.shouldSample(component) {
		return
	}
	
	// Record metrics
	m.initErrorCount.WithLabelValues(component).Inc()
	
	// If aggregation is enabled, buffer the duration
	if m.aggregationEnabled {
		m.bufferDuration(component, duration)
	} else {
		// Otherwise, record directly
		m.initDuration.WithLabelValues(component).Observe(duration.Seconds())
	}
	
	// Update status
	m.componentStatus.WithLabelValues(component).Set(0)
}

// RecordMemoryUsage records memory usage for a component
func (m *AdaptiveMetrics) RecordMemoryUsage(component string, bytes int64) {
	// Check if sampling is enabled and if we should sample this event
	if m.samplingEnabled && !m.shouldSample(component) {
		return
	}
	
	// Record metrics
	m.memoryUsage.WithLabelValues(component).Set(float64(bytes))
}

// shouldSample determines if an event should be sampled
func (m *AdaptiveMetrics) shouldSample(component string) bool {
	m.samplingRateMu.RLock()
	rate := m.samplingRate
	m.samplingRateMu.RUnlock()
	
	// Always sample if rate is 1.0
	if rate >= 1.0 {
		return true
	}
	
	// Never sample if rate is 0.0
	if rate <= 0.0 {
		return false
	}
	
	// Increment counter and check if we should sample
	m.samplingCountersMu.Lock()
	defer m.samplingCountersMu.Unlock()
	
	m.samplingCounters[component]++
	counter := m.samplingCounters[component]
	
	// Sample based on rate (e.g., if rate is 0.1, sample every 10th event)
	return counter%int(1.0/rate) == 0
}

// bufferDuration buffers a duration for aggregation
func (m *AdaptiveMetrics) bufferDuration(component string, duration time.Duration) {
	m.aggregationMu.Lock()
	defer m.aggregationMu.Unlock()
	
	// Initialize buffer if needed
	if _, ok := m.aggregationBuffer[component]; !ok {
		m.aggregationBuffer[component] = make([]time.Duration, 0)
	}
	
	// Add duration to buffer
	m.aggregationBuffer[component] = append(m.aggregationBuffer[component], duration)
}

// startAggregation starts the aggregation timer
func (m *AdaptiveMetrics) startAggregation() {
	m.aggregationTimer = time.NewTicker(m.aggregationInterval)
	
	go func() {
		for {
			select {
			case <-m.aggregationTimer.C:
				m.flushAggregation()
			case <-m.aggregationStopCh:
				m.aggregationTimer.Stop()
				return
			}
		}
	}()
}

// flushAggregation flushes the aggregation buffer
func (m *AdaptiveMetrics) flushAggregation() {
	m.aggregationMu.Lock()
	defer m.aggregationMu.Unlock()
	
	// Process each component
	for component, durations := range m.aggregationBuffer {
		if len(durations) == 0 {
			continue
		}
		
		// Record each duration
		for _, duration := range durations {
			m.initDuration.WithLabelValues(component).Observe(duration.Seconds())
		}
		
		// Clear buffer
		m.aggregationBuffer[component] = make([]time.Duration, 0)
	}
}

// stopAggregation stops the aggregation timer
func (m *AdaptiveMetrics) stopAggregation() {
	close(m.aggregationStopCh)
	
	// Flush any remaining data
	m.flushAggregation()
}

// SetSamplingRate sets the sampling rate (0.0-1.0)
func (m *AdaptiveMetrics) SetSamplingRate(rate float64) {
	m.samplingRateMu.Lock()
	defer m.samplingRateMu.Unlock()
	
	// Clamp rate to valid range
	if rate < 0.0 {
		rate = 0.0
	} else if rate > 1.0 {
		rate = 1.0
	}
	
	m.samplingRate = rate
}

// EnableSampling enables or disables sampling
func (m *AdaptiveMetrics) EnableSampling(enabled bool) {
	m.samplingRateMu.Lock()
	defer m.samplingRateMu.Unlock()
	
	m.samplingEnabled = enabled
}

// SetAggregationInterval sets the aggregation interval
func (m *AdaptiveMetrics) SetAggregationInterval(interval time.Duration) {
	m.aggregationMu.Lock()
	defer m.aggregationMu.Unlock()
	
	// Stop current timer
	if m.aggregationTimer != nil {
		m.aggregationTimer.Stop()
	}
	
	m.aggregationInterval = interval
	
	// Restart timer if enabled
	if m.aggregationEnabled {
		m.aggregationTimer = time.NewTicker(interval)
	}
}

// EnableAggregation enables or disables aggregation
func (m *AdaptiveMetrics) EnableAggregation(enabled bool) {
	m.aggregationMu.Lock()
	defer m.aggregationMu.Unlock()
	
	// If changing state
	if m.aggregationEnabled != enabled {
		m.aggregationEnabled = enabled
		
		// Start or stop aggregation
		if enabled {
			m.startAggregation()
		} else {
			m.stopAggregation()
		}
	}
}

// GetInitializationCount returns the initialization count for a component
func (m *AdaptiveMetrics) GetInitializationCount(component string) float64 {
	return getCounterValue(m.initCount, component)
}

// GetInitializationErrorCount returns the initialization error count for a component
func (m *AdaptiveMetrics) GetInitializationErrorCount(component string) float64 {
	return getCounterValue(m.initErrorCount, component)
}

// GetAverageInitializationTime returns the average initialization time for a component
func (m *AdaptiveMetrics) GetAverageInitializationTime(component string) time.Duration {
	// This is an approximation since Prometheus histograms don't directly provide averages
	// In a real implementation, we might want to track this separately
	return time.Duration(0)
}

// getCounterValue gets the value of a counter for a component
func getCounterValue(counter *prometheus.CounterVec, component string) float64 {
	// This is a placeholder for a real implementation
	// In a real system, we would use the Prometheus API to get the value
	return 0
}

// Close closes the metrics collector
func (m *AdaptiveMetrics) Close() {
	// Stop aggregation if enabled
	if m.aggregationEnabled {
		m.stopAggregation()
	}
}

