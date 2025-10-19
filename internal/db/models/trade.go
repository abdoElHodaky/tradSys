package models

import (
	"time"
	"gorm.io/gorm"
)

// Trade represents a trade in the database
type Trade struct {
	gorm.Model
	ID                  string    `gorm:"primaryKey;type:uuid"`
	OrderID             string    `gorm:"index"`
	Symbol              string    `gorm:"index"`
	Side                string
	Price               float64
	Quantity            float64
	ExecutedAt          time.Time `gorm:"index"`
	Fee                 float64
	FeeCurrency         string
	CounterPartyOrderID string
	Metadata            string `gorm:"type:jsonb"`
}

// TableName returns the table name for the Trade model
func (Trade) TableName() string {
	return "trades"
}
