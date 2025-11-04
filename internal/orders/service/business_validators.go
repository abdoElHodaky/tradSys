package service

import (
	"errors"
	"fmt"
	"time"
)

// BusinessRuleValidator validates business rules
type BusinessRuleValidator struct {
	maxOrderValue  float64
	maxDailyOrders int
	allowedSymbols map[string]bool
	tradingHours   TradingHours
}

// TradingHours represents trading hours configuration
type TradingHours struct {
	Start time.Time
	End   time.Time
}

// NewBusinessRuleValidator creates a new business rule validator
func NewBusinessRuleValidator() *BusinessRuleValidator {
	return &BusinessRuleValidator{
		maxOrderValue:  1000000.0,
		maxDailyOrders: 1000,
		allowedSymbols: make(map[string]bool),
		tradingHours: TradingHours{
			Start: time.Date(0, 1, 1, 9, 30, 0, 0, time.UTC),
			End:   time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC),
		},
	}
}

// ValidateBusinessRules validates business rules using early return pattern
func (v *BusinessRuleValidator) ValidateBusinessRules(req *OrderRequest) error {
	if req == nil {
		return errors.New("order request cannot be nil")
	}

	if err := v.validateOrderValue(req); err != nil {
		return err
	}

	if err := v.validateSymbolAllowed(req.Symbol); err != nil {
		return err
	}

	if err := v.validateTradingHours(); err != nil {
		return err
	}

	return nil
}

// validateOrderValue validates the total order value
func (v *BusinessRuleValidator) validateOrderValue(req *OrderRequest) error {
	orderValue := req.Price * req.Quantity
	if orderValue > v.maxOrderValue {
		return fmt.Errorf("order value %.2f exceeds maximum allowed %.2f",
			orderValue, v.maxOrderValue)
	}
	return nil
}

// validateSymbolAllowed checks if symbol is allowed for trading
func (v *BusinessRuleValidator) validateSymbolAllowed(symbol string) error {
	if len(v.allowedSymbols) == 0 {
		return nil // No restrictions if list is empty
	}

	if !v.allowedSymbols[symbol] {
		return fmt.Errorf("symbol %s is not allowed for trading", symbol)
	}

	return nil
}

// validateTradingHours checks if current time is within trading hours
func (v *BusinessRuleValidator) validateTradingHours() error {
	now := time.Now()
	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	if currentTime.Before(v.tradingHours.Start) {
		return errors.New("trading has not started yet")
	}
	if currentTime.After(v.tradingHours.End) {
		return errors.New("trading has ended for the day")
	}

	return nil
}

// RiskValidator validates risk-related constraints
type RiskValidator struct {
	maxPositionSize float64
	maxLeverage     float64
}

// NewRiskValidator creates a new risk validator
func NewRiskValidator() *RiskValidator {
	return &RiskValidator{
		maxPositionSize: 100000.0,
		maxLeverage:     10.0,
	}
}

// ValidateRiskConstraints validates risk constraints using early return pattern
func (v *RiskValidator) ValidateRiskConstraints(req *OrderRequest, currentPosition float64) error {
	if req == nil {
		return errors.New("order request cannot be nil")
	}

	if err := v.validatePositionSize(req, currentPosition); err != nil {
		return err
	}

	if err := v.validateLeverage(req); err != nil {
		return err
	}

	return nil
}

// validatePositionSize validates position size limits
func (v *RiskValidator) validatePositionSize(req *OrderRequest, currentPosition float64) error {
	newPosition := currentPosition
	if req.Side == OrderSideBuy {
		newPosition += req.Quantity
	} else {
		newPosition -= req.Quantity
	}

	if abs(newPosition) > v.maxPositionSize {
		return fmt.Errorf("position size %.2f would exceed maximum allowed %.2f",
			abs(newPosition), v.maxPositionSize)
	}

	return nil
}

// validateLeverage validates leverage limits
func (v *RiskValidator) validateLeverage(req *OrderRequest) error {
	// Simplified leverage calculation
	leverage := req.Quantity * req.Price / 10000 // Assuming 10k account balance

	if leverage > v.maxLeverage {
		return fmt.Errorf("leverage %.2f exceeds maximum allowed %.2f",
			leverage, v.maxLeverage)
	}

	return nil
}

// CancelValidator validates order cancellation requests
type CancelValidator struct{}

// NewCancelValidator creates a new cancel validator
func NewCancelValidator() *CancelValidator {
	return &CancelValidator{}
}

// ValidateCancelRequest validates order cancellation using early return pattern
func (v *CancelValidator) ValidateCancelRequest(req *OrderCancelRequest) error {
	if req == nil {
		return errors.New("cancel request cannot be nil")
	}

	if req.UserID == "" {
		return errors.New("user ID is required for cancellation")
	}

	if req.OrderID == "" && req.ClientOrderID == "" {
		return errors.New("either order ID or client order ID is required")
	}

	if req.Symbol == "" {
		return errors.New("symbol is required for cancellation")
	}

	return nil
}

// UpdateValidator validates order update requests
type UpdateValidator struct{}

// NewUpdateValidator creates a new update validator
func NewUpdateValidator() *UpdateValidator {
	return &UpdateValidator{}
}

// ValidateUpdateRequest validates order updates using early return pattern
func (v *UpdateValidator) ValidateUpdateRequest(req *OrderUpdateRequest) error {
	if req == nil {
		return errors.New("update request cannot be nil")
	}

	if req.UserID == "" {
		return errors.New("user ID is required for update")
	}

	if req.OrderID == "" && req.ClientOrderID == "" {
		return errors.New("either order ID or client order ID is required")
	}

	if req.Symbol == "" {
		return errors.New("symbol is required for update")
	}

	// Validate updated fields
	if req.Price <= 0 && req.Quantity <= 0 {
		return errors.New("at least one field (price or quantity) must be updated")
	}

	if req.Price > 0 && req.Price > 1000000 {
		return errors.New("updated price exceeds maximum limit")
	}

	if req.Quantity > 0 && req.Quantity > 1000000 {
		return errors.New("updated quantity exceeds maximum limit")
	}

	return nil
}

// Helper functions

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
