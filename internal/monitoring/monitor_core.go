package monitoring

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrMonitorAlreadyRunning = errors.New("monitor is already running")
	ErrMonitorNotRunning     = errors.New("monitor is not running")
	ErrInvalidConfig         = errors.New("invalid monitor configuration")
)

// NewUnifiedMonitor creates a new unified monitor
func NewUnifiedMonitor(config *MonitorConfig, logger *zap.Logger) (*UnifiedMonitor, error) {
	if config == nil {
		return nil, ErrInvalidConfig
	}

	// Set default values
	if config.MetricsInterval == 0 {
		config.MetricsInterval = 30 * time.Second
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 60 * time.Second
	}
	if config.AlertCheckInterval == 0 {
		config.AlertCheckInterval = 15 * time.Second
	}
	if config.MaxMetricsHistory == 0 {
		config.MaxMetricsHistory = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &UnifiedMonitor{
		config:   config,
		logger:   logger,
		registry: prometheus.NewRegistry(),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize components
	monitor.metricsCollector = NewUnifiedMetricsCollector(config, logger)
	monitor.alertManager = NewUnifiedAlertManager(config, logger)
	monitor.healthChecker = NewHealthChecker(config, logger)
	monitor.performanceTracker = NewPerformanceTracker(config, logger)

	return monitor, nil
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
func (m *UnifiedMonitor) GetHealth() (*UnifiedHealthStatus, error) {
	return m.healthChecker.GetCurrentHealth()
}

// GetAlerts returns active alerts
func (m *UnifiedMonitor) GetAlerts() ([]*UnifiedAlert, error) {
	return m.alertManager.GetActiveAlerts()
}

// RecordMetric records a custom metric
func (m *UnifiedMonitor) RecordMetric(name string, value float64, labels map[string]string) {
	m.metricsCollector.RecordMetric(name, value, labels)
}

// TriggerAlert triggers a custom alert
func (m *UnifiedMonitor) TriggerAlert(alert *UnifiedAlert) {
	m.alertManager.TriggerAlert(alert)
}

// RegisterHealthCheck registers a health check
func (m *UnifiedMonitor) RegisterHealthCheck(name string, check UnifiedHealthCheck) {
	m.healthChecker.RegisterCheck(name, check)
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

			// Update performance tracker
			if metrics, err := m.metricsCollector.GetCurrentMetrics(); err == nil {
				m.performanceTracker.RecordMetrics(metrics)
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
			if err := m.healthChecker.RunChecks(); err != nil {
				m.logger.Error("Failed to run health checks", zap.Error(err))
			}
		case <-m.ctx.Done():
			return
		}
	}
}

// alertCheckLoop runs the alert check loop
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

// NewUnifiedMetricsCollector creates a new metrics collector
func NewUnifiedMetricsCollector(config *MonitorConfig, logger *zap.Logger) *UnifiedMetricsCollector {
	return &UnifiedMetricsCollector{
		config:  config,
		logger:  logger,
		metrics: &SystemMetrics{},
	}
}

// Start starts the metrics collector
func (mc *UnifiedMetricsCollector) Start() error {
	mc.logger.Info("Starting metrics collector")
	return nil
}

// Stop stops the metrics collector
func (mc *UnifiedMetricsCollector) Stop() error {
	mc.logger.Info("Stopping metrics collector")
	return nil
}

// CollectMetrics collects current system metrics
func (mc *UnifiedMetricsCollector) CollectMetrics() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Collect metrics from various sources
	mc.metrics = &SystemMetrics{
		OrdersPerSecond: mc.getOrdersPerSecond(),
		TradesPerSecond: mc.getTradesPerSecond(),
		MatchingLatency: mc.getMatchingLatency(),
		CPUUsage:        mc.getCPUUsage(),
		MemoryUsage:     mc.getMemoryUsage(),
		ErrorRate:       mc.getErrorRate(),
		ResponseTime:    mc.getResponseTime(),
		Timestamp:       time.Now(),
	}

	return nil
}

// GetCurrentMetrics returns current metrics
func (mc *UnifiedMetricsCollector) GetCurrentMetrics() (*SystemMetrics, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy
	metrics := *mc.metrics
	return &metrics, nil
}

// RecordMetric records a custom metric
func (mc *UnifiedMetricsCollector) RecordMetric(name string, value float64, labels map[string]string) {
	mc.logger.Debug("Recording custom metric",
		zap.String("name", name),
		zap.Float64("value", value))
}

// Helper methods for metrics collection
func (mc *UnifiedMetricsCollector) getOrdersPerSecond() float64 {
	// Placeholder implementation
	return 100.0
}

func (mc *UnifiedMetricsCollector) getTradesPerSecond() float64 {
	// Placeholder implementation
	return 50.0
}

func (mc *UnifiedMetricsCollector) getMatchingLatency() float64 {
	// Placeholder implementation
	return 2.5
}

func (mc *UnifiedMetricsCollector) getCPUUsage() float64 {
	// Placeholder implementation
	return 45.0
}

func (mc *UnifiedMetricsCollector) getMemoryUsage() float64 {
	// Placeholder implementation
	return 60.0
}

func (mc *UnifiedMetricsCollector) getErrorRate() float64 {
	// Placeholder implementation
	return 0.1
}

func (mc *UnifiedMetricsCollector) getResponseTime() float64 {
	// Placeholder implementation
	return 150.0
}

// NewUnifiedAlertManager creates a new alert manager
func NewUnifiedAlertManager(config *MonitorConfig, logger *zap.Logger) *UnifiedAlertManager {
	return &UnifiedAlertManager{
		config: config,
		logger: logger,
		alerts: make(map[string]*UnifiedAlert),
		rules:  make([]*AlertRule, 0),
	}
}

// Start starts the alert manager
func (am *UnifiedAlertManager) Start() error {
	am.logger.Info("Starting alert manager")
	return nil
}

// Stop stops the alert manager
func (am *UnifiedAlertManager) Stop() error {
	am.logger.Info("Stopping alert manager")
	return nil
}

// CheckAlerts checks alert conditions
func (am *UnifiedAlertManager) CheckAlerts() error {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Check alert rules
	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		// Placeholder alert checking logic
		am.logger.Debug("Checking alert rule", zap.String("rule", rule.Name))
	}

	return nil
}

// GetActiveAlerts returns active alerts
func (am *UnifiedAlertManager) GetActiveAlerts() ([]*UnifiedAlert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []*UnifiedAlert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// TriggerAlert triggers an alert
func (am *UnifiedAlertManager) TriggerAlert(alert *UnifiedAlert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.alerts[alert.ID] = alert
	am.logger.Info("Alert triggered",
		zap.String("id", alert.ID),
		zap.String("type", string(alert.Type)),
		zap.String("severity", string(alert.Severity)),
		zap.String("message", alert.Message))
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *MonitorConfig, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		config: config,
		logger: logger,
		checks: make(map[string]UnifiedHealthCheck),
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
func (hc *HealthChecker) RegisterCheck(name string, check UnifiedHealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = check
	hc.logger.Info("Registered health check", zap.String("name", name))
}

// RunChecks runs all health checks
func (hc *HealthChecker) RunChecks() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	components := make(map[string]HealthState)
	details := make(map[string]interface{})
	overall := HealthStateHealthy

	for name, check := range hc.checks {
		state, detail, err := check()
		if err != nil {
			hc.logger.Error("Health check failed",
				zap.String("check", name),
				zap.Error(err))
			state = HealthStateUnknown
		}

		components[name] = state
		if detail != nil {
			details[name] = detail
		}

		// Determine overall health
		if state == HealthStateCritical {
			overall = HealthStateCritical
		} else if state == HealthStateWarning && overall != HealthStateCritical {
			overall = HealthStateWarning
		}
	}

	hc.lastStatus = &UnifiedHealthStatus{
		Overall:    overall,
		Components: components,
		Timestamp:  time.Now(),
		Details:    details,
	}

	return nil
}

// GetCurrentHealth returns current health status
func (hc *HealthChecker) GetCurrentHealth() (*UnifiedHealthStatus, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.lastStatus == nil {
		return &UnifiedHealthStatus{
			Overall:   HealthStateUnknown,
			Timestamp: time.Now(),
		}, nil
	}

	// Return a copy
	status := *hc.lastStatus
	return &status, nil
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker(config *MonitorConfig, logger *zap.Logger) *PerformanceTracker {
	return &PerformanceTracker{
		config:         config,
		logger:         logger,
		metricsHistory: make([]*SystemMetrics, 0, config.MaxMetricsHistory),
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

// RecordMetrics records metrics for performance tracking
func (pt *PerformanceTracker) RecordMetrics(metrics *SystemMetrics) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Add to history
	pt.metricsHistory = append(pt.metricsHistory, metrics)

	// Trim history if needed
	if len(pt.metricsHistory) > pt.config.MaxMetricsHistory {
		pt.metricsHistory = pt.metricsHistory[1:]
	}
}
