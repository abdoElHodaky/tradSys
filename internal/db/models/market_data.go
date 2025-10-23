package models

import (
	"time"

	"gorm.io/gorm"
)

// Quote represents a market data quote
type Quote struct {
	gorm.Model
	Symbol    string `gorm:"index:idx_quote_symbol_exchange;not null"`
	Exchange  string `gorm:"index:idx_quote_symbol_exchange;not null"`
	Bid       float64
	Ask       float64
	BidSize   float64
	AskSize   float64
	Timestamp time.Time `gorm:"index;not null"`
}

// OHLCV represents Open-High-Low-Close-Volume data
type OHLCV struct {
	gorm.Model
	Symbol    string `gorm:"index:idx_ohlcv_symbol_timeframe;not null"`
	Timeframe string `gorm:"index:idx_ohlcv_symbol_timeframe;not null"`
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp time.Time `gorm:"index;not null"`
}

// MarketDepth represents a level in the order book
type MarketDepth struct {
	gorm.Model
	Symbol    string `gorm:"index:idx_depth_symbol_exchange;not null"`
	Exchange  string `gorm:"index:idx_depth_symbol_exchange;not null"`
	Level     int    `gorm:"not null"`
	BidPrice  float64
	BidSize   float64
	AskPrice  float64
	AskSize   float64
	Timestamp time.Time `gorm:"index;not null"`
}
