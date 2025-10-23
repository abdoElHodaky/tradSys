package models

import (
	"time"

	"gorm.io/gorm"
)

// PairStatus represents the status of a trading pair
type PairStatus string

const (
	PairStatusActive     PairStatus = "active"
	PairStatusInactive   PairStatus = "inactive"
	PairStatusMonitoring PairStatus = "monitoring"
)

// Pair represents a trading pair for statistical arbitrage
type Pair struct {
	gorm.Model
	PairID               string     `gorm:"uniqueIndex;not null"`
	Symbol1              string     `gorm:"index;not null"` // First instrument in the pair
	Symbol2              string     `gorm:"index;not null"` // Second instrument in the pair
	Ratio                float64    // Trading ratio between instruments
	Status               PairStatus `gorm:"not null"`
	Correlation          float64    // Current correlation coefficient
	Cointegration        float64    // Cointegration test statistic
	ZScoreThresholdEntry float64    // Z-score threshold for entry
	ZScoreThresholdExit  float64    // Z-score threshold for exit
	LookbackPeriod       int        `gorm:"not null"` // Period for statistical calculations
	HalfLife             int        // Half-life of mean reversion
	CreatedBy            uint       `gorm:"index"` // User who created the pair
	Notes                string
}

// PairStatistics represents statistical data for a pair
type PairStatistics struct {
	gorm.Model
	PairID        string    `gorm:"index;not null"`
	Timestamp     time.Time `gorm:"index"`
	Correlation   float64
	Cointegration float64
	SpreadMean    float64
	SpreadStdDev  float64
	CurrentZScore float64
	SpreadValue   float64
}

// PairPosition represents an open position in a pair
type PairPosition struct {
	gorm.Model
	PairID         string `gorm:"index;not null"`
	EntryTimestamp time.Time
	Symbol1        string
	Symbol2        string
	Quantity1      float64 // Position size in first instrument
	Quantity2      float64 // Position size in second instrument
	EntryPrice1    float64
	EntryPrice2    float64
	CurrentPrice1  float64
	CurrentPrice2  float64
	EntrySpread    float64
	CurrentSpread  float64
	EntryZScore    float64
	CurrentZScore  float64
	PnL            float64 // Current profit/loss
	Status         string  // "open" or "closed"
	ExitTimestamp  time.Time
}
