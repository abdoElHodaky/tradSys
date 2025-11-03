// Package common provides unified types for all TradSys v3 services
package common

import (
	"time"
)

// AssetType represents different types of assets
type AssetType int

const (
	AssetTypeStock AssetType = iota
	AssetTypeBond
	AssetTypeETF
	AssetTypeREIT
	AssetTypeMutualFund
	AssetTypeCommodity
	AssetTypeCrypto
	AssetTypeForex
	AssetTypeGovernmentBond
	AssetTypeCorporateBond
	AssetTypeIslamicInstrument
	AssetTypeSukuk
	AssetTypeIslamicFund
	AssetTypeIslamicREIT
)

// String returns the string representation of AssetType
func (at AssetType) String() string {
	switch at {
	case AssetTypeStock:
		return "STOCK"
	case AssetTypeBond:
		return "BOND"
	case AssetTypeETF:
		return "ETF"
	case AssetTypeREIT:
		return "REIT"
	case AssetTypeMutualFund:
		return "MUTUAL_FUND"
	case AssetTypeCommodity:
		return "COMMODITY"
	case AssetTypeCrypto:
		return "CRYPTO"
	case AssetTypeForex:
		return "FOREX"
	case AssetTypeGovernmentBond:
		return "GOVERNMENT_BOND"
	case AssetTypeCorporateBond:
		return "CORPORATE_BOND"
	case AssetTypeIslamicInstrument:
		return "ISLAMIC_INSTRUMENT"
	case AssetTypeSukuk:
		return "SUKUK"
	case AssetTypeIslamicFund:
		return "ISLAMIC_FUND"
	case AssetTypeIslamicREIT:
		return "ISLAMIC_REIT"
	default:
		return "UNKNOWN"
	}
}

// OrderType represents different types of orders
type OrderType int

const (
	OrderTypeMarket OrderType = iota
	OrderTypeLimit
	OrderTypeStop
	OrderTypeStopLimit
	OrderTypeTrailingStop
	OrderTypeGoodTillCancelled
	OrderTypeGoodTillDate
)

// String returns the string representation of OrderType
func (ot OrderType) String() string {
	switch ot {
	case OrderTypeMarket:
		return "MARKET"
	case OrderTypeLimit:
		return "LIMIT"
	case OrderTypeStop:
		return "STOP"
	case OrderTypeStopLimit:
		return "STOP_LIMIT"
	case OrderTypeTrailingStop:
		return "TRAILING_STOP"
	case OrderTypeGoodTillCancelled:
		return "GOOD_TILL_CANCELLED"
	case OrderTypeGoodTillDate:
		return "GOOD_TILL_DATE"
	default:
		return "UNKNOWN"
	}
}

// OrderSide represents the side of an order
type OrderSide int

const (
	OrderSideBuy OrderSide = iota
	OrderSideSell
)

// String returns the string representation of OrderSide
func (os OrderSide) String() string {
	switch os {
	case OrderSideBuy:
		return "BUY"
	case OrderSideSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

// OrderStatus represents the status of an order
type OrderStatus int

const (
	OrderStatusPending OrderStatus = iota
	OrderStatusNew
	OrderStatusPartiallyFilled
	OrderStatusFilled
	OrderStatusCancelled
	OrderStatusRejected
	OrderStatusExpired
	OrderStatusSuspended
)

// String returns the string representation of OrderStatus
func (os OrderStatus) String() string {
	switch os {
	case OrderStatusPending:
		return "PENDING"
	case OrderStatusNew:
		return "NEW"
	case OrderStatusPartiallyFilled:
		return "PARTIALLY_FILLED"
	case OrderStatusFilled:
		return "FILLED"
	case OrderStatusCancelled:
		return "CANCELLED"
	case OrderStatusRejected:
		return "REJECTED"
	case OrderStatusExpired:
		return "EXPIRED"
	case OrderStatusSuspended:
		return "SUSPENDED"
	default:
		return "UNKNOWN"
	}
}

// ExchangeType represents different exchanges
type ExchangeType int

const (
	ExchangeEGX ExchangeType = iota
	ExchangeNASDAQ
	ExchangeNYSE
	ExchangeLSE
	ExchangeTSE
	ExchangeHKEX
	ExchangeSSE
	ExchangeSZSE
	ExchangeNSE
	ExchangeBSE
	ExchangeJSE
	ExchangeASX
	ExchangeTSX
	ExchangeEuronext
	ExchangeXETRA
	ExchangeBinance
	ExchangeCoinbase
	ExchangeKraken
	ExchangeBitfinex
)

// String returns the string representation of ExchangeType
func (et ExchangeType) String() string {
	switch et {
	case ExchangeEGX:
		return "EGX"
	case ExchangeNASDAQ:
		return "NASDAQ"
	case ExchangeNYSE:
		return "NYSE"
	case ExchangeLSE:
		return "LSE"
	case ExchangeTSE:
		return "TSE"
	case ExchangeHKEX:
		return "HKEX"
	case ExchangeSSE:
		return "SSE"
	case ExchangeSZSE:
		return "SZSE"
	case ExchangeNSE:
		return "NSE"
	case ExchangeBSE:
		return "BSE"
	case ExchangeJSE:
		return "JSE"
	case ExchangeASX:
		return "ASX"
	case ExchangeTSX:
		return "TSX"
	case ExchangeEuronext:
		return "EURONEXT"
	case ExchangeXETRA:
		return "XETRA"
	case ExchangeBinance:
		return "BINANCE"
	case ExchangeCoinbase:
		return "COINBASE"
	case ExchangeKraken:
		return "KRAKEN"
	case ExchangeBitfinex:
		return "BITFINEX"
	default:
		return "UNKNOWN"
	}
}

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

// Order represents a trading order
type Order struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Symbol      string                 `json:"symbol"`
	AssetType   AssetType              `json:"asset_type"`
	Exchange    string                 `json:"exchange"`
	Type        OrderType              `json:"type"`
	Side        OrderSide              `json:"side"`
	Quantity    float64                `json:"quantity"`
	Price       float64                `json:"price,omitempty"`
	StopPrice   float64                `json:"stop_price,omitempty"`
	Status      OrderStatus            `json:"status"`
	FilledQty   float64                `json:"filled_qty"`
	AvgPrice    float64                `json:"avg_price,omitempty"`
	Commission  float64                `json:"commission,omitempty"`
	Fees        float64                `json:"fees,omitempty"`
	TimeInForce string                 `json:"time_in_force,omitempty"`
	ExpiryTime  *time.Time             `json:"expiry_time,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	FilledAt    *time.Time             `json:"filled_at,omitempty"`
}

// Trade represents an executed trade
type Trade struct {
	ID         string     `json:"id"`
	OrderID    string     `json:"order_id"`
	UserID     string     `json:"user_id"`
	Symbol     string     `json:"symbol"`
	Side       OrderSide  `json:"side"`
	Quantity   float64    `json:"quantity"`
	Price      float64    `json:"price"`
	Value      float64    `json:"value"`
	Commission float64    `json:"commission"`
	Fees       float64    `json:"fees"`
	Exchange   string     `json:"exchange"`
	ExecutedAt time.Time  `json:"executed_at"`
	SettledAt  *time.Time `json:"settled_at,omitempty"`
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
