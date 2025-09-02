package marketdata

import (
	"time"
)

// MarketDataType represents the type of market data
type MarketDataType int32

const (
	MarketDataType_TRADE      MarketDataType = 0
	MarketDataType_ORDERBOOK  MarketDataType = 1
	MarketDataType_TICKER     MarketDataType = 2
	MarketDataType_OHLCV      MarketDataType = 3
	MarketDataType_QUOTE      MarketDataType = 4
	MarketDataType_DEPTH      MarketDataType = 5
	MarketDataType_STATISTICS MarketDataType = 6
)

// MarketDataResponse represents a market data response
type MarketDataResponse struct {
	Type      MarketDataType
	Symbol    string
	Timestamp time.Time
	Data      interface{}
}

// OrderBookEntry represents an entry in the order book
type OrderBookEntry struct {
	Price  float64
	Amount float64
	Count  int32
}

// OrderBookData represents order book data
type OrderBookData struct {
	Bids []OrderBookEntry
	Asks []OrderBookEntry
}

// TradeData represents trade data
type TradeData struct {
	ID        string
	Price     float64
	Amount    float64
	Side      string
	Timestamp time.Time
}

// TickerData represents ticker data
type TickerData struct {
	Last      float64
	High      float64
	Low       float64
	Volume    float64
	Bid       float64
	Ask       float64
	Timestamp time.Time
}

// OHLCVData represents OHLCV (candle) data
type OHLCVData struct {
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp time.Time
	Period    string
}
