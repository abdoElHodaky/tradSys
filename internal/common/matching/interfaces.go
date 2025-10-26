package matching

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of an order
type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
)

// Order represents a trading order with all necessary fields
type Order struct {
	ID             string    `json:"id"`
	ClientOrderID  string    `json:"client_order_id"`
	UserID         string    `json:"user_id"`
	Symbol         string    `json:"symbol"`
	Side           OrderSide `json:"side"`
	Type           OrderType `json:"type"`
	Price          float64   `json:"price"`
	Quantity       float64   `json:"quantity"`
	FilledQuantity float64   `json:"filled_quantity"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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

// MatchingEngine defines the interface for order matching engines
type MatchingEngine interface {
	// ProcessOrder processes a new order and returns resulting trades
	ProcessOrder(ctx context.Context, order *Order) ([]*Trade, error)
	
	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, orderID string) error
	
	// GetOrderBook returns the current order book state
	GetOrderBook(symbol string) (*OrderBook, error)
	
	// GetMetrics returns engine performance metrics
	GetMetrics() *EngineMetrics
	
	// Start starts the matching engine
	Start(ctx context.Context) error
	
	// Stop stops the matching engine gracefully
	Stop(ctx context.Context) error
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

// EngineMetrics contains performance metrics for the matching engine
type EngineMetrics struct {
	OrdersProcessed   uint64        `json:"orders_processed"`
	TradesExecuted    uint64        `json:"trades_executed"`
	AverageLatency    time.Duration `json:"average_latency"`
	ThroughputPerSec  float64       `json:"throughput_per_sec"`
	LastProcessedAt   time.Time     `json:"last_processed_at"`
}

// EngineConfig contains configuration for matching engines
type EngineConfig struct {
	MaxOrdersPerSymbol int           `json:"max_orders_per_symbol"`
	TickSize           float64       `json:"tick_size"`
	ProcessingTimeout  time.Duration `json:"processing_timeout"`
	EnableMetrics      bool          `json:"enable_metrics"`
	PoolSize           int           `json:"pool_size"`
}

// EngineFactory creates matching engines based on configuration
type EngineFactory interface {
	CreateEngine(config *EngineConfig) (MatchingEngine, error)
	GetSupportedEngineTypes() []string
}

// OrderValidator validates orders before processing
type OrderValidator interface {
	ValidateOrder(order *Order) error
}

// TradeNotifier notifies about completed trades
type TradeNotifier interface {
	NotifyTrade(trade *Trade) error
}

// PerformanceOptimizer optimizes engine performance
type PerformanceOptimizer interface {
	OptimizeEngine(engine MatchingEngine) error
	GetOptimizationMetrics() map[string]interface{}
}
