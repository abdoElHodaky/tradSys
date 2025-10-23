package compliance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
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

// ComplianceConfig contains configuration for compliance engine
type ComplianceConfig struct {
	EnabledRegulations   []string      `json:"enabled_regulations"`
	ReportingEnabled     bool          `json:"reporting_enabled"`
	AuditTrailEnabled    bool          `json:"audit_trail_enabled"`
	AlertingEnabled      bool          `json:"alerting_enabled"`
	ReportingInterval    time.Duration `json:"reporting_interval"`
	RetentionPeriod      time.Duration `json:"retention_period"`
	MaxViolationsPerDay  int           `json:"max_violations_per_day"`
	AutoReportingEnabled bool          `json:"auto_reporting_enabled"`
}

// ComplianceMetrics tracks compliance performance
type ComplianceMetrics struct {
	TotalChecks        int64         `json:"total_checks"`
	PassedChecks       int64         `json:"passed_checks"`
	FailedChecks       int64         `json:"failed_checks"`
	ViolationsDetected int64         `json:"violations_detected"`
	ReportsGenerated   int64         `json:"reports_generated"`
	AlertsTriggered    int64         `json:"alerts_triggered"`
	AverageCheckTime   time.Duration `json:"average_check_time"`
	LastUpdateTime     time.Time     `json:"last_update_time"`
}

// RuleEngine manages compliance rules and checks
type RuleEngine struct {
	rules      map[string]ComplianceRule
	violations []ComplianceViolation
	logger     *zap.Logger
	mu         sync.RWMutex
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

// ComplianceRule defines a compliance rule
type ComplianceRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Regulation  string                 `json:"regulation"`
	RuleType    ComplianceRuleType     `json:"rule_type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Enabled     bool                   `json:"enabled"`
	Severity    ViolationSeverity      `json:"severity"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ComplianceRuleType defines types of compliance rules
type ComplianceRuleType string

const (
	RuleTypePositionLimit      ComplianceRuleType = "position_limit"
	RuleTypeOrderSize          ComplianceRuleType = "order_size"
	RuleTypeTradingHours       ComplianceRuleType = "trading_hours"
	RuleTypeMarketManipulation ComplianceRuleType = "market_manipulation"
	RuleTypeInsiderTrading     ComplianceRuleType = "insider_trading"
	RuleTypeRiskLimit          ComplianceRuleType = "risk_limit"
	RuleTypeReporting          ComplianceRuleType = "reporting"
)

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	OrderID     string                 `json:"order_id,omitempty"`
	UserID      string                 `json:"user_id"`
	Symbol      string                 `json:"symbol,omitempty"`
	Severity    ViolationSeverity      `json:"severity"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	Status      ViolationStatus        `json:"status"`
	DetectedAt  time.Time              `json:"detected_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// ViolationSeverity defines severity levels for violations
type ViolationSeverity string

const (
	SeverityLow      ViolationSeverity = "low"
	SeverityMedium   ViolationSeverity = "medium"
	SeverityHigh     ViolationSeverity = "high"
	SeverityCritical ViolationSeverity = "critical"
)

// ViolationStatus defines status of violations
type ViolationStatus string

const (
	ViolationStatusOpen          ViolationStatus = "open"
	ViolationStatusInvestigating ViolationStatus = "investigating"
	ViolationStatusResolved      ViolationStatus = "resolved"
	ViolationStatusFalsePositive ViolationStatus = "false_positive"
)

// ComplianceReport represents a regulatory report
type ComplianceReport struct {
	ID          string                 `json:"id"`
	Type        ReportType             `json:"type"`
	Regulation  string                 `json:"regulation"`
	Period      ReportPeriod           `json:"period"`
	Data        map[string]interface{} `json:"data"`
	Status      ReportStatus           `json:"status"`
	GeneratedAt time.Time              `json:"generated_at"`
	SubmittedAt *time.Time             `json:"submitted_at,omitempty"`
	FilePath    string                 `json:"file_path,omitempty"`
}

// ReportType defines types of compliance reports
type ReportType string

const (
	ReportTypeDaily     ReportType = "daily"
	ReportTypeWeekly    ReportType = "weekly"
	ReportTypeMonthly   ReportType = "monthly"
	ReportTypeQuarterly ReportType = "quarterly"
	ReportTypeAnnual    ReportType = "annual"
	ReportTypeAdHoc     ReportType = "adhoc"
)

// ReportPeriod defines the period for a report
type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// ReportStatus defines status of reports
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusGenerated ReportStatus = "generated"
	ReportStatusSubmitted ReportStatus = "submitted"
	ReportStatusFailed    ReportStatus = "failed"
)

// ReportTemplate defines a template for generating reports
type ReportTemplate struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       ReportType `json:"type"`
	Regulation string     `json:"regulation"`
	Fields     []string   `json:"fields"`
	Format     string     `json:"format"`
	Template   string     `json:"template"`
	CreatedAt  time.Time  `json:"created_at"`
}

// AuditEntry represents an audit trail entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	EventType AuditEventType         `json:"event_type"`
	UserID    string                 `json:"user_id"`
	OrderID   string                 `json:"order_id,omitempty"`
	Symbol    string                 `json:"symbol,omitempty"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// AuditEventType defines types of audit events
type AuditEventType string

const (
	AuditEventOrderSubmit AuditEventType = "order_submit"
	AuditEventOrderCancel AuditEventType = "order_cancel"
	AuditEventOrderModify AuditEventType = "order_modify"
	AuditEventTrade       AuditEventType = "trade"
	AuditEventLogin       AuditEventType = "login"
	AuditEventLogout      AuditEventType = "logout"
	AuditEventRiskBreach  AuditEventType = "risk_breach"
	AuditEventCompliance  AuditEventType = "compliance"
)

// ComplianceAlert represents a compliance alert
type ComplianceAlert struct {
	ID             string                 `json:"id"`
	Type           AlertType              `json:"type"`
	Severity       ViolationSeverity      `json:"severity"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Details        map[string]interface{} `json:"details"`
	Status         AlertStatus            `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
}

// AlertType defines types of compliance alerts
type AlertType string

const (
	AlertTypeViolation   AlertType = "violation"
	AlertTypeRiskBreach  AlertType = "risk_breach"
	AlertTypeSystemError AlertType = "system_error"
	AlertTypeReporting   AlertType = "reporting"
)

// AlertStatus defines status of alerts
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// AlertHandler defines the interface for alert handlers
type AlertHandler interface {
	HandleAlert(alert *ComplianceAlert) error
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

// ComplianceResult represents the result of a compliance check
type ComplianceResult struct {
	OrderID    string                `json:"order_id"`
	UserID     string                `json:"user_id"`
	Passed     bool                  `json:"passed"`
	Violations []ComplianceViolation `json:"violations,omitempty"`
	CheckedAt  time.Time             `json:"checked_at"`
	CheckTime  time.Duration         `json:"check_time"`
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
					"end_hour":     endHour,
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
