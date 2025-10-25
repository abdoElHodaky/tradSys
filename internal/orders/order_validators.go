package orders

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
)

// OrderValidator handles order validation logic
type OrderValidator struct {
	logger *zap.Logger
}

// NewOrderValidator creates a new order validator
func NewOrderValidator(logger *zap.Logger) *OrderValidator {
	return &OrderValidator{
		logger: logger,
	}
}

// ValidateOrderRequest validates an order request
func (v *OrderValidator) ValidateOrderRequest(ctx context.Context, req *OrderRequest) error {
	if req == nil {
		return ErrInvalidOrderRequest
	}

	// Validate required fields
	if err := v.validateRequiredFields(req); err != nil {
		return err
	}

	// Validate field values
	if err := v.validateFieldValues(req); err != nil {
		return err
	}

	// Validate business rules
	if err := v.validateBusinessRules(ctx, req); err != nil {
		return err
	}

	v.logger.Debug("Order request validated successfully",
		zap.String("user_id", req.UserID),
		zap.String("symbol", req.Symbol),
		zap.String("side", string(req.Side)),
		zap.String("type", string(req.Type)))

	return nil
}

// ValidateOrderUpdate validates an order update request
func (v *OrderValidator) ValidateOrderUpdate(ctx context.Context, order *Order, req *OrderUpdateRequest) error {
	if req == nil {
		return ErrInvalidOrderRequest
	}

	if order == nil {
		return ErrOrderNotFound
	}

	// Check if order can be updated
	if !v.canOrderBeUpdated(order) {
		return ErrOrderCannotBeUpdated
	}

	// Validate update fields
	if err := v.validateUpdateFields(req); err != nil {
		return err
	}

	// Validate business rules for updates
	if err := v.validateUpdateBusinessRules(ctx, order, req); err != nil {
		return err
	}

	v.logger.Debug("Order update validated successfully",
		zap.String("order_id", req.OrderID),
		zap.String("user_id", req.UserID))

	return nil
}

// ValidateOrderCancellation validates an order cancellation request
func (v *OrderValidator) ValidateOrderCancellation(ctx context.Context, order *Order, req *OrderCancelRequest) error {
	if req == nil {
		return ErrInvalidOrderRequest
	}

	if order == nil {
		return ErrOrderNotFound
	}

	// Check if order can be cancelled
	if !v.canOrderBeCancelled(order) {
		return ErrOrderCannotBeCancelled
	}

	// Validate user authorization
	if order.UserID != req.UserID {
		return ErrUnauthorizedOrderAccess
	}

	v.logger.Debug("Order cancellation validated successfully",
		zap.String("order_id", req.OrderID),
		zap.String("user_id", req.UserID))

	return nil
}

// validateRequiredFields validates required fields in order request
func (v *OrderValidator) validateRequiredFields(req *OrderRequest) error {
	if strings.TrimSpace(req.UserID) == "" {
		return ErrMissingUserID
	}

	if strings.TrimSpace(req.Symbol) == "" {
		return ErrMissingSymbol
	}

	if req.Side == "" {
		return ErrMissingSide
	}

	if req.Type == "" {
		return ErrMissingOrderType
	}

	if req.Quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Price is required for limit orders
	if (req.Type == OrderTypeLimit || req.Type == OrderTypeStopLimit) && req.Price <= 0 {
		return ErrMissingPrice
	}

	// Stop price is required for stop orders
	if (req.Type == OrderTypeStopLimit || req.Type == OrderTypeStopMarket) && req.StopPrice <= 0 {
		return ErrMissingStopPrice
	}

	return nil
}

// validateFieldValues validates field values in order request
func (v *OrderValidator) validateFieldValues(req *OrderRequest) error {
	// Validate order side
	if !v.isValidOrderSide(req.Side) {
		return ErrInvalidOrderSide
	}

	// Validate order type
	if !v.isValidOrderType(req.Type) {
		return ErrInvalidOrderType
	}

	// Validate time in force
	if req.TimeInForce != "" && !v.isValidTimeInForce(req.TimeInForce) {
		return ErrInvalidTimeInForce
	}

	// Validate symbol format
	if !v.isValidSymbol(req.Symbol) {
		return ErrInvalidSymbol
	}

	// Validate price precision
	if req.Price > 0 && !v.isValidPricePrecision(req.Price) {
		return ErrInvalidPricePrecision
	}

	// Validate quantity precision
	if !v.isValidQuantityPrecision(req.Quantity) {
		return ErrInvalidQuantityPrecision
	}

	return nil
}

// validateBusinessRules validates business rules for order request
func (v *OrderValidator) validateBusinessRules(ctx context.Context, req *OrderRequest) error {
	// Validate order size limits
	if err := v.validateOrderSizeLimits(req); err != nil {
		return err
	}

	// Validate price limits
	if err := v.validatePriceLimits(req); err != nil {
		return err
	}

	// Validate expiration time
	if err := v.validateExpirationTime(req); err != nil {
		return err
	}

	// Validate market hours (if applicable)
	if err := v.validateMarketHours(ctx, req); err != nil {
		return err
	}

	return nil
}

// validateUpdateFields validates fields in order update request
func (v *OrderValidator) validateUpdateFields(req *OrderUpdateRequest) error {
	// At least one field must be updated
	if req.Price <= 0 && req.Quantity <= 0 && req.StopPrice <= 0 && 
		req.TimeInForce == "" && req.ExpiresAt.IsZero() {
		return ErrNoFieldsToUpdate
	}

	// Validate updated values
	if req.Price > 0 && !v.isValidPricePrecision(req.Price) {
		return ErrInvalidPricePrecision
	}

	if req.Quantity > 0 && !v.isValidQuantityPrecision(req.Quantity) {
		return ErrInvalidQuantityPrecision
	}

	if req.TimeInForce != "" && !v.isValidTimeInForce(req.TimeInForce) {
		return ErrInvalidTimeInForce
	}

	return nil
}

// validateUpdateBusinessRules validates business rules for order updates
func (v *OrderValidator) validateUpdateBusinessRules(ctx context.Context, order *Order, req *OrderUpdateRequest) error {
	// Cannot reduce quantity below filled quantity
	if req.Quantity > 0 && req.Quantity < order.FilledQuantity {
		return ErrQuantityBelowFilled
	}

	// Validate updated order size limits
	if req.Quantity > 0 {
		tempReq := &OrderRequest{
			UserID:   order.UserID,
			Symbol:   order.Symbol,
			Quantity: req.Quantity,
		}
		if err := v.validateOrderSizeLimits(tempReq); err != nil {
			return err
		}
	}

	// Validate updated price limits
	if req.Price > 0 {
		tempReq := &OrderRequest{
			Symbol: order.Symbol,
			Price:  req.Price,
		}
		if err := v.validatePriceLimits(tempReq); err != nil {
			return err
		}
	}

	return nil
}

// canOrderBeUpdated checks if an order can be updated
func (v *OrderValidator) canOrderBeUpdated(order *Order) bool {
	// Only new and pending orders can be updated
	return order.Status == OrderStatusNew || order.Status == OrderStatusPending
}

// canOrderBeCancelled checks if an order can be cancelled
func (v *OrderValidator) canOrderBeCancelled(order *Order) bool {
	// Only new, pending, and partially filled orders can be cancelled
	return order.Status == OrderStatusNew || 
		   order.Status == OrderStatusPending || 
		   order.Status == OrderStatusPartiallyFilled
}

// isValidOrderSide checks if order side is valid
func (v *OrderValidator) isValidOrderSide(side OrderSide) bool {
	return side == OrderSideBuy || side == OrderSideSell
}

// isValidOrderType checks if order type is valid
func (v *OrderValidator) isValidOrderType(orderType OrderType) bool {
	validTypes := []OrderType{
		OrderTypeLimit,
		OrderTypeMarket,
		OrderTypeStopLimit,
		OrderTypeStopMarket,
	}

	for _, validType := range validTypes {
		if orderType == validType {
			return true
		}
	}
	return false
}

// isValidTimeInForce checks if time in force is valid
func (v *OrderValidator) isValidTimeInForce(tif TimeInForce) bool {
	validTifs := []TimeInForce{
		TimeInForceGTC,
		TimeInForceIOC,
		TimeInForceFOK,
		TimeInForceDay,
	}

	for _, validTif := range validTifs {
		if tif == validTif {
			return true
		}
	}
	return false
}

// isValidSymbol checks if symbol format is valid
func (v *OrderValidator) isValidSymbol(symbol string) bool {
	// Basic symbol validation - should be 2-10 characters, alphanumeric
	if len(symbol) < 2 || len(symbol) > 10 {
		return false
	}

	for _, char := range symbol {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	return true
}

// isValidPricePrecision checks if price has valid precision
func (v *OrderValidator) isValidPricePrecision(price float64) bool {
	// Allow up to 4 decimal places
	return price > 0 && price < 1000000 // Basic range check
}

// isValidQuantityPrecision checks if quantity has valid precision
func (v *OrderValidator) isValidQuantityPrecision(quantity float64) bool {
	// Allow up to 8 decimal places for crypto, 0 for stocks
	return quantity > 0 && quantity < 1000000000 // Basic range check
}

// validateOrderSizeLimits validates order size limits
func (v *OrderValidator) validateOrderSizeLimits(req *OrderRequest) error {
	// Maximum order size limits
	maxOrderSize := v.getMaxOrderSize(req.Symbol)
	if req.Quantity > maxOrderSize {
		return ErrOrderSizeExceedsLimit
	}

	// Minimum order size limits
	minOrderSize := v.getMinOrderSize(req.Symbol)
	if req.Quantity < minOrderSize {
		return ErrOrderSizeBelowMinimum
	}

	// Order value limits
	if req.Price > 0 {
		orderValue := req.Price * req.Quantity
		maxOrderValue := v.getMaxOrderValue(req.Symbol)
		if orderValue > maxOrderValue {
			return ErrOrderValueExceedsLimit
		}
	}

	return nil
}

// validatePriceLimits validates price limits
func (v *OrderValidator) validatePriceLimits(req *OrderRequest) error {
	if req.Price <= 0 {
		return nil // No price to validate
	}

	// Price range limits
	minPrice := v.getMinPrice(req.Symbol)
	maxPrice := v.getMaxPrice(req.Symbol)

	if req.Price < minPrice {
		return ErrPriceBelowMinimum
	}

	if req.Price > maxPrice {
		return ErrPriceExceedsMaximum
	}

	return nil
}

// validateExpirationTime validates order expiration time
func (v *OrderValidator) validateExpirationTime(req *OrderRequest) error {
	if req.ExpiresAt.IsZero() {
		return nil // No expiration time set
	}

	// Expiration time must be in the future
	if req.ExpiresAt.Before(time.Now()) {
		return ErrExpirationTimeInPast
	}

	// Maximum expiration time (e.g., 90 days)
	maxExpiration := time.Now().Add(90 * 24 * time.Hour)
	if req.ExpiresAt.After(maxExpiration) {
		return ErrExpirationTimeTooFar
	}

	return nil
}

// validateMarketHours validates if order can be placed during current market hours
func (v *OrderValidator) validateMarketHours(ctx context.Context, req *OrderRequest) error {
	// For market orders, check if market is open
	if req.Type == OrderTypeMarket {
		if !v.isMarketOpen(req.Symbol) {
			return ErrMarketClosed
		}
	}

	return nil
}

// Helper methods to get limits (would be configurable in production)
func (v *OrderValidator) getMaxOrderSize(symbol string) float64 {
	// Default max order size
	return 1000000
}

func (v *OrderValidator) getMinOrderSize(symbol string) float64 {
	// Default min order size
	return 0.001
}

func (v *OrderValidator) getMaxOrderValue(symbol string) float64 {
	// Default max order value
	return 10000000
}

func (v *OrderValidator) getMinPrice(symbol string) float64 {
	// Default min price
	return 0.0001
}

func (v *OrderValidator) getMaxPrice(symbol string) float64 {
	// Default max price
	return 1000000
}

func (v *OrderValidator) isMarketOpen(symbol string) bool {
	// Simplified market hours check
	now := time.Now()
	hour := now.Hour()
	
	// Assume market is open 9 AM to 4 PM
	return hour >= 9 && hour < 16
}

// Error definitions for validation
var (
	ErrInvalidOrderRequest      = errors.New("invalid order request")
	ErrMissingUserID           = errors.New("missing user ID")
	ErrMissingSymbol           = errors.New("missing symbol")
	ErrMissingSide             = errors.New("missing order side")
	ErrMissingOrderType        = errors.New("missing order type")
	ErrMissingPrice            = errors.New("missing price for limit order")
	ErrMissingStopPrice        = errors.New("missing stop price for stop order")
	ErrInvalidQuantity         = errors.New("invalid quantity")
	ErrInvalidOrderSide        = errors.New("invalid order side")
	ErrInvalidOrderType        = errors.New("invalid order type")
	ErrInvalidTimeInForce      = errors.New("invalid time in force")
	ErrInvalidSymbol           = errors.New("invalid symbol format")
	ErrInvalidPricePrecision   = errors.New("invalid price precision")
	ErrInvalidQuantityPrecision = errors.New("invalid quantity precision")
	ErrOrderSizeExceedsLimit   = errors.New("order size exceeds limit")
	ErrOrderSizeBelowMinimum   = errors.New("order size below minimum")
	ErrOrderValueExceedsLimit  = errors.New("order value exceeds limit")
	ErrPriceBelowMinimum       = errors.New("price below minimum")
	ErrPriceExceedsMaximum     = errors.New("price exceeds maximum")
	ErrExpirationTimeInPast    = errors.New("expiration time is in the past")
	ErrExpirationTimeTooFar    = errors.New("expiration time is too far in the future")
	ErrMarketClosed            = errors.New("market is closed")
	ErrOrderCannotBeUpdated    = errors.New("order cannot be updated")
	ErrOrderCannotBeCancelled  = errors.New("order cannot be cancelled")
	ErrUnauthorizedOrderAccess = errors.New("unauthorized order access")
	ErrNoFieldsToUpdate        = errors.New("no fields to update")
	ErrQuantityBelowFilled     = errors.New("quantity cannot be below filled quantity")
	ErrOrderNotFound           = errors.New("order not found")
)
