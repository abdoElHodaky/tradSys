// Package types provides canonical trading type definitions for the TradSys public API.
// These types serve as the single source of truth for trading operations across all packages.
package types

import "time"

// OrderSide is available in pkg/types

// OrderType is available in pkg/types

// OrderStatus is available in pkg/types

// Order is available in pkg/types

// Trade is defined in common_core.go to avoid duplication

// TradeFeesInfo is available in pkg/types

// OrderValidationError is available in pkg/types

// IsValidOrderSide checks if the given string is a valid order side
func IsValidOrderSide(side string) bool {
	return side == string(OrderSideBuy) || side == string(OrderSideSell)
}

// IsValidOrderType checks if the given string is a valid order type
func IsValidOrderType(orderType string) bool {
	switch OrderType(orderType) {
	case OrderTypeMarket, OrderTypeLimit, OrderTypeStop, OrderTypeStopLimit, OrderTypeStopMarket:
		return true
	default:
		return false
	}
}

// IsValidOrderStatus checks if the given string is a valid order status
func IsValidOrderStatus(status string) bool {
	switch OrderStatus(status) {
	case OrderStatusNew, OrderStatusPartiallyFilled, OrderStatusFilled, 
		 OrderStatusCanceled, OrderStatusCancelled, OrderStatusRejected, OrderStatusExpired:
		return true
	default:
		return false
	}
}
