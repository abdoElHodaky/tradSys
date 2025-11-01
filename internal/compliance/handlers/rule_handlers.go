// Package handlers provides compliance rule handlers using the strategy pattern.
// This replaces complex switch statements with extensible handler interfaces.
package handlers

import (
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
	"go.uber.org/zap"
)

// RuleHandler defines the interface for compliance rule handlers
type RuleHandler interface {
	// Handle processes a compliance rule against an order
	Handle(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation
	
	// GetRuleType returns the rule type this handler supports
	GetRuleType() string
	
	// GetDescription returns a description of what this handler does
	GetDescription() string
}

// ComplianceRule represents a compliance rule
type ComplianceRule struct {
	ID         string                 `json:"id"`
	RuleType   string                 `json:"rule_type"`
	Severity   string                 `json:"severity"`
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	OrderID     string                 `json:"order_id"`
	UserID      string                 `json:"user_id"`
	Symbol      string                 `json:"symbol"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	Status      string                 `json:"status"`
	DetectedAt  time.Time              `json:"detected_at"`
}

// Rule types constants
const (
	RuleTypeOrderSize    = "order_size"
	RuleTypePositionLimit = "position_limit"
	RuleTypeTradingHours = "trading_hours"
	RuleTypeRiskLimit    = "risk_limit"
	RuleTypeAML          = "aml"
	RuleTypeKYC          = "kyc"
	RuleTypeSanctions    = "sanctions"
)

// Violation status constants
const (
	ViolationStatusOpen     = "open"
	ViolationStatusResolved = "resolved"
	ViolationStatusIgnored  = "ignored"
)

// RuleDispatcher manages and dispatches rules to appropriate handlers
type RuleDispatcher struct {
	handlers map[string]RuleHandler
	logger   *zap.Logger
}

// NewRuleDispatcher creates a new rule dispatcher
func NewRuleDispatcher(logger *zap.Logger) *RuleDispatcher {
	dispatcher := &RuleDispatcher{
		handlers: make(map[string]RuleHandler),
		logger:   logger,
	}
	
	// Register default handlers
	dispatcher.RegisterHandler(&OrderSizeHandler{logger: logger})
	dispatcher.RegisterHandler(&PositionLimitHandler{logger: logger})
	dispatcher.RegisterHandler(&TradingHoursHandler{logger: logger})
	dispatcher.RegisterHandler(&RiskLimitHandler{logger: logger})
	dispatcher.RegisterHandler(&AMLHandler{logger: logger})
	dispatcher.RegisterHandler(&KYCHandler{logger: logger})
	dispatcher.RegisterHandler(&SanctionsHandler{logger: logger})
	
	return dispatcher
}

// RegisterHandler registers a new rule handler
func (d *RuleDispatcher) RegisterHandler(handler RuleHandler) {
	d.handlers[handler.GetRuleType()] = handler
	d.logger.Info("Registered compliance rule handler",
		zap.String("rule_type", handler.GetRuleType()),
		zap.String("description", handler.GetDescription()),
	)
}

// Dispatch processes a rule using the appropriate handler
func (d *RuleDispatcher) Dispatch(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	if !rule.Enabled {
		return nil
	}
	
	handler, exists := d.handlers[rule.RuleType]
	if !exists {
		d.logger.Warn("No handler found for rule type",
			zap.String("rule_type", rule.RuleType),
			zap.String("rule_id", rule.ID),
		)
		return nil
	}
	
	d.logger.Debug("Processing compliance rule",
		zap.String("rule_type", rule.RuleType),
		zap.String("rule_id", rule.ID),
		zap.String("order_id", order.ID),
	)
	
	return handler.Handle(rule, order, userID)
}

// GetSupportedRuleTypes returns all supported rule types
func (d *RuleDispatcher) GetSupportedRuleTypes() []string {
	types := make([]string, 0, len(d.handlers))
	for ruleType := range d.handlers {
		types = append(types, ruleType)
	}
	return types
}

// OrderSizeHandler handles order size compliance rules
type OrderSizeHandler struct {
	logger *zap.Logger
}

func (h *OrderSizeHandler) GetRuleType() string {
	return RuleTypeOrderSize
}

func (h *OrderSizeHandler) GetDescription() string {
	return "Validates order size against maximum allowed limits"
}

func (h *OrderSizeHandler) Handle(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	maxSize, ok := rule.Parameters["max_order_size"].(float64)
	if !ok || order.Quantity <= maxSize {
		return nil
	}
	
	h.logger.Warn("Order size violation detected",
		zap.String("order_id", order.ID),
		zap.Float64("order_size", order.Quantity),
		zap.Float64("max_size", maxSize),
	)
	
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

// PositionLimitHandler handles position limit compliance rules
type PositionLimitHandler struct {
	logger *zap.Logger
}

func (h *PositionLimitHandler) GetRuleType() string {
	return RuleTypePositionLimit
}

func (h *PositionLimitHandler) GetDescription() string {
	return "Validates position size against maximum allowed limits"
}

func (h *PositionLimitHandler) Handle(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	maxPosition, ok := rule.Parameters["max_position"].(float64)
	if !ok || order.Quantity <= maxPosition {
		return nil
	}
	
	h.logger.Warn("Position limit violation detected",
		zap.String("order_id", order.ID),
		zap.Float64("position_size", order.Quantity),
		zap.Float64("max_position", maxPosition),
	)
	
	return &ComplianceViolation{
		ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
		RuleID:      rule.ID,
		OrderID:     order.ID,
		UserID:      userID,
		Symbol:      order.Symbol,
		Severity:    rule.Severity,
		Description: fmt.Sprintf("Position size %.2f exceeds maximum allowed position %.2f", order.Quantity, maxPosition),
		Details: map[string]interface{}{
			"position_size": order.Quantity,
			"max_position":  maxPosition,
		},
		Status:     ViolationStatusOpen,
		DetectedAt: time.Now(),
	}
}

// TradingHoursHandler handles trading hours compliance rules
type TradingHoursHandler struct {
	logger *zap.Logger
}

func (h *TradingHoursHandler) GetRuleType() string {
	return RuleTypeTradingHours
}

func (h *TradingHoursHandler) GetDescription() string {
	return "Validates orders are placed within allowed trading hours"
}

func (h *TradingHoursHandler) Handle(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	now := time.Now().UTC()
	startHour, _ := rule.Parameters["start_hour"].(int)
	endHour, _ := rule.Parameters["end_hour"].(int)
	
	if now.Hour() >= startHour && now.Hour() < endHour {
		return nil
	}
	
	h.logger.Warn("Trading hours violation detected",
		zap.String("order_id", order.ID),
		zap.Int("current_hour", now.Hour()),
		zap.Int("start_hour", startHour),
		zap.Int("end_hour", endHour),
	)
	
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

// RiskLimitHandler handles risk limit compliance rules
type RiskLimitHandler struct {
	logger *zap.Logger
}

func (h *RiskLimitHandler) GetRuleType() string {
	return RuleTypeRiskLimit
}

func (h *RiskLimitHandler) GetDescription() string {
	return "Validates risk exposure against maximum allowed limits"
}

func (h *RiskLimitHandler) Handle(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	maxRisk, ok := rule.Parameters["max_risk_exposure"].(float64)
	if !ok {
		return nil
	}
	
	// Calculate risk exposure (simplified)
	riskExposure := order.Quantity * order.Price
	if riskExposure <= maxRisk {
		return nil
	}
	
	h.logger.Warn("Risk limit violation detected",
		zap.String("order_id", order.ID),
		zap.Float64("risk_exposure", riskExposure),
		zap.Float64("max_risk", maxRisk),
	)
	
	return &ComplianceViolation{
		ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
		RuleID:      rule.ID,
		OrderID:     order.ID,
		UserID:      userID,
		Symbol:      order.Symbol,
		Severity:    rule.Severity,
		Description: fmt.Sprintf("Risk exposure %.2f exceeds maximum allowed %.2f", riskExposure, maxRisk),
		Details: map[string]interface{}{
			"risk_exposure": riskExposure,
			"max_risk":      maxRisk,
		},
		Status:     ViolationStatusOpen,
		DetectedAt: time.Now(),
	}
}

// AMLHandler handles Anti-Money Laundering compliance rules
type AMLHandler struct {
	logger *zap.Logger
}

func (h *AMLHandler) GetRuleType() string {
	return RuleTypeAML
}

func (h *AMLHandler) GetDescription() string {
	return "Performs Anti-Money Laundering checks on orders"
}

func (h *AMLHandler) Handle(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	// AML logic would be more complex in practice
	suspiciousThreshold, ok := rule.Parameters["suspicious_threshold"].(float64)
	if !ok {
		return nil
	}
	
	orderValue := order.Quantity * order.Price
	if orderValue <= suspiciousThreshold {
		return nil
	}
	
	h.logger.Warn("AML suspicious activity detected",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.Float64("order_value", orderValue),
	)
	
	return &ComplianceViolation{
		ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
		RuleID:      rule.ID,
		OrderID:     order.ID,
		UserID:      userID,
		Symbol:      order.Symbol,
		Severity:    rule.Severity,
		Description: fmt.Sprintf("Suspicious transaction detected: value %.2f exceeds threshold %.2f", orderValue, suspiciousThreshold),
		Details: map[string]interface{}{
			"order_value":           orderValue,
			"suspicious_threshold":  suspiciousThreshold,
			"requires_investigation": true,
		},
		Status:     ViolationStatusOpen,
		DetectedAt: time.Now(),
	}
}

// KYCHandler handles Know Your Customer compliance rules
type KYCHandler struct {
	logger *zap.Logger
}

func (h *KYCHandler) GetRuleType() string {
	return RuleTypeKYC
}

func (h *KYCHandler) GetDescription() string {
	return "Validates Know Your Customer requirements for orders"
}

func (h *KYCHandler) Handle(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	// KYC logic would check user verification status
	requiresVerification, ok := rule.Parameters["requires_verification"].(bool)
	if !ok || !requiresVerification {
		return nil
	}
	
	// In practice, this would check user's KYC status from database
	// For now, simulate a KYC violation
	if userID == "" {
		h.logger.Warn("KYC violation: missing user ID",
			zap.String("order_id", order.ID),
		)
		
		return &ComplianceViolation{
			ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
			RuleID:      rule.ID,
			OrderID:     order.ID,
			UserID:      userID,
			Symbol:      order.Symbol,
			Severity:    rule.Severity,
			Description: "KYC verification required but user ID is missing",
			Details: map[string]interface{}{
				"requires_verification": requiresVerification,
				"user_verified":         false,
			},
			Status:     ViolationStatusOpen,
			DetectedAt: time.Now(),
		}
	}
	
	return nil
}

// SanctionsHandler handles sanctions compliance rules
type SanctionsHandler struct {
	logger *zap.Logger
}

func (h *SanctionsHandler) GetRuleType() string {
	return RuleTypeSanctions
}

func (h *SanctionsHandler) GetDescription() string {
	return "Checks orders against sanctions lists and restricted entities"
}

func (h *SanctionsHandler) Handle(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	// Sanctions logic would check against OFAC and other sanctions lists
	checkSanctions, ok := rule.Parameters["check_sanctions"].(bool)
	if !ok || !checkSanctions {
		return nil
	}
	
	// In practice, this would check against actual sanctions databases
	// For now, simulate based on symbol patterns
	if len(order.Symbol) > 0 && order.Symbol[0] == 'X' {
		h.logger.Warn("Sanctions violation detected",
			zap.String("order_id", order.ID),
			zap.String("symbol", order.Symbol),
			zap.String("user_id", userID),
		)
		
		return &ComplianceViolation{
			ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
			RuleID:      rule.ID,
			OrderID:     order.ID,
			UserID:      userID,
			Symbol:      order.Symbol,
			Severity:    rule.Severity,
			Description: fmt.Sprintf("Symbol %s matches sanctions screening criteria", order.Symbol),
			Details: map[string]interface{}{
				"sanctions_match": true,
				"symbol":          order.Symbol,
				"requires_review": true,
			},
			Status:     ViolationStatusOpen,
			DetectedAt: time.Now(),
		}
	}
	
	return nil
}
