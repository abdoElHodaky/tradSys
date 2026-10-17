// Package compliance provides centralized compliance types for TradSys v3
// This package follows KISS (Keep It Simple, Stupid) and DRY (Don't Repeat Yourself) principles
// by centralizing all compliance-related type definitions in one location for easy reuse
// by all packages in the system.
package compliance

import (
	"time"
)

// ============================================================================
// Core Compliance Types
// ============================================================================

// ComplianceRule defines a compliance rule for regulatory validation
type ComplianceRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Regulation  string                 `json:"regulation"`
	RuleType    RuleType               `json:"rule_type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Enabled     bool                   `json:"enabled"`
	Severity    ViolationSeverity      `json:"severity"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ComplianceRuleType is an alias for RuleType for backward compatibility
type ComplianceRuleType = RuleType

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

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name,omitempty"`
	OrderID     string                 `json:"order_id,omitempty"`
	UserID      string                 `json:"user_id"`
	Symbol      string                 `json:"symbol,omitempty"`
	Severity    ViolationSeverity      `json:"severity"`
	Description string                 `json:"description"`
	Details     map[string]any         `json:"details"`
	Status      ViolationStatus        `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	DetectedAt  time.Time              `json:"detected_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// ============================================================================
// Reporting Types
// ============================================================================

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

// ReportStatus defines status of reports
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusGenerated ReportStatus = "generated"
	ReportStatusSubmitted ReportStatus = "submitted"
	ReportStatusFailed    ReportStatus = "failed"
)

// ComplianceReport represents a regulatory report
type ComplianceReport struct {
	ID          string              `json:"id"`
	Type        ReportType          `json:"type"`
	Regulation  string              `json:"regulation"`
	Period      ReportPeriod        `json:"period"`
	Data        map[string]any      `json:"data"`
	Status      ReportStatus        `json:"status"`
	GeneratedAt time.Time           `json:"generated_at"`
	SubmittedAt *time.Time         `json:"submitted_at,omitempty"`
	FilePath    string              `json:"file_path,omitempty"`
}

// ReportPeriod defines the period for a report
type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

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

// ============================================================================
// Audit Types
// ============================================================================

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

// AuditEntry represents an audit trail entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	EventType AuditEventType         `json:"event_type"`
	UserID    string                 `json:"user_id"`
	OrderID   string                 `json:"order_id,omitempty"`
	Symbol    string                 `json:"symbol,omitempty"`
	Action    string                 `json:"action"`
	Details   map[string]any         `json:"details"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ============================================================================
// Alert Types
// ============================================================================

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

// ComplianceAlert represents a compliance alert
type ComplianceAlert struct {
	ID             string                 `json:"id"`
	Type           AlertType              `json:"type"`
	Severity       ViolationSeverity      `json:"severity"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Details        map[string]any         `json:"details"`
	Status         AlertStatus            `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
}

// AlertHandler defines the interface for alert handlers
type AlertHandler interface {
	HandleAlert(alert *ComplianceAlert) error
}

// ============================================================================
// Compliance Result Types
// ============================================================================

// ComplianceResult represents the result of a compliance check
type ComplianceResult struct {
	OrderID    string                `json:"order_id"`
	UserID     string                `json:"user_id"`
	Passed     bool                  `json:"passed"`
	Violations []ComplianceViolation `json:"violations,omitempty"`
	CheckedAt  time.Time             `json:"checked_at"`
	CheckTime  time.Duration         `json:"check_time"`
}

// ============================================================================
// KYC and Regulatory Types
// ============================================================================

// KYCRequirements defines KYC (Know Your Customer) requirements
type KYCRequirements struct {
	RequiredDocuments []string
	VerificationLevel int
	RenewalPeriod     time.Duration
}

// ReportingRules defines reporting requirements
type ReportingRules struct {
	DailyReports   []string
	MonthlyReports []string
	AnnualReports  []string
}

// ============================================================================
// Compliance Configuration Types
// ============================================================================

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

// ============================================================================
// Violation and Warning Aliases
// ============================================================================

// Violation is an alias for ComplianceViolation for backward compatibility
type Violation = ComplianceViolation

// Warning is an alias for ComplianceWarning for backward compatibility
type Warning = ComplianceWarning
