package interfaces

import (
	"context"
	"time"
)

// Supporting types for optimized interfaces

// OrderResult represents the result of an order operation
type OrderResult struct {
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderModification represents modifications to an existing order
type OrderModification struct {
	Quantity *float64 `json:"quantity,omitempty"`
	Price    *float64 `json:"price,omitempty"`
}

// RiskCheckResult represents the result of a risk check
type RiskCheckResult struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

// Position represents a trading position
type Position struct {
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// Portfolio represents a trading portfolio
type Portfolio struct {
	Positions []Position `json:"positions"`
	Balance   float64    `json:"balance"`
}

// Ticker represents market ticker data
type Ticker struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

// Candle represents OHLCV candle data
type Candle struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// Trade represents a completed trade
type Trade struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	BuyOrderID   string    `json:"buy_order_id"`
	SellOrderID  string    `json:"sell_order_id"`
	Price        float64   `json:"price"`
	Quantity     float64   `json:"quantity"`
	Timestamp    time.Time `json:"timestamp"`
	BuyUserID    string    `json:"buy_user_id"`
	SellUserID   string    `json:"sell_user_id"`
}

// OrderBook represents the current state of buy and sell orders
type OrderBook struct {
	Symbol string             `json:"symbol"`
	Bids   []*OrderBookLevel  `json:"bids"`
	Asks   []*OrderBookLevel  `json:"asks"`
}

// OrderBookLevel represents a price level in the order book
type OrderBookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Count    int     `json:"count"`
}

// PerformanceReport represents performance metrics
type PerformanceReport struct {
	Timestamp       time.Time `json:"timestamp"`
	OrdersProcessed uint64    `json:"orders_processed"`
	TradesExecuted  uint64    `json:"trades_executed"`
	Latency         time.Duration `json:"latency"`
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

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



// Common Types and Structures
