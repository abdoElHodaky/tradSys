package orders

import (
	"time"

	"github.com/google/uuid"
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
	// TimeInForceGTC represents Good Till Cancelled
	TimeInForceGTC TimeInForce = "GTC"
	// TimeInForceIOC represents Immediate Or Cancel
	TimeInForceIOC TimeInForce = "IOC"
	// TimeInForceFOK represents Fill Or Kill
	TimeInForceFOK TimeInForce = "FOK"
	// TimeInForceDAY represents Day order
	TimeInForceDAY TimeInForce = "DAY"
)

// Order represents a trading order
type Order struct {
	ID                string      `json:"id"`
	ClientOrderID     string      `json:"client_order_id"`
	Symbol            string      `json:"symbol"`
	Side              OrderSide   `json:"side"`
	Type              OrderType   `json:"type"`
	Quantity          float64     `json:"quantity"`
	Price             float64     `json:"price"`
	StopPrice         float64     `json:"stop_price,omitempty"`
	TimeInForce       TimeInForce `json:"time_in_force"`
	Status            OrderStatus `json:"status"`
	FilledQuantity    float64     `json:"filled_quantity"`
	RemainingQuantity float64     `json:"remaining_quantity"`
	AveragePrice      float64     `json:"average_price"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	ExpiresAt         *time.Time  `json:"expires_at,omitempty"`
	UserID            string      `json:"user_id"`
	AccountID         string      `json:"account_id"`
	
	// Internal fields
	OriginalQuantity float64 `json:"original_quantity"`
	LastTradePrice   float64 `json:"last_trade_price"`
	LastTradeTime    *time.Time `json:"last_trade_time,omitempty"`
	
	// Trades associated with this order
	Trades []*Trade `json:"trades,omitempty"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Trade represents a trade execution
type Trade struct {
	ID           string    `json:"id"`
	OrderID      string    `json:"order_id"`
	Symbol       string    `json:"symbol"`
	Side         OrderSide `json:"side"`
	Quantity     float64   `json:"quantity"`
	Price        float64   `json:"price"`
	Fee          float64   `json:"fee"`
	FeeAsset     string    `json:"fee_asset"`
	Timestamp    time.Time `json:"timestamp"`
	IsMaker      bool      `json:"is_maker"`
	
	// Execution information
	ExecutedAt time.Time `json:"executed_at"`
	
	// Fee information
	FeeCurrency string `json:"fee_currency"`
	
	// Counterparty information
	CounterOrderID      string `json:"counter_order_id,omitempty"`
	CounterPartyOrderID string `json:"counter_party_order_id,omitempty"`
	
	// Settlement information
	SettlementStatus string `json:"settlement_status"`
	SettledAt        *time.Time `json:"settled_at,omitempty"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// OrderFilter represents filters for querying orders
type OrderFilter struct {
	UserID      string       `json:"user_id,omitempty"`
	AccountID   string       `json:"account_id,omitempty"`
	Symbol      string       `json:"symbol,omitempty"`
	Side        *OrderSide   `json:"side,omitempty"`
	Type        *OrderType   `json:"type,omitempty"`
	Status      *OrderStatus `json:"status,omitempty"`
	StartTime   *time.Time   `json:"start_time,omitempty"`
	EndTime     *time.Time   `json:"end_time,omitempty"`
	Limit       int          `json:"limit,omitempty"`
	Offset      int          `json:"offset,omitempty"`
}

// OrderRequest represents a request to place an order
type OrderRequest struct {
	ClientOrderID string      `json:"client_order_id,omitempty"`
	Symbol        string      `json:"symbol" validate:"required"`
	Side          OrderSide   `json:"side" validate:"required"`
	Type          OrderType   `json:"type" validate:"required"`
	Quantity      float64     `json:"quantity" validate:"required,gt=0"`
	Price         float64     `json:"price,omitempty"`
	StopPrice     float64     `json:"stop_price,omitempty"`
	TimeInForce   TimeInForce `json:"time_in_force,omitempty"`
	UserID        string      `json:"user_id" validate:"required"`
	AccountID     string      `json:"account_id" validate:"required"`
	ExpiresAt     *time.Time  `json:"expires_at,omitempty"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// OrderCancelRequest represents a request to cancel an order
type OrderCancelRequest struct {
	OrderID       string `json:"order_id,omitempty"`
	ClientOrderID string `json:"client_order_id,omitempty"`
	UserID        string `json:"user_id" validate:"required"`
	AccountID     string `json:"account_id" validate:"required"`
	Symbol        string `json:"symbol,omitempty"`
}

// OrderUpdateRequest represents a request to update an order
type OrderUpdateRequest struct {
	OrderID       string  `json:"order_id,omitempty"`
	ClientOrderID string  `json:"client_order_id,omitempty"`
	UserID        string  `json:"user_id" validate:"required"`
	AccountID     string  `json:"account_id" validate:"required"`
	Quantity      float64     `json:"quantity,omitempty"`
	Price         float64     `json:"price,omitempty"`
	StopPrice     float64     `json:"stop_price,omitempty"`
	TimeInForce   TimeInForce `json:"time_in_force,omitempty"`
	ExpiresAt     *time.Time  `json:"expires_at,omitempty"`
	
	// Metadata updates
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewOrder creates a new order with default values
func NewOrder() *Order {
	return &Order{
		ID:        uuid.New().String(),
		Status:    OrderStatusNew,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// IsActive returns true if the order is in an active state
func (o *Order) IsActive() bool {
	return o.Status == OrderStatusNew || 
		   o.Status == OrderStatusPending || 
		   o.Status == OrderStatusPartiallyFilled
}

// IsFinal returns true if the order is in a final state
func (o *Order) IsFinal() bool {
	return o.Status == OrderStatusFilled || 
		   o.Status == OrderStatusCancelled || 
		   o.Status == OrderStatusRejected || 
		   o.Status == OrderStatusExpired
}

// IsExpired returns true if the order has expired
func (o *Order) IsExpired() bool {
	if o.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*o.ExpiresAt)
}

// UpdateStatus updates the order status and timestamp
func (o *Order) UpdateStatus(status OrderStatus) {
	o.Status = status
	o.UpdatedAt = time.Now()
}

// AddTrade adds a trade to the order and updates quantities
func (o *Order) AddTrade(trade *Trade) {
	o.FilledQuantity += trade.Quantity
	o.RemainingQuantity = o.Quantity - o.FilledQuantity
	o.LastTradePrice = trade.Price
	now := trade.Timestamp
	o.LastTradeTime = &now
	o.UpdatedAt = time.Now()
	
	// Update average price
	if o.FilledQuantity > 0 {
		// Weighted average calculation would go here
		// For simplicity, using last trade price
		o.AveragePrice = trade.Price
	}
	
	// Update status based on fill
	if o.RemainingQuantity <= 0 {
		o.UpdateStatus(OrderStatusFilled)
	} else if o.FilledQuantity > 0 {
		o.UpdateStatus(OrderStatusPartiallyFilled)
	}
}
