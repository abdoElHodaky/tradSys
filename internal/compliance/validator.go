package compliance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Validator handles compliance validation and rule checking
type Validator struct {
	rules      map[string]*Rule
	ruleEngine *RuleEngine
	logger     *zap.Logger
	mu         sync.RWMutex
}

// Rule represents a compliance rule
type Rule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        RuleType               `json:"type"`
	Regulation  RegulationType         `json:"regulation"`
	Parameters  map[string]interface{} `json:"parameters"`
	Enabled     bool                   `json:"enabled"`
	Severity    ViolationSeverity      `json:"severity"`
	Validator   func(context.Context, *ValidationRequest) (*ValidationResult, error) `json:"-"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Use types from unified_compliance.go
type RuleType = ComplianceRuleType

const (
	RuleTypeOrderValidation     = "order_validation"
	RuleTypeAMLCheck            = "aml_check"
	RuleTypeKYCVerification     = "kyc_verification"
	RuleTypeShariaCompliance    = "sharia_compliance"
	RuleTypeReportingRequirement = "reporting_requirement"
)

// RegulationType defines regulatory frameworks
type RegulationType string

const (
	RegulationSEC     RegulationType = "sec"     // US Securities and Exchange Commission
	RegulationMiFID   RegulationType = "mifid"   // EU Markets in Financial Instruments Directive
	RegulationSCA     RegulationType = "sca"     // UAE Securities and Commodities Authority
	RegulationADGM    RegulationType = "adgm"    // Abu Dhabi Global Market
	RegulationDIFC    RegulationType = "difc"    // Dubai International Financial Centre
	RegulationSharia  RegulationType = "sharia"  // Islamic finance compliance
	RegulationFATCA   RegulationType = "fatca"   // Foreign Account Tax Compliance Act
	RegulationEMIR    RegulationType = "emir"    // European Market Infrastructure Regulation
)

// Use ViolationSeverity from unified_compliance.go

// ValidationRequest represents a compliance validation request
type ValidationRequest struct {
	Type       ValidationType         `json:"type"`
	UserID     string                 `json:"user_id"`
	OrderData  *OrderValidationData   `json:"order_data,omitempty"`
	UserData   *UserValidationData    `json:"user_data,omitempty"`
	TradeData  *TradeValidationData   `json:"trade_data,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ValidationType defines types of validation
type ValidationType string

const (
	ValidationTypeOrder    ValidationType = "order"
	ValidationTypeUser     ValidationType = "user"
	ValidationTypeTrade    ValidationType = "trade"
	ValidationTypePosition ValidationType = "position"
	ValidationTypeAccount  ValidationType = "account"
)

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
	UserID      string                 `json:"user_id"`
	AccountType string                 `json:"account_type"`
	Jurisdiction string                `json:"jurisdiction"`
	KYCStatus   string                 `json:"kyc_status"`
	AMLStatus   string                 `json:"aml_status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TradeValidationData contains trade-specific validation data
type TradeValidationData struct {
	TradeID     string    `json:"trade_id"`
	Symbol      string    `json:"symbol"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Timestamp   time.Time `json:"timestamp"`
	BuyUserID   string    `json:"buy_user_id"`
	SellUserID  string    `json:"sell_user_id"`
}

// ValidationResult represents the result of a compliance validation
type ValidationResult struct {
	Passed      bool                   `json:"passed"`
	Violations  []*Violation           `json:"violations,omitempty"`
	Warnings    []*Warning             `json:"warnings,omitempty"`
	Score       float64                `json:"score"`
	Details     map[string]interface{} `json:"details,omitempty"`
	ValidatedAt time.Time              `json:"validated_at"`
}

// Violation represents a compliance violation
type Violation struct {
	RuleID      string            `json:"rule_id"`
	RuleName    string            `json:"rule_name"`
	Severity    ViolationSeverity `json:"severity"`
	Description string            `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Warning represents a compliance warning
type Warning struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewValidator creates a new compliance validator
func NewValidator(logger *zap.Logger) *Validator {
	validator := &Validator{
		rules:      make(map[string]*Rule),
		ruleEngine: NewRuleEngine(logger),
		logger:     logger,
	}
	
	// Initialize default rules
	validator.initializeDefaultRules()
	
	return validator
}

// ValidateCompliance validates a request against compliance rules
func (v *Validator) ValidateCompliance(ctx context.Context, request *ValidationRequest) (*ValidationResult, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	result := &ValidationResult{
		Passed:      true,
		Violations:  make([]*Violation, 0),
		Warnings:    make([]*Warning, 0),
		Score:       100.0,
		Details:     make(map[string]interface{}),
		ValidatedAt: time.Now(),
	}
	
	// Run applicable rules
	for _, rule := range v.rules {
		if !rule.Enabled {
			continue
		}
		
		// Check if rule applies to this validation type
		if !v.ruleApplies(rule, request) {
			continue
		}
		
		// Execute rule validation
		ruleResult, err := rule.Validator(ctx, request)
		if err != nil {
			v.logger.Error("Rule validation failed",
				zap.String("rule_id", rule.ID),
				zap.Error(err))
			continue
		}
		
		// Process rule result
		if !ruleResult.Passed {
			result.Passed = false
			result.Violations = append(result.Violations, ruleResult.Violations...)
			
			// Adjust score based on severity using lookup table
			for _, violation := range ruleResult.Violations {
				result.Score -= GetSeverityScore(violation.Severity)
			}
		}
		
		// Add warnings
		result.Warnings = append(result.Warnings, ruleResult.Warnings...)
	}
	
	// Ensure score doesn't go below 0
	if result.Score < 0 {
		result.Score = 0
	}
	
	v.logger.Debug("Compliance validation completed",
		zap.String("type", string(request.Type)),
		zap.String("user_id", request.UserID),
		zap.Bool("passed", result.Passed),
		zap.Float64("score", result.Score),
		zap.Int("violations", len(result.Violations)),
		zap.Int("warnings", len(result.Warnings)))
	
	return result, nil
}

// AddRule adds a new compliance rule
func (v *Validator) AddRule(rule *Rule) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	if rule.ID == "" {
		return ErrInvalidRuleID
	}
	
	if rule.Validator == nil {
		return ErrMissingRuleValidator
	}
	
	rule.UpdatedAt = time.Now()
	v.rules[rule.ID] = rule
	
	v.logger.Info("Compliance rule added",
		zap.String("rule_id", rule.ID),
		zap.String("rule_name", rule.Name),
		zap.String("regulation", string(rule.Regulation)))
	
	return nil
}

// RemoveRule removes a compliance rule
func (v *Validator) RemoveRule(ruleID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	if _, exists := v.rules[ruleID]; !exists {
		return ErrRuleNotFound
	}
	
	delete(v.rules, ruleID)
	
	v.logger.Info("Compliance rule removed",
		zap.String("rule_id", ruleID))
	
	return nil
}

// EnableRule enables a compliance rule
func (v *Validator) EnableRule(ruleID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	rule, exists := v.rules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}
	
	rule.Enabled = true
	rule.UpdatedAt = time.Now()
	
	v.logger.Info("Compliance rule enabled",
		zap.String("rule_id", ruleID))
	
	return nil
}

// DisableRule disables a compliance rule
func (v *Validator) DisableRule(ruleID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	rule, exists := v.rules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}
	
	rule.Enabled = false
	rule.UpdatedAt = time.Now()
	
	v.logger.Info("Compliance rule disabled",
		zap.String("rule_id", ruleID))
	
	return nil
}

// GetRules returns all compliance rules
func (v *Validator) GetRules() map[string]*Rule {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	rules := make(map[string]*Rule)
	for id, rule := range v.rules {
		rules[id] = rule
	}
	
	return rules
}

// ruleApplies checks if a rule applies to a validation request
func (v *Validator) ruleApplies(rule *Rule, request *ValidationRequest) bool {
	switch rule.Type {
	case RuleTypeOrderValidation:
		return request.Type == ValidationTypeOrder && request.OrderData != nil
	case RuleTypePositionLimit, RuleTypeRiskLimit:
		return request.Type == ValidationTypePosition || request.Type == ValidationTypeOrder
	case RuleTypeKYCVerification, RuleTypeAMLCheck:
		return request.Type == ValidationTypeUser && request.UserData != nil
	case RuleTypeMarketManipulation, RuleTypeInsiderTrading:
		return request.Type == ValidationTypeTrade && request.TradeData != nil
	case RuleTypeShariaCompliance:
		return true // Sharia compliance applies to all types
	default:
		return true
	}
}

// initializeDefaultRules initializes default compliance rules
func (v *Validator) initializeDefaultRules() {
	// Order size limit rule
	v.AddRule(&Rule{
		ID:          "order_size_limit",
		Name:        "Order Size Limit",
		Description: "Validates order size against maximum limits",
		Type:        RuleTypeOrderValidation,
		Regulation:  RegulationSEC,
		Enabled:     true,
		Severity:    SeverityHigh,
		Validator:   v.validateOrderSizeLimit,
		CreatedAt:   time.Now(),
	})
	
	// KYC verification rule
	v.AddRule(&Rule{
		ID:          "kyc_verification",
		Name:        "KYC Verification",
		Description: "Validates user KYC status",
		Type:        RuleTypeKYCVerification,
		Regulation:  RegulationAMLGeneral,
		Enabled:     true,
		Severity:    SeverityCritical,
		Validator:   v.validateKYCStatus,
		CreatedAt:   time.Now(),
	})
	
	// Sharia compliance rule
	v.AddRule(&Rule{
		ID:          "sharia_compliance",
		Name:        "Sharia Compliance",
		Description: "Validates Islamic finance compliance",
		Type:        RuleTypeShariaCompliance,
		Regulation:  RegulationSharia,
		Enabled:     true,
		Severity:    SeverityHigh,
		Validator:   v.validateShariaCompliance,
		CreatedAt:   time.Now(),
	})
}

// Rule validators

// validateOrderSizeLimit validates order size limits
func (v *Validator) validateOrderSizeLimit(ctx context.Context, request *ValidationRequest) (*ValidationResult, error) {
	result := &ValidationResult{
		Passed:      true,
		Violations:  make([]*Violation, 0),
		Warnings:    make([]*Warning, 0),
		ValidatedAt: time.Now(),
	}
	
	if request.OrderData == nil {
		return result, nil
	}
	
	// Check maximum order size (example: $1M)
	maxOrderValue := 1000000.0
	orderValue := request.OrderData.Quantity * request.OrderData.Price
	
	if orderValue > maxOrderValue {
		result.Passed = false
		result.Violations = append(result.Violations, &Violation{
			RuleID:      "order_size_limit",
			RuleName:    "Order Size Limit",
			Severity:    SeverityHigh,
			Description: fmt.Sprintf("Order value %.2f exceeds maximum limit %.2f", orderValue, maxOrderValue),
			Timestamp:   time.Now(),
		})
	}
	
	return result, nil
}

// validateKYCStatus validates user KYC status
func (v *Validator) validateKYCStatus(ctx context.Context, request *ValidationRequest) (*ValidationResult, error) {
	result := &ValidationResult{
		Passed:      true,
		Violations:  make([]*Violation, 0),
		Warnings:    make([]*Warning, 0),
		ValidatedAt: time.Now(),
	}
	
	if request.UserData == nil {
		return result, nil
	}
	
	// Check KYC status
	if request.UserData.KYCStatus != "verified" {
		result.Passed = false
		result.Violations = append(result.Violations, &Violation{
			RuleID:      "kyc_verification",
			RuleName:    "KYC Verification",
			Severity:    SeverityCritical,
			Description: "User KYC status is not verified",
			Details:     map[string]interface{}{"kyc_status": request.UserData.KYCStatus},
			Timestamp:   time.Now(),
		})
	}
	
	return result, nil
}

// validateShariaCompliance validates Islamic finance compliance
func (v *Validator) validateShariaCompliance(ctx context.Context, request *ValidationRequest) (*ValidationResult, error) {
	result := &ValidationResult{
		Passed:      true,
		Violations:  make([]*Violation, 0),
		Warnings:    make([]*Warning, 0),
		ValidatedAt: time.Now(),
	}
	
	// Check if order involves prohibited instruments
	if request.OrderData != nil {
		if v.isProhibitedInstrument(request.OrderData.Symbol) {
			result.Passed = false
			result.Violations = append(result.Violations, &Violation{
				RuleID:      "sharia_compliance",
				RuleName:    "Sharia Compliance",
				Severity:    SeverityHigh,
				Description: "Instrument is not Sharia compliant",
				Details:     map[string]interface{}{"symbol": request.OrderData.Symbol},
				Timestamp:   time.Now(),
			})
		}
	}
	
	return result, nil
}

// isProhibitedInstrument checks if an instrument is prohibited under Sharia law
func (v *Validator) isProhibitedInstrument(symbol string) bool {
	// Simplified check - in production would use comprehensive database
	prohibitedSectors := []string{"BANK", "INSURANCE", "ALCOHOL", "GAMBLING", "TOBACCO"}
	
	for _, sector := range prohibitedSectors {
		if contains(symbol, sector) {
			return true
		}
	}
	
	return false
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// Additional regulation types
const (
	RegulationAMLGeneral RegulationType = "aml_general"
)

// Use RuleEngine from unified_compliance.go

// Error definitions
var (
	ErrInvalidRuleID        = errors.New("invalid rule ID")
	ErrMissingRuleValidator = errors.New("missing rule validator")
	ErrRuleNotFound         = errors.New("rule not found")
)
