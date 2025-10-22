package services

import (
	"context"
	"time"
)

// OrderService defines the interface for order management operations
type OrderService interface {
	// Order lifecycle operations
	CreateOrder(ctx context.Context, order *Order) (*Order, error)
	UpdateOrder(ctx context.Context, id string, updates *OrderUpdate) (*Order, error)
	CancelOrder(ctx context.Context, id string) error
	GetOrder(ctx context.Context, id string) (*Order, error)
	ListOrders(ctx context.Context, filter *OrderFilter) ([]*Order, error)
	
	// Order execution operations
	ExecuteOrder(ctx context.Context, id string) (*ExecutionResult, error)
	GetOrderStatus(ctx context.Context, id string) (*OrderStatus, error)
}

// SettlementService defines the interface for settlement processing
type SettlementService interface {
	// Settlement operations
	ProcessSettlement(ctx context.Context, trade *Trade) (*Settlement, error)
	GetSettlement(ctx context.Context, id string) (*Settlement, error)
	ListSettlements(ctx context.Context, filter *SettlementFilter) ([]*Settlement, error)
	
	// Batch operations
	ProcessBatchSettlement(ctx context.Context, trades []*Trade) ([]*Settlement, error)
	GetPendingSettlements(ctx context.Context) ([]*Settlement, error)
}

// RiskService defines the interface for risk management operations
type RiskService interface {
	// Risk assessment
	CheckRisk(ctx context.Context, order *Order) (*RiskCheckResult, error)
	ValidatePosition(ctx context.Context, position *Position) error
	GetRiskMetrics(ctx context.Context, portfolio *Portfolio) (*RiskMetrics, error)
	
	// Risk monitoring
	MonitorRisk(ctx context.Context) error
	GetRiskLimits(ctx context.Context, accountID string) (*RiskLimits, error)
	UpdateRiskLimits(ctx context.Context, accountID string, limits *RiskLimits) error
}

// StrategyService defines the interface for trading strategy operations
type StrategyService interface {
	// Strategy management
	CreateStrategy(ctx context.Context, strategy *Strategy) (*Strategy, error)
	UpdateStrategy(ctx context.Context, id string, updates *StrategyUpdate) (*Strategy, error)
	DeleteStrategy(ctx context.Context, id string) error
	GetStrategy(ctx context.Context, id string) (*Strategy, error)
	ListStrategies(ctx context.Context, filter *StrategyFilter) ([]*Strategy, error)
	
	// Strategy execution
	StartStrategy(ctx context.Context, id string) error
	StopStrategy(ctx context.Context, id string) error
	GetStrategyStatus(ctx context.Context, id string) (*StrategyStatus, error)
}

// PairsService defines the interface for trading pairs operations
type PairsService interface {
	// Pairs management
	GetPair(ctx context.Context, symbol string) (*TradingPair, error)
	ListPairs(ctx context.Context, filter *PairFilter) ([]*TradingPair, error)
	GetPairInfo(ctx context.Context, symbol string) (*PairInfo, error)
	
	// Market data
	GetTicker(ctx context.Context, symbol string) (*Ticker, error)
	GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error)
	GetTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error)
}

// Common types used across services

// Order represents a trading order
type Order struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"` // "buy" or "sell"
	Type        string    `json:"type"` // "market", "limit", "stop", etc.
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price,omitempty"`
	StopPrice   float64   `json:"stop_price,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ExecutedAt  *time.Time `json:"executed_at,omitempty"`
}

// OrderUpdate represents order update parameters
type OrderUpdate struct {
	Quantity  *float64 `json:"quantity,omitempty"`
	Price     *float64 `json:"price,omitempty"`
	StopPrice *float64 `json:"stop_price,omitempty"`
}

// OrderFilter represents order filtering parameters
type OrderFilter struct {
	AccountID *string    `json:"account_id,omitempty"`
	Symbol    *string    `json:"symbol,omitempty"`
	Side      *string    `json:"side,omitempty"`
	Status    *string    `json:"status,omitempty"`
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// OrderStatus represents order execution status
type OrderStatus struct {
	ID              string    `json:"id"`
	Status          string    `json:"status"`
	FilledQuantity  float64   `json:"filled_quantity"`
	RemainingQuantity float64 `json:"remaining_quantity"`
	AveragePrice    float64   `json:"average_price"`
	LastUpdated     time.Time `json:"last_updated"`
}

// ExecutionResult represents order execution result
type ExecutionResult struct {
	OrderID       string    `json:"order_id"`
	ExecutedPrice float64   `json:"executed_price"`
	ExecutedQty   float64   `json:"executed_quantity"`
	Commission    float64   `json:"commission"`
	ExecutedAt    time.Time `json:"executed_at"`
}

// Trade represents a completed trade
type Trade struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Commission float64  `json:"commission"`
	Timestamp time.Time `json:"timestamp"`
}

// Settlement represents a trade settlement
type Settlement struct {
	ID          string    `json:"id"`
	TradeID     string    `json:"trade_id"`
	Status      string    `json:"status"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SettlementFilter represents settlement filtering parameters
type SettlementFilter struct {
	TradeID   *string    `json:"trade_id,omitempty"`
	Status    *string    `json:"status,omitempty"`
	Currency  *string    `json:"currency,omitempty"`
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// Position represents a trading position
type Position struct {
	ID           string  `json:"id"`
	AccountID    string  `json:"account_id"`
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	Quantity     float64 `json:"quantity"`
	AveragePrice float64 `json:"average_price"`
	MarketValue  float64 `json:"market_value"`
	UnrealizedPL float64 `json:"unrealized_pl"`
}

// Portfolio represents a trading portfolio
type Portfolio struct {
	ID         string      `json:"id"`
	AccountID  string      `json:"account_id"`
	Positions  []*Position `json:"positions"`
	TotalValue float64     `json:"total_value"`
	Cash       float64     `json:"cash"`
}

// RiskCheckResult represents risk assessment result
type RiskCheckResult struct {
	Approved bool     `json:"approved"`
	Reasons  []string `json:"reasons,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// RiskMetrics represents portfolio risk metrics
type RiskMetrics struct {
	VaR           float64 `json:"var"`           // Value at Risk
	MaxDrawdown   float64 `json:"max_drawdown"`
	Sharpe        float64 `json:"sharpe"`
	Beta          float64 `json:"beta"`
	Volatility    float64 `json:"volatility"`
	Exposure      float64 `json:"exposure"`
}

// RiskLimits represents risk management limits
type RiskLimits struct {
	MaxPositionSize float64 `json:"max_position_size"`
	MaxDailyLoss    float64 `json:"max_daily_loss"`
	MaxLeverage     float64 `json:"max_leverage"`
	MaxExposure     float64 `json:"max_exposure"`
}

// Strategy represents a trading strategy
type Strategy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// StrategyUpdate represents strategy update parameters
type StrategyUpdate struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Parameters  *map[string]interface{} `json:"parameters,omitempty"`
}

// StrategyFilter represents strategy filtering parameters
type StrategyFilter struct {
	Type   *string `json:"type,omitempty"`
	Status *string `json:"status,omitempty"`
	Limit  int     `json:"limit,omitempty"`
	Offset int     `json:"offset,omitempty"`
}

// StrategyStatus represents strategy execution status
type StrategyStatus struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	RunningTime   time.Duration `json:"running_time"`
	OrdersPlaced  int       `json:"orders_placed"`
	ProfitLoss    float64   `json:"profit_loss"`
	LastUpdated   time.Time `json:"last_updated"`
}

// TradingPair represents a trading pair
type TradingPair struct {
	Symbol      string  `json:"symbol"`
	BaseAsset   string  `json:"base_asset"`
	QuoteAsset  string  `json:"quote_asset"`
	Status      string  `json:"status"`
	MinQuantity float64 `json:"min_quantity"`
	MaxQuantity float64 `json:"max_quantity"`
	StepSize    float64 `json:"step_size"`
	MinPrice    float64 `json:"min_price"`
	MaxPrice    float64 `json:"max_price"`
	TickSize    float64 `json:"tick_size"`
}

// PairInfo represents detailed pair information
type PairInfo struct {
	Symbol       string  `json:"symbol"`
	Volume24h    float64 `json:"volume_24h"`
	PriceChange  float64 `json:"price_change"`
	PriceChangePercent float64 `json:"price_change_percent"`
	HighPrice    float64 `json:"high_price"`
	LowPrice     float64 `json:"low_price"`
	LastPrice    float64 `json:"last_price"`
}

// PairFilter represents pair filtering parameters
type PairFilter struct {
	BaseAsset  *string `json:"base_asset,omitempty"`
	QuoteAsset *string `json:"quote_asset,omitempty"`
	Status     *string `json:"status,omitempty"`
	Limit      int     `json:"limit,omitempty"`
	Offset     int     `json:"offset,omitempty"`
}

// Ticker represents market ticker data
type Ticker struct {
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	Bid      float64 `json:"bid"`
	Ask      float64 `json:"ask"`
	Volume   float64 `json:"volume"`
	Change   float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderBook represents market order book
type OrderBook struct {
	Symbol    string           `json:"symbol"`
	Bids      []OrderBookEntry `json:"bids"`
	Asks      []OrderBookEntry `json:"asks"`
	Timestamp time.Time        `json:"timestamp"`
}

// OrderBookEntry represents an order book entry
type OrderBookEntry struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}
