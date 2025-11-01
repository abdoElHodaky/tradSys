// Package types provides canonical type definitions for the trading system.
// These types serve as the single source of truth across all packages.
package types

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Engine represents the core interface for all matching engines
type Engine interface {
	// Start initializes and starts the engine
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the engine
	Stop(ctx context.Context) error
	
	// ProcessOrder processes a single order
	ProcessOrder(order *Order) (*Trade, error)
	
	// GetStats returns current engine statistics
	GetStats() *EngineStats
	
	// GetConfig returns the engine configuration
	GetConfig() *EngineConfig
	
	// IsHealthy returns the health status of the engine
	IsHealthy() bool
}

// EngineConfig holds configuration for all engine types
type EngineConfig struct {
	// Core Configuration
	Symbol              string        `json:"symbol" yaml:"symbol"`
	MaxOrderBookDepth   int           `json:"max_order_book_depth" yaml:"max_order_book_depth"`
	TickSize            float64       `json:"tick_size" yaml:"tick_size"`
	LotSize             float64       `json:"lot_size" yaml:"lot_size"`
	
	// Performance Configuration
	TradeChannelBuffer  int           `json:"trade_channel_buffer" yaml:"trade_channel_buffer"`
	OrderChannelBuffer  int           `json:"order_channel_buffer" yaml:"order_channel_buffer"`
	BatchSize           int           `json:"batch_size" yaml:"batch_size"`
	FlushInterval       time.Duration `json:"flush_interval" yaml:"flush_interval"`
	
	// Risk Configuration
	MaxOrderSize        float64       `json:"max_order_size" yaml:"max_order_size"`
	MaxPositionSize     float64       `json:"max_position_size" yaml:"max_position_size"`
	PriceDeviationLimit float64       `json:"price_deviation_limit" yaml:"price_deviation_limit"`
	
	// HFT-specific Configuration
	MinLatencyTarget    time.Duration `json:"min_latency_target" yaml:"min_latency_target"`
	MaxLatencyTarget    time.Duration `json:"max_latency_target" yaml:"max_latency_target"`
	EnableOptimizations bool          `json:"enable_optimizations" yaml:"enable_optimizations"`
	
	// Monitoring Configuration
	EnableMetrics       bool          `json:"enable_metrics" yaml:"enable_metrics"`
	MetricsInterval     time.Duration `json:"metrics_interval" yaml:"metrics_interval"`
	EnableTracing       bool          `json:"enable_tracing" yaml:"enable_tracing"`
}

// EngineStats holds runtime statistics for all engine types
type EngineStats struct {
	// Basic Statistics
	OrdersProcessed     uint64        `json:"orders_processed"`
	TradesExecuted      uint64        `json:"trades_executed"`
	TotalVolume         float64       `json:"total_volume"`
	TotalValue          float64       `json:"total_value"`
	
	// Performance Statistics
	AverageLatency      time.Duration `json:"average_latency"`
	MinLatency          time.Duration `json:"min_latency"`
	MaxLatency          time.Duration `json:"max_latency"`
	MinLatencyNanos     int64         `json:"min_latency_nanos"`
	MaxLatencyNanos     int64         `json:"max_latency_nanos"`
	
	// Throughput Statistics
	OrdersPerSecond     float64       `json:"orders_per_second"`
	TradesPerSecond     float64       `json:"trades_per_second"`
	MessagesPerSecond   float64       `json:"messages_per_second"`
	
	// Error Statistics
	ErrorCount          uint64        `json:"error_count"`
	RejectedOrders      uint64        `json:"rejected_orders"`
	FailedTrades        uint64        `json:"failed_trades"`
	
	// Resource Statistics
	MemoryUsage         uint64        `json:"memory_usage"`
	CPUUsage            float64       `json:"cpu_usage"`
	GoroutineCount      int           `json:"goroutine_count"`
	
	// Timestamps
	StartTime           time.Time     `json:"start_time"`
	LastUpdateTime      time.Time     `json:"last_update_time"`
	UptimeSeconds       float64       `json:"uptime_seconds"`
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Orders   int     `json:"orders"`
}

// Trade represents a completed trade
type Trade struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	BuyOrderID   string    `json:"buy_order_id"`
	SellOrderID  string    `json:"sell_order_id"`
	Price        float64   `json:"price"`
	Quantity     float64   `json:"quantity"`
	Value        float64   `json:"value"`
	Timestamp    time.Time `json:"timestamp"`
	TakerSide    string    `json:"taker_side"`
	MakerFee     float64   `json:"maker_fee"`
	TakerFee     float64   `json:"taker_fee"`
}

// Order represents a trading order
type Order struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`        // "buy" or "sell"
	Type        string    `json:"type"`        // "market", "limit", "stop"
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	Filled      float64   `json:"filled"`
	Remaining   float64   `json:"remaining"`
	Status      string    `json:"status"`      // "pending", "partial", "filled", "cancelled"
	UserID      string    `json:"user_id"`
	Timestamp   time.Time `json:"timestamp"`
	TimeInForce string    `json:"time_in_force"` // "GTC", "IOC", "FOK"
}

// EngineError represents an engine-specific error
type EngineError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"` // "low", "medium", "high", "critical"
}

// Error implements the error interface
func (e *EngineError) Error() string {
	return e.Message
}

// NewEngineError creates a new engine error
func NewEngineError(code, message, details, severity string) *EngineError {
	return &EngineError{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Severity:  severity,
	}
}

// Constants for engine configuration
const (
	// Default buffer sizes
	DefaultTradeChannelBuffer = 1000
	DefaultOrderChannelBuffer = 1000
	DefaultBatchSize          = 100
	
	// Default limits
	MaxOrderBookDepth         = 1000
	DefaultFlushInterval      = 100 * time.Millisecond
	DefaultMetricsInterval    = 1 * time.Second
	
	// Order sides
	OrderSideBuy  = "buy"
	OrderSideSell = "sell"
	
	// Order types
	OrderTypeMarket = "market"
	OrderTypeLimit  = "limit"
	OrderTypeStop   = "stop"
	
	// Order status
	OrderStatusPending   = "pending"
	OrderStatusPartial   = "partial"
	OrderStatusFilled    = "filled"
	OrderStatusCancelled = "cancelled"
	
	// Time in force
	TimeInForceGTC = "GTC" // Good Till Cancelled
	TimeInForceIOC = "IOC" // Immediate Or Cancel
	TimeInForceFOK = "FOK" // Fill Or Kill
	
	// Error severities
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// DefaultEngineConfig returns a default engine configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MaxOrderBookDepth:   MaxOrderBookDepth,
		TradeChannelBuffer:  DefaultTradeChannelBuffer,
		OrderChannelBuffer:  DefaultOrderChannelBuffer,
		BatchSize:           DefaultBatchSize,
		FlushInterval:       DefaultFlushInterval,
		MaxOrderSize:        1000000.0,
		MaxPositionSize:     10000000.0,
		PriceDeviationLimit: 0.1,
		MinLatencyTarget:    1 * time.Microsecond,
		MaxLatencyTarget:    1 * time.Millisecond,
		EnableOptimizations: true,
		EnableMetrics:       true,
		MetricsInterval:     DefaultMetricsInterval,
		EnableTracing:       false,
	}
}

// NewEngineStats creates a new engine stats instance
func NewEngineStats() *EngineStats {
	return &EngineStats{
		StartTime:      time.Now(),
		LastUpdateTime: time.Now(),
	}
}

// Update updates the engine statistics
func (s *EngineStats) Update() {
	s.LastUpdateTime = time.Now()
	s.UptimeSeconds = time.Since(s.StartTime).Seconds()
}

// Reset resets all statistics
func (s *EngineStats) Reset() {
	*s = EngineStats{
		StartTime:      time.Now(),
		LastUpdateTime: time.Now(),
	}
}

// BaseEngine provides common functionality for all engine implementations
type BaseEngine struct {
	config *EngineConfig
	stats  *EngineStats
	logger *zap.Logger
	mu     sync.RWMutex
	
	// Channels
	orderChan chan *Order
	tradeChan chan *Trade
	
	// State
	running bool
	healthy bool
}

// NewBaseEngine creates a new base engine
func NewBaseEngine(config *EngineConfig, logger *zap.Logger) *BaseEngine {
	if config == nil {
		config = DefaultEngineConfig()
	}
	
	return &BaseEngine{
		config:    config,
		stats:     NewEngineStats(),
		logger:    logger,
		orderChan: make(chan *Order, config.OrderChannelBuffer),
		tradeChan: make(chan *Trade, config.TradeChannelBuffer),
		healthy:   true,
	}
}

// GetConfig returns the engine configuration
func (e *BaseEngine) GetConfig() *EngineConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// GetStats returns the engine statistics
func (e *BaseEngine) GetStats() *EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.stats.Update()
	return e.stats
}

// IsHealthy returns the health status
func (e *BaseEngine) IsHealthy() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.healthy
}

// IsRunning returns the running status
func (e *BaseEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// SetRunning sets the running status
func (e *BaseEngine) SetRunning(running bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = running
}

// SetHealthy sets the health status
func (e *BaseEngine) SetHealthy(healthy bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.healthy = healthy
}
