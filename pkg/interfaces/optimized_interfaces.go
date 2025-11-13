package interfaces

import (
	"context"
	"time"
)

// Core Trading Interfaces

// TradingEngine defines the core trading engine interface
type TradingEngine interface {
	// Order management
	SubmitOrder(ctx context.Context, order *Order) (*OrderResult, error)
	CancelOrder(ctx context.Context, orderID string) error
	ModifyOrder(ctx context.Context, orderID string, modification *OrderModification) error
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	
	// Market data
	GetOrderBook(ctx context.Context, symbol string) (*OrderBook, error)
	GetTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error)
	
	// Engine lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
}

// RiskManager defines the risk management interface
type RiskManager interface {
	// Pre-trade checks
	PreTradeCheck(ctx context.Context, order *Order) (*RiskCheckResult, error)
	
	// Post-trade monitoring
	PostTradeCheck(ctx context.Context, trade *Trade) error
	
	// Position management
	GetPosition(ctx context.Context, symbol string) (*Position, error)
	GetPortfolio(ctx context.Context, accountID string) (*Portfolio, error)
	
	// Risk metrics
	GetRiskMetrics(ctx context.Context) (*RiskMetrics, error)
	GetVaR(ctx context.Context, portfolio *Portfolio) (float64, error)
	
	// Limits management
	SetPositionLimit(ctx context.Context, symbol string, limit float64) error
	SetOrderSizeLimit(ctx context.Context, symbol string, limit float64) error
	
	// Circuit breaker
	TripCircuitBreaker(ctx context.Context, reason string) error
	ResetCircuitBreaker(ctx context.Context) error
	IsCircuitBreakerTripped(ctx context.Context) bool
}

// MatchingEngine is defined in core_interfaces.go

// MarketDataProvider defines the market data interface
type MarketDataProvider interface {
	// Real-time data
	Subscribe(ctx context.Context, symbols []string) (<-chan *MarketData, error)
	Unsubscribe(ctx context.Context, symbols []string) error
	
	// Historical data
	GetHistoricalTrades(ctx context.Context, symbol string, from, to time.Time) ([]*Trade, error)
	GetHistoricalCandles(ctx context.Context, symbol string, interval time.Duration, from, to time.Time) ([]*Candle, error)
	
	// Current data
	GetTicker(ctx context.Context, symbol string) (*Ticker, error)
	GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error)
}

// PositionManager defines the position management interface
type PositionManager interface {
	// Position operations
	UpdatePosition(ctx context.Context, trade *Trade) error
	GetPosition(ctx context.Context, symbol string) (*Position, error)
	GetAllPositions(ctx context.Context) ([]*Position, error)
	
	// Portfolio operations
	GetPortfolio(ctx context.Context, accountID string) (*Portfolio, error)
	CalculatePortfolioValue(ctx context.Context, portfolio *Portfolio) (float64, error)
	
	// P&L calculations
	CalculateUnrealizedPnL(ctx context.Context, position *Position, currentPrice float64) (float64, error)
	CalculateRealizedPnL(ctx context.Context, trade *Trade) (float64, error)
}

// Performance and Monitoring Interfaces

// MetricsCollector is defined in core_interfaces.go

// HealthChecker is defined in common_interfaces.go

// PerformanceMonitor defines the performance monitoring interface
type PerformanceMonitor interface {
	// Latency monitoring
	RecordLatency(operation string, duration time.Duration)
	GetAverageLatency(operation string) time.Duration
	GetMaxLatency(operation string) time.Duration
	
	// Throughput monitoring
	RecordThroughput(operation string, count int64)
	GetThroughput(operation string) float64
	
	// Resource monitoring
	GetCPUUsage() float64
	GetMemoryUsage() uint64
	GetGoroutineCount() int
	
	// Performance reports
	GeneratePerformanceReport(ctx context.Context) (*PerformanceReport, error)
}

// Storage and Persistence Interfaces

// OrderRepository is defined in core_interfaces.go

// TradeRepository is defined in core_interfaces.go

// PositionRepository defines the position storage interface
type PositionRepository interface {
	// CRUD operations
	Create(ctx context.Context, position *Position) error
	GetBySymbol(ctx context.Context, symbol string) (*Position, error)
	Update(ctx context.Context, position *Position) error
	Delete(ctx context.Context, symbol string) error
	
	// Query operations
	GetAll(ctx context.Context) ([]*Position, error)
	FindByAccount(ctx context.Context, accountID string) ([]*Position, error)
}

// Communication Interfaces

// WebSocketHandler defines the WebSocket handling interface
type WebSocketHandler interface {
	// Connection management
	HandleConnection(ctx context.Context, conn WebSocketConnection) error
	CloseConnection(ctx context.Context, connectionID string) error
	
	// Message handling
	HandleMessage(ctx context.Context, connectionID string, message []byte) error
	BroadcastMessage(ctx context.Context, message []byte) error
	SendToConnection(ctx context.Context, connectionID string, message []byte) error
	
	// Subscription management
	Subscribe(ctx context.Context, connectionID string, channels []string) error
	Unsubscribe(ctx context.Context, connectionID string, channels []string) error
}

// EventPublisher is defined in core_interfaces.go

// EventSubscriber is defined in core_interfaces.go

// Configuration Interface

// ConfigManager is defined in core_interfaces.go

// Common Types and Structures

// Order is defined in exchange_interface.go

// Trade represents a completed trade
type Trade struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Side      OrderSide `json:"side"`
	Timestamp time.Time `json:"timestamp"`
	BuyOrderID  string  `json:"buy_order_id"`
	SellOrderID string  `json:"sell_order_id"`
}

// Position represents a trading position
type Position struct {
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`
	AveragePrice   float64   `json:"average_price"`
	MarketPrice    float64   `json:"market_price"`
	UnrealizedPnL  float64   `json:"unrealized_pnl"`
	RealizedPnL    float64   `json:"realized_pnl"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// OrderBook represents the order book for a symbol
type OrderBook struct {
	Symbol    string        `json:"symbol"`
	Bids      []*PriceLevel `json:"bids"`
	Asks      []*PriceLevel `json:"asks"`
	Timestamp time.Time     `json:"timestamp"`
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Orders   int     `json:"orders"`
}

// Enums and Constants

// OrderSide is defined in exchange_interface.go

// OrderType is defined in exchange_interface.go

// OrderStatus is defined in exchange_interface.go

// Result and Status Types

type OrderResult struct {
	Order  *Order   `json:"order"`
	Trades []*Trade `json:"trades"`
	Error  string   `json:"error,omitempty"`
}

type RiskCheckResult struct {
	Passed       bool      `json:"passed"`
	Reason       string    `json:"reason,omitempty"`
	RiskScore    float64   `json:"risk_score"`
	Timestamp    time.Time `json:"timestamp"`
	CheckLatency time.Duration `json:"check_latency"`
}

// HealthStatus is defined in core_interfaces.go

type ComponentHealth struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Latency   time.Duration `json:"latency"`
}

// Event Types

// Event is defined in common_interfaces.go

// EventHandler is defined in common_interfaces.go

// WebSocket Types

type WebSocketConnection interface {
	ID() string
	Send(message []byte) error
	Close() error
	IsAlive() bool
}

// Additional Types

type OrderModification struct {
	Quantity *float64 `json:"quantity,omitempty"`
	Price    *float64 `json:"price,omitempty"`
}

type Portfolio struct {
	AccountID string      `json:"account_id"`
	Positions []*Position `json:"positions"`
	TotalValue float64    `json:"total_value"`
	Cash       float64    `json:"cash"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// RiskMetrics is defined in core_interfaces.go

type MatchingStats struct {
	OrdersProcessed   int64     `json:"orders_processed"`
	TradesExecuted    int64     `json:"trades_executed"`
	AverageLatency    time.Duration `json:"average_latency"`
	MaxLatency        time.Duration `json:"max_latency"`
	Timestamp         time.Time `json:"timestamp"`
}

// MarketData is defined in exchange_interface.go

type Ticker struct {
	Symbol    string    `json:"symbol"`
	LastPrice float64   `json:"last_price"`
	BidPrice  float64   `json:"bid_price"`
	AskPrice  float64   `json:"ask_price"`
	Volume    float64   `json:"volume"`
	Change    float64   `json:"change"`
	Timestamp time.Time `json:"timestamp"`
}

type Candle struct {
	Symbol    string    `json:"symbol"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

type PerformanceReport struct {
	Latencies   map[string]time.Duration `json:"latencies"`
	Throughput  map[string]float64       `json:"throughput"`
	CPUUsage    float64                  `json:"cpu_usage"`
	MemoryUsage uint64                   `json:"memory_usage"`
	Timestamp   time.Time                `json:"timestamp"`
}
