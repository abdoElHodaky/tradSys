package marketdata

import (
	"time"
)

// MarketDataResponse represents market data for a specific symbol
type MarketDataResponse struct {
	// Symbol identifier (e.g., "BTC-USD")
	Symbol string `json:"symbol"`
	
	// Timestamp of the market data
	Timestamp time.Time `json:"timestamp"`
	
	// Price information
	LastPrice  float64 `json:"last_price"`
	BidPrice   float64 `json:"bid_price"`
	AskPrice   float64 `json:"ask_price"`
	HighPrice  float64 `json:"high_price"`
	LowPrice   float64 `json:"low_price"`
	OpenPrice  float64 `json:"open_price"`
	ClosePrice float64 `json:"close_price"`
	
	// Volume information
	Volume         float64 `json:"volume"`
	QuoteVolume    float64 `json:"quote_volume"`
	BidSize        float64 `json:"bid_size"`
	AskSize        float64 `json:"ask_size"`
	
	// Order book information
	Bids []OrderBookEntry `json:"bids,omitempty"`
	Asks []OrderBookEntry `json:"asks,omitempty"`
	
	// Trade information
	Trades []Trade `json:"trades,omitempty"`
	
	// Market statistics
	VWAP           float64 `json:"vwap,omitempty"`
	PriceChange    float64 `json:"price_change,omitempty"`
	PriceChangePct float64 `json:"price_change_pct,omitempty"`
	
	// Source of the market data
	Source string `json:"source"`
	
	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// OrderBookEntry represents a single entry in the order book
type OrderBookEntry struct {
	Price  float64 `json:"price"`
	Size   float64 `json:"size"`
	Count  int     `json:"count,omitempty"`
	Orders []Order `json:"orders,omitempty"`
}

// Order represents an order in the order book
type Order struct {
	ID        string  `json:"id"`
	Price     float64 `json:"price"`
	Size      float64 `json:"size"`
	Timestamp time.Time `json:"timestamp"`
}

// Trade represents a single trade
type Trade struct {
	ID        string    `json:"id"`
	Price     float64   `json:"price"`
	Size      float64   `json:"size"`
	Side      string    `json:"side"` // "buy" or "sell"
	Timestamp time.Time `json:"timestamp"`
}

// NewMarketDataResponse creates a new MarketDataResponse with default values
func NewMarketDataResponse(symbol string) *MarketDataResponse {
	return &MarketDataResponse{
		Symbol:    symbol,
		Timestamp: time.Now(),
		Source:    "default",
		Metadata:  make(map[string]interface{}),
		Bids:      make([]OrderBookEntry, 0),
		Asks:      make([]OrderBookEntry, 0),
		Trades:    make([]Trade, 0),
	}
}

