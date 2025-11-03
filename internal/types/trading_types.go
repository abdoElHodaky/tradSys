// Package types provides canonical trading type definitions for the TradSys public API.
// These types serve as the single source of truth for trading operations across all packages.
package types

import "time"

// OrderSide is available in pkg/types

// OrderType is available in pkg/types

// OrderStatus is available in pkg/types

// Order is available in pkg/types

// Trade represents a completed trade in the trading system
type Trade struct {
	// ID is the unique identifier for the trade
	ID string `json:"id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// BuyOrderID is the ID of the buy order
	BuyOrderID string `json:"buy_order_id"`
	// SellOrderID is the ID of the sell order
	SellOrderID string `json:"sell_order_id"`
	// Price is the execution price
	Price float64 `json:"price"`
	// Quantity is the traded quantity
	Quantity float64 `json:"quantity"`
	// Timestamp is the time of the trade
	Timestamp time.Time `json:"timestamp"`
	// BuyerID is the buyer user ID
	BuyerID string `json:"buyer_id"`
	// SellerID is the seller user ID
	SellerID string `json:"seller_id"`
	// TakerSide indicates which side was the taker
	TakerSide OrderSide `json:"taker_side"`
	// Fees contains fee information
	Fees TradeFeesInfo `json:"fees,omitempty"`
}

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
