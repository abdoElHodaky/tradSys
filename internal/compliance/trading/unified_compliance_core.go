package compliance

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// NewUnifiedComplianceEngine creates a new unified compliance engine
func NewUnifiedComplianceEngine(config *ComplianceConfig, logger *zap.Logger) *UnifiedComplianceEngine {
	engine := &UnifiedComplianceEngine{
		config:      config,
		logger:      logger,
		metrics:     &ComplianceMetrics{LastUpdateTime: time.Now()},
		stopChannel: make(chan struct{}),
	}

	// Initialize rule engine
	engine.ruleEngine = &RuleEngine{
		rules:  make(map[string]ComplianceRule),
		logger: logger.Named("rules"),
	}

	// Initialize report generator
	engine.reportGenerator = &ReportGenerator{
		templates: make(map[string]ReportTemplate),
		logger:    logger.Named("reports"),
	}

	// Initialize audit trail
	engine.auditTrail = &AuditTrail{
		maxEntries: 100000, // Keep last 100k entries
		logger:     logger.Named("audit"),
	}

	// Initialize alert manager
	engine.alertManager = &AlertManager{
		logger: logger.Named("alerts"),
	}

	return engine
}

// Start starts the unified compliance engine
func (c *UnifiedComplianceEngine) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&c.isRunning, 0, 1) {
		return fmt.Errorf("compliance engine is already running")
	}

	c.logger.Info("Starting unified compliance engine",
		zap.Any("config", c.config))

	// Load default compliance rules
	c.loadDefaultRules()

	// Start periodic reporting if enabled
	if c.config.ReportingEnabled && c.config.AutoReportingEnabled {
		go c.periodicReporting(ctx)
	}

	// Start audit trail cleanup
	go c.auditTrailCleanup(ctx)

	c.logger.Info("Unified compliance engine started successfully")
	return nil
}

// Stop stops the unified compliance engine
func (c *UnifiedComplianceEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&c.isRunning, 1, 0) {
		return fmt.Errorf("compliance engine is not running")
	}

	c.logger.Info("Stopping unified compliance engine")
	close(c.stopChannel)
	c.logger.Info("Unified compliance engine stopped")
	return nil
}

// CheckCompliance performs compliance checks on an order
func (c *UnifiedComplianceEngine) CheckCompliance(order *types.Order, userID string) (*ComplianceResult, error) {
	startTime := time.Now()
	defer func() {
		checkTime := time.Since(startTime)
		c.updateMetrics(checkTime)
	}()

	result := &ComplianceResult{
		Passed:    true,
		CheckTime: time.Since(startTime),
		Timestamp: time.Now(),
	}

	// Record audit entry
	if c.config.AuditTrailEnabled {
		c.auditTrail.AddEntry(&AuditEntry{
			ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
			EventType: AuditEventViolationDetected,
			UserID:    userID,
			OrderID:   order.ID,
			Action:    "compliance_check",
			Details: map[string]interface{}{
				"order_type": order.Type,
				"quantity":   order.Quantity,
				"price":      order.Price,
			},
			Timestamp: time.Now(),
		})
	}

	// Run compliance rules
	violations := c.ruleEngine.CheckRules(order, userID)
	if len(violations) > 0 {
		result.Passed = false
		result.Violations = violations

		// Trigger alerts for violations
		for _, violation := range violations {
			c.triggerViolationAlert(&violation)
		}

		atomic.AddInt64(&c.metrics.FailedChecks, 1)
		atomic.AddInt64(&c.metrics.ViolationsDetected, int64(len(violations)))
	} else {
		atomic.AddInt64(&c.metrics.PassedChecks, 1)
	}

	atomic.AddInt64(&c.metrics.TotalChecks, 1)
	return result, nil
}

// loadDefaultRules loads default compliance rules
func (c *UnifiedComplianceEngine) loadDefaultRules() {
	// Position limit rule
	c.ruleEngine.AddRule(ComplianceRule{
		ID:          "position_limit_001",
		Name:        "Maximum Position Limit",
		Description: "Prevents positions from exceeding maximum allowed size",
		Type:        RuleTypePositionLimit,
		Enabled:     true,
		Priority:    1,
		Parameters: map[string]interface{}{
			"max_position_size": 1000000.0,
			"currency":          "USD",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Trading hours rule
	c.ruleEngine.AddRule(ComplianceRule{
		ID:          "trading_hours_001",
		Name:        "Trading Hours Restriction",
		Description: "Prevents trading outside of allowed hours",
		Type:        RuleTypeTradingHours,
		Enabled:     true,
		Priority:    2,
		Parameters: map[string]interface{}{
			"start_time": "09:00",
			"end_time":   "16:00",
			"timezone":   "UTC",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Risk limit rule
	c.ruleEngine.AddRule(ComplianceRule{
		ID:          "risk_limit_001",
		Name:        "Daily Risk Limit",
		Description: "Prevents daily losses from exceeding limit",
		Type:        RuleTypeRiskLimit,
		Enabled:     true,
		Priority:    1,
		Parameters: map[string]interface{}{
			"max_daily_loss": 50000.0,
			"currency":       "USD",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	c.logger.Info("Default compliance rules loaded", zap.Int("count", 3))
}

// periodicReporting runs periodic compliance reporting
func (c *UnifiedComplianceEngine) periodicReporting(ctx context.Context) {
	ticker := time.NewTicker(c.config.ReportingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChannel:
			return
		case <-ticker.C:
			c.generatePeriodicReports()
		}
	}
}

// generatePeriodicReports generates periodic compliance reports
func (c *UnifiedComplianceEngine) generatePeriodicReports() {
	c.logger.Info("Generating periodic compliance reports")

	// Generate daily report
	report := c.reportGenerator.GenerateReport(ReportTypeDaily,
		time.Now().AddDate(0, 0, -1), time.Now())

	if report != nil {
		atomic.AddInt64(&c.metrics.ReportsGenerated, 1)
		c.logger.Info("Daily compliance report generated", zap.String("report_id", report.ID))
	}
}

// auditTrailCleanup periodically cleans up old audit trail entries
func (c *UnifiedComplianceEngine) auditTrailCleanup(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChannel:
			return
		case <-ticker.C:
			if c.config.AuditTrailEnabled {
				c.auditTrail.Cleanup(c.config.RetentionPeriod)
			}
		}
	}
}

// triggerViolationAlert triggers an alert for a compliance violation
func (c *UnifiedComplianceEngine) triggerViolationAlert(violation *ComplianceViolation) {
	if !c.config.AlertingEnabled {
		return
	}

	alert := &ComplianceAlert{
		ID:       fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Type:     AlertTypeViolation,
		Severity: violation.Severity,
		Title:    fmt.Sprintf("Compliance Violation: %s", violation.ViolationType),
		Message:  violation.Description,
		Details: map[string]interface{}{
			"violation_id": violation.ID,
			"rule_id":      violation.RuleID,
			"user_id":      violation.UserID,
			"order_id":     violation.OrderID,
		},
		Status:    AlertStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	c.alertManager.TriggerAlert(alert)
	atomic.AddInt64(&c.metrics.AlertsTriggered, 1)

	c.logger.Warn("Compliance violation alert triggered",
		zap.String("alert_id", alert.ID),
		zap.String("violation_type", violation.ViolationType),
		zap.String("severity", string(violation.Severity)))
}

// updateMetrics updates compliance metrics
func (c *UnifiedComplianceEngine) updateMetrics(checkTime time.Duration) {
	c.metrics.AverageCheckTime = checkTime
	c.metrics.LastUpdateTime = time.Now()
}

// GetMetrics returns current compliance metrics
func (c *UnifiedComplianceEngine) GetMetrics() *ComplianceMetrics {
	return c.metrics
}
