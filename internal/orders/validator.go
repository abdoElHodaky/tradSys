package orders

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// OrderValidator handles validation of order requests
type OrderValidator struct {
	logger *zap.Logger

	// Configuration
	maxQuantity    float64
	maxPrice       float64
	minPrice       float64
	minQuantity    float64
	allowedSymbols map[string]bool
	marketHours    map[string]MarketHours

	// Validation rules
	symbolPattern   *regexp.Regexp
	clientIDPattern *regexp.Regexp
}

// MarketHours represents trading hours for a market
type MarketHours struct {
	Open     time.Time
	Close    time.Time
	Timezone string
	Days     []time.Weekday
}

// ValidationResult represents the result of order validation
type ValidationResult struct {
	Valid    bool                   `json:"valid"`
	Errors   []ValidationError      `json:"errors,omitempty"`
	Warnings []ValidationWarning    `json:"warnings,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string      `json:"field"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// NewOrderValidator creates a new order validator
func NewOrderValidator(logger *zap.Logger) *OrderValidator {
	// Default symbol pattern (e.g., BTC-USD, AAPL, etc.)
	symbolPattern := regexp.MustCompile(`^[A-Z0-9]{1,10}(-[A-Z0-9]{1,10})?$`)

	// Client order ID pattern (alphanumeric with hyphens/underscores)
	clientIDPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

	return &OrderValidator{
		logger:          logger,
		maxQuantity:     1000000.0, // Default max quantity
		maxPrice:        1000000.0, // Default max price
		minPrice:        0.01,      // Default min price
		minQuantity:     0.001,     // Default min quantity
		allowedSymbols:  make(map[string]bool),
		marketHours:     make(map[string]MarketHours),
		symbolPattern:   symbolPattern,
		clientIDPattern: clientIDPattern,
	}
}

// SetLimits sets validation limits
func (v *OrderValidator) SetLimits(maxQuantity, maxPrice, minPrice, minQuantity float64) {
	v.maxQuantity = maxQuantity
	v.maxPrice = maxPrice
	v.minPrice = minPrice
	v.minQuantity = minQuantity
}

// AddAllowedSymbol adds a symbol to the allowed list
func (v *OrderValidator) AddAllowedSymbol(symbol string) {
	v.allowedSymbols[strings.ToUpper(symbol)] = true
}

// SetMarketHours sets market hours for a symbol
func (v *OrderValidator) SetMarketHours(symbol string, hours MarketHours) {
	v.marketHours[strings.ToUpper(symbol)] = hours
}

// ValidateOrderRequest validates an order request
func (v *OrderValidator) ValidateOrderRequest(ctx context.Context, request *OrderRequest) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
		Details:  make(map[string]interface{}),
	}

	// Validate required fields
	v.validateRequiredFields(request, result)

	// Validate field formats
	v.validateFieldFormats(request, result)

	// Validate business rules
	v.validateBusinessRules(request, result)

	// Validate market hours
	v.validateMarketHours(request, result)

	// Validate limits
	v.validateLimits(request, result)

	// Set overall validity
	result.Valid = len(result.Errors) == 0

	// Log validation result
	if !result.Valid {
		v.logger.Warn("Order validation failed",
			zap.String("user_id", request.UserID),
			zap.String("symbol", request.Symbol),
			zap.Int("error_count", len(result.Errors)))
	}

	return result
}

// validateRequiredFields validates required fields
func (v *OrderValidator) validateRequiredFields(request *OrderRequest, result *ValidationResult) {
	if request.UserID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "user_id",
			Code:    ErrCodeValidationFailed,
			Message: "User ID is required",
		})
	}

	if request.AccountID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "account_id",
			Code:    ErrCodeValidationFailed,
			Message: "Account ID is required",
		})
	}

	if request.Symbol == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "symbol",
			Code:    ErrCodeInvalidSymbol,
			Message: "Symbol is required",
		})
	}

	if request.Side == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "side",
			Code:    ErrCodeInvalidSide,
			Message: "Order side is required",
		})
	}

	if request.Type == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "type",
			Code:    ErrCodeInvalidType,
			Message: "Order type is required",
		})
	}

	if request.Quantity <= 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "quantity",
			Code:    ErrCodeInsufficientQuantity,
			Message: "Quantity must be greater than 0",
			Value:   request.Quantity,
		})
	}
}

// validateFieldFormats validates field formats
func (v *OrderValidator) validateFieldFormats(request *OrderRequest, result *ValidationResult) {
	// Validate symbol format
	if request.Symbol != "" && !v.symbolPattern.MatchString(strings.ToUpper(request.Symbol)) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "symbol",
			Code:    ErrCodeInvalidSymbol,
			Message: "Invalid symbol format",
			Value:   request.Symbol,
		})
	}

	// Validate client order ID format
	if request.ClientOrderID != "" && !v.clientIDPattern.MatchString(request.ClientOrderID) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "client_order_id",
			Code:    ErrCodeInvalidRequest,
			Message: "Invalid client order ID format",
			Value:   request.ClientOrderID,
		})
	}

	// Validate order side
	if request.Side != "" && request.Side != OrderSideBuy && request.Side != OrderSideSell {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "side",
			Code:    ErrCodeInvalidSide,
			Message: "Invalid order side, must be 'buy' or 'sell'",
			Value:   request.Side,
		})
	}

	// Validate order type
	validTypes := []OrderType{OrderTypeLimit, OrderTypeMarket, OrderTypeStopLimit, OrderTypeStopMarket}
	if request.Type != "" {
		valid := false
		for _, validType := range validTypes {
			if request.Type == validType {
				valid = true
				break
			}
		}
		if !valid {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "type",
				Code:    ErrCodeInvalidType,
				Message: fmt.Sprintf("Invalid order type, must be one of: %v", validTypes),
				Value:   request.Type,
			})
		}
	}

	// Validate time in force
	if request.TimeInForce != "" {
		validTIF := []TimeInForce{TimeInForceGTC, TimeInForceIOC, TimeInForceFOK, TimeInForceDAY}
		valid := false
		for _, validType := range validTIF {
			if request.TimeInForce == validType {
				valid = true
				break
			}
		}
		if !valid {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "time_in_force",
				Code:    ErrCodeInvalidTimeInForce,
				Message: fmt.Sprintf("Invalid time in force, must be one of: %v", validTIF),
				Value:   request.TimeInForce,
			})
		}
	}
}

// validateBusinessRules validates business logic rules
func (v *OrderValidator) validateBusinessRules(request *OrderRequest, result *ValidationResult) {
	// Check price for limit orders
	if (request.Type == OrderTypeLimit || request.Type == OrderTypeStopLimit) && request.Price <= 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "price",
			Code:    ErrCodeInvalidPrice,
			Message: "Price is required for limit orders and must be greater than 0",
			Value:   request.Price,
		})
	}

	// Check stop price for stop orders
	if (request.Type == OrderTypeStopLimit || request.Type == OrderTypeStopMarket) && request.StopPrice <= 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "stop_price",
			Code:    ErrCodeInvalidPrice,
			Message: "Stop price is required for stop orders and must be greater than 0",
			Value:   request.StopPrice,
		})
	}

	// Validate stop limit order logic
	if request.Type == OrderTypeStopLimit && request.Price > 0 && request.StopPrice > 0 {
		if request.Side == OrderSideBuy && request.StopPrice <= request.Price {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "stop_price",
				Code:    "UNUSUAL_STOP_PRICE",
				Message: "Stop price should typically be higher than limit price for buy stop orders",
				Value:   map[string]float64{"stop_price": request.StopPrice, "price": request.Price},
			})
		} else if request.Side == OrderSideSell && request.StopPrice >= request.Price {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "stop_price",
				Code:    "UNUSUAL_STOP_PRICE",
				Message: "Stop price should typically be lower than limit price for sell stop orders",
				Value:   map[string]float64{"stop_price": request.StopPrice, "price": request.Price},
			})
		}
	}

	// Check allowed symbols
	if len(v.allowedSymbols) > 0 && request.Symbol != "" {
		if !v.allowedSymbols[strings.ToUpper(request.Symbol)] {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "symbol",
				Code:    ErrCodeInvalidSymbol,
				Message: "Symbol is not allowed for trading",
				Value:   request.Symbol,
			})
		}
	}

	// Validate expiration time
	if request.ExpiresAt != nil && request.ExpiresAt.Before(time.Now()) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "expires_at",
			Code:    ErrCodeOrderExpired,
			Message: "Expiration time cannot be in the past",
			Value:   request.ExpiresAt,
		})
	}
}

// validateMarketHours validates market hours
func (v *OrderValidator) validateMarketHours(request *OrderRequest, result *ValidationResult) {
	if request.Symbol == "" {
		return
	}

	hours, exists := v.marketHours[strings.ToUpper(request.Symbol)]
	if !exists {
		// No market hours configured, allow trading
		return
	}

	now := time.Now()

	// Check if current day is a trading day
	validDay := false
	for _, day := range hours.Days {
		if now.Weekday() == day {
			validDay = true
			break
		}
	}

	if !validDay {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "symbol",
			Code:    "OUTSIDE_TRADING_DAYS",
			Message: "Order placed outside of regular trading days",
			Value: map[string]interface{}{
				"current_day":  now.Weekday().String(),
				"trading_days": hours.Days,
			},
		})
		return
	}

	// Check trading hours (simplified - assumes same timezone)
	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)
	openTime := time.Date(0, 1, 1, hours.Open.Hour(), hours.Open.Minute(), hours.Open.Second(), 0, time.UTC)
	closeTime := time.Date(0, 1, 1, hours.Close.Hour(), hours.Close.Minute(), hours.Close.Second(), 0, time.UTC)

	if currentTime.Before(openTime) || currentTime.After(closeTime) {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "symbol",
			Code:    "OUTSIDE_TRADING_HOURS",
			Message: "Order placed outside of regular trading hours",
			Value: map[string]interface{}{
				"current_time": now.Format("15:04:05"),
				"market_open":  hours.Open.Format("15:04:05"),
				"market_close": hours.Close.Format("15:04:05"),
			},
		})
	}
}

// validateLimits validates quantity and price limits
func (v *OrderValidator) validateLimits(request *OrderRequest, result *ValidationResult) {
	// Validate quantity limits
	if request.Quantity < v.minQuantity {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "quantity",
			Code:    ErrCodeInsufficientQuantity,
			Message: fmt.Sprintf("Quantity must be at least %f", v.minQuantity),
			Value:   request.Quantity,
		})
	}

	if request.Quantity > v.maxQuantity {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "quantity",
			Code:    ErrCodeInsufficientQuantity,
			Message: fmt.Sprintf("Quantity cannot exceed %f", v.maxQuantity),
			Value:   request.Quantity,
		})
	}

	// Validate price limits
	if request.Price > 0 {
		if request.Price < v.minPrice {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "price",
				Code:    ErrCodeInvalidPrice,
				Message: fmt.Sprintf("Price must be at least %f", v.minPrice),
				Value:   request.Price,
			})
		}

		if request.Price > v.maxPrice {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "price",
				Code:    ErrCodeInvalidPrice,
				Message: fmt.Sprintf("Price cannot exceed %f", v.maxPrice),
				Value:   request.Price,
			})
		}
	}

	// Validate stop price limits
	if request.StopPrice > 0 {
		if request.StopPrice < v.minPrice {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "stop_price",
				Code:    ErrCodeInvalidPrice,
				Message: fmt.Sprintf("Stop price must be at least %f", v.minPrice),
				Value:   request.StopPrice,
			})
		}

		if request.StopPrice > v.maxPrice {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "stop_price",
				Code:    ErrCodeInvalidPrice,
				Message: fmt.Sprintf("Stop price cannot exceed %f", v.maxPrice),
				Value:   request.StopPrice,
			})
		}
	}
}

// ValidateOrderUpdate validates an order update request
func (v *OrderValidator) ValidateOrderUpdate(ctx context.Context, request *OrderUpdateRequest) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
		Details:  make(map[string]interface{}),
	}

	// Validate required fields
	if request.UserID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "user_id",
			Code:    ErrCodeValidationFailed,
			Message: "User ID is required",
		})
	}

	if request.AccountID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "account_id",
			Code:    ErrCodeValidationFailed,
			Message: "Account ID is required",
		})
	}

	if request.OrderID == "" && request.ClientOrderID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "order_id",
			Code:    ErrCodeValidationFailed,
			Message: "Either order ID or client order ID is required",
		})
	}

	// Validate update fields
	if request.Quantity > 0 {
		if request.Quantity < v.minQuantity {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "quantity",
				Code:    ErrCodeInsufficientQuantity,
				Message: fmt.Sprintf("Quantity must be at least %f", v.minQuantity),
				Value:   request.Quantity,
			})
		}

		if request.Quantity > v.maxQuantity {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "quantity",
				Code:    ErrCodeInsufficientQuantity,
				Message: fmt.Sprintf("Quantity cannot exceed %f", v.maxQuantity),
				Value:   request.Quantity,
			})
		}
	}

	if request.Price > 0 {
		if request.Price < v.minPrice || request.Price > v.maxPrice {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "price",
				Code:    ErrCodeInvalidPrice,
				Message: fmt.Sprintf("Price must be between %f and %f", v.minPrice, v.maxPrice),
				Value:   request.Price,
			})
		}
	}

	if request.StopPrice > 0 {
		if request.StopPrice < v.minPrice || request.StopPrice > v.maxPrice {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "stop_price",
				Code:    ErrCodeInvalidPrice,
				Message: fmt.Sprintf("Stop price must be between %f and %f", v.minPrice, v.maxPrice),
				Value:   request.StopPrice,
			})
		}
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// ValidateOrderCancel validates an order cancel request
func (v *OrderValidator) ValidateOrderCancel(ctx context.Context, request *OrderCancelRequest) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
		Details:  make(map[string]interface{}),
	}

	// Validate required fields
	if request.UserID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "user_id",
			Code:    ErrCodeValidationFailed,
			Message: "User ID is required",
		})
	}

	if request.AccountID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "account_id",
			Code:    ErrCodeValidationFailed,
			Message: "Account ID is required",
		})
	}

	if request.OrderID == "" && request.ClientOrderID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "order_id",
			Code:    ErrCodeValidationFailed,
			Message: "Either order ID or client order ID is required",
		})
	}

	result.Valid = len(result.Errors) == 0
	return result
}
