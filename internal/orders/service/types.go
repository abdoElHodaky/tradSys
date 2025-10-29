package service

import (
	"time"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	// OrderStatusNew represents a new order
	OrderStatusNew OrderStatus = "new"
	// OrderStatusPending represents a pending order
	OrderStatusPending OrderStatus = "pending"
	// OrderStatusPartiallyFilled represents a partially filled order
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	// OrderStatusFilled represents a filled order
	OrderStatusFilled OrderStatus = "filled"
	// OrderStatusCancelled represents a cancelled order
	OrderStatusCancelled OrderStatus = "cancelled"
	// OrderStatusRejected represents a rejected order
	OrderStatusRejected OrderStatus = "rejected"
	// OrderStatusExpired represents an expired order
	OrderStatusExpired OrderStatus = "expired"
)

// OrderType represents the type of order
type OrderType string

const (
	// OrderTypeLimit represents a limit order
	OrderTypeLimit OrderType = "limit"
	// OrderTypeMarket represents a market order
	OrderTypeMarket OrderType = "market"
	// OrderTypeStopLimit represents a stop limit order
	OrderTypeStopLimit OrderType = "stop_limit"
	// OrderTypeStopMarket represents a stop market order
	OrderTypeStopMarket OrderType = "stop_market"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	// OrderSideBuy represents a buy order
	OrderSideBuy OrderSide = "buy"
	// OrderSideSell represents a sell order
	OrderSideSell OrderSide = "sell"
)

// TimeInForce represents the time in force of an order
type TimeInForce string

const (
	// TimeInForceGTC represents a good-till-cancelled order
	TimeInForceGTC TimeInForce = "GTC"
	// TimeInForceIOC represents an immediate-or-cancel order
	TimeInForceIOC TimeInForce = "IOC"
	// TimeInForceFOK represents a fill-or-kill order
	TimeInForceFOK TimeInForce = "FOK"
	// TimeInForceDay represents a day order
	TimeInForceDay TimeInForce = "DAY"
)

// Order represents an order
type Order struct {
	// ID is the unique identifier for the order
	ID string
	// UserID is the user ID
	UserID string
	// ClientOrderID is the client order ID
	ClientOrderID string
	// Symbol is the trading symbol
	Symbol string
	// Side is the side of the order (buy or sell)
	Side OrderSide
	// Type is the type of the order
	Type OrderType
	// Price is the price of the order
	Price float64
	// StopPrice is the stop price for stop orders
	StopPrice float64
	// Quantity is the quantity of the order
	Quantity float64
	// FilledQuantity is the filled quantity of the order
	FilledQuantity float64
	// Status is the status of the order
	Status OrderStatus
	// TimeInForce is the time in force of the order
	TimeInForce TimeInForce
	// CreatedAt is the time the order was created
	CreatedAt time.Time
	// UpdatedAt is the time the order was last updated
	UpdatedAt time.Time
	// ExpiresAt is the time the order expires
	ExpiresAt time.Time
	// Trades is the trades associated with the order
	Trades []*Trade
	// Metadata is additional metadata for the order
	Metadata map[string]interface{}
}

// Trade represents a trade
type Trade struct {
	// ID is the unique identifier for the trade
	ID string
	// OrderID is the order ID
	OrderID string
	// Symbol is the trading symbol
	Symbol string
	// Side is the side of the trade (buy or sell)
	Side OrderSide
	// Price is the price of the trade
	Price float64
	// Quantity is the quantity of the trade
	Quantity float64
	// ExecutedAt is the time the trade was executed
	ExecutedAt time.Time
	// Fee is the fee for the trade
	Fee float64
	// FeeCurrency is the currency of the fee
	FeeCurrency string
	// CounterPartyOrderID is the counter party order ID
	CounterPartyOrderID string
	// Metadata is additional metadata for the trade
	Metadata map[string]interface{}
}

// OrderFilter represents a filter for orders
type OrderFilter struct {
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Side is the side of the order (buy or sell)
	Side OrderSide
	// Type is the type of the order
	Type OrderType
	// Status is the status of the order
	Status OrderStatus
	// StartTime is the start time for the filter
	StartTime time.Time
	// EndTime is the end time for the filter
	EndTime time.Time
}

// OrderRequest represents an order request
type OrderRequest struct {
	// UserID is the user ID
	UserID string
	// ClientOrderID is the client order ID
	ClientOrderID string
	// Symbol is the trading symbol
	Symbol string
	// Side is the side of the order (buy or sell)
	Side OrderSide
	// Type is the type of the order
	Type OrderType
	// Price is the price of the order
	Price float64
	// StopPrice is the stop price for stop orders
	StopPrice float64
	// Quantity is the quantity of the order
	Quantity float64
	// TimeInForce is the time in force of the order
	TimeInForce TimeInForce
	// ExpiresAt is the time the order expires
	ExpiresAt time.Time
	// Metadata is additional metadata for the order
	Metadata map[string]interface{}
}

// OrderCancelRequest represents an order cancel request
type OrderCancelRequest struct {
	// UserID is the user ID
	UserID string
	// OrderID is the order ID
	OrderID string
	// ClientOrderID is the client order ID
	ClientOrderID string
	// Symbol is the trading symbol
	Symbol string
}

// OrderUpdateRequest represents an order update request
type OrderUpdateRequest struct {
	// UserID is the user ID
	UserID string
	// OrderID is the order ID
	OrderID string
	// ClientOrderID is the client order ID
	ClientOrderID string
	// Symbol is the trading symbol
	Symbol string
	// Price is the price of the order
	Price float64
	// StopPrice is the stop price for stop orders
	StopPrice float64
	// Quantity is the quantity of the order
	Quantity float64
	// TimeInForce is the time in force of the order
	TimeInForce TimeInForce
	// ExpiresAt is the time the order expires
	ExpiresAt time.Time
}

// orderOperation represents a batch operation on orders
type orderOperation struct {
	opType    string
	order     *Order
	requestID string
	resultCh  chan orderOperationResult
}

// orderOperationResult represents the result of a batch operation
type orderOperationResult struct {
	order *Order
	err   error
}
