package models

import (
	"time"

	"gorm.io/gorm"
)

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

// TableName returns the table name for the Position model
func (Position) TableName() string {
	return "positions"
}
