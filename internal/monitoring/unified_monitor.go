package monitoring

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// UnifiedMonitor provides comprehensive system monitoring and metrics collection
type UnifiedMonitor struct {
	// Core components
	metricsCollector *MetricsCollector
	alertManager     *AlertManager
	healthChecker    *HealthChecker
	performanceTracker *PerformanceTracker
	
	// Configuration
	config *MonitorConfig
	logger *zap.Logger
	
	// Metrics registry
	registry *prometheus.Registry
	
	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	
	// System state
	isRunning bool
	startTime time.Time
}

// MonitorConfig contains monitoring configuration
type MonitorConfig struct {
	MetricsInterval     time.Duration `json:"metrics_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	AlertCheckInterval  time.Duration `json:"alert_check_interval"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	EnablePrometheus    bool          `json:"enable_prometheus"`
	EnableAlerts        bool          `json:"enable_alerts"`
	EnableHealthChecks  bool          `json:"enable_health_checks"`
	MaxMetricsHistory   int           `json:"max_metrics_history"`
}

// SystemMetrics represents comprehensive system metrics
type SystemMetrics struct {
	// Trading metrics
	OrdersPerSecond     float64 `json:"orders_per_second"`
	TradesPerSecond     float64 `json:"trades_per_second"`
	MatchingLatency     float64 `json:"matching_latency_ms"`
	OrderBookDepth      int     `json:"order_book_depth"`
	ActiveConnections   int64   `json:"active_connections"`
	
	// Performance metrics
	CPUUsage           float64 `json:"cpu_usage_percent"`
	MemoryUsage        float64 `json:"memory_usage_percent"`
	DiskUsage          float64 `json:"disk_usage_percent"`
	NetworkThroughput  float64 `json:"network_throughput_mbps"`
	
	// Application metrics
	ErrorRate          float64 `json:"error_rate_percent"`
	ResponseTime       float64 `json:"response_time_ms"`
	ThroughputRPS      float64 `json:"throughput_rps"`
	CacheHitRate       float64 `json:"cache_hit_rate_percent"`
	
	// Business metrics
	TotalVolume        float64   `json:"total_volume"`
	TotalTrades        int64     `json:"total_trades"`
	ActiveUsers        int64     `json:"active_users"`
	ComplianceScore    float64   `json:"compliance_score"`
	
	// Timestamp
	Timestamp time.Time `json:"timestamp"`
}

// HealthStatus is defined in production.go to avoid redeclaration

// HealthState represents health state
type HealthState string

const (
	HealthStateHealthy   HealthState = "healthy"
	HealthStateWarning   HealthState = "warning"
	HealthStateCritical  HealthState = "critical"
	HealthStateUnknown   HealthState = "unknown"
)

// Alert is defined in alerts.go to avoid redeclaration

// AlertType defines types of alerts
type AlertType string

const (
	AlertTypePerformance AlertType = "performance"
	AlertTypeError       AlertType = "error"
	AlertTypeCapacity    AlertType = "capacity"
	AlertTypeSecurity    AlertType = "security"
	AlertTypeCompliance  AlertType = "compliance"
	AlertTypeHealth      AlertType = "health"
)

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// NewUnifiedMonitor creates a new unified monitor
func NewUnifiedMonitor(config *MonitorConfig, logger *zap.Logger) *UnifiedMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &UnifiedMonitor{
		config:    config,
		logger:    logger,
		registry:  prometheus.NewRegistry(),
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}
	
	// Initialize components
	monitor.metricsCollector = NewMetricsCollector(logger)
	monitor.alertManager = NewAlertManager(logger)
	monitor.healthChecker = NewHealthChecker(config, logger)
	monitor.performanceTracker = NewPerformanceTracker(config, logger)
	
	return monitor
}

// Start starts the unified monitor
func (m *UnifiedMonitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.isRunning {
		return ErrMonitorAlreadyRunning
	}
	
	m.logger.Info("Starting unified monitor",
		zap.Duration("metrics_interval", m.config.MetricsInterval),
		zap.Duration("health_check_interval", m.config.HealthCheckInterval))
	
	// Start components
	if err := m.metricsCollector.Start(); err != nil {
		return err
	}
	
	if m.config.EnableAlerts {
		if err := m.alertManager.Start(); err != nil {
			return err
		}
	}
	
	if m.config.EnableHealthChecks {
		if err := m.healthChecker.Start(); err != nil {
			return err
		}
	}
	
	if err := m.performanceTracker.Start(); err != nil {
		return err
	}
	
	// Start background processes
	go m.metricsLoop()
	go m.healthCheckLoop()
	go m.alertCheckLoop()
	
	m.isRunning = true
	m.startTime = time.Now()
	
	return nil
}

// Stop stops the unified monitor
func (m *UnifiedMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.isRunning {
		return ErrMonitorNotRunning
	}
	
	m.logger.Info("Stopping unified monitor")
	
	m.cancel()
	
	// Stop components
	m.metricsCollector.Stop()
	m.alertManager.Stop()
	m.healthChecker.Stop()
	m.performanceTracker.Stop()
	
	m.isRunning = false
	
	return nil
}

// GetMetrics returns current system metrics
func (m *UnifiedMonitor) GetMetrics() (*SystemMetrics, error) {
	return m.metricsCollector.GetCurrentMetrics()
}

// GetHealth returns current system health
func (m *UnifiedMonitor) GetHealth() (*HealthStatus, error) {
	return m.healthChecker.GetCurrentHealth()
}

// GetAlerts returns active alerts
func (m *UnifiedMonitor) GetAlerts() ([]*Alert, error) {
	alerts := m.alertManager.GetActiveAlerts()
	return alerts, nil
}

// RecordMetric records a custom metric
func (m *UnifiedMonitor) RecordMetric(name string, value float64, labels map[string]string) {
	m.metricsCollector.RecordMetric(name, value, labels)
}

// TriggerAlert triggers a custom alert
func (m *UnifiedMonitor) TriggerAlert(alert *Alert) {
	m.alertManager.TriggerAlert(alert)
}

// RegisterHealthCheck registers a custom health check
func (m *UnifiedMonitor) RegisterHealthCheck(name string, checker HealthCheckFunc) {
	m.healthChecker.RegisterCheck(name, checker)
}

// metricsLoop runs the metrics collection loop
func (m *UnifiedMonitor) metricsLoop() {
	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := m.metricsCollector.CollectMetrics(); err != nil {
				m.logger.Error("Failed to collect metrics", zap.Error(err))
			}
		case <-m.ctx.Done():
			return
		}
	}
}

// healthCheckLoop runs the health check loop
func (m *UnifiedMonitor) healthCheckLoop() {
	if !m.config.EnableHealthChecks {
		return
	}
	
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := m.healthChecker.RunHealthChecks(); err != nil {
				m.logger.Error("Failed to run health checks", zap.Error(err))
			}
		case <-m.ctx.Done():
			return
		}
	}
}

// alertCheckLoop runs the alert checking loop
func (m *UnifiedMonitor) alertCheckLoop() {
	if !m.config.EnableAlerts {
		return
	}
	
	ticker := time.NewTicker(m.config.AlertCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := m.alertManager.CheckAlerts(); err != nil {
				m.logger.Error("Failed to check alerts", zap.Error(err))
			}
		case <-m.ctx.Done():
			return
		}
	}
}

// GetUptime returns system uptime
func (m *UnifiedMonitor) GetUptime() time.Duration {
	return time.Since(m.startTime)
}

// IsRunning returns whether the monitor is running
func (m *UnifiedMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetPrometheusRegistry returns the Prometheus registry
func (m *UnifiedMonitor) GetPrometheusRegistry() *prometheus.Registry {
	return m.registry
}

// Component interfaces and types

// MetricsCollector and NewMetricsCollector are defined in metrics.go to avoid redeclaration

// AlertManager and NewAlertManager are defined in alerts.go to avoid redeclaration

// HealthChecker performs health checks
type HealthChecker struct {
	config  *MonitorConfig
	logger  *zap.Logger
	checks  map[string]HealthCheckFunc
	status  *HealthStatus
	mu      sync.RWMutex
}

// HealthCheckFunc represents a health check function
type HealthCheckFunc func() HealthState

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *MonitorConfig, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		config: config,
		logger: logger,
		checks: make(map[string]HealthCheckFunc),
		status: &HealthStatus{
			Overall:    HealthStateHealthy,
			Components: make(map[string]HealthState),
			Timestamp:  time.Now(),
		},
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start() error {
	hc.logger.Info("Starting health checker")
	return nil
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() error {
	hc.logger.Info("Stopping health checker")
	return nil
}

// RegisterCheck registers a health check
func (hc *HealthChecker) RegisterCheck(name string, checker HealthCheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.checks[name] = checker
	hc.logger.Info("Health check registered", zap.String("name", name))
}

// RunHealthChecks runs all health checks
func (hc *HealthChecker) RunHealthChecks() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	components := make(map[string]HealthState)
	overall := HealthStateHealthy
	
	for name, checker := range hc.checks {
		state := checker()
		components[name] = state
		
		// Determine overall health
		if state == HealthStateCritical {
			overall = HealthStateCritical
		} else if state == HealthStateWarning && overall != HealthStateCritical {
			overall = HealthStateWarning
		}
	}
	
	hc.status = &HealthStatus{
		Overall:    overall,
		Components: components,
		Timestamp:  time.Now(),
	}
	
	return nil
}

// GetCurrentHealth returns current health status
func (hc *HealthChecker) GetCurrentHealth() (*HealthStatus, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	// Return a copy
	status := *hc.status
	return &status, nil
}

// PerformanceTracker tracks performance metrics
type PerformanceTracker struct {
	config *MonitorConfig
	logger *zap.Logger
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker(config *MonitorConfig, logger *zap.Logger) *PerformanceTracker {
	return &PerformanceTracker{
		config: config,
		logger: logger,
	}
}

// Start starts the performance tracker
func (pt *PerformanceTracker) Start() error {
	pt.logger.Info("Starting performance tracker")
	return nil
}

// Stop stops the performance tracker
func (pt *PerformanceTracker) Stop() error {
	pt.logger.Info("Stopping performance tracker")
	return nil
}

// Error definitions
var (
	ErrMonitorAlreadyRunning = errors.New("monitor is already running")
	ErrMonitorNotRunning     = errors.New("monitor is not running")
)
