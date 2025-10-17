package external

import (
	"context"
	"time"
)

// MarketDataType represents the type of market data
type MarketDataType string

const (
	// MarketDataTypeOrderBook represents order book data
	MarketDataTypeOrderBook MarketDataType = "order_book"
	// MarketDataTypeTrade represents trade data
	MarketDataTypeTrade MarketDataType = "trade"
	// MarketDataTypeTicker represents ticker data
	MarketDataTypeTicker MarketDataType = "ticker"
	// MarketDataTypeOHLCV represents OHLCV data
	MarketDataTypeOHLCV MarketDataType = "ohlcv"
)

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64
	Quantity float64
}

// OrderBookData represents order book data
type OrderBookData struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the bids
	Bids []PriceLevel
	// Asks is the asks
	Asks []PriceLevel
	// Timestamp is the time of the update
	Timestamp time.Time
}

// TradeData represents trade data
type TradeData struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the price of the trade
	Price float64
	// Quantity is the quantity of the trade
	Quantity float64
	// Side is the side of the trade
	Side string
	// Timestamp is the time of the trade
	Timestamp time.Time
	// TradeID is the trade ID
	TradeID string
}

// TickerData represents ticker data
type TickerData struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the current price
	Price float64
	// Volume is the 24-hour volume
	Volume float64
	// Change is the 24-hour price change
	Change float64
	// ChangePercent is the 24-hour price change percentage
	ChangePercent float64
	// High is the 24-hour high price
	High float64
	// Low is the 24-hour low price
	Low float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// OHLCVData represents OHLCV data
type OHLCVData struct {
	// Symbol is the trading symbol
	Symbol string
	// Interval is the interval
	Interval string
	// Open is the open price
	Open float64
	// High is the high price
	High float64
	// Low is the low price
	Low float64
	// Close is the close price
	Close float64
	// Volume is the volume
	Volume float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// MarketDataCallback is a callback function for market data
type MarketDataCallback func(interface{})

// Provider represents a market data provider
type Provider interface {
	// Name returns the name of the provider
	Name() string
	// Connect connects to the provider
	Connect(ctx context.Context) error
	// Disconnect disconnects from the provider
	Disconnect(ctx context.Context) error
	// SubscribeOrderBook subscribes to order book updates
	SubscribeOrderBook(ctx context.Context, symbol string, callback MarketDataCallback) error
	// UnsubscribeOrderBook unsubscribes from order book updates
	UnsubscribeOrderBook(ctx context.Context, symbol string) error
	// SubscribeTrades subscribes to trade updates
	SubscribeTrades(ctx context.Context, symbol string, callback MarketDataCallback) error
	// UnsubscribeTrades unsubscribes from trade updates
	UnsubscribeTrades(ctx context.Context, symbol string) error
	// SubscribeTicker subscribes to ticker updates
	SubscribeTicker(ctx context.Context, symbol string, callback MarketDataCallback) error
	// UnsubscribeTicker unsubscribes from ticker updates
	UnsubscribeTicker(ctx context.Context, symbol string) error
	// SubscribeOHLCV subscribes to OHLCV updates
	SubscribeOHLCV(ctx context.Context, symbol, interval string, callback MarketDataCallback) error
	// UnsubscribeOHLCV unsubscribes from OHLCV updates
	UnsubscribeOHLCV(ctx context.Context, symbol, interval string) error
	// GetOrderBook gets the order book
	GetOrderBook(ctx context.Context, symbol string) (*OrderBookData, error)
	// GetTrades gets trades
	GetTrades(ctx context.Context, symbol string, limit int) ([]TradeData, error)
	// GetTicker gets the ticker
	GetTicker(ctx context.Context, symbol string) (*TickerData, error)
	// GetOHLCV gets OHLCV data
	GetOHLCV(ctx context.Context, symbol, interval string, limit int) ([]OHLCVData, error)
}
