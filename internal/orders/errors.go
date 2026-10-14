package orders

import "errors"

// Base errors for order operations - used for polymorphism and encapsulation
// These errors can be wrapped or compared using errors.Is for type checking
// DRY: Centralized error definitions

var (
	// Order errors (use from trading/types/base.go where applicable)
	ErrInvalidOrder         = errors.New("invalid order")
	ErrInvalidOrderStatus   = errors.New("invalid order status")
	ErrOrderNotFound        = errors.New("order not found")
	ErrOrderAlreadyExists   = errors.New("order already exists")
	ErrInvalidOrderRequest  = errors.New("invalid order request")

	// Validation errors (use from trading/types/base.go where applicable)
	ErrMissingUserID          = errors.New("missing user ID")
	ErrMissingSymbol          = errors.New("missing symbol")
	ErrMissingSide            = errors.New("missing order side")
	ErrMissingOrderType       = errors.New("missing order type")
	ErrMissingPrice           = errors.New("missing price")
	ErrMissingQuantity        = errors.New("missing quantity")
	ErrMissingStopPrice       = errors.New("missing stop price")
	ErrInvalidQuantity        = errors.New("invalid quantity")
	ErrInvalidPrice           = errors.New("invalid price")
	ErrInvalidSymbol          = errors.New("invalid symbol")
	ErrInvalidOrderSide       = errors.New("invalid order side")
	ErrInvalidOrderType       = errors.New("invalid order type")
	ErrInvalidStopPrice       = errors.New("invalid stop price")
	ErrInvalidTimeInForce     = errors.New("invalid time in force")
	ErrExpirationTimeInPast   = errors.New("expiration time is in the past")
	ErrExpirationTimeTooFar   = errors.New("expiration time is too far in the future")
	ErrMarketClosed           = errors.New("market is closed")
	ErrOrderSizeBelowMinimum  = errors.New("order size below minimum")
	ErrOrderSizeExceedsLimit  = errors.New("order size exceeds limit")
	ErrOrderValueExceedsLimit = errors.New("order value exceeds limit")
	ErrPriceBelowMinimum      = errors.New("price below minimum")
	ErrPriceExceedsMaximum    = errors.New("price exceeds maximum")
	ErrInvalidPricePrecision        = errors.New("invalid price precision")
	ErrInvalidQuantityPrecision     = errors.New("invalid quantity precision")
	ErrTimeInForce                = errors.New("invalid time in force")

	// Request errors
	ErrInvalidRequest          = errors.New("invalid request")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrDuplicateClientOrderID  = errors.New("duplicate client order ID")
	ErrOrderCannotBeUpdated    = errors.New("order cannot be updated")
	ErrOrderCannotBeCancelled  = errors.New("order cannot be cancelled")
	ErrUnauthorizedOrderAccess = errors.New("unauthorized order access")
	ErrNoFieldsToUpdate        = errors.New("no fields to update")
	ErrQuantityBelowFilled     = errors.New("quantity cannot be below filled quantity")
)