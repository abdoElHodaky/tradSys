package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	
	"github.com/abdoElHodaky/tradSys/internal/hft/config"
	"github.com/abdoElHodaky/tradSys/internal/hft/memory"
	"github.com/abdoElHodaky/tradSys/internal/hft/metrics"
)

// HFTMonitoringConfig contains monitoring configuration
type HFTMonitoringConfig struct {
	// Metrics collection
	EnablePrometheus    bool          `yaml:"enable_prometheus" default:"true"`
	EnableCustomMetrics bool          `yaml:"enable_custom_metrics" default:"true"`
	MetricsInterval     time.Duration `yaml:"metrics_interval" default:"10s"`
	
	// Health checks
	EnableHealthChecks  bool          `yaml:"enable_health_checks" default:"true"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" default:"30s"`
	
	// Alerting
	EnableAlerting      bool          `yaml:"enable_alerting" default:"true"`
	AlertThresholds     AlertThresholds `yaml:"alert_thresholds"`
	
	// Performance monitoring
	EnablePerformanceMonitoring bool          `yaml:"enable_performance_monitoring" default:"true"`
	PerformanceInterval         time.Duration `yaml:"performance_interval" default:"5s"`
	
	// Dashboard
	EnableDashboard bool   `yaml:"enable_dashboard" default:"true"`
	DashboardPort   int    `yaml:"dashboard_port" default:"9090"`
	DashboardPath   string `yaml:"dashboard_path" default:"/dashboard"`
}

// AlertThresholds contains alerting thresholds
type AlertThresholds struct {
	MaxLatency        time.Duration `yaml:"max_latency" default:"100ms"`
	MaxErrorRate      float64       `yaml:"max_error_rate" default:"0.01"`      // 1%
	MaxMemoryUsage    int64         `yaml:"max_memory_usage" default:"1073741824"` // 1GB
	MaxGCPauseTime    time.Duration `yaml:"max_gc_pause_time" default:"10ms"`
	MinThroughput     int64         `yaml:"min_throughput" default:"1000"`     // requests/sec
}

// HFTProductionMonitor provides comprehensive production monitoring
type HFTProductionMonitor struct {
	config *HFTMonitoringConfig
	
	// Prometheus metrics
	requestDuration    *prometheus.HistogramVec
	requestCount       *prometheus.CounterVec
	errorCount         *prometheus.CounterVec
	memoryUsage        prometheus.Gauge
	gcPauseTime        prometheus.Gauge
	activeConnections  prometheus.Gauge
	throughput         prometheus.Gauge
	
	// Health checks
	healthChecks map[string]HealthCheck
	healthStatus atomic.Value // HealthStatus
	
	// Performance tracking
	performanceMetrics *PerformanceMetrics
	
	// Alerting
	alertManager *AlertManager
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	mu sync.RWMutex
}

// HealthCheck represents a health check function
type HealthCheck func() error

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
	Uptime    time.Duration          `json:"uptime"`
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Status  string        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency"`
}

// PerformanceMetrics tracks performance metrics
type PerformanceMetrics struct {
	RequestsPerSecond   float64   `json:"requests_per_second"`
	AverageLatency      time.Duration `json:"average_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	P99Latency          time.Duration `json:"p99_latency"`
	ErrorRate           float64   `json:"error_rate"`
	MemoryUsage         uint64    `json:"memory_usage"`
	GCPauseTime         time.Duration `json:"gc_pause_time"`
	ActiveConnections   int64     `json:"active_connections"`
	Timestamp           time.Time `json:"timestamp"`
}

// NewHFTProductionMonitor creates a new production monitor
func NewHFTProductionMonitor(config *HFTMonitoringConfig) *HFTProductionMonitor {
	if config == nil {
		config = &HFTMonitoringConfig{
			EnablePrometheus:            true,
			EnableCustomMetrics:         true,
			MetricsInterval:            10 * time.Second,
			EnableHealthChecks:         true,
			HealthCheckInterval:        30 * time.Second,
			EnableAlerting:             true,
			EnablePerformanceMonitoring: true,
			PerformanceInterval:        5 * time.Second,
			EnableDashboard:            true,
			DashboardPort:              9090,
			DashboardPath:              "/dashboard",
			AlertThresholds: AlertThresholds{
				MaxLatency:     100 * time.Millisecond,
				MaxErrorRate:   0.01,
				MaxMemoryUsage: 1073741824, // 1GB
				MaxGCPauseTime: 10 * time.Millisecond,
				MinThroughput:  1000,
			},
		}
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &HFTProductionMonitor{
		config:         config,
		healthChecks:   make(map[string]HealthCheck),
		performanceMetrics: &PerformanceMetrics{},
		alertManager:   NewAlertManager(&config.AlertThresholds),
		ctx:           ctx,
		cancel:        cancel,
	}
	
	// Initialize Prometheus metrics
	if config.EnablePrometheus {
		monitor.initPrometheusMetrics()
	}
	
	// Initialize health status
	monitor.healthStatus.Store(HealthStatus{
		Status:    "unknown",
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckResult),
	})
	
	// Start monitoring goroutines
	monitor.start()
	
	return monitor
}

// initPrometheusMetrics initializes Prometheus metrics
func (m *HFTProductionMonitor) initPrometheusMetrics() {
	m.requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hft_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"method", "endpoint", "status"},
	)
	
	m.requestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hft_requests_total",
			Help: "Total number of requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	
	m.errorCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hft_errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "endpoint"},
	)
	
	m.memoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hft_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
	)
	
	m.gcPauseTime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hft_gc_pause_time_seconds",
			Help: "Last GC pause time in seconds",
		},
	)
	
	m.activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hft_active_connections",
			Help: "Number of active connections",
		},
	)
	
	m.throughput = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hft_throughput_requests_per_second",
			Help: "Current throughput in requests per second",
		},
	)
}

// start starts the monitoring goroutines
func (m *HFTProductionMonitor) start() {
	if m.config.EnableCustomMetrics {
		m.wg.Add(1)
		go m.metricsCollectionLoop()
	}
	
	if m.config.EnableHealthChecks {
		m.wg.Add(1)
		go m.healthCheckLoop()
	}
	
	if m.config.EnablePerformanceMonitoring {
		m.wg.Add(1)
		go m.performanceMonitoringLoop()
	}
	
	if m.config.EnableDashboard {
		m.wg.Add(1)
		go m.startDashboard()
	}
}

// metricsCollectionLoop collects metrics periodically
func (m *HFTProductionMonitor) metricsCollectionLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collectMetrics()
		}
	}
}

// collectMetrics collects various metrics
func (m *HFTProductionMonitor) collectMetrics() {
	// Collect memory metrics
	if memory.GlobalMemoryManager != nil {
		memStats := memory.GlobalMemoryManager.GetMemoryStats()
		if m.memoryUsage != nil {
			m.memoryUsage.Set(float64(memStats.HeapAlloc))
		}
		if m.gcPauseTime != nil && len(memStats.PauseTotal) > 0 {
			m.gcPauseTime.Set(memStats.PauseTotal.Seconds())
		}
	}
	
	// Collect GC metrics
	gcStats := config.GetGCStats()
	if gcStats != nil && m.gcPauseTime != nil {
		m.gcPauseTime.Set(gcStats.PauseTotal.Seconds())
	}
	
	// Collect connection metrics
	if metrics.GlobalMetrics != nil {
		// This would integrate with your WebSocket manager
		// m.activeConnections.Set(float64(wsManager.GetConnectionCount()))
	}
}

// healthCheckLoop runs health checks periodically
func (m *HFTProductionMonitor) healthCheckLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.runHealthChecks()
		}
	}
}

// runHealthChecks runs all registered health checks
func (m *HFTProductionMonitor) runHealthChecks() {
	m.mu.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range m.healthChecks {
		checks[name] = check
	}
	m.mu.RUnlock()
	
	results := make(map[string]CheckResult)
	overallStatus := "healthy"
	
	for name, check := range checks {
		start := time.Now()
		err := check()
		latency := time.Since(start)
		
		if err != nil {
			results[name] = CheckResult{
				Status:  "unhealthy",
				Message: err.Error(),
				Latency: latency,
			}
			overallStatus = "unhealthy"
		} else {
			results[name] = CheckResult{
				Status:  "healthy",
				Latency: latency,
			}
		}
	}
	
	// Update health status
	status := HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    results,
		Uptime:    time.Since(time.Now()), // This would be calculated from start time
	}
	
	m.healthStatus.Store(status)
}

// performanceMonitoringLoop monitors performance metrics
func (m *HFTProductionMonitor) performanceMonitoringLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.PerformanceInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updatePerformanceMetrics()
		}
	}
}

// updatePerformanceMetrics updates performance metrics
func (m *HFTProductionMonitor) updatePerformanceMetrics() {
	// This would integrate with your metrics system
	if metrics.GlobalMetrics != nil {
		// Update performance metrics based on collected data
		m.performanceMetrics.Timestamp = time.Now()
		
		// Check thresholds and trigger alerts
		if m.config.EnableAlerting {
			m.checkAlertThresholds()
		}
	}
}

// checkAlertThresholds checks if any alert thresholds are exceeded
func (m *HFTProductionMonitor) checkAlertThresholds() {
	thresholds := &m.config.AlertThresholds
	
	// Check latency
	if m.performanceMetrics.P99Latency > thresholds.MaxLatency {
		m.alertManager.TriggerAlert("high_latency", fmt.Sprintf(
			"P99 latency %v exceeds threshold %v",
			m.performanceMetrics.P99Latency,
			thresholds.MaxLatency,
		))
	}
	
	// Check error rate
	if m.performanceMetrics.ErrorRate > thresholds.MaxErrorRate {
		m.alertManager.TriggerAlert("high_error_rate", fmt.Sprintf(
			"Error rate %.4f exceeds threshold %.4f",
			m.performanceMetrics.ErrorRate,
			thresholds.MaxErrorRate,
		))
	}
	
	// Check memory usage
	if int64(m.performanceMetrics.MemoryUsage) > thresholds.MaxMemoryUsage {
		m.alertManager.TriggerAlert("high_memory_usage", fmt.Sprintf(
			"Memory usage %d bytes exceeds threshold %d bytes",
			m.performanceMetrics.MemoryUsage,
			thresholds.MaxMemoryUsage,
		))
	}
	
	// Check GC pause time
	if m.performanceMetrics.GCPauseTime > thresholds.MaxGCPauseTime {
		m.alertManager.TriggerAlert("high_gc_pause", fmt.Sprintf(
			"GC pause time %v exceeds threshold %v",
			m.performanceMetrics.GCPauseTime,
			thresholds.MaxGCPauseTime,
		))
	}
}

// startDashboard starts the monitoring dashboard
func (m *HFTProductionMonitor) startDashboard() {
	defer m.wg.Done()
	
	router := gin.New()
	router.Use(gin.Recovery())
	
	// Health endpoint
	router.GET("/health", m.healthHandler)
	
	// Metrics endpoint
	router.GET("/metrics", m.metricsHandler)
	
	// Performance endpoint
	router.GET("/performance", m.performanceHandler)
	
	// Dashboard endpoint
	router.GET(m.config.DashboardPath, m.dashboardHandler)
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.DashboardPort),
		Handler: router,
	}
	
	go func() {
		<-m.ctx.Done()
		server.Shutdown(context.Background())
	}()
	
	server.ListenAndServe()
}

// healthHandler handles health check requests
func (m *HFTProductionMonitor) healthHandler(c *gin.Context) {
	status := m.healthStatus.Load().(HealthStatus)
	
	if status.Status == "healthy" {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}

// metricsHandler handles metrics requests
func (m *HFTProductionMonitor) metricsHandler(c *gin.Context) {
	// Return custom metrics in JSON format
	metrics := map[string]interface{}{
		"memory": memory.GlobalMemoryManager.GetMemoryStats(),
		"gc":     config.GetGCStats(),
		"pools":  memory.GlobalMemoryManager.GetPoolStats(),
	}
	
	c.JSON(http.StatusOK, metrics)
}

// performanceHandler handles performance metrics requests
func (m *HFTProductionMonitor) performanceHandler(c *gin.Context) {
	c.JSON(http.StatusOK, m.performanceMetrics)
}

// dashboardHandler serves the monitoring dashboard
func (m *HFTProductionMonitor) dashboardHandler(c *gin.Context) {
	// This would serve a web dashboard
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "HFT Monitoring Dashboard",
	})
}

// RegisterHealthCheck registers a health check
func (m *HFTProductionMonitor) RegisterHealthCheck(name string, check HealthCheck) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.healthChecks[name] = check
}

// RecordRequest records a request for monitoring
func (m *HFTProductionMonitor) RecordRequest(method, endpoint string, status int, duration time.Duration) {
	if m.requestDuration != nil {
		m.requestDuration.WithLabelValues(method, endpoint, fmt.Sprintf("%d", status)).Observe(duration.Seconds())
	}
	
	if m.requestCount != nil {
		m.requestCount.WithLabelValues(method, endpoint, fmt.Sprintf("%d", status)).Inc()
	}
}

// RecordError records an error for monitoring
func (m *HFTProductionMonitor) RecordError(errorType, endpoint string) {
	if m.errorCount != nil {
		m.errorCount.WithLabelValues(errorType, endpoint).Inc()
	}
}

// GetHealthStatus returns the current health status
func (m *HFTProductionMonitor) GetHealthStatus() HealthStatus {
	return m.healthStatus.Load().(HealthStatus)
}

// GetPerformanceMetrics returns current performance metrics
func (m *HFTProductionMonitor) GetPerformanceMetrics() *PerformanceMetrics {
	return m.performanceMetrics
}

// Close shuts down the monitor
func (m *HFTProductionMonitor) Close() {
	m.cancel()
	m.wg.Wait()
}

// AlertManager manages alerts
type AlertManager struct {
	thresholds *AlertThresholds
	alerts     map[string]time.Time
	mu         sync.RWMutex
}

// NewAlertManager creates a new alert manager
func NewAlertManager(thresholds *AlertThresholds) *AlertManager {
	return &AlertManager{
		thresholds: thresholds,
		alerts:     make(map[string]time.Time),
	}
}

// TriggerAlert triggers an alert
func (am *AlertManager) TriggerAlert(alertType, message string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	// Check if we've already alerted for this recently (rate limiting)
	if lastAlert, exists := am.alerts[alertType]; exists {
		if time.Since(lastAlert) < 5*time.Minute {
			return // Don't spam alerts
		}
	}
	
	am.alerts[alertType] = time.Now()
	
	// Log the alert (in production, this would send to alerting system)
	fmt.Printf("[ALERT] %s: %s\n", alertType, message)
}

// Global production monitor instance
var GlobalProductionMonitor *HFTProductionMonitor

// InitProductionMonitor initializes the global production monitor
func InitProductionMonitor(config *HFTMonitoringConfig) {
	GlobalProductionMonitor = NewHFTProductionMonitor(config)
}
