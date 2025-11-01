// 🎯 **HFT Engine Types**
// Generated using TradSys Code Splitting Standards
//
// This file contains type definitions, constants, and data structures
// for the High-Frequency Trading Engine component. All types follow the established
// naming conventions and include comprehensive documentation for ultra-low latency operations.
//
// Performance Requirements: <100μs latency, zero-allocation where possible
// File size limit: 300 lines

package order_matching

import (
	"context"
	"sync"
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
	AvgLatencyNanos   uint64
	MaxLatencyNanos   uint64
	MinLatencyNanos   uint64
	TotalLatencyNanos uint64
	LastUpdateTime    int64 // Unix nanoseconds
}

// HFTOrderBook represents a high-frequency trading optimized order book
type HFTOrderBook struct {
	Symbol string

	// Lock-free order book using atomic operations
	bids unsafe.Pointer // *PriceLevelTree
	asks unsafe.Pointer // *PriceLevelTree

	// Order lookup map with RWMutex for better read performance
	orders sync.Map // map[string]*Order

	// Last trade price (atomic)
	lastPrice uint64 // float64 as uint64 for atomic operations

	// Performance counters
	orderCount  uint64
	tradeCount  uint64
	lastUpdated int64 // Unix nanoseconds

	// Logger
	logger *zap.Logger
}

// PriceLevelTree represents a price level tree for efficient order book management
type PriceLevelTree struct {
	// Root node of the tree
	root *PriceLevelNode

	// Side of the tree (buy or sell)
	side OrderSide

	// Node count for quick size checks
	nodeCount uint32

	// RWMutex for concurrent access
	mu sync.RWMutex
}

// PriceLevelNode represents a node in the price level tree
type PriceLevelNode struct {
	// Price level
	price float64

	// Orders at this price level (FIFO queue)
	orders []*Order

	// Total quantity at this price level
	totalQuantity float64

	// Tree structure
	left   *PriceLevelNode
	right  *PriceLevelNode
	parent *PriceLevelNode
	height int

	// Order count at this level
	orderCount uint32
}

// FastOrder type alias for pool optimization
type FastOrder = pool.FastOrder

// HFTEngineConfig contains configuration for HFT engine
type HFTEngineConfig struct {
	// Performance settings
	MaxLatencyNanos    uint64 `json:"max_latency_nanos"`    // Target: <100,000 (100μs)
	TradeChannelBuffer int    `json:"trade_channel_buffer"` // Default: 10,000
	WorkerPoolSize     int    `json:"worker_pool_size"`     // Default: 2x CPU cores
	OrderPoolSize      int    `json:"order_pool_size"`      // Default: 10,000
	TradePoolSize      int    `json:"trade_pool_size"`      // Default: 1,000

	// Order book settings
	MaxPriceLevels    int     `json:"max_price_levels"`     // Default: 1,000
	MaxOrdersPerLevel int     `json:"max_orders_per_level"` // Default: 100
	PriceTickSize     float64 `json:"price_tick_size"`      // Minimum price increment

	// Monitoring settings
	EnableMetrics      bool `json:"enable_metrics"`       // Default: true
	MetricsInterval    int  `json:"metrics_interval_ms"`  // Default: 1000ms
	EnableTradeLogging bool `json:"enable_trade_logging"` // Default: false (performance)

	// Memory management
	EnableGCOptimization bool `json:"enable_gc_optimization"` // Default: true
	PreallocateMemory    bool `json:"preallocate_memory"`     // Default: true
}

// HFTPerformanceMetrics contains detailed performance metrics
type HFTPerformanceMetrics struct {
	// Latency metrics (nanoseconds)
	OrderProcessingLatency struct {
		Min  uint64 `json:"min"`
		Max  uint64 `json:"max"`
		Avg  uint64 `json:"avg"`
		P50  uint64 `json:"p50"`
		P95  uint64 `json:"p95"`
		P99  uint64 `json:"p99"`
		P999 uint64 `json:"p999"`
	} `json:"order_processing_latency"`

	// Throughput metrics
	OrdersPerSecond   float64 `json:"orders_per_second"`
	TradesPerSecond   float64 `json:"trades_per_second"`
	MessagesPerSecond float64 `json:"messages_per_second"`

	// Memory metrics
	MemoryUsageBytes uint64  `json:"memory_usage_bytes"`
	PoolUtilization  float64 `json:"pool_utilization"`
	GCPauseTimeNanos uint64  `json:"gc_pause_time_nanos"`

	// Order book metrics
	AverageSpread   float64 `json:"average_spread"`
	OrderBookDepth  int     `json:"order_book_depth"`
	PriceLevelCount int     `json:"price_level_count"`

	// System metrics
	CPUUtilization      float64 `json:"cpu_utilization"`
	NetworkLatencyNanos uint64  `json:"network_latency_nanos"`

	// Error metrics
	RejectedOrders uint64 `json:"rejected_orders"`
	FailedTrades   uint64 `json:"failed_trades"`
	TimeoutCount   uint64 `json:"timeout_count"`

	// Timestamp
	LastUpdateTime int64 `json:"last_update_time"`
}

// HFTOrderBookSnapshot represents a point-in-time snapshot of the order book
type HFTOrderBookSnapshot struct {
	Symbol    string `json:"symbol"`
	Timestamp int64  `json:"timestamp"`

	// Best bid/ask
	BestBid float64 `json:"best_bid"`
	BestAsk float64 `json:"best_ask"`
	Spread  float64 `json:"spread"`

	// Depth
	BidLevels []PriceLevel `json:"bid_levels"`
	AskLevels []PriceLevel `json:"ask_levels"`

	// Statistics
	TotalBidQuantity float64 `json:"total_bid_quantity"`
	TotalAskQuantity float64 `json:"total_ask_quantity"`
	LastTradePrice   float64 `json:"last_trade_price"`
	LastTradeTime    int64   `json:"last_trade_time"`
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Orders   int     `json:"orders"`
}

// HFTTradeExecution represents a completed trade execution
type HFTTradeExecution struct {
	TradeID         string  `json:"trade_id"`
	Symbol          string  `json:"symbol"`
	Price           float64 `json:"price"`
	Quantity        float64 `json:"quantity"`
	BuyOrderID      string  `json:"buy_order_id"`
	SellOrderID     string  `json:"sell_order_id"`
	ExecutionTime   int64   `json:"execution_time"`
	ProcessingTime  uint64  `json:"processing_time_nanos"`
	MatchingLatency uint64  `json:"matching_latency_nanos"`
}

// HFTEngineState represents the current state of the HFT engine
type HFTEngineState struct {
	IsRunning      bool   `json:"is_running"`
	StartTime      int64  `json:"start_time"`
	UptimeSeconds  int64  `json:"uptime_seconds"`
	ActiveSymbols  int    `json:"active_symbols"`
	TotalOrders    uint64 `json:"total_orders"`
	TotalTrades    uint64 `json:"total_trades"`
	CurrentLatency uint64 `json:"current_latency_nanos"`
	HealthStatus   string `json:"health_status"`
}

// Constants for HFT engine operation
const (
	// Performance targets
	MaxTargetLatencyNanos = 100000 // 100 microseconds
	MaxOrderBookDepth     = 1000   // Maximum price levels
	MaxOrdersPerLevel     = 100    // Maximum orders per price level

	// Buffer sizes
	DefaultTradeChannelBuffer = 10000
	DefaultOrderPoolSize      = 10000
	DefaultTradePoolSize      = 1000

	// Monitoring intervals
	DefaultMetricsIntervalMs = 1000
	DefaultHealthCheckMs     = 100

	// Memory optimization
	DefaultGCTargetPercent = 50 // Lower GC pressure for HFT
)

// Health status constants
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusDegraded  = "degraded"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusCritical  = "critical"
)
