package models

import (
	"time"

	"gorm.io/gorm"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "NEW"
	OrderStatusAccepted  OrderStatus = "ACCEPTED"
	OrderStatusRejected  OrderStatus = "REJECTED"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusPartial   OrderStatus = "PARTIAL"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// OrderType represents the type of an order
type OrderType string

const (
	OrderTypeMarket    OrderType = "MARKET"
	OrderTypeLimit     OrderType = "LIMIT"
	OrderTypeStop      OrderType = "STOP"
	OrderTypeStopLimit OrderType = "STOP_LIMIT"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
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
	ClientID   string      `gorm:"index"`
	Status     OrderStatus `gorm:"not null"`
	FilledQty  float64     `gorm:"default:0"`
	AvgPrice   float64     `gorm:"default:0"`
	Exchange   string
	ExternalID string
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
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

