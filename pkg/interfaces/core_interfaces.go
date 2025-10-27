package interfaces

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// MatchingEngine defines the interface for order matching engines
type MatchingEngine interface {
	// ProcessOrder processes a new order and returns resulting trades
	ProcessOrder(ctx context.Context, order *types.Order) ([]*types.Trade, error)
	
	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, orderID string) error
	
	// GetOrderBook returns the current order book state
	GetOrderBook(symbol string) (*types.OrderBook, error)
	
	// GetMetrics returns engine performance metrics
	GetMetrics() *EngineMetrics
	
	// Start starts the matching engine
	Start(ctx context.Context) error
	
	// Stop stops the matching engine gracefully
	Stop(ctx context.Context) error
	
	// Subscribe to order book updates
	SubscribeOrderBook(symbol string, callback func(*types.OrderBook)) error
	
	// Subscribe to trade updates
	SubscribeTrades(symbol string, callback func(*types.Trade)) error
}

// OrderService defines the interface for order management
type OrderService interface {
	// CreateOrder creates a new order
	CreateOrder(ctx context.Context, order *types.Order) error
	
	// GetOrder retrieves an order by ID
	GetOrder(ctx context.Context, orderID string) (*types.Order, error)
	
	// GetOrderByClientID retrieves an order by client order ID
	GetOrderByClientID(ctx context.Context, userID, clientOrderID string) (*types.Order, error)
	
	// ListOrders lists orders for a user with optional filters
	ListOrders(ctx context.Context, userID string, filters *OrderFilters) ([]*types.Order, error)
	
	// UpdateOrder updates an existing order
	UpdateOrder(ctx context.Context, order *types.Order) error
	
	// CancelOrder cancels an order
	CancelOrder(ctx context.Context, orderID string) error
	
	// CancelAllOrders cancels all orders for a user
	CancelAllOrders(ctx context.Context, userID string) error
}

// OrderRepository defines the interface for order data access
type OrderRepository interface {
	// Create creates a new order
	Create(ctx context.Context, order *types.Order) error
	
	// GetByID retrieves an order by ID
	GetByID(ctx context.Context, orderID string) (*types.Order, error)
	
	// GetByClientOrderID retrieves an order by client order ID
	GetByClientOrderID(ctx context.Context, userID, clientOrderID string) (*types.Order, error)
	
	// Update updates an existing order
	Update(ctx context.Context, order *types.Order) error
	
	// Delete deletes an order
	Delete(ctx context.Context, orderID string) error
	
	// ListByUser lists orders for a user
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*types.Order, error)
	
	// ListBySymbol lists orders for a symbol
	ListBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*types.Order, error)
	
	// ListActiveOrders lists all active orders
	ListActiveOrders(ctx context.Context) ([]*types.Order, error)
}

// TradeService defines the interface for trade management
type TradeService interface {
	// CreateTrade creates a new trade
	CreateTrade(ctx context.Context, trade *types.Trade) error
	
	// GetTrade retrieves a trade by ID
	GetTrade(ctx context.Context, tradeID string) (*types.Trade, error)
	
	// ListTrades lists trades with optional filters
	ListTrades(ctx context.Context, filters *TradeFilters) ([]*types.Trade, error)
	
	// GetTradesByOrder gets all trades for an order
	GetTradesByOrder(ctx context.Context, orderID string) ([]*types.Trade, error)
	
	// GetTradesByUser gets all trades for a user
	GetTradesByUser(ctx context.Context, userID string, limit, offset int) ([]*types.Trade, error)
}

// TradeRepository defines the interface for trade data access
type TradeRepository interface {
	// Create creates a new trade
	Create(ctx context.Context, trade *types.Trade) error
	
	// GetByID retrieves a trade by ID
	GetByID(ctx context.Context, tradeID string) (*types.Trade, error)
	
	// ListBySymbol lists trades for a symbol
	ListBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*types.Trade, error)
	
	// ListByUser lists trades for a user
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*types.Trade, error)
	
	// ListByTimeRange lists trades within a time range
	ListByTimeRange(ctx context.Context, start, end time.Time, limit, offset int) ([]*types.Trade, error)
}

// MarketDataService defines the interface for market data management
type MarketDataService interface {
	// GetMarketData gets current market data for a symbol
	GetMarketData(ctx context.Context, symbol string) (*types.MarketData, error)
	
	// GetOHLCV gets OHLCV data for a symbol
	GetOHLCV(ctx context.Context, symbol string, interval string, limit int) ([]*types.OHLCV, error)
	
	// SubscribeMarketData subscribes to market data updates
	SubscribeMarketData(symbol string, callback func(*types.MarketData)) error
	
	// SubscribeOHLCV subscribes to OHLCV updates
	SubscribeOHLCV(symbol string, interval string, callback func(*types.OHLCV)) error
	
	// GetSymbols gets all available symbols
	GetSymbols(ctx context.Context) ([]*types.Symbol, error)
}

// PositionService defines the interface for position management
type PositionService interface {
	// GetPosition gets a position for a user and symbol
	GetPosition(ctx context.Context, userID, symbol string) (*types.Position, error)
	
	// ListPositions lists all positions for a user
	ListPositions(ctx context.Context, userID string) ([]*types.Position, error)
	
	// UpdatePosition updates a position
	UpdatePosition(ctx context.Context, position *types.Position) error
	
	// CalculateUnrealizedPnL calculates unrealized PnL for a position
	CalculateUnrealizedPnL(ctx context.Context, position *types.Position, currentPrice float64) (float64, error)
}

// RiskService defines the interface for risk management
type RiskService interface {
	// ValidateOrder validates an order against risk limits
	ValidateOrder(ctx context.Context, order *types.Order) error
	
	// CheckPositionLimits checks if a position exceeds limits
	CheckPositionLimits(ctx context.Context, userID, symbol string, quantity float64) error
	
	// CheckDailyLimits checks if daily trading limits are exceeded
	CheckDailyLimits(ctx context.Context, userID string, value float64) error
	
	// GetRiskMetrics gets risk metrics for a user
	GetRiskMetrics(ctx context.Context, userID string) (*RiskMetrics, error)
}

// OrderValidator defines the interface for order validation
type OrderValidator interface {
	// ValidateOrder validates an order
	ValidateOrder(order *types.Order) error
	
	// ValidatePrice validates a price
	ValidatePrice(symbol string, price float64) error
	
	// ValidateQuantity validates a quantity
	ValidateQuantity(symbol string, quantity float64) error
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	// PublishOrderEvent publishes an order event
	PublishOrderEvent(ctx context.Context, event *OrderEvent) error
	
	// PublishTradeEvent publishes a trade event
	PublishTradeEvent(ctx context.Context, event *TradeEvent) error
	
	// PublishMarketDataEvent publishes a market data event
	PublishMarketDataEvent(ctx context.Context, event *MarketDataEvent) error
}

// EventSubscriber defines the interface for subscribing to events
type EventSubscriber interface {
	// SubscribeOrderEvents subscribes to order events
	SubscribeOrderEvents(callback func(*OrderEvent)) error
	
	// SubscribeTradeEvents subscribes to trade events
	SubscribeTradeEvents(callback func(*TradeEvent)) error
	
	// SubscribeMarketDataEvents subscribes to market data events
	SubscribeMarketDataEvents(callback func(*MarketDataEvent)) error
}

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	// GetConfig gets configuration by key
	GetConfig(key string) (interface{}, error)
	
	// SetConfig sets configuration by key
	SetConfig(key string, value interface{}) error
	
	// LoadConfig loads configuration from source
	LoadConfig(source string) error
	
	// SaveConfig saves configuration to destination
	SaveConfig(destination string) error
}



// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// IncrementCounter increments a counter metric
	IncrementCounter(name string, tags map[string]string)
	
	// RecordGauge records a gauge metric
	RecordGauge(name string, value float64, tags map[string]string)
	
	// RecordHistogram records a histogram metric
	RecordHistogram(name string, value float64, tags map[string]string)
	
	// RecordTimer records a timer metric
	RecordTimer(name string, duration time.Duration, tags map[string]string)
}



// Supporting types and structures

// OrderFilters represents filters for order queries
type OrderFilters struct {
	Symbol    string             `json:"symbol,omitempty"`
	Side      types.OrderSide    `json:"side,omitempty"`
	Type      types.OrderType    `json:"type,omitempty"`
	Status    types.OrderStatus  `json:"status,omitempty"`
	StartTime *time.Time         `json:"start_time,omitempty"`
	EndTime   *time.Time         `json:"end_time,omitempty"`
	Limit     int                `json:"limit,omitempty"`
	Offset    int                `json:"offset,omitempty"`
}

// TradeFilters represents filters for trade queries
type TradeFilters struct {
	Symbol    string     `json:"symbol,omitempty"`
	UserID    string     `json:"user_id,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// EngineMetrics contains performance metrics for matching engines
type EngineMetrics struct {
	OrdersProcessed   uint64        `json:"orders_processed"`
	TradesExecuted    uint64        `json:"trades_executed"`
	AverageLatency    time.Duration `json:"average_latency"`
	ThroughputPerSec  float64       `json:"throughput_per_sec"`
	LastProcessedAt   time.Time     `json:"last_processed_at"`
	ActiveOrders      int           `json:"active_orders"`
	QueueDepth        int           `json:"queue_depth"`
}

// RiskMetrics contains risk metrics for users
type RiskMetrics struct {
	UserID              string    `json:"user_id"`
	TotalExposure       float64   `json:"total_exposure"`
	DailyVolume         float64   `json:"daily_volume"`
	MaxPositionSize     float64   `json:"max_position_size"`
	CurrentPositions    int       `json:"current_positions"`
	RiskScore           float64   `json:"risk_score"`
	LastUpdated         time.Time `json:"last_updated"`
}

// OrderEvent represents an order-related event
type OrderEvent struct {
	Type      string       `json:"type"`
	Order     *types.Order `json:"order"`
	Timestamp time.Time    `json:"timestamp"`
	UserID    string       `json:"user_id"`
}

// TradeEvent represents a trade-related event
type TradeEvent struct {
	Type      string       `json:"type"`
	Trade     *types.Trade `json:"trade"`
	Timestamp time.Time    `json:"timestamp"`
}

// MarketDataEvent represents a market data event
type MarketDataEvent struct {
	Type       string             `json:"type"`
	Symbol     string             `json:"symbol"`
	MarketData *types.MarketData  `json:"market_data,omitempty"`
	OrderBook  *types.OrderBook   `json:"order_book,omitempty"`
	OHLCV      *types.OHLCV       `json:"ohlcv,omitempty"`
	Timestamp  time.Time          `json:"timestamp"`
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status    string            `json:"status"`
	Message   string            `json:"message,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Event types constants
const (
	OrderEventCreated   = "order.created"
	OrderEventUpdated   = "order.updated"
	OrderEventCanceled  = "order.canceled"
	OrderEventFilled    = "order.filled"
	
	TradeEventExecuted  = "trade.executed"
	
	MarketDataEventTick = "marketdata.tick"
	MarketDataEventOHLCV = "marketdata.ohlcv"
	MarketDataEventOrderBook = "marketdata.orderbook"
)

// Health status constants
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusDegraded  = "degraded"
	HealthStatusUnhealthy = "unhealthy"
)
