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
		rules:      make(map[string]ComplianceRule),
		violations: make([]ComplianceViolation, 0),
		logger:     logger.Named("rules"),
	}

	// Initialize report generator
	engine.reportGenerator = &ReportGenerator{
		templates: make(map[string]ReportTemplate),
		reports:   make([]ComplianceReport, 0),
		logger:    logger.Named("reports"),
	}

	// Initialize audit trail
	engine.auditTrail = &AuditTrail{
		entries:    make([]AuditEntry, 0),
		maxEntries: 100000, // Keep last 100k entries
		logger:     logger.Named("audit"),
	}

	// Initialize alert manager
	engine.alertManager = &AlertManager{
		alerts:      make([]ComplianceAlert, 0),
		subscribers: make([]AlertHandler, 0),
		logger:      logger.Named("alerts"),
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
			EventType: AuditEventCompliance,
			UserID:    userID,
			OrderID:   order.ID,
			Symbol:    order.Symbol,
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
		Description: "Ensures positions don't exceed maximum allowed size",
		Regulation:  "INTERNAL",
		RuleType:    RuleTypePositionLimit,
		Parameters: map[string]interface{}{
			"max_position": 1000000.0,
		},
		Enabled:   true,
		Severity:  SeverityHigh,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Order size rule
	c.ruleEngine.AddRule(ComplianceRule{
		ID:          "order_size_001",
		Name:        "Maximum Order Size",
		Description: "Ensures orders don't exceed maximum allowed size",
		Regulation:  "INTERNAL",
		RuleType:    RuleTypeOrderSize,
		Parameters: map[string]interface{}{
			"max_order_size": 100000.0,
		},
		Enabled:   true,
		Severity:  SeverityMedium,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Trading hours rule
	c.ruleEngine.AddRule(ComplianceRule{
		ID:          "trading_hours_001",
		Name:        "Trading Hours Compliance",
		Description: "Ensures trading only occurs during allowed hours",
		Regulation:  "EXCHANGE",
		RuleType:    RuleTypeTradingHours,
		Parameters: map[string]interface{}{
			"start_hour": 9,
			"end_hour":   16,
			"timezone":   "UTC",
		},
		Enabled:   true,
		Severity:  SeverityMedium,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Risk limit rule
	c.ruleEngine.AddRule(ComplianceRule{
		ID:          "risk_limit_001",
		Name:        "Maximum Risk Exposure",
		Description: "Ensures risk exposure doesn't exceed limits",
		Regulation:  "INTERNAL",
		RuleType:    RuleTypeRiskLimit,
		Parameters: map[string]interface{}{
			"max_risk_exposure": 500000.0,
			"var_limit":         50000.0,
		},
		Enabled:   true,
		Severity:  SeverityCritical,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Market manipulation detection rule
	c.ruleEngine.AddRule(ComplianceRule{
		ID:          "market_manipulation_001",
		Name:        "Market Manipulation Detection",
		Description: "Detects potential market manipulation patterns",
		Regulation:  "REGULATORY",
		RuleType:    RuleTypeMarketManipulation,
		Parameters: map[string]interface{}{
			"order_frequency_threshold": 100,
			"price_deviation_threshold": 0.05,
			"volume_threshold":          10000.0,
		},
		Enabled:   true,
		Severity:  SeverityCritical,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	c.logger.Info("Loaded default compliance rules", zap.Int("rule_count", 5))
}

// periodicReporting generates periodic compliance reports
func (c *UnifiedComplianceEngine) periodicReporting(ctx context.Context) {
	ticker := time.NewTicker(c.config.ReportingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.generatePeriodicReports()
		case <-ctx.Done():
			return
		case <-c.stopChannel:
			return
		}
	}
}

// generatePeriodicReports generates periodic reports
func (c *UnifiedComplianceEngine) generatePeriodicReports() {
	c.logger.Info("Generating periodic compliance reports")

	// Generate daily report
	report := c.reportGenerator.GenerateReport(ReportTypeDaily, time.Now().AddDate(0, 0, -1), time.Now())
	if report != nil {
		atomic.AddInt64(&c.metrics.ReportsGenerated, 1)
		c.logger.Info("Generated daily compliance report", zap.String("report_id", report.ID))
	}

	// Generate weekly report on Sundays
	if time.Now().Weekday() == time.Sunday {
		weeklyReport := c.reportGenerator.GenerateReport(ReportTypeWeekly, time.Now().AddDate(0, 0, -7), time.Now())
		if weeklyReport != nil {
			atomic.AddInt64(&c.metrics.ReportsGenerated, 1)
			c.logger.Info("Generated weekly compliance report", zap.String("report_id", weeklyReport.ID))
		}
	}

	// Generate monthly report on the first day of the month
	if time.Now().Day() == 1 {
		monthlyReport := c.reportGenerator.GenerateReport(ReportTypeMonthly, time.Now().AddDate(0, -1, 0), time.Now())
		if monthlyReport != nil {
			atomic.AddInt64(&c.metrics.ReportsGenerated, 1)
			c.logger.Info("Generated monthly compliance report", zap.String("report_id", monthlyReport.ID))
		}
	}
}

// auditTrailCleanup cleans up old audit trail entries
func (c *UnifiedComplianceEngine) auditTrailCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.auditTrail.Cleanup(c.config.RetentionPeriod)
		case <-ctx.Done():
			return
		case <-c.stopChannel:
			return
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
		Title:    fmt.Sprintf("Compliance Violation: %s", violation.RuleID),
		Message:  violation.Description,
		Details: map[string]interface{}{
			"violation_id": violation.ID,
			"rule_id":      violation.RuleID,
			"user_id":      violation.UserID,
			"symbol":       violation.Symbol,
		},
		Status:    AlertStatusActive,
		CreatedAt: time.Now(),
	}

	c.alertManager.TriggerAlert(alert)
	atomic.AddInt64(&c.metrics.AlertsTriggered, 1)
}

// updateMetrics updates compliance metrics
func (c *UnifiedComplianceEngine) updateMetrics(checkTime time.Duration) {
	// Simple moving average for check time
	if c.metrics.AverageCheckTime == 0 {
		c.metrics.AverageCheckTime = checkTime
	} else {
		c.metrics.AverageCheckTime = (c.metrics.AverageCheckTime + checkTime) / 2
	}
	c.metrics.LastUpdateTime = time.Now()
}

// GetMetrics returns current compliance metrics
func (c *UnifiedComplianceEngine) GetMetrics() *ComplianceMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Return a copy of metrics to avoid race conditions
	metricsCopy := *c.metrics
	return &metricsCopy
}

// GetViolations returns all violations from the rule engine
func (c *UnifiedComplianceEngine) GetViolations() []ComplianceViolation {
	return c.ruleEngine.GetViolations()
}

// GetReports returns all generated reports
func (c *UnifiedComplianceEngine) GetReports() []ComplianceReport {
	return c.reportGenerator.GetReports()
}

// GetAuditEntries returns audit trail entries
func (c *UnifiedComplianceEngine) GetAuditEntries(limit int) []AuditEntry {
	return c.auditTrail.GetEntries(limit)
}

// GetAlerts returns all alerts
func (c *UnifiedComplianceEngine) GetAlerts() []ComplianceAlert {
	return c.alertManager.GetAlerts()
}
