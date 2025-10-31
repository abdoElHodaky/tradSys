package strategies

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// NewUnifiedStrategyEngine creates a new unified strategy engine
func NewUnifiedStrategyEngine(config *StrategyConfig, logger *zap.Logger) *UnifiedStrategyEngine {
	return &UnifiedStrategyEngine{
		config:      config,
		logger:      logger,
		strategies:  make(map[string]TradingStrategy),
		executor:    NewStrategyExecutor(logger),
		monitor:     NewStrategyMonitor(logger),
		metrics:     &StrategyMetrics{LastUpdateTime: time.Now()},
		stopChannel: make(chan struct{}),
	}
}

// NewStrategyExecutor creates a new strategy executor
func NewStrategyExecutor(logger *zap.Logger) *StrategyExecutor {
	return &StrategyExecutor{
		orderChannel: make(chan *types.Order, 1000),
		logger:       logger,
	}
}

// NewStrategyMonitor creates a new strategy monitor
func NewStrategyMonitor(logger *zap.Logger) *StrategyMonitor {
	return &StrategyMonitor{
		positions: make(map[string]*Position),
		logger:    logger,
	}
}

// Start starts the unified strategy engine
func (e *UnifiedStrategyEngine) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 0, 1) {
		return fmt.Errorf("strategy engine is already running")
	}

	e.logger.Info("Starting unified strategy engine",
		zap.Int("enabled_strategies", len(e.config.EnabledStrategies)),
		zap.Duration("execution_interval", e.config.ExecutionInterval))

	// Start execution loop
	go e.executionLoop(ctx)

	// Start monitoring loop if enabled
	if e.config.MonitoringEnabled {
		go e.monitoringLoop(ctx)
	}

	return nil
}

// Stop stops the unified strategy engine
func (e *UnifiedStrategyEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 1, 0) {
		return fmt.Errorf("strategy engine is not running")
	}

	e.logger.Info("Stopping unified strategy engine")

	// Signal stop to all goroutines
	close(e.stopChannel)

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

	return nil
}

// RegisterStrategy registers a new trading strategy
func (e *UnifiedStrategyEngine) RegisterStrategy(strategy TradingStrategy) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	strategyID := strategy.GetID()
	if _, exists := e.strategies[strategyID]; exists {
		return fmt.Errorf("strategy %s already registered", strategyID)
	}

	e.strategies[strategyID] = strategy
	e.logger.Info("Registered strategy",
		zap.String("strategy_id", strategyID),
		zap.String("strategy_name", strategy.GetName()))

	return nil
}

// UnregisterStrategy unregisters a trading strategy
func (e *UnifiedStrategyEngine) UnregisterStrategy(strategyID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	strategy, exists := e.strategies[strategyID]
	if !exists {
		return fmt.Errorf("strategy %s not found", strategyID)
	}

	// Stop the strategy
	if err := strategy.Stop(); err != nil {
		e.logger.Error("Failed to stop strategy during unregistration",
			zap.String("strategy", strategyID),
			zap.Error(err))
	}

	delete(e.strategies, strategyID)
	e.logger.Info("Unregistered strategy", zap.String("strategy_id", strategyID))

	return nil
}

// ProcessMarketData processes market data through all enabled strategies
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

		// Process each signal
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

// GetStrategy returns a strategy by ID
func (e *UnifiedStrategyEngine) GetStrategy(strategyID string) (TradingStrategy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	strategy, exists := e.strategies[strategyID]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", strategyID)
	}

	return strategy, nil
}

// GetAllStrategies returns all registered strategies
func (e *UnifiedStrategyEngine) GetAllStrategies() map[string]TradingStrategy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	strategies := make(map[string]TradingStrategy)
	for id, strategy := range e.strategies {
		strategies[id] = strategy
	}

	return strategies
}

// GetEnabledStrategies returns all enabled strategies
func (e *UnifiedStrategyEngine) GetEnabledStrategies() map[string]TradingStrategy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	strategies := make(map[string]TradingStrategy)
	for id, strategy := range e.strategies {
		if strategy.IsEnabled() {
			strategies[id] = strategy
		}
	}

	return strategies
}

// UpdatePosition updates a position for monitoring
func (e *UnifiedStrategyEngine) UpdatePosition(position *Position) error {
	return e.monitor.UpdatePosition(position)
}

// UpdatePosition updates a position in the monitor
func (m *StrategyMonitor) UpdatePosition(position *Position) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.positions[position.Symbol] = position
	m.logger.Debug("Updated position",
		zap.String("symbol", position.Symbol),
		zap.Float64("quantity", position.Quantity),
		zap.Float64("unrealized_pnl", position.UnrealizedPnL))

	return nil
}

// GetPosition returns a position by symbol
func (m *StrategyMonitor) GetPosition(symbol string) (*Position, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	position, exists := m.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("position for symbol %s not found", symbol)
	}

	return position, nil
}

// GetAllPositions returns all positions
func (m *StrategyMonitor) GetAllPositions() map[string]*Position {
	m.mu.RLock()
	defer m.mu.RUnlock()

	positions := make(map[string]*Position)
	for symbol, position := range m.positions {
		positions[symbol] = position
	}

	return positions
}

// NewMomentumStrategy creates a new momentum strategy
func NewMomentumStrategy(id, name string, threshold float64, lookback int, logger *zap.Logger) *MomentumStrategy {
	return &MomentumStrategy{
		id:        id,
		name:      name,
		enabled:   true,
		threshold: threshold,
		lookback:  lookback,
		metrics:   &StrategyMetrics{LastUpdateTime: time.Now()},
		logger:    logger,
	}
}

// GetID returns the strategy ID
func (s *MomentumStrategy) GetID() string {
	return s.id
}

// GetName returns the strategy name
func (s *MomentumStrategy) GetName() string {
	return s.name
}

// Initialize initializes the strategy with configuration
func (s *MomentumStrategy) Initialize(config map[string]interface{}) error {
	if threshold, ok := config["threshold"].(float64); ok {
		s.threshold = threshold
	}
	if lookback, ok := config["lookback"].(int); ok {
		s.lookback = lookback
	}
	
	s.logger.Info("Initialized momentum strategy",
		zap.String("strategy_id", s.id),
		zap.Float64("threshold", s.threshold),
		zap.Int("lookback", s.lookback))
	
	return nil
}

// GenerateSignals generates trading signals based on momentum
func (s *MomentumStrategy) GenerateSignals(data *MarketData) ([]*TradingSignal, error) {
	var signals []*TradingSignal

	// Simple momentum calculation (in real implementation, would use historical data)
	// This is a simplified example
	momentum := data.Price * 0.01 // Placeholder calculation

	if momentum > s.threshold {
		// Strong upward momentum, buy signal
		signals = append(signals, &TradingSignal{
			StrategyID: s.id,
			Symbol:     data.Symbol,
			Action:     SignalActionBuy,
			Quantity:   100, // Fixed quantity for simplicity
			Confidence: math.Min(momentum/s.threshold, 1.0),
			Timestamp:  time.Now(),
		})
	} else if momentum < -s.threshold {
		// Strong downward momentum, sell signal
		signals = append(signals, &TradingSignal{
			StrategyID: s.id,
			Symbol:     data.Symbol,
			Action:     SignalActionSell,
			Quantity:   100, // Fixed quantity for simplicity
			Confidence: math.Min(math.Abs(momentum)/s.threshold, 1.0),
			Timestamp:  time.Now(),
		})
	}

	return signals, nil
}

// UpdatePosition updates the strategy's position
func (s *MomentumStrategy) UpdatePosition(position *Position) error {
	// Update strategy metrics based on position
	return nil
}

// GetMetrics returns strategy metrics
func (s *MomentumStrategy) GetMetrics() *StrategyMetrics {
	return s.metrics
}

// IsEnabled returns whether the strategy is enabled
func (s *MomentumStrategy) IsEnabled() bool {
	return s.enabled
}

// Stop stops the strategy
func (s *MomentumStrategy) Stop() error {
	s.enabled = false
	return nil
}
