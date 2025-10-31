// 🎯 **Standard Engine Types**
// Generated using TradSys Code Splitting Standards
//
// This file contains type definitions, constants, and data structures
// for the Standard Order Matching Engine component. All types follow the established
// naming conventions and include comprehensive documentation for standard trading operations.
//
// Performance Requirements: Standard latency, heap-based order book management
// File size limit: 300 lines

package order_matching

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
	OrderStatusOpen            = types.OrderStatusNew // Alias for consistency
)

// Trade represents a trade execution
type Trade struct {
	// ID is the unique identifier for the trade
	ID string `json:"id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Price is the price of the trade
	Price float64 `json:"price"`
	// Quantity is the quantity of the trade
	Quantity float64 `json:"quantity"`
	// BuyOrderID is the buy order ID
	BuyOrderID string `json:"buy_order_id"`
	// SellOrderID is the sell order ID
	SellOrderID string `json:"sell_order_id"`
	// TakerOrderID is the taker order ID
	TakerOrderID string `json:"taker_order_id"`
	// MakerOrderID is the maker order ID
	MakerOrderID string `json:"maker_order_id"`
	// Timestamp is the time the trade was executed
	Timestamp time.Time `json:"timestamp"`
	// TakerSide is the side of the taker
	TakerSide OrderSide `json:"taker_side"`
	// MakerSide is the side of the maker
	MakerSide OrderSide `json:"maker_side"`
	// TakerFee is the fee for the taker
	TakerFee float64 `json:"taker_fee"`
	// MakerFee is the fee for the maker
	MakerFee float64 `json:"maker_fee"`
}

// OrderBook represents an order book for a symbol
type OrderBook struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the buy orders (heap-based)
	Bids *OrderHeap
	// Asks is the sell orders (heap-based)
	Asks *OrderHeap
	// Orders is a map of order ID to order for fast lookup
	Orders map[string]*Order
	// StopBids is the stop buy orders
	StopBids *OrderHeap
	// StopAsks is the stop sell orders
	StopAsks *OrderHeap
	// LastPrice is the last traded price
	LastPrice float64
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger for structured logging
	logger *zap.Logger
}

// OrderHeap is a heap-based priority queue of orders
type OrderHeap struct {
	// Orders is the slice of orders
	Orders []*Order
	// Side is the side of the orders (buy or sell)
	Side OrderSide
}

// Engine represents the standard order matching engine
type Engine struct {
	// OrderBooks is a map of symbol to order book
	OrderBooks map[string]*OrderBook
	// TradeChannel is the channel for publishing trades
	TradeChannel chan *Trade
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger for structured logging
	logger *zap.Logger
}

// EngineConfig is defined in advanced_engine.go

// OrderBookSnapshot represents a point-in-time snapshot of the order book
type OrderBookSnapshot struct {
	Symbol    string      `json:"symbol"`
	Timestamp time.Time   `json:"timestamp"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	LastPrice float64     `json:"last_price"`
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Orders   int     `json:"orders"`
}

// EngineStats represents engine statistics
type EngineStats struct {
	TotalOrders     int64   `json:"total_orders"`
	TotalTrades     int64   `json:"total_trades"`
	ActiveSymbols   int     `json:"active_symbols"`
	AverageSpread   float64 `json:"average_spread"`
	LastUpdateTime  time.Time `json:"last_update_time"`
}

// TradeExecution represents a completed trade execution with details
type TradeExecution struct {
	TradeID       string    `json:"trade_id"`
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Quantity      float64   `json:"quantity"`
	TakerOrderID  string    `json:"taker_order_id"`
	MakerOrderID  string    `json:"maker_order_id"`
	ExecutionTime time.Time `json:"execution_time"`
	TakerSide     OrderSide `json:"taker_side"`
	TakerFee      float64   `json:"taker_fee"`
	MakerFee      float64   `json:"maker_fee"`
}

// OrderBookState represents the current state of an order book
type OrderBookState struct {
	Symbol        string    `json:"symbol"`
	BidCount      int       `json:"bid_count"`
	AskCount      int       `json:"ask_count"`
	StopBidCount  int       `json:"stop_bid_count"`
	StopAskCount  int       `json:"stop_ask_count"`
	BestBid       float64   `json:"best_bid"`
	BestAsk       float64   `json:"best_ask"`
	Spread        float64   `json:"spread"`
	LastPrice     float64   `json:"last_price"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// Constants for standard engine operation
const (
	// Default configuration values
	DefaultTradeChannelBuffer = 1000
	DefaultMaxOrdersPerSymbol = 10000
	
	// Order book limits
	MaxOrderBookDepth = 100
	MaxPriceLevels    = 50
	
	// Performance thresholds (less strict than HFT)
	MaxProcessingTimeMs = 10 // 10ms target
)

// Error definitions
var (
	ErrOrderBookNotFound = NewEngineError("order book not found")
	ErrOrderNotFound     = NewEngineError("order not found")
	ErrInvalidOrder      = NewEngineError("invalid order")
	ErrInvalidQuantity   = NewEngineError("invalid quantity")
	ErrInvalidPrice      = NewEngineError("invalid price")
	ErrOrderExpired      = NewEngineError("order expired")
	ErrInsufficientFunds = NewEngineError("insufficient funds")
)

// EngineError represents an engine-specific error
type EngineError struct {
	Message string
}

// Error implements the error interface
func (e *EngineError) Error() string {
	return e.Message
}

// NewEngineError creates a new engine error
func NewEngineError(message string) *EngineError {
	return &EngineError{Message: message}
}

// OrderMatchResult represents the result of order matching
type OrderMatchResult struct {
	Trades       []*Trade `json:"trades"`
	RemainingQty float64  `json:"remaining_qty"`
	FullyFilled  bool     `json:"fully_filled"`
}

// StopOrderTrigger represents a stop order trigger event
type StopOrderTrigger struct {
	OrderID     string    `json:"order_id"`
	Symbol      string    `json:"symbol"`
	TriggerPrice float64  `json:"trigger_price"`
	CurrentPrice float64  `json:"current_price"`
	TriggerTime  time.Time `json:"trigger_time"`
}
