package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderStatus represents the status of an order
type OrderStatus string

// Order statuses
const (
	OrderStatusNew        OrderStatus = "new"
	OrderStatusPartial    OrderStatus = "partial"
	OrderStatusFilled     OrderStatus = "filled"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRejected   OrderStatus = "rejected"
	OrderStatusExpired    OrderStatus = "expired"
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
)

// OrderSide represents the side of an order
type OrderSide string

// Order sides
const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of an order
type OrderType string

// Order types
const (
	OrderTypeMarket     OrderType = "market"
	OrderTypeLimit      OrderType = "limit"
	OrderTypeStop       OrderType = "stop"
	OrderTypeStopLimit  OrderType = "stop_limit"
	OrderTypeTrailing   OrderType = "trailing"
	OrderTypeIOC        OrderType = "ioc"
	OrderTypeFOK        OrderType = "fok"
	OrderTypeConditional OrderType = "conditional"
)

// Order represents an order in the trading system
type Order struct {
	ID           string      `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       string      `gorm:"type:varchar(36);index" json:"user_id"`
	AccountID    string      `gorm:"type:varchar(36);index" json:"account_id"`
	Symbol       string      `gorm:"type:varchar(20);index" json:"symbol"`
	Side         OrderSide   `gorm:"type:varchar(10);index" json:"side"`
	Type         OrderType   `gorm:"type:varchar(20);index" json:"type"`
	Quantity     float64     `gorm:"type:decimal(20,8)" json:"quantity"`
	Price        float64     `gorm:"type:decimal(20,8)" json:"price"`
	StopPrice    float64     `gorm:"type:decimal(20,8)" json:"stop_price"`
	TrailingOffset float64   `gorm:"type:decimal(20,8)" json:"trailing_offset"`
	TimeInForce  string      `gorm:"type:varchar(10)" json:"time_in_force"`
	Status       OrderStatus `gorm:"type:varchar(20);index" json:"status"`
	FilledQty    float64     `gorm:"type:decimal(20,8)" json:"filled_qty"`
	AvgPrice     float64     `gorm:"type:decimal(20,8)" json:"avg_price"`
	ClientOrderID string     `gorm:"type:varchar(50);index" json:"client_order_id"`
	ExchangeOrderID string   `gorm:"type:varchar(50);index" json:"exchange_order_id"`
	Notes        string      `gorm:"type:text" json:"notes"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	ExpiresAt    *time.Time  `json:"expires_at"`
}

// OrderWithTriggers represents an order with stop loss and take profit triggers
type OrderWithTriggers struct {
	Order
	StopLoss   *float64 `gorm:"-" json:"stop_loss"`
	TakeProfit *float64 `gorm:"-" json:"take_profit"`
}

// Reset resets the order to its zero state for pool reuse
func (o *Order) Reset() {
	o.ID = ""
	o.UserID = ""
	o.AccountID = ""
	o.Symbol = ""
	o.Side = ""
	o.Type = ""
	o.Quantity = 0
	o.Price = 0
	o.StopPrice = 0
	o.TrailingOffset = 0
	o.TimeInForce = ""
	o.Status = ""
	o.FilledQty = 0
	o.AvgPrice = 0
	o.ClientOrderID = ""
	o.ExchangeOrderID = ""
	o.Notes = ""
	o.CreatedAt = time.Time{}
	o.UpdatedAt = time.Time{}
	o.ExpiresAt = nil
}

// BeforeCreate is a GORM hook that runs before creating a new order
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

// OrderHistory represents the history of an order
type OrderHistory struct {
	ID        string      `gorm:"primaryKey;type:varchar(36)" json:"id"`
	OrderID   string      `gorm:"type:varchar(36);index" json:"order_id"`
	Status    OrderStatus `gorm:"type:varchar(20)" json:"status"`
	Quantity  float64     `gorm:"type:decimal(20,8)" json:"quantity"`
	Price     float64     `gorm:"type:decimal(20,8)" json:"price"`
	Notes     string      `gorm:"type:text" json:"notes"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// BeforeCreate is a GORM hook that runs before creating a new order history entry
func (oh *OrderHistory) BeforeCreate(tx *gorm.DB) error {
	if oh.ID == "" {
		oh.ID = uuid.New().String()
	}
	return nil
}
