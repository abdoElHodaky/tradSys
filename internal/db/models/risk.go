package models

import (
	"time"

	"gorm.io/gorm"
)

// RiskLimit represents risk limits for an account
type RiskLimit struct {
	gorm.Model
	AccountID     string  `gorm:"index:idx_risk_account_symbol;not null"`
	Symbol        string  `gorm:"index:idx_risk_account_symbol;not null"`
	MaxPosition   float64 `gorm:"not null"`
	MaxOrderSize  float64 `gorm:"not null"`
	MaxDailyLoss  float64 `gorm:"not null"`
	CurrentDailyLoss float64
	Active        bool    `gorm:"not null;default:true"`
}

// CircuitBreaker represents a circuit breaker for a symbol
type CircuitBreaker struct {
	gorm.Model
	Symbol      string    `gorm:"uniqueIndex;not null"`
	Triggered   bool      `gorm:"not null;default:false"`
	Reason      string
	TriggerTime time.Time
	ResetTime   time.Time
}

// RiskCheck represents a record of a risk check
type RiskCheck struct {
	gorm.Model
	OrderID     string    `gorm:"index;not null"`
	AccountID   string    `gorm:"index;not null"`
	Symbol      string    `gorm:"index;not null"`
	Approved    bool      `gorm:"not null"`
	Reason      string
	CheckTime   time.Time `gorm:"not null"`
}

