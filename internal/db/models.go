package db

import (
	"time"

	"gorm.io/gorm"
)

// Order represents an order in the database
type Order struct {
	gorm.Model
	ID             string `gorm:"primaryKey;type:uuid"`
	UserID         string `gorm:"index"`
	ClientOrderID  string `gorm:"index"`
	Symbol         string `gorm:"index"`
	Side           string `gorm:"index"`
	Type           string `gorm:"index"`
	Price          float64
	StopPrice      float64
	Quantity       float64
	FilledQuantity float64
	Status         string `gorm:"index"`
	TimeInForce    string
	ExpiresAt      time.Time
	Trades         []Trade `gorm:"foreignKey:OrderID"`
	Metadata       string  `gorm:"type:jsonb"`
}

// Trade represents a trade in the database
type Trade struct {
	gorm.Model
	ID                  string `gorm:"primaryKey;type:uuid"`
	OrderID             string `gorm:"index"`
	Symbol              string `gorm:"index"`
	Side                string
	Price               float64
	Quantity            float64
	ExecutedAt          time.Time `gorm:"index"`
	Fee                 float64
	FeeCurrency         string
	CounterPartyOrderID string
	Metadata            string `gorm:"type:jsonb"`
}

// Position represents a position in the database
type Position struct {
	gorm.Model
	UserID            string `gorm:"index"`
	Symbol            string `gorm:"index"`
	Quantity          float64
	AverageEntryPrice float64
	UnrealizedPnL     float64
	RealizedPnL       float64
	LastUpdated       time.Time
}

// RiskLimit represents a risk limit in the database
type RiskLimit struct {
	gorm.Model
	ID      string `gorm:"primaryKey;type:uuid"`
	UserID  string `gorm:"index"`
	Symbol  string
	Type    string `gorm:"index"`
	Value   float64
	Enabled bool
}

// MarketData represents market data in the database
type MarketData struct {
	gorm.Model
	Symbol    string `gorm:"index"`
	Type      string `gorm:"index"`
	Price     float64
	Volume    float64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Timestamp time.Time `gorm:"index"`
	Data      string    `gorm:"type:jsonb"`
}

// CircuitBreaker represents a circuit breaker in the database
type CircuitBreaker struct {
	gorm.Model
	Symbol              string `gorm:"primaryKey"`
	PercentageThreshold float64
	TimeWindow          int64 // stored in seconds
	CooldownPeriod      int64 // stored in seconds
	LastTriggered       time.Time
	Triggered           bool
	ReferencePrice      float64
}

// TableName returns the table name for the Order model
func (Order) TableName() string {
	return "orders"
}

// TableName returns the table name for the Trade model
func (Trade) TableName() string {
	return "trades"
}

// TableName returns the table name for the Position model
func (Position) TableName() string {
	return "positions"
}

// TableName returns the table name for the RiskLimit model
func (RiskLimit) TableName() string {
	return "risk_limits"
}

// TableName returns the table name for the MarketData model
func (MarketData) TableName() string {
	return "market_data"
}

// TableName returns the table name for the CircuitBreaker model
func (CircuitBreaker) TableName() string {
	return "circuit_breakers"
}
