package orders

import (
	"errors"
	"time"
)

// KISS: Simple type definitions - no external dependencies needed
// DRY: Types used across the orders package

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "new"
	OrderStatusPending         OrderStatus = "pending"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusCancelled       OrderStatus = "cancelled"
	OrderStatusRejected        OrderStatus = "rejected"
	OrderStatusExpired         OrderStatus = "expired"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeLimit     OrderType = "limit"
	OrderTypeMarket    OrderType = "market"
	OrderTypeStop      OrderType = "stop"
	OrderTypeStopLimit OrderType = "stop_limit"
	OrderTypeStopMarket OrderType = "stop_market"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// TimeInForce represents the time in force of an order
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC"
	TimeInForceIOC TimeInForce = "IOC"
	TimeInForceFOK TimeInForce = "FOK"
	TimeInForceDay TimeInForce = "DAY"
)

// Order represents an order (DRY: single definition)
type Order struct {
	ID              string
	UserID          string
	ClientOrderID   string
	Symbol          string
	Side            OrderSide
	Type            OrderType
	Price           float64
	StopPrice       float64
	Quantity        float64
	FilledQuantity  float64
	Status          OrderStatus
	TimeInForce     TimeInForce
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ExpiresAt       time.Time
	Trades          []*Trade
	Metadata        map[string]any
}

// Trade represents a trade
type Trade struct {
	ID                  string
	OrderID             string
	Symbol              string
	Side                OrderSide
	Price               float64
	Quantity            float64
	ExecutedAt          time.Time
	Fee                 float64
	FeeCurrency         string
	CounterPartyOrderID string
	Metadata            map[string]any
}

// OrderFilter represents a filter for orders
type OrderFilter struct {
	UserID    string
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Status    OrderStatus
	StartTime time.Time
	EndTime   time.Time
}

// OrderRequest represents an order request
type OrderRequest struct {
	UserID        string
	ClientOrderID string
	Symbol        string
	Side          OrderSide
	Type          OrderType
	Price         float64
	StopPrice     float64
	Quantity      float64
	TimeInForce   TimeInForce
	ExpiresAt     time.Time
	Metadata      map[string]any
}

// OrderCancelRequest represents an order cancel request
type OrderCancelRequest struct {
	UserID        string
	OrderID       string
	ClientOrderID string
	Symbol        string
}

// OrderUpdateRequest represents an order update request
type OrderUpdateRequest struct {
	UserID        string
	OrderID       string
	ClientOrderID string
	Symbol        string
	Price         float64
	StopPrice     float64
	Quantity      float64
	TimeInForce   TimeInForce
	ExpiresAt     time.Time
}

// OrderCarrier interface for polymorphic order operations (SOLID: Interface Segregation)
type OrderCarrier interface {
	GetOrder() *Order
	GetOrderID() string
	Validate() error
}

// GetOrder returns itself for OrderCarrier compatibility
func (o *Order) GetOrder() *Order { return o }

// GetOrderID returns the order's ID
func (o *Order) GetOrderID() string { return o.ID }

// Validate validates the order's fields
func (o *Order) Validate() error {
	if o.ID == "" {
		return errors.New("order ID is required")
	}
	if o.Symbol == "" {
		return errors.New("invalid symbol")
	}
	if o.Quantity <= 0 {
		return errors.New("invalid quantity")
	}
	if o.Price <= 0 && (o.Type == OrderTypeLimit || o.Type == OrderTypeStopLimit) {
		return errors.New("invalid price")
	}
	return nil
}

// IsIceberg returns true if this is an iceberg order
func (o *Order) IsIceberg() bool { return false }

// RemainingQuantity returns the remaining quantity to be filled
func (o *Order) RemainingQuantity() float64 {
	return o.Quantity - o.FilledQuantity
}

// IsFilled returns true if the order is completely filled
func (o *Order) IsFilled() bool {
	return o.FilledQuantity >= o.Quantity
}