package compliance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	compliance_types "github.com/abdoElHodaky/tradSys/pkg/types/compliance"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// Type aliases for backward compatibility - these point to the centralized types from pkg/types/compliance
type ComplianceConfig = compliance_types.ComplianceConfig
type ComplianceMetrics = compliance_types.ComplianceMetrics
type ComplianceRule = compliance_types.ComplianceRule
type ComplianceRuleType = compliance_types.ComplianceRuleType
type ComplianceViolation = compliance_types.ComplianceViolation
type ViolationSeverity = compliance_types.ViolationSeverity
type ViolationStatus = compliance_types.ViolationStatus
type ComplianceReport = compliance_types.ComplianceReport
type ReportType = compliance_types.ReportType
type ReportPeriod = compliance_types.ReportPeriod
type ReportStatus = compliance_types.ReportStatus
type ReportTemplate = compliance_types.ReportTemplate
type AuditEntry = compliance_types.AuditEntry
type AuditEventType = compliance_types.AuditEventType
type ComplianceAlert = compliance_types.ComplianceAlert
type AlertType = compliance_types.AlertType
type AlertStatus = compliance_types.AlertStatus
type AlertHandler = compliance_types.AlertHandler
type ComplianceResult = compliance_types.ComplianceResult

// Constants for rules, severity, and other compliance-related values
const (
	// Rule types - unified from RuleType in validation.go
	RuleTypeOrderValidation      RuleType = "order_validation"
	RuleTypeAMLCheck             RuleType = "aml_check"
	RuleTypeKYCVerification      RuleType = "kyc_verification"
	RuleTypeShariaCompliance     RuleType = "sharia_compliance"
	RuleTypeReportingRequirement RuleType = "reporting_requirement"

	RuleTypePositionLimit      RuleType = "position_limit"
	RuleTypeOrderSize          RuleType = "order_size"
	RuleTypeTradingHours       RuleType = "trading_hours"
	RuleTypeMarketManipulation RuleType = "market_manipulation"
	RuleTypeInsiderTrading     RuleType = "insider_trading"
	RuleTypeRiskLimit          RuleType = "risk_limit"
	RuleTypeReporting          RuleType = "reporting"

	// Severity levels
	SeverityLow      ViolationSeverity = compliance_types.SeverityLow
	SeverityMedium   ViolationSeverity = compliance_types.SeverityMedium
	SeverityHigh     ViolationSeverity = compliance_types.SeverityHigh
	SeverityCritical ViolationSeverity = compliance_types.SeverityCritical

	// Audit event types
	AuditEventCompliance     AuditEventType = compliance_types.AuditEventCompliance
	AuditEventOrderSubmit    AuditEventType = compliance_types.AuditEventOrderSubmit
	AuditEventOrderCancel    AuditEventType = compliance_types.AuditEventOrderCancel
	AuditEventOrderModify    AuditEventType = compliance_types.AuditEventOrderModify
	AuditEventTrade          AuditEventType = compliance_types.AuditEventTrade
	AuditEventLogin          AuditEventType = compliance_types.AuditEventLogin
	AuditEventLogout         AuditEventType = compliance_types.AuditEventLogout
	AuditEventRiskBreach     AuditEventType = compliance_types.AuditEventRiskBreach

	// Alert types and statuses
	AlertTypeViolation   AlertType = compliance_types.AlertTypeViolation
	AlertTypeRiskBreach  AlertType = compliance_types.AlertTypeRiskBreach
	AlertTypeSystemError AlertType = compliance_types.AlertTypeSystemError
	AlertTypeReporting   AlertType = compliance_types.AlertTypeReporting

	AlertStatusActive       AlertStatus = compliance_types.AlertStatusActive
	AlertStatusAcknowledged AlertStatus = compliance_types.AlertStatusAcknowledged
	AlertStatusResolved     AlertStatus = compliance_types.AlertStatusResolved

	// Report types and statuses
	ReportTypeDaily     ReportType = compliance_types.ReportTypeDaily
	ReportTypeWeekly    ReportType = compliance_types.ReportTypeWeekly
	ReportTypeMonthly   ReportType = compliance_types.ReportTypeMonthly
	ReportTypeQuarterly ReportType = compliance_types.ReportTypeQuarterly
	ReportTypeAnnual    ReportType = compliance_types.ReportTypeAnnual
	ReportTypeAdHoc     ReportType = compliance_types.ReportTypeAdHoc

	ReportStatusPending   ReportStatus = compliance_types.ReportStatusPending
	ReportStatusGenerated ReportStatus = compliance_types.ReportStatusGenerated
	ReportStatusSubmitted ReportStatus = compliance_types.ReportStatusSubmitted
	ReportStatusFailed    ReportStatus = compliance_types.ReportStatusFailed

	// Validation types
	ValidationTypeOrder    = "order"
	ValidationTypeUser     = "user"
	ValidationTypeTrade    = "trade"
	ValidationTypePosition = "position"
	ValidationTypeAccount  = "account"

	// Violation status
	ViolationStatusOpen          ViolationStatus = compliance_types.ViolationStatusOpen
	ViolationStatusInvestigating ViolationStatus = compliance_types.ViolationStatusInvestigating
	ViolationStatusResolved      ViolationStatus = compliance_types.ViolationStatusResolved
	ViolationStatusFalsePositive ViolationStatus = compliance_types.ViolationStatusFalsePositive
)

// UnifiedComplianceEngine provides comprehensive compliance and regulatory reporting
type UnifiedComplianceEngine struct {
	config          *ComplianceConfig
	logger          *zap.Logger
	ruleEngine      *RuleEngine
	reportGenerator *ReportGenerator
	auditTrail      *AuditTrail
	alertManager    *AlertManager
	metrics         *ComplianceMetrics
	isRunning       int32
	stopChannel     chan struct{}
	mu              sync.RWMutex
}

// RuleEngine manages compliance rules and checks
type RuleEngine struct {
	rules      map[string]ComplianceRule
	violations []ComplianceViolation
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(logger *zap.Logger) *RuleEngine {
	return &RuleEngine{
		rules:      make(map[string]ComplianceRule),
		violations: make([]ComplianceViolation, 0),
		logger:     logger,
	}
}

// ReportGenerator generates regulatory reports
type ReportGenerator struct {
	templates map[string]ReportTemplate
	reports   []ComplianceReport
	logger    *zap.Logger
	mu        sync.RWMutex
}

// AuditTrail maintains comprehensive audit logs
type AuditTrail struct {
	entries    []AuditEntry
	maxEntries int
	logger     *zap.Logger
	mu         sync.RWMutex
}

// AlertManager handles compliance alerts and notifications
type AlertManager struct {
	alerts      []ComplianceAlert
	subscribers []AlertHandler
	logger      *zap.Logger
	mu          sync.RWMutex
}

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
		OrderID:   order.ID,
		UserID:    userID,
		Passed:    true,
		CheckedAt: time.Now(),
		CheckTime: time.Since(startTime),
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

	c.logger.Info("Loaded default compliance rules", zap.Int("rule_count", 3))
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
	// Simple moving average
	c.metrics.AverageCheckTime = (c.metrics.AverageCheckTime + checkTime) / 2
	c.metrics.LastUpdateTime = time.Now()
}

// GetMetrics returns current compliance metrics
func (c *UnifiedComplianceEngine) GetMetrics() *ComplianceMetrics {
	return c.metrics
}

// AddRule adds a compliance rule
func (r *RuleEngine) AddRule(rule ComplianceRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID] = rule
}

// CheckRules checks all enabled rules against an order
func (r *RuleEngine) CheckRules(order *types.Order, userID string) []ComplianceViolation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var violations []ComplianceViolation

	for _, rule := range r.rules {
		if !rule.Enabled {
			continue
		}

		if violation := r.checkRule(rule, order, userID); violation != nil {
			violations = append(violations, *violation)
		}
	}

	return violations
}

// checkRule checks a specific rule against an order
func (r *RuleEngine) checkRule(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	switch rule.RuleType {
	case RuleTypeOrderSize:
		maxSize, ok := rule.Parameters["max_order_size"].(float64)
		if ok && order.Quantity > maxSize {
			return &ComplianceViolation{
				ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
				RuleID:      rule.ID,
				OrderID:     order.ID,
				UserID:      userID,
				Symbol:      order.Symbol,
				Severity:    rule.Severity,
				Description: fmt.Sprintf("Order size %.2f exceeds maximum allowed size %.2f", order.Quantity, maxSize),
				Details: map[string]interface{}{
					"order_size": order.Quantity,
					"max_size":   maxSize,
				},
				Status:     ViolationStatusOpen,
				DetectedAt: time.Now(),
			}
		}
	case RuleTypeTradingHours:
		// Simplified trading hours check
		now := time.Now().UTC()
		startHour, _ := rule.Parameters["start_hour"].(int)
		endHour, _ := rule.Parameters["end_hour"].(int)

		if now.Hour() < startHour || now.Hour() >= endHour {
			return &ComplianceViolation{
				ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
				RuleID:      rule.ID,
				OrderID:     order.ID,
				UserID:      userID,
				Symbol:      order.Symbol,
				Severity:    rule.Severity,
				Description: fmt.Sprintf("Trading outside allowed hours (%d:00-%d:00 UTC)", startHour, endHour),
				Details: map[string]interface{}{
					"current_hour": now.Hour(),
					"start_hour":   startHour,
					"end_hour":   endHour,
				},
				Status:     ViolationStatusOpen,
				DetectedAt: time.Now(),
			}
		}
	}

	return nil
}

// GenerateReport generates a compliance report
func (g *ReportGenerator) GenerateReport(reportType ReportType, startDate, endDate time.Time) *ComplianceReport {
	g.mu.Lock()
	defer g.mu.Unlock()

	report := &ComplianceReport{
		ID:   fmt.Sprintf("report_%d", time.Now().UnixNano()),
		Type: reportType,
		Period: ReportPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Status:      ReportStatusGenerated,
		GeneratedAt: time.Now(),
		Data: map[string]interface{}{
			"period":      fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
			"generated":   time.Now(),
			"report_type": reportType,
		},
	}

	g.reports = append(g.reports, *report)
	return report
}

// AddEntry adds an entry to the audit trail
func (a *AuditTrail) AddEntry(entry *AuditEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.entries = append(a.entries, *entry)

	// Keep only the last maxEntries
	if len(a.entries) > a.maxEntries {
		a.entries = a.entries[len(a.entries)-a.maxEntries:]
	}
}

// Cleanup removes old audit entries
func (a *AuditTrail) Cleanup(retentionPeriod time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cutoff := time.Now().Add(-retentionPeriod)
	var kept []AuditEntry

	for _, entry := range a.entries {
		if entry.Timestamp.After(cutoff) {
			kept = append(kept, entry)
		}
	}

	removed := len(a.entries) - len(kept)
	a.entries = kept

	if removed > 0 {
		a.logger.Info("Cleaned up audit trail entries",
			zap.Int("removed", removed),
			zap.Int("remaining", len(kept)))
	}
}

// TriggerAlert triggers a compliance alert
func (m *AlertManager) TriggerAlert(alert *ComplianceAlert) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.alerts = append(m.alerts, *alert)

	// Notify subscribers
	for _, handler := range m.subscribers {
		if err := handler.HandleAlert(alert); err != nil {
			m.logger.Error("Alert handler failed", zap.Error(err))
		}
	}

	m.logger.Warn("Compliance alert triggered",
		zap.String("alert_id", alert.ID),
		zap.String("type", string(alert.Type)),
		zap.String("severity", string(alert.Severity)),
		zap.String("title", alert.Title))
}

// Subscribe subscribes to compliance alerts
func (m *AlertManager) Subscribe(handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, handler)
}
