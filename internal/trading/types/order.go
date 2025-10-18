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
	ID string
	// Symbol is the trading symbol
	Symbol string
	// Side is the side of the order (buy or sell)
	Side OrderSide
	// Type is the type of the order
	Type OrderType
	// Price is the price of the order
	Price float64
	// Quantity is the quantity of the order
	Quantity float64
	// FilledQuantity is the filled quantity of the order
	FilledQuantity float64
	// Status is the status of the order
	Status OrderStatus
	// CreatedAt is the time the order was created
	CreatedAt time.Time
	// UpdatedAt is the time the order was last updated
	UpdatedAt time.Time
	// ClientOrderID is the client order ID
	ClientOrderID string
	// UserID is the user ID
	UserID string
	// StopPrice is the stop price for stop orders
	StopPrice float64
	// TimeInForce is the time in force for the order
	TimeInForce string
	// Index is the index in the heap
	Index int
}

// Reset resets the Order struct for object pooling
func (o *Order) Reset() {
	o.ID = ""
	o.Symbol = ""
	o.Side = ""
	o.Type = ""
	o.Price = 0
	o.Quantity = 0
	o.FilledQuantity = 0
	o.Status = ""
	o.CreatedAt = time.Time{}
	o.UpdatedAt = time.Time{}
	o.ClientOrderID = ""
	o.UserID = ""
	o.StopPrice = 0
	o.TimeInForce = ""
	o.Index = 0
}
