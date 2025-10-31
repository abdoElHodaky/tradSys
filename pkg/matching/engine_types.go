package matching

import (
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// Use types from the shared types package
type OrderType = types.OrderType
type OrderSide = types.OrderSide
type OrderStatus = types.OrderStatus
type Order = types.Order

// Constants from types package
const (
	OrderTypeLimit             = types.OrderTypeLimit
	OrderTypeMarket            = types.OrderTypeMarket
	OrderTypeStop              = types.OrderTypeStop
	OrderTypeStopLimit         = types.OrderTypeStopLimit
	OrderTypeStopMarket        = types.OrderTypeStopMarket
	OrderSideBuy               = types.OrderSideBuy
	OrderSideSell              = types.OrderSideSell
	OrderStatusNew             = types.OrderStatusNew
	OrderStatusPartiallyFilled = types.OrderStatusPartiallyFilled
	OrderStatusFilled          = types.OrderStatusFilled
	OrderStatusCanceled        = types.OrderStatusCanceled
	OrderStatusCancelled       = types.OrderStatusCancelled
	OrderStatusRejected        = types.OrderStatusRejected
	OrderStatusExpired         = types.OrderStatusExpired
)

// Trade represents a trade
type Trade struct {
	// ID is the unique identifier for the trade
	ID string
	// Symbol is the trading symbol
	Symbol string
	// Price is the price of the trade
	Price float64
	// Quantity is the quantity of the trade
	Quantity float64
	// BuyOrderID is the buy order ID
	BuyOrderID string
	// SellOrderID is the sell order ID
	SellOrderID string
	// Timestamp is the time the trade was executed
	Timestamp time.Time
	// TakerSide is the side of the taker
	TakerSide OrderSide
	// MakerSide is the side of the maker
	MakerSide OrderSide
	// TakerFee is the fee for the taker
	TakerFee float64
	// MakerFee is the fee for the maker
	MakerFee float64
}

// OrderBook represents an order book for a symbol
type OrderBook struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the buy orders
	Bids *OrderHeap
	// Asks is the sell orders
	Asks *OrderHeap
	// Orders is a map of order ID to order
	Orders map[string]*Order
	// StopBids is the stop buy orders
	StopBids *OrderHeap
	// StopAsks is the stop sell orders
	StopAsks *OrderHeap
	// LastPrice is the last traded price
	LastPrice float64
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
}

// OrderHeap is a heap of orders
type OrderHeap struct {
	// Orders is the slice of orders
	Orders []*Order
	// Side is the side of the orders
	Side OrderSide
}

// MatchingEngine represents the matching engine
type MatchingEngine struct {
	// OrderBooks is a map of symbol to order book
	OrderBooks map[string]*OrderBook
	// Trades is a slice of trades
	Trades []*Trade
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// TradeChan is the channel for trades
	TradeChan chan *Trade
	// OrderChan is the channel for orders
	OrderChan chan *Order
	// CancelChan is the channel for order cancellations
	CancelChan chan string
	// StopChan is the channel for stopping the engine
	StopChan chan struct{}
	// Running indicates if the engine is running
	Running bool
	// Metrics
	Metrics *EngineMetrics
}

// EngineMetrics represents metrics for the matching engine
type EngineMetrics struct {
	// TotalTrades is the total number of trades
	TotalTrades int64
	// TotalOrders is the total number of orders
	TotalOrders int64
	// TotalVolume is the total volume traded
	TotalVolume float64
	// AverageLatency is the average latency
	AverageLatency time.Duration
	// MaxLatency is the maximum latency
	MaxLatency time.Duration
	// MinLatency is the minimum latency
	MinLatency time.Duration
	// LastTradeTime is the time of the last trade
	LastTradeTime time.Time
	// OrdersPerSecond is the number of orders per second
	OrdersPerSecond float64
	// TradesPerSecond is the number of trades per second
	TradesPerSecond float64
	// Mutex for thread safety
	mu sync.RWMutex
}

// MatchResult represents the result of a match operation
type MatchResult struct {
	// Trades is the slice of trades generated
	Trades []*Trade
	// UpdatedOrders is the slice of updated orders
	UpdatedOrders []*Order
	// RemovedOrders is the slice of removed order IDs
	RemovedOrders []string
	// Latency is the latency of the match operation
	Latency time.Duration
}

// OrderBookSnapshot represents a snapshot of an order book
type OrderBookSnapshot struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the slice of bid orders
	Bids []*Order
	// Asks is the slice of ask orders
	Asks []*Order
	// LastPrice is the last traded price
	LastPrice float64
	// Timestamp is the time of the snapshot
	Timestamp time.Time
	// BidDepth is the total bid depth
	BidDepth float64
	// AskDepth is the total ask depth
	AskDepth float64
	// Spread is the bid-ask spread
	Spread float64
}

// MarketData represents market data for a symbol
type MarketData struct {
	// Symbol is the trading symbol
	Symbol string
	// LastPrice is the last traded price
	LastPrice float64
	// BestBid is the best bid price
	BestBid float64
	// BestAsk is the best ask price
	BestAsk float64
	// BidSize is the size at the best bid
	BidSize float64
	// AskSize is the size at the best ask
	AskSize float64
	// Volume is the 24h volume
	Volume float64
	// High is the 24h high
	High float64
	// Low is the 24h low
	Low float64
	// Open is the 24h open
	Open float64
	// Close is the 24h close
	Close float64
	// Change is the 24h change
	Change float64
	// ChangePercent is the 24h change percentage
	ChangePercent float64
	// Timestamp is the time of the market data
	Timestamp time.Time
}

// EngineConfig represents configuration for the matching engine
type EngineConfig struct {
	// MaxOrdersPerSymbol is the maximum number of orders per symbol
	MaxOrdersPerSymbol int
	// MaxTradesHistory is the maximum number of trades to keep in history
	MaxTradesHistory int
	// EnableMetrics indicates if metrics should be collected
	EnableMetrics bool
	// MetricsInterval is the interval for metrics collection
	MetricsInterval time.Duration
	// OrderChannelSize is the size of the order channel
	OrderChannelSize int
	// TradeChannelSize is the size of the trade channel
	TradeChannelSize int
	// CancelChannelSize is the size of the cancel channel
	CancelChannelSize int
	// EnableStopOrders indicates if stop orders are enabled
	EnableStopOrders bool
	// EnableOrderExpiry indicates if order expiry is enabled
	EnableOrderExpiry bool
	// DefaultOrderTTL is the default time to live for orders
	DefaultOrderTTL time.Duration
}

// OrderValidationResult represents the result of order validation
type OrderValidationResult struct {
	// Valid indicates if the order is valid
	Valid bool
	// Error is the validation error
	Error string
	// Warnings is a slice of validation warnings
	Warnings []string
}

// CancelResult represents the result of an order cancellation
type CancelResult struct {
	// Success indicates if the cancellation was successful
	Success bool
	// OrderID is the ID of the cancelled order
	OrderID string
	// Error is the cancellation error
	Error string
	// CancelledOrder is the cancelled order
	CancelledOrder *Order
}

// EngineStatus represents the status of the matching engine
type EngineStatus struct {
	// Running indicates if the engine is running
	Running bool
	// TotalSymbols is the total number of symbols
	TotalSymbols int
	// TotalOrders is the total number of active orders
	TotalOrders int
	// TotalTrades is the total number of trades
	TotalTrades int64
	// Uptime is the uptime of the engine
	Uptime time.Duration
	// StartTime is the start time of the engine
	StartTime time.Time
	// LastTradeTime is the time of the last trade
	LastTradeTime time.Time
	// OrdersPerSecond is the current orders per second
	OrdersPerSecond float64
	// TradesPerSecond is the current trades per second
	TradesPerSecond float64
	// AverageLatency is the average matching latency
	AverageLatency time.Duration
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	// Price is the price level
	Price float64
	// Quantity is the total quantity at this price level
	Quantity float64
	// OrderCount is the number of orders at this price level
	OrderCount int
	// Orders is the slice of orders at this price level
	Orders []*Order
}

// OrderBookDepth represents the depth of an order book
type OrderBookDepth struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the slice of bid price levels
	Bids []*PriceLevel
	// Asks is the slice of ask price levels
	Asks []*PriceLevel
	// Timestamp is the time of the depth snapshot
	Timestamp time.Time
}

// TradeHistory represents trade history for a symbol
type TradeHistory struct {
	// Symbol is the trading symbol
	Symbol string
	// Trades is the slice of trades
	Trades []*Trade
	// StartTime is the start time of the history
	StartTime time.Time
	// EndTime is the end time of the history
	EndTime time.Time
	// TotalVolume is the total volume in the history
	TotalVolume float64
	// TotalTrades is the total number of trades
	TotalTrades int
	// VWAP is the volume weighted average price
	VWAP float64
}

// EngineEvent represents an event from the matching engine
type EngineEvent struct {
	// Type is the event type
	Type string
	// Symbol is the trading symbol
	Symbol string
	// Data is the event data
	Data interface{}
	// Timestamp is the time of the event
	Timestamp time.Time
}

// Event types
const (
	EventTypeOrderAdded    = "order_added"
	EventTypeOrderCanceled = "order_canceled"
	EventTypeOrderFilled   = "order_filled"
	EventTypeTradeExecuted = "trade_executed"
	EventTypeOrderBookUpdated = "order_book_updated"
	EventTypeEngineStarted = "engine_started"
	EventTypeEngineStopped = "engine_stopped"
)
