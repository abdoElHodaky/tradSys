package matching

import (
	"context"
	"errors"
	"sync"
	"time"
	"unsafe"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"go.uber.org/zap"
)

// HFTEngine represents a high-frequency trading optimized order matching engine
type HFTEngine struct {
	// OrderBooks is a map of symbol to order book (lock-free access)
	orderBooks unsafe.Pointer // map[string]*HFTOrderBook

	// Trade channel with buffering for high throughput
	TradeChannel chan *Trade

	// Order pools for zero-allocation order processing
	fastOrderPool *pool.FastOrderPool
	tradePool     *pool.TradePool

	// Performance metrics
	ordersProcessed uint64
	tradesExecuted  uint64
	avgLatency      uint64 // nanoseconds

	// Logger
	logger *zap.Logger

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Worker pool for parallel processing
	workerPool chan struct{}

	// Lock-free statistics
	stats *EngineStats
}

// EngineStats represents engine performance statistics
type EngineStats struct {
	OrdersProcessed   uint64
	TradesExecuted    uint64
	AvgLatencyNs      uint64
	MaxLatencyNs      uint64
	MinLatencyNs      uint64
	TotalVolumeTraded uint64
	ActiveOrders      uint64
	CancelledOrders   uint64
	RejectedOrders    uint64
	LastUpdateTime    time.Time
}

// HFTOrderBook represents a high-frequency trading optimized order book
type HFTOrderBook struct {
	Symbol string

	// Lock-free order storage using atomic operations
	buyOrders  unsafe.Pointer // *OrderLevel
	sellOrders unsafe.Pointer // *OrderLevel

	// Fast lookup maps for order management
	orderMap sync.Map // map[string]*HFTOrder

	// Performance counters
	totalOrders   uint64
	totalTrades   uint64
	totalVolume   uint64
	lastTradeTime time.Time

	// Spread tracking
	bestBid uint64 // atomic
	bestAsk uint64 // atomic
	spread  uint64 // atomic

	// Lock for critical sections (minimal usage)
	mu sync.RWMutex
}

// OrderLevel represents a price level in the order book
type OrderLevel struct {
	Price    uint64 // Fixed-point price representation
	Quantity uint64
	Orders   unsafe.Pointer // *HFTOrder (linked list)
	Next     unsafe.Pointer // *OrderLevel
}

// HFTOrder represents a trading order optimized for HFT
type HFTOrder struct {
	ID        string
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Price     uint64 // Fixed-point representation
	Quantity  uint64
	Filled    uint64
	Status    OrderStatus
	Timestamp time.Time
	UserID    string

	// Linked list pointers for order book
	Next *HFTOrder
	Prev *HFTOrder

	// Performance tracking
	LatencyNs uint64
}

// Use types from engine.go to avoid duplication
// Use OrderStatus constants from engine.go
// Use Trade from engine.go

// Error definitions
var (
	ErrOrderBookNotFound = errors.New("order book not found")
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidOrder      = errors.New("invalid order")
)
