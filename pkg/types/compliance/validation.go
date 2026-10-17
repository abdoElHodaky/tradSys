// Package compliance provides centralized compliance types for TradSys v3
// Validation types for compliance checking and rule evaluation
package compliance

import (
	"context"
	"time"
)

// ============================================================================
// Validation Types
// ============================================================================

// Rule represents a compliance rule with validation logic
type Rule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        RuleType               `json:"type"`
	Regulation  RegulationType         `json:"regulation"`
	Parameters  map[string]any         `json:"parameters"`
	Enabled     bool                   `json:"enabled"`
	Severity    ViolationSeverity      `json:"severity"`
	Validator   func(context.Context, *ValidationRequest) (*ValidationResult, error) `json:"-"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// RuleType defines types of compliance rules
type RuleType string

const (
	// Validation-specific rule types
	RuleTypeOrderValidation      RuleType = "order_validation"
	RuleTypeAMLCheck             RuleType = "aml_check"
	RuleTypeKYCVerification      RuleType = "kyc_verification"
	RuleTypeShariaCompliance     RuleType = "sharia_compliance"
	RuleTypeReportingRequirement RuleType = "reporting_requirement"

	// Compliance rule types
	RuleTypePositionLimit      RuleType = "position_limit"
	RuleTypeOrderSize          RuleType = "order_size"
	RuleTypeTradingHours       RuleType = "trading_hours"
	RuleTypeMarketManipulation RuleType = "market_manipulation"
	RuleTypeInsiderTrading     RuleType = "insider_trading"
	RuleTypeRiskLimit          RuleType = "risk_limit"
	RuleTypeReporting          RuleType = "reporting"
)

// RegulationType defines regulatory frameworks
type RegulationType string

const (
	RegulationSEC              RegulationType = "sec"           // US Securities and Exchange Commission
	RegulationMiFID            RegulationType = "mifid"         // EU Markets in Financial Instruments Directive
	RegulationSCA              RegulationType = "sca"            // UAE Securities and Commodities Authority
	RegulationADGM             RegulationType = "adgm"           // Abu Dhabi Global Market
	RegulationDIFC             RegulationType = "difc"          // Dubai International Financial Centre
	RegulationSharia           RegulationType = "sharia"        // Islamic finance compliance
	RegulationFATCA            RegulationType = "fatca"         // Foreign Account Tax Compliance Act
	RegulationEMIR             RegulationType = "emir"          // European Market Infrastructure Regulation
	RegulationAMLGeneral       RegulationType = "aml_general"
)

// ValidationType defines types of validation
type ValidationType string

const (
	ValidationTypeOrder    ValidationType = "order"
	ValidationTypeUser     ValidationType = "user"
	ValidationTypeTrade    ValidationType = "trade"
	ValidationTypePosition ValidationType = "position"
	ValidationTypeAccount  ValidationType = "account"
)

// ValidationRequest represents a compliance validation request
type ValidationRequest struct {
	Type       ValidationType           `json:"type"`
	UserID     string                   `json:"user_id"`
	OrderData  *OrderValidationData     `json:"order_data,omitempty"`
	UserData   *UserValidationData      `json:"user_data,omitempty"`
	TradeData  *TradeValidationData     `json:"trade_data,omitempty"`
	Context    map[string]any           `json:"context,omitempty"`
	Timestamp  time.Time                `json:"timestamp"`
}

// OrderValidationData contains order-specific validation data
type OrderValidationData struct {
	OrderID   string  `json:"order_id"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Quantity  float64 `json:"quantity"`
	Price     float64 `json:"price"`
	OrderType string  `json:"order_type"`
	UserID    string  `json:"user_id"`
}

// UserValidationData contains user-specific validation data
type UserValidationData struct {
	UserID       string                 `json:"user_id"`
	AccountType  string                 `json:"account_type"`
	Jurisdiction string                 `json:"jurisdiction"`
	KYCStatus    string                 `json:"kyc_status"`
	AMLStatus    string                 `json:"aml_status"`
	Metadata     map[string]any         `json:"metadata,omitempty"`
}

// TradeValidationData contains trade-specific validation data
type TradeValidationData struct {
	TradeID    string    `json:"trade_id"`
	Symbol     string    `json:"symbol"`
	Quantity   float64   `json:"quantity"`
	Price      float64   `json:"price"`
	Timestamp  time.Time `json:"timestamp"`
	BuyUserID  string    `json:"buy_user_id"`
	SellUserID string    `json:"sell_user_id"`
}

// ValidationResult represents the result of a compliance validation
type ValidationResult struct {
	Passed     bool                    `json:"passed"`
	Violations []*ComplianceViolation  `json:"violations,omitempty"`
	Warnings   []*ComplianceWarning    `json:"warnings,omitempty"`
	Score      float64                 `json:"score"`
	Details    map[string]any          `json:"details,omitempty"`
	ValidatedAt time.Time              `json:"validated_at"`
}

// ComplianceWarning represents a compliance warning
type ComplianceWarning struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Description string                 `json:"description"`
	Details     map[string]any         `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ============================================================================
// Error Types
// ============================================================================

// RuleNotFoundError indicates a rule was not found
type RuleNotFoundError struct {
	RuleID string
}

func (e *RuleNotFoundError) Error() string {
	return "rule not found: " + e.RuleID
}

// ErrInvalidRuleID indicates an invalid rule ID
type ErrInvalidRuleID struct {
	ID string
}

func (e *ErrInvalidRuleID) Error() string {
	return "invalid rule ID: " + e.ID
}

// ErrMissingRuleValidator indicates a rule is missing its validator function
type ErrMissingRuleValidator struct {
	RuleID string
}

func (e *ErrMissingRuleValidator) Error() string {
	return "missing rule validator for rule: " + e.RuleID
}

// ErrRuleNotFound indicates a rule was not found
type ErrRuleNotFound struct {
	RuleID string
}

func (e *ErrRuleNotFound) Error() string {
	return "rule not found: " + e.RuleID
}

// InvalidRuleError indicates an invalid rule definition
type InvalidRuleError struct {
	Field   string
	Message string
}

func (e *InvalidRuleError) Error() string {
	return "invalid rule: " + e.Field + " - " + e.Message
}
