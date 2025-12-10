// Package types provides unified types for all TradSys v3 services
package types

import (
	"time"
)









// MarketDataType represents different types of market data
type MarketDataType int

const (
	MarketDataTypeTick MarketDataType = iota
	MarketDataTypeQuote
	MarketDataTypeTrade
	MarketDataTypeOrderBook
	MarketDataTypeCandle
	MarketDataTypeVolume
	MarketDataTypeNews
	MarketDataTypeEconomicData
)

// String returns the string representation of MarketDataType
func (mdt MarketDataType) String() string {
	switch mdt {
	case MarketDataTypeTick:
		return "TICK"
	case MarketDataTypeQuote:
		return "QUOTE"
	case MarketDataTypeTrade:
		return "TRADE"
	case MarketDataTypeOrderBook:
		return "ORDER_BOOK"
	case MarketDataTypeCandle:
		return "CANDLE"
	case MarketDataTypeVolume:
		return "VOLUME"
	case MarketDataTypeNews:
		return "NEWS"
	case MarketDataTypeEconomicData:
		return "ECONOMIC_DATA"
	default:
		return "UNKNOWN"
	}
}

// TimeFrame represents different time frames for market data
type TimeFrame int

const (
	TimeFrame1Second TimeFrame = iota
	TimeFrame5Second
	TimeFrame10Second
	TimeFrame30Second
	TimeFrame1Minute
	TimeFrame5Minute
	TimeFrame15Minute
	TimeFrame30Minute
	TimeFrame1Hour
	TimeFrame4Hour
	TimeFrame1Day
	TimeFrame1Week
	TimeFrame1Month
)

// String returns the string representation of TimeFrame
func (tf TimeFrame) String() string {
	switch tf {
	case TimeFrame1Second:
		return "1s"
	case TimeFrame5Second:
		return "5s"
	case TimeFrame10Second:
		return "10s"
	case TimeFrame30Second:
		return "30s"
	case TimeFrame1Minute:
		return "1m"
	case TimeFrame5Minute:
		return "5m"
	case TimeFrame15Minute:
		return "15m"
	case TimeFrame30Minute:
		return "30m"
	case TimeFrame1Hour:
		return "1h"
	case TimeFrame4Hour:
		return "4h"
	case TimeFrame1Day:
		return "1d"
	case TimeFrame1Week:
		return "1w"
	case TimeFrame1Month:
		return "1M"
	default:
		return "UNKNOWN"
	}
}

// Core Business Types

// Asset represents a tradeable asset
type Asset struct {
	ID        string                 `json:"id"`
	Symbol    string                 `json:"symbol"`
	Name      string                 `json:"name"`
	AssetType AssetType              `json:"asset_type"`
	Exchange  string                 `json:"exchange"`
	Region    string                 `json:"region"`
	Currency  string                 `json:"currency"`
	ISIN      string                 `json:"isin,omitempty"`
	Sector    string                 `json:"sector,omitempty"`
	Industry  string                 `json:"industry,omitempty"`
	MarketCap float64                `json:"market_cap,omitempty"`
	IsActive  bool                   `json:"is_active"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}





// Position represents a trading position
type Position struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Symbol          string     `json:"symbol"`
	AssetType       AssetType  `json:"asset_type"`
	Exchange        string     `json:"exchange"`
	Quantity        float64    `json:"quantity"`
	AveragePrice    float64    `json:"average_price"`
	MarketPrice     float64    `json:"market_price"`
	MarketValue     float64    `json:"market_value"`
	UnrealizedPL    float64    `json:"unrealized_pl"`
	RealizedPL      float64    `json:"realized_pl"`
	TotalCost       float64    `json:"total_cost"`
	TotalCommission float64    `json:"total_commission"`
	TotalFees       float64    `json:"total_fees"`
	OpenedAt        time.Time  `json:"opened_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
}
