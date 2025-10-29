package service

import (
	"errors"
	"fmt"
	"time"
)

// OrderValidator provides validation functionality for orders
type OrderValidator struct {
	validators []ValidatorFunc
}

// ValidatorFunc represents a validation function
type ValidatorFunc func(*OrderRequest) error

// NewOrderValidator creates a new order validator with default validation rules
func NewOrderValidator() *OrderValidator {
	return &OrderValidator{
		validators: []ValidatorFunc{
			validateUserID,
			validateSymbol,
			validateSide,
			validateOrderType,
			validateQuantity,
			validatePrice,
			validateTimeInForce,
			validateStopPrice,
			validateExpiration,
		},
	}
}

// Validate validates an order request using early return pattern
func (v *OrderValidator) Validate(req *OrderRequest) error {
	if req == nil {
		return errors.New("order request cannot be nil")
	}

	for _, validator := range v.validators {
		if err := validator(req); err != nil {
			return err
		}
	}

	return nil
}

// validateUserID validates the user ID field
func validateUserID(req *OrderRequest) error {
	if req.UserID == "" {
		return errors.New("user ID is required")
	}
	if len(req.UserID) < 3 {
		return errors.New("user ID must be at least 3 characters")
	}
	if len(req.UserID) > 50 {
		return errors.New("user ID must be less than 50 characters")
	}
	return nil
}

// validateSymbol validates the trading symbol
func validateSymbol(req *OrderRequest) error {
	if req.Symbol == "" {
		return errors.New("symbol is required")
	}
	if len(req.Symbol) < 2 {
		return errors.New("symbol must be at least 2 characters")
	}
	if len(req.Symbol) > 20 {
		return errors.New("symbol must be less than 20 characters")
	}
	if !isValidSymbolFormat(req.Symbol) {
		return errors.New("symbol contains invalid characters")
	}
	return nil
}

// validateSide validates the order side
func validateSide(req *OrderRequest) error {
	if req.Side == "" {
		return errors.New("order side is required")
	}
	if req.Side != OrderSideBuy && req.Side != OrderSideSell {
		return fmt.Errorf("invalid order side: %s", req.Side)
	}
	return nil
}

// validateOrderType validates the order type
func validateOrderType(req *OrderRequest) error {
	if req.Type == "" {
		return errors.New("order type is required")
	}
	
	validTypes := []OrderType{
		OrderTypeLimit,
		OrderTypeMarket,
		OrderTypeStopLimit,
		OrderTypeStopMarket,
	}
	
	for _, validType := range validTypes {
		if req.Type == validType {
			return nil
		}
	}
	
	return fmt.Errorf("invalid order type: %s", req.Type)
}

// validateQuantity validates the order quantity
func validateQuantity(req *OrderRequest) error {
	if req.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if req.Quantity > 1000000 {
		return errors.New("quantity exceeds maximum limit")
	}
	return nil
}

// validatePrice validates the order price using early returns
func validatePrice(req *OrderRequest) error {
	// Market orders don't require price validation
	if req.Type == OrderTypeMarket || req.Type == OrderTypeStopMarket {
		return nil
	}
	
	if req.Price <= 0 {
		return errors.New("price must be positive for limit orders")
	}
	if req.Price > 1000000 {
		return errors.New("price exceeds maximum limit")
	}
	
	return nil
}

// validateStopPrice validates stop price for stop orders
func validateStopPrice(req *OrderRequest) error {
	// Only validate stop price for stop orders
	if req.Type != OrderTypeStopLimit && req.Type != OrderTypeStopMarket {
		return nil
	}
	
	if req.StopPrice <= 0 {
		return errors.New("stop price must be positive for stop orders")
	}
	if req.StopPrice > 1000000 {
		return errors.New("stop price exceeds maximum limit")
	}
	
	return nil
}

// validateTimeInForce validates the time in force
func validateTimeInForce(req *OrderRequest) error {
	if req.TimeInForce == "" {
		return errors.New("time in force is required")
	}
	
	validTIF := []TimeInForce{
		TimeInForceGTC,
		TimeInForceIOC,
		TimeInForceFOK,
		TimeInForceDay,
	}
	
	for _, validType := range validTIF {
		if req.TimeInForce == validType {
			return nil
		}
	}
	
	return fmt.Errorf("invalid time in force: %s", req.TimeInForce)
}

// validateExpiration validates order expiration
func validateExpiration(req *OrderRequest) error {
	// Only validate expiration for day orders
	if req.TimeInForce != TimeInForceDay {
		return nil
	}
	
	if req.ExpiresAt.IsZero() {
		return errors.New("expiration time is required for day orders")
	}
	if req.ExpiresAt.Before(time.Now()) {
		return errors.New("expiration time cannot be in the past")
	}
	if req.ExpiresAt.After(time.Now().Add(24 * time.Hour)) {
		return errors.New("expiration time cannot be more than 24 hours in the future")
	}
	
	return nil
}

// Helper functions

// isValidSymbolFormat checks if symbol format is valid
func isValidSymbolFormat(symbol string) bool {
	// Allow alphanumeric characters and common separators
	for _, char := range symbol {
		if !((char >= 'A' && char <= 'Z') || 
			 (char >= 'a' && char <= 'z') || 
			 (char >= '0' && char <= '9') || 
			 char == '-' || char == '_' || char == '.') {
			return false
		}
	}
	return true
}
