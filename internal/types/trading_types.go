// Package types provides canonical trading type definitions for the TradSys public API.
// These types serve as the single source of truth for trading operations across all packages.
package types

import "time"

// OrderSide represents the side of an order
type OrderSide string

const (
	// OrderSideBuy represents a buy order
	OrderSideBuy OrderSide = "buy"
	// OrderSideSell represents a sell order
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of an order
type OrderType string

const (
	// OrderTypeMarket represents a market order
	OrderTypeMarket OrderType = "market"
	// OrderTypeLimit represents a limit order
	OrderTypeLimit OrderType = "limit"
	// OrderTypeStop represents a stop order
	OrderTypeStop OrderType = "stop"
	// OrderTypeStopLimit represents a stop limit order
	OrderTypeStopLimit OrderType = "stop_limit"
	// OrderTypeStopMarket represents a stop market order
	OrderTypeStopMarket OrderType = "stop_market"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	// OrderStatusNew represents a new order
	OrderStatusNew OrderStatus = "new"
	// OrderStatusPartiallyFilled represents a partially filled order
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	// OrderStatusFilled represents a filled order
	OrderStatusFilled OrderStatus = "filled"
	// OrderStatusCanceled represents a canceled order
	OrderStatusCanceled OrderStatus = "canceled"
	// OrderStatusCancelled represents a cancelled order (alternative spelling)
	OrderStatusCancelled OrderStatus = "cancelled"
	// OrderStatusRejected represents a rejected order
	OrderStatusRejected OrderStatus = "rejected"
	// OrderStatusExpired represents an expired order
	OrderStatusExpired OrderStatus = "expired"
)

// Order represents an order in the trading system
type Order struct {
	// ID is the unique identifier for the order
	ID string `json:"id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// AssetType is the type of asset being traded
	AssetType AssetType `json:"asset_type"`
	// Side is the side of the order (buy or sell)
	Side OrderSide `json:"side"`
	// Type is the type of the order
	Type OrderType `json:"type"`
	// Price is the price of the order
	Price float64 `json:"price"`
	// Quantity is the quantity of the order
	Quantity float64 `json:"quantity"`
	// FilledQuantity is the filled quantity of the order
	FilledQuantity float64 `json:"filled_quantity"`
	// Status is the status of the order
	Status OrderStatus `json:"status"`
	// CreatedAt is the time the order was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the time the order was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// ClientOrderID is the client order ID
	ClientOrderID string `json:"client_order_id"`
	// UserID is the user ID
	UserID string `json:"user_id"`
	// StopPrice is the stop price for stop orders
	StopPrice float64 `json:"stop_price,omitempty"`
	// TimeInForce is the time in force for the order
	TimeInForce string `json:"time_in_force"`
	// Index is the index in the heap
	Index int `json:"-"`

	// Advanced order features
	// DisplayQuantity is the visible quantity for iceberg orders
	DisplayQuantity float64 `json:"display_quantity,omitempty"`
	// IsHidden indicates if this is a hidden order
	IsHidden bool `json:"is_hidden,omitempty"`
	// IsPriceImproved indicates if price improvement was applied
	IsPriceImproved bool `json:"is_price_improved,omitempty"`
	// EstimatedImpact is the estimated market impact
	EstimatedImpact float64 `json:"estimated_impact,omitempty"`
	// ParentOrderID is the parent order ID for child orders
	ParentOrderID string `json:"parent_order_id,omitempty"`
	// IsIcebergChild indicates if this is a child of an iceberg order
	IsIcebergChild bool `json:"is_iceberg_child,omitempty"`
	// Priority is the order priority for matching
	Priority int64 `json:"priority,omitempty"`
}

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

// TradeFeesInfo contains fee information for a trade
type TradeFeesInfo struct {
	// BuyerFee is the fee charged to the buyer
	BuyerFee float64 `json:"buyer_fee"`
	// SellerFee is the fee charged to the seller
	SellerFee float64 `json:"seller_fee"`
	// FeeCurrency is the currency in which fees are charged
	FeeCurrency string `json:"fee_currency"`
}

// OrderValidationError represents an order validation error
type OrderValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Error implements the error interface
func (e OrderValidationError) Error() string {
	return e.Message
}

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
