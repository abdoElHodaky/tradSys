package compliance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	compliance_types "github.com/abdoElHodaky/tradSys/pkg/types/compliance"
	"go.uber.org/zap"
)

// Use types from centralized compliance package
// RuleType constants are already defined in validation.go
// ComplianceRuleType constants are already defined in types.go
type Rule = compliance_types.Rule
type RuleType = compliance_types.RuleType
type ValidationRequest = compliance_types.ValidationRequest
type ValidationType = compliance_types.ValidationType
type OrderValidationData = compliance_types.OrderValidationData
type UserValidationData = compliance_types.UserValidationData
type TradeValidationData = compliance_types.TradeValidationData
type ValidationResult = compliance_types.ValidationResult
type Violation = compliance_types.Violation
type Warning = compliance_types.Warning
type RegulationType = compliance_types.RegulationType

// Error types from validation.go
type ErrInvalidRuleID = compliance_types.ErrInvalidRuleID
type ErrMissingRuleValidator = compliance_types.ErrMissingRuleValidator
type ErrRuleNotFound = compliance_types.ErrRuleNotFound

// Validator handles compliance validation and rule checking
type Validator struct {
	rules      map[string]*Rule
	ruleEngine *RuleEngine
	logger     *zap.Logger
	mu         sync.RWMutex
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

			// Adjust score based on severity
			for _, violation := range ruleResult.Violations {
				switch violation.Severity {
				case "critical":
					result.Score -= 50
				case "high":
					result.Score -= 25
				case "medium":
					result.Score -= 10
				case "low":
					result.Score -= 5
				}
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
		return errors.New("invalid rule ID")
	}

	if rule.Validator == nil {
		return errors.New("missing rule validator")
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
		return errors.New("rule not found")
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
		return errors.New("rule not found")
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
		return errors.New("rule not found")
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
		Regulation:  "sec",
		Enabled:     true,
		Severity:    "high",
		Validator:   v.validateOrderSizeLimit,
		CreatedAt:   time.Now(),
	})

	// KYC verification rule
	v.AddRule(&Rule{
		ID:          "kyc_verification",
		Name:        "KYC Verification",
		Description: "Validates user KYC status",
		Type:        RuleTypeKYCVerification,
		Regulation:  "aml_general",
		Enabled:     true,
		Severity:    "critical",
		Validator:   v.validateKYCStatus,
		CreatedAt:   time.Now(),
	})

	// Sharia compliance rule
	v.AddRule(&Rule{
		ID:          "sharia_compliance",
		Name:        "Sharia Compliance",
		Description: "Validates Islamic finance compliance",
		Type:        RuleTypeShariaCompliance,
		Regulation:  "sharia",
		Enabled:     true,
		Severity:    "high",
		Validator:   v.validateShariaCompliance,
		CreatedAt:   time.Now(),
	})
}

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
			Severity:    "high",
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
			Severity:    "critical",
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
				Severity:    "high",
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
