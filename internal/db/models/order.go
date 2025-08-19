package models

import (
	"time"

	"gorm.io/gorm"
)

// OrderStatus represents the status of an order
type OrderStatus string

// Order status constants
const (
	OrderStatusNew       OrderStatus = "NEW"       // Order has been created but not processed
	OrderStatusAccepted  OrderStatus = "ACCEPTED"  // Order has been accepted by the system
	OrderStatusRejected  OrderStatus = "REJECTED"  // Order has been rejected
	OrderStatusFilled    OrderStatus = "FILLED"    // Order has been completely filled
	OrderStatusPartial   OrderStatus = "PARTIAL"   // Order has been partially filled
	OrderStatusCancelled OrderStatus = "CANCELLED" // Order has been cancelled
)

// OrderType represents the type of an order
type OrderType string

// Order type constants
const (
	OrderTypeMarket    OrderType = "MARKET"     // Market order - executed at current market price
	OrderTypeLimit     OrderType = "LIMIT"      // Limit order - executed at specified price or better
	OrderTypeStop      OrderType = "STOP"       // Stop order - becomes market order when price reaches stop price
	OrderTypeStopLimit OrderType = "STOP_LIMIT" // Stop-limit order - becomes limit order when price reaches stop price
)

// OrderSide represents the side of an order
type OrderSide string

// Order side constants
const (
	OrderSideBuy  OrderSide = "BUY"  // Buy order
	OrderSideSell OrderSide = "SELL" // Sell order
)

// Order represents an order in the system
type Order struct {
	gorm.Model
	OrderID    string      `gorm:"uniqueIndex;not null"`
	Symbol     string      `gorm:"index;not null"`
	Type       OrderType   `gorm:"not null"`
	Side       OrderSide   `gorm:"not null"`
	Quantity   float64     `gorm:"not null"`
	Price      float64
//<<<<<<< codegen-bot/pairs-management-implementation
	StopLoss   float64     `gorm:"default:0"` // Stop loss price level
	TakeProfit float64     `gorm:"default:0"` // Take profit price level
//=======
//>>>>>>> main
	ClientID   string      `gorm:"index"`
	Status     OrderStatus `gorm:"not null"`
	FilledQty  float64     `gorm:"default:0"`
	AvgPrice   float64     `gorm:"default:0"`
	Exchange   string
	ExternalID string
//<<<<<<< codegen-bot/pairs-management-implementation
	Strategy   string      `gorm:"index"` // Strategy that generated this order
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Timestamp  time.Time   `gorm:"index"` // Time when the order was created by the strategy
//=======
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
//>>>>>>> main
}

// Trade represents an executed trade
type Trade struct {
	gorm.Model
	TradeID   string    `gorm:"uniqueIndex;not null"`
	OrderID   string    `gorm:"index;not null"`
	Symbol    string    `gorm:"index;not null"`
	Side      OrderSide `gorm:"not null"`
	Quantity  float64   `gorm:"not null"`
	Price     float64   `gorm:"not null"`
	Timestamp time.Time `gorm:"index;not null"`
	Exchange  string
	Fee       float64
	FeeCcy    string
}

// Position represents a trading position
type Position struct {
	gorm.Model
	AccountID     string    `gorm:"index;not null"`
	Symbol        string    `gorm:"index;not null"`
	Quantity      float64   `gorm:"not null"`
	AveragePrice  float64   `gorm:"not null"`
	UnrealizedPnL float64
	RealizedPnL   float64
	LastUpdated   time.Time `gorm:"index;not null"`
}
