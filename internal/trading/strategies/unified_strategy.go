package strategies

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// UnifiedStrategyEngine manages algorithmic trading strategies
type UnifiedStrategyEngine struct {
	config      *StrategyConfig
	logger      *zap.Logger
	strategies  map[string]TradingStrategy
	executor    *StrategyExecutor
	monitor     *StrategyMonitor
	metrics     *StrategyMetrics
	isRunning   int32
	stopChannel chan struct{}
	mu          sync.RWMutex
}

// StrategyConfig contains configuration for strategy engine
type StrategyConfig struct {
	EnabledStrategies   []string      `json:"enabled_strategies"`
	MaxConcurrentOrders int           `json:"max_concurrent_orders"`
	RiskLimits          RiskLimits    `json:"risk_limits"`
	ExecutionInterval   time.Duration `json:"execution_interval"`
	MonitoringEnabled   bool          `json:"monitoring_enabled"`
}

// RiskLimits defines risk limits for strategies
type RiskLimits struct {
	MaxPositionSize float64 `json:"max_position_size"`
	MaxDailyLoss    float64 `json:"max_daily_loss"`
	MaxDrawdown     float64 `json:"max_drawdown"`
}

// StrategyMetrics tracks strategy performance
type StrategyMetrics struct {
	TotalOrders      int64     `json:"total_orders"`
	SuccessfulTrades int64     `json:"successful_trades"`
	TotalPnL         float64   `json:"total_pnl"`
	WinRate          float64   `json:"win_rate"`
	AverageReturn    float64   `json:"average_return"`
	MaxDrawdown      float64   `json:"max_drawdown"`
	SharpeRatio      float64   `json:"sharpe_ratio"`
	LastUpdateTime   time.Time `json:"last_update_time"`
}

// TradingStrategy defines the interface for trading strategies
type TradingStrategy interface {
	GetID() string
	GetName() string
	Initialize(config map[string]interface{}) error
	GenerateSignals(marketData *MarketData) ([]*TradingSignal, error)
	UpdatePosition(position *Position) error
	GetMetrics() *StrategyMetrics
	IsEnabled() bool
	Stop() error
}

// StrategyExecutor executes trading signals
type StrategyExecutor struct {
	orderChannel chan *types.Order
	logger       *zap.Logger
	mu           sync.RWMutex
}

// StrategyMonitor monitors strategy performance
type StrategyMonitor struct {
	positions map[string]*Position
	logger    *zap.Logger
	mu        sync.RWMutex
}

// MarketData represents market data for strategies
type MarketData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	OHLC      *OHLC     `json:"ohlc,omitempty"`
}

// OHLC represents open, high, low, close data
type OHLC struct {
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

// TradingSignal represents a trading signal
type TradingSignal struct {
	StrategyID string                 `json:"strategy_id"`
	Symbol     string                 `json:"symbol"`
	Action     SignalAction           `json:"action"`
	Quantity   float64                `json:"quantity"`
	Price      float64                `json:"price,omitempty"`
	Confidence float64                `json:"confidence"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// SignalAction defines types of trading signals
type SignalAction string

const (
	SignalActionBuy   SignalAction = "buy"
	SignalActionSell  SignalAction = "sell"
	SignalActionHold  SignalAction = "hold"
	SignalActionClose SignalAction = "close"
)

// Position represents a trading position
type Position struct {
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	AveragePrice  float64   `json:"average_price"`
	CurrentPrice  float64   `json:"current_price"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	RealizedPnL   float64   `json:"realized_pnl"`
	LastUpdate    time.Time `json:"last_update"`
}

// Note: MeanReversionStrategy and MomentumStrategy are defined in their respective files

// NewUnifiedStrategyEngine creates a new unified strategy engine
func NewUnifiedStrategyEngine(config *StrategyConfig, logger *zap.Logger) *UnifiedStrategyEngine {
	engine := &UnifiedStrategyEngine{
		config:      config,
		logger:      logger,
		strategies:  make(map[string]TradingStrategy),
		metrics:     &StrategyMetrics{LastUpdateTime: time.Now()},
		stopChannel: make(chan struct{}),
	}

	// Initialize executor
	engine.executor = &StrategyExecutor{
		orderChannel: make(chan *types.Order, 1000),
		logger:       logger.Named("executor"),
	}

	// Initialize monitor
	engine.monitor = &StrategyMonitor{
		positions: make(map[string]*Position),
		logger:    logger.Named("monitor"),
	}

	return engine
}

// Start starts the unified strategy engine
func (e *UnifiedStrategyEngine) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 0, 1) {
		return fmt.Errorf("strategy engine is already running")
	}

	e.logger.Info("Starting unified strategy engine",
		zap.Any("config", e.config))

	// Register default strategies
	e.registerDefaultStrategies()

	// Start strategy execution loop
	go e.executionLoop(ctx)

	// Start monitoring if enabled
	if e.config.MonitoringEnabled {
		go e.monitoringLoop(ctx)
	}

	e.logger.Info("Unified strategy engine started successfully")
	return nil
}

// Stop stops the unified strategy engine
func (e *UnifiedStrategyEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 1, 0) {
		return fmt.Errorf("strategy engine is not running")
	}

	e.logger.Info("Stopping unified strategy engine")

	// Stop all strategies
	e.mu.RLock()
	for _, strategy := range e.strategies {
		if err := strategy.Stop(); err != nil {
			e.logger.Error("Failed to stop strategy",
				zap.String("strategy", strategy.GetID()),
				zap.Error(err))
		}
	}
	e.mu.RUnlock()

	close(e.stopChannel)
	e.logger.Info("Unified strategy engine stopped")
	return nil
}

// registerDefaultStrategies registers default trading strategies
func (e *UnifiedStrategyEngine) registerDefaultStrategies() {
	// Mean reversion strategy
	meanReversion := &MeanReversionStrategy{
		id:        "mean_reversion_001",
		name:      "Mean Reversion Strategy",
		enabled:   true,
		lookback:  20,
		threshold: 2.0,
		metrics:   &StrategyMetrics{LastUpdateTime: time.Now()},
		logger:    e.logger.Named("mean_reversion"),
	}

	// Momentum strategy
	momentum := &MomentumStrategy{
		id:        "momentum_001",
		name:      "Momentum Strategy",
		enabled:   true,
		period:    10,
		threshold: 0.02,
		metrics:   &StrategyMetrics{LastUpdateTime: time.Now()},
		logger:    e.logger.Named("momentum"),
	}

	e.RegisterStrategy(meanReversion)
	e.RegisterStrategy(momentum)

	e.logger.Info("Registered default strategies", zap.Int("count", 2))
}

// RegisterStrategy registers a trading strategy
func (e *UnifiedStrategyEngine) RegisterStrategy(strategy TradingStrategy) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.strategies[strategy.GetID()] = strategy
	e.logger.Info("Registered strategy",
		zap.String("id", strategy.GetID()),
		zap.String("name", strategy.GetName()))
}

// ProcessMarketData processes market data and generates signals
func (e *UnifiedStrategyEngine) ProcessMarketData(data *MarketData) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, strategy := range e.strategies {
		if !strategy.IsEnabled() {
			continue
		}

		signals, err := strategy.GenerateSignals(data)
		if err != nil {
			e.logger.Error("Strategy failed to generate signals",
				zap.String("strategy", strategy.GetID()),
				zap.Error(err))
			continue
		}

		// Process signals
		for _, signal := range signals {
			e.processSignal(signal)
		}
	}

	return nil
}

// processSignal processes a trading signal
func (e *UnifiedStrategyEngine) processSignal(signal *TradingSignal) {
	if signal.Action == SignalActionHold {
		return
	}

	// Convert signal to order
	order := &types.Order{
		ID:        fmt.Sprintf("strategy_%s_%d", signal.StrategyID, time.Now().UnixNano()),
		Symbol:    signal.Symbol,
		Quantity:  signal.Quantity,
		Type:      types.OrderTypeMarket,
		Status:    types.OrderStatusNew,
		CreatedAt: time.Now(),
		UserID:    signal.StrategyID,
	}

	switch signal.Action {
	case SignalActionBuy:
		order.Side = types.OrderSideBuy
	case SignalActionSell:
		order.Side = types.OrderSideSell
	}

	if signal.Price > 0 {
		order.Type = types.OrderTypeLimit
		order.Price = signal.Price
	}

	// Queue order for execution
	select {
	case e.executor.orderChannel <- order:
		atomic.AddInt64(&e.metrics.TotalOrders, 1)
		e.logger.Info("Queued strategy order",
			zap.String("strategy", signal.StrategyID),
			zap.String("symbol", signal.Symbol),
			zap.String("action", string(signal.Action)),
			zap.Float64("quantity", signal.Quantity))
	default:
		e.logger.Warn("Order queue full, dropping signal",
			zap.String("strategy", signal.StrategyID),
			zap.String("symbol", signal.Symbol))
	}
}

// executionLoop runs the strategy execution loop
func (e *UnifiedStrategyEngine) executionLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.ExecutionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Process any queued orders
			e.processQueuedOrders()
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// processQueuedOrders processes queued orders
func (e *UnifiedStrategyEngine) processQueuedOrders() {
	for {
		select {
		case order := <-e.executor.orderChannel:
			// In a real implementation, this would submit to the trading engine
			e.logger.Info("Processing strategy order",
				zap.String("order_id", order.ID),
				zap.String("symbol", order.Symbol),
				zap.String("side", string(order.Side)),
				zap.Float64("quantity", order.Quantity))
		default:
			return // No more orders to process
		}
	}
}

// monitoringLoop runs the strategy monitoring loop
func (e *UnifiedStrategyEngine) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute) // Monitor every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.updateMetrics()
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// updateMetrics updates strategy metrics
func (e *UnifiedStrategyEngine) updateMetrics() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	totalPnL := 0.0
	totalTrades := int64(0)

	for _, strategy := range e.strategies {
		metrics := strategy.GetMetrics()
		totalPnL += metrics.TotalPnL
		totalTrades += metrics.SuccessfulTrades
	}

	e.metrics.TotalPnL = totalPnL
	e.metrics.SuccessfulTrades = totalTrades
	if totalTrades > 0 {
		e.metrics.AverageReturn = totalPnL / float64(totalTrades)
	}
	e.metrics.LastUpdateTime = time.Now()
}

// GetMetrics returns current strategy metrics
func (e *UnifiedStrategyEngine) GetMetrics() *StrategyMetrics {
	return e.metrics
}

// Note: Strategy implementations are in their respective files
// - MeanReversionStrategy methods are in mean_reversion.go
// - MomentumStrategy methods are in momentum.go
