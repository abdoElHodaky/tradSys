package orders

import "errors"

// Order service errors
var (
	// ErrOrderNotFound is returned when an order is not found
	ErrOrderNotFound = errors.New("order not found")

	// ErrInvalidOrder is returned when an order is invalid
	ErrInvalidOrder = errors.New("invalid order")

	// ErrInvalidOrderRequest is returned when an order request is invalid
	ErrInvalidOrderRequest = errors.New("invalid order request")

	// ErrOrderAlreadyExists is returned when an order already exists
	ErrOrderAlreadyExists = errors.New("order already exists")

	// ErrOrderNotActive is returned when trying to modify an inactive order
	ErrOrderNotActive = errors.New("order is not active")

	// ErrOrderExpired is returned when an order has expired
	ErrOrderExpired = errors.New("order has expired")

	// ErrInsufficientQuantity is returned when there's insufficient quantity
	ErrInsufficientQuantity = errors.New("insufficient quantity")

	// ErrInvalidPrice is returned when the price is invalid
	ErrInvalidPrice = errors.New("invalid price")

	// ErrInvalidSymbol is returned when the symbol is invalid
	ErrInvalidSymbol = errors.New("invalid symbol")

	// ErrInvalidSide is returned when the order side is invalid
	ErrInvalidSide = errors.New("invalid order side")

	// ErrInvalidType is returned when the order type is invalid
	ErrInvalidType = errors.New("invalid order type")

	// ErrInvalidTimeInForce is returned when the time in force is invalid
	ErrInvalidTimeInForce = errors.New("invalid time in force")

	// ErrClientOrderIDExists is returned when client order ID already exists
	ErrClientOrderIDExists = errors.New("client order ID already exists")

	// ErrUnauthorized is returned when user is not authorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidOrderStatus is returned when order status is invalid
	ErrInvalidOrderStatus = errors.New("invalid order status")

	// ErrInvalidRequest is returned when request is invalid
	ErrInvalidRequest = errors.New("invalid request")

	// ErrDuplicateClientOrderID is returned when client order ID is duplicate
	ErrDuplicateClientOrderID = errors.New("duplicate client order ID")

	// ErrRateLimitExceeded is returned when rate limit is exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrServiceUnavailable is returned when service is unavailable
	ErrServiceUnavailable = errors.New("service unavailable")

	// Batch processing errors
	ErrBatchChannelFull      = errors.New("batch channel is full")
	ErrInvalidOperationType  = errors.New("invalid operation type")
	ErrBatchProcessingFailed = errors.New("batch processing failed")

	// Validation errors
	ErrMissingUserID            = errors.New("missing user ID")
	ErrMissingAccountID         = errors.New("missing account ID")
	ErrMissingSymbol            = errors.New("missing symbol")
	ErrMissingQuantity          = errors.New("missing quantity")
	ErrMissingSide              = errors.New("missing order side")
	ErrMissingType              = errors.New("missing order type")
	ErrMissingOrderType         = errors.New("missing order type")
	ErrMissingPrice             = errors.New("missing price for limit order")
	ErrMissingStopPrice         = errors.New("missing stop price for stop order")
	ErrInvalidQuantity          = errors.New("invalid quantity")
	ErrInvalidOrderSide         = errors.New("invalid order side")
	ErrInvalidOrderType         = errors.New("invalid order type")
	ErrInvalidPricePrecision    = errors.New("invalid price precision")
	ErrInvalidQuantityPrecision = errors.New("invalid quantity precision")
	ErrOrderSizeExceedsLimit    = errors.New("order size exceeds limit")
	ErrOrderSizeBelowMinimum    = errors.New("order size below minimum")
	ErrOrderValueExceedsLimit   = errors.New("order value exceeds limit")
	ErrPriceBelowMinimum        = errors.New("price below minimum")
	ErrPriceExceedsMaximum      = errors.New("price exceeds maximum")
	ErrExpirationTimeInPast     = errors.New("expiration time is in the past")
	ErrExpirationTimeTooFar     = errors.New("expiration time is too far in the future")
	ErrMarketClosed             = errors.New("market is closed")
	ErrOrderCannotBeUpdated     = errors.New("order cannot be updated")
	ErrOrderCannotBeCancelled   = errors.New("order cannot be cancelled")
	ErrUnauthorizedOrderAccess  = errors.New("unauthorized order access")
	ErrNoFieldsToUpdate         = errors.New("no fields to update")
	ErrQuantityBelowFilled      = errors.New("quantity cannot be below filled quantity")

	// Market data errors
	ErrMarketDataUnavailable = errors.New("market data unavailable")
	ErrInvalidMarketHours    = errors.New("invalid market hours")

	// Risk management errors
	ErrRiskLimitExceeded     = errors.New("risk limit exceeded")
	ErrPositionLimitExceeded = errors.New("position limit exceeded")
	ErrExposureLimitExceeded = errors.New("exposure limit exceeded")
)

// OrderError represents an order-specific error with additional context
type OrderError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	OrderID string                 `json:"order_id,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
	Err     error                  `json:"-"`
}

// Error implements the error interface
func (e *OrderError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *OrderError) Unwrap() error {
	return e.Err
}

// NewOrderError creates a new order error
func NewOrderError(code, message string, err error) *OrderError {
	return &OrderError{
		Code:    code,
		Message: message,
		Err:     err,
		Details: make(map[string]interface{}),
	}
}

// WithOrderID adds order ID to the error
func (e *OrderError) WithOrderID(orderID string) *OrderError {
	e.OrderID = orderID
	return e
}

// WithDetail adds a detail to the error
func (e *OrderError) WithDetail(key string, value interface{}) *OrderError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// Error codes
const (
	ErrCodeOrderNotFound         = "ORDER_NOT_FOUND"
	ErrCodeInvalidOrder          = "INVALID_ORDER"
	ErrCodeInvalidRequest        = "INVALID_REQUEST"
	ErrCodeOrderExists           = "ORDER_EXISTS"
	ErrCodeOrderNotActive        = "ORDER_NOT_ACTIVE"
	ErrCodeOrderExpired          = "ORDER_EXPIRED"
	ErrCodeInsufficientQuantity  = "INSUFFICIENT_QUANTITY"
	ErrCodeInvalidPrice          = "INVALID_PRICE"
	ErrCodeInvalidSymbol         = "INVALID_SYMBOL"
	ErrCodeInvalidSide           = "INVALID_SIDE"
	ErrCodeInvalidType           = "INVALID_TYPE"
	ErrCodeInvalidTimeInForce    = "INVALID_TIME_IN_FORCE"
	ErrCodeClientOrderIDExists   = "CLIENT_ORDER_ID_EXISTS"
	ErrCodeUnauthorized          = "UNAUTHORIZED"
	ErrCodeRateLimitExceeded     = "RATE_LIMIT_EXCEEDED"
	ErrCodeServiceUnavailable    = "SERVICE_UNAVAILABLE"
	ErrCodeBatchChannelFull      = "BATCH_CHANNEL_FULL"
	ErrCodeInvalidOperationType  = "INVALID_OPERATION_TYPE"
	ErrCodeBatchProcessingFailed = "BATCH_PROCESSING_FAILED"
	ErrCodeValidationFailed      = "VALIDATION_FAILED"
	ErrCodeMarketDataUnavailable = "MARKET_DATA_UNAVAILABLE"
	ErrCodeInvalidMarketHours    = "INVALID_MARKET_HOURS"
	ErrCodeRiskLimitExceeded     = "RISK_LIMIT_EXCEEDED"
	ErrCodePositionLimitExceeded = "POSITION_LIMIT_EXCEEDED"
	ErrCodeExposureLimitExceeded = "EXPOSURE_LIMIT_EXCEEDED"
)

// IsRetryableError returns true if the error is retryable
func IsRetryableError(err error) bool {
	switch err {
	case ErrServiceUnavailable, ErrBatchChannelFull, ErrRateLimitExceeded:
		return true
	default:
		return false
	}
}

// IsValidationError returns true if the error is a validation error
func IsValidationError(err error) bool {
	switch err {
	case ErrInvalidOrder, ErrInvalidOrderRequest, ErrInvalidPrice,
		ErrInvalidSymbol, ErrInvalidSide, ErrInvalidType,
		ErrInvalidTimeInForce, ErrMissingUserID, ErrMissingAccountID,
		ErrMissingSymbol, ErrMissingQuantity, ErrMissingSide, ErrMissingType:
		return true
	default:
		return false
	}
}

// IsBusinessError returns true if the error is a business logic error
func IsBusinessError(err error) bool {
	switch err {
	case ErrOrderNotActive, ErrOrderExpired, ErrInsufficientQuantity,
		ErrClientOrderIDExists, ErrRiskLimitExceeded,
		ErrPositionLimitExceeded, ErrExposureLimitExceeded:
		return true
	default:
		return false
	}
}
