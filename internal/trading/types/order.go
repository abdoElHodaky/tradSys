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

	// Advanced order features
	// DisplayQuantity is the visible quantity for iceberg orders
	DisplayQuantity float64
	// IsHidden indicates if this is a hidden order
	IsHidden bool
	// IsPriceImproved indicates if price improvement was applied
	IsPriceImproved bool
	// EstimatedImpact is the estimated market impact
	EstimatedImpact float64
	// ParentOrderID is the parent order ID for child orders
	ParentOrderID string
	// IsIcebergChild indicates if this is a child of an iceberg order
	IsIcebergChild bool
	// Priority is the order priority for matching
	Priority int64
	// MinQuantity is the minimum quantity for execution
	MinQuantity float64
	// MaxFloor is the maximum floor quantity
	MaxFloor float64
	// ExpireTime is the expiration time for the order
	ExpireTime time.Time
	// Tags are custom tags for the order
	Tags map[string]string
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

	// Reset advanced features
	o.DisplayQuantity = 0
	o.IsHidden = false
	o.IsPriceImproved = false
	o.EstimatedImpact = 0
	o.ParentOrderID = ""
	o.IsIcebergChild = false
	o.Priority = 0
	o.MinQuantity = 0
	o.MaxFloor = 0
	o.ExpireTime = time.Time{}
	o.Tags = nil
}

// IsIceberg returns true if this is an iceberg order
func (o *Order) IsIceberg() bool {
	return o.DisplayQuantity > 0 && o.DisplayQuantity < o.Quantity
}

// RemainingQuantity returns the remaining quantity to be filled
func (o *Order) RemainingQuantity() float64 {
	return o.Quantity - o.FilledQuantity
}

// IsFilled returns true if the order is completely filled
func (o *Order) IsFilled() bool {
	return o.FilledQuantity >= o.Quantity
}

// IsPartiallyFilled returns true if the order is partially filled
func (o *Order) IsPartiallyFilled() bool {
	return o.FilledQuantity > 0 && o.FilledQuantity < o.Quantity
}

// IsExpired returns true if the order has expired
func (o *Order) IsExpired() bool {
	return !o.ExpireTime.IsZero() && time.Now().After(o.ExpireTime)
}

// CanMatch returns true if this order can match with another order
func (o *Order) CanMatch(other *Order) bool {
	if o.Symbol != other.Symbol {
		return false
	}
	if o.Side == other.Side {
		return false
	}
	if o.IsExpired() || other.IsExpired() {
		return false
	}

	// Price matching logic
	if o.Side == OrderSideBuy && other.Side == OrderSideSell {
		return o.Price >= other.Price
	}
	if o.Side == OrderSideSell && other.Side == OrderSideBuy {
		return o.Price <= other.Price
	}

	return false
}

// GetEffectiveQuantity returns the effective quantity for matching
func (o *Order) GetEffectiveQuantity() float64 {
	if o.IsIceberg() {
		return o.DisplayQuantity
	}
	return o.RemainingQuantity()
}

// SetTag sets a custom tag on the order
func (o *Order) SetTag(key, value string) {
	if o.Tags == nil {
		o.Tags = make(map[string]string)
	}
	o.Tags[key] = value
}

// GetTag gets a custom tag from the order
func (o *Order) GetTag(key string) (string, bool) {
	if o.Tags == nil {
		return "", false
	}
	value, exists := o.Tags[key]
	return value, exists
}
