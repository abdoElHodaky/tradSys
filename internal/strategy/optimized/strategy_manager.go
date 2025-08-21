package optimized

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/resilience"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/workerpool"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// Common strategy errors
var (
	ErrStrategyNotFound        = errors.New("strategy not found")
	ErrStrategyAlreadyRunning  = errors.New("strategy already running")
	ErrStrategyNotRunning      = errors.New("strategy not running")
	ErrStrategyAlreadyRegistered = errors.New("strategy already registered")
)

// Strategy defines the interface for trading strategies
type Strategy interface {
	// GetName returns the name of the strategy
	GetName() string
	
	// Initialize initializes the strategy
	Initialize(ctx context.Context) error
	
	// Start starts the strategy
	Start(ctx context.Context) error
	
	// Stop stops the strategy
	Stop(ctx context.Context) error
	
	// IsRunning returns whether the strategy is running
	IsRunning() bool
	
	// OnMarketData processes market data updates
	OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error
	
	// OnOrderUpdate processes order updates
	OnOrderUpdate(ctx context.Context, order *orders.OrderResponse) error
	
	// GetPerformanceMetrics returns performance metrics for the strategy
	GetPerformanceMetrics() map[string]interface{}
}

// StrategyStats contains statistics for a strategy
type StrategyStats struct {
	Name            string
	ProcessedUpdates int64
	ExecutedTrades   int64
	PnL             float64
	LastUpdate      time.Time
}

// StrategyManager manages trading strategies with optimized performance
type StrategyManager struct {
	logger           *zap.Logger
	workerPool       *workerpool.WorkerPoolFactory
	circuitBreaker   *resilience.CircuitBreakerFactory
	metrics          *StrategyMetrics
	
	strategies       map[string]Strategy
	strategyPriorities map[string]int
	running          map[string]bool
	mu               sync.RWMutex
	
	// Statistics
	processedMarketData atomic.Int64
	processedOrders     atomic.Int64
}

// NewStrategyManager creates a new optimized strategy manager
func NewStrategyManager(
	logger *zap.Logger,
	workerPool *workerpool.WorkerPoolFactory,
	circuitBreaker *resilience.CircuitBreakerFactory,
	metrics *StrategyMetrics,
) *StrategyManager {
	return &StrategyManager{
		logger:             logger,
		workerPool:         workerPool,
		circuitBreaker:     circuitBreaker,
		metrics:            metrics,
		strategies:         make(map[string]Strategy),
		strategyPriorities: make(map[string]int),
		running:            make(map[string]bool),
	}
}

// RegisterStrategy registers a strategy with optional priority
// Higher priority (lower number) strategies are executed first
func (m *StrategyManager) RegisterStrategy(strategy Strategy, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	name := strategy.GetName()
	if _, exists := m.strategies[name]; exists {
		return ErrStrategyAlreadyRegistered
	}
	
	m.strategies[name] = strategy
	m.strategyPriorities[name] = priority
	m.running[name] = false
	
	m.logger.Info("Strategy registered", 
		zap.String("name", name),
		zap.Int("priority", priority))
	
	return nil
}

// UnregisterStrategy unregisters a strategy
func (m *StrategyManager) UnregisterStrategy(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.strategies[name]; !exists {
		return ErrStrategyNotFound
	}
	
	// Stop the strategy if it's running
	if m.running[name] {
		m.mu.Unlock()
		if err := m.StopStrategy(context.Background(), name); err != nil {
			m.mu.Lock()
			return err
		}
		m.mu.Lock()
	}
	
	delete(m.strategies, name)
	delete(m.strategyPriorities, name)
	delete(m.running, name)
	
	m.logger.Info("Strategy unregistered", zap.String("name", name))
	
	return nil
}

// StartStrategy starts a strategy with circuit breaker protection
func (m *StrategyManager) StartStrategy(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	strategy, exists := m.strategies[name]
	if !exists {
		return ErrStrategyNotFound
	}
	
	if m.running[name] {
		return ErrStrategyAlreadyRunning
	}
	
	// Use circuit breaker to protect against strategy initialization failures
	result := m.circuitBreaker.ExecuteWithFallback(
		"strategy-start-"+name,
		func() (interface{}, error) {
			startTime := time.Now()
			err := strategy.Start(ctx)
			m.metrics.RecordStrategyOperation(name, "start", time.Since(startTime))
			return nil, err
		},
		func(err error) (interface{}, error) {
			m.logger.Error("Circuit breaker triggered fallback for strategy start",
				zap.String("strategy", name),
				zap.Error(err))
			return nil, err
		},
	)
	
	if result.Error != nil {
		return result.Error
	}
	
	m.running[name] = true
	
	m.logger.Info("Strategy started", zap.String("name", name))
	
	return nil
}

// StopStrategy stops a strategy with circuit breaker protection
func (m *StrategyManager) StopStrategy(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	strategy, exists := m.strategies[name]
	if !exists {
		return ErrStrategyNotFound
	}
	
	if !m.running[name] {
		return ErrStrategyNotRunning
	}
	
	// Use circuit breaker to protect against strategy shutdown failures
	result := m.circuitBreaker.ExecuteWithFallback(
		"strategy-stop-"+name,
		func() (interface{}, error) {
			startTime := time.Now()
			err := strategy.Stop(ctx)
			m.metrics.RecordStrategyOperation(name, "stop", time.Since(startTime))
			return nil, err
		},
		func(err error) (interface{}, error) {
			m.logger.Error("Circuit breaker triggered fallback for strategy stop",
				zap.String("strategy", name),
				zap.Error(err))
			return nil, err
		},
	)
	
	if result.Error != nil {
		return result.Error
	}
	
	m.running[name] = false
	
	m.logger.Info("Strategy stopped", zap.String("name", name))
	
	return nil
}

// GetStrategy returns a strategy
func (m *StrategyManager) GetStrategy(name string) (Strategy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	strategy, exists := m.strategies[name]
	if !exists {
		return nil, ErrStrategyNotFound
	}
	
	return strategy, nil
}

// ListStrategies returns a list of registered strategies
func (m *StrategyManager) ListStrategies() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var strategies []string
	for name := range m.strategies {
		strategies = append(strategies, name)
	}
	
	return strategies
}

// IsStrategyRunning checks if a strategy is running
func (m *StrategyManager) IsStrategyRunning(name string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if _, exists := m.strategies[name]; !exists {
		return false, ErrStrategyNotFound
	}
	
	return m.running[name], nil
}

// ProcessMarketData processes market data updates for all running strategies
// using the worker pool and circuit breaker for protection
func (m *StrategyManager) ProcessMarketData(ctx context.Context, data *marketdata.MarketDataResponse) {
	// Increment processed count
	m.processedMarketData.Add(1)
	
	// Track market data processing
	startTime := time.Now()
	defer m.metrics.RecordMarketDataProcessing(data.Symbol, time.Since(startTime))
	
	// Get prioritized strategies
	strategies := m.getPrioritizedStrategies()
	if len(strategies) == 0 {
		return
	}
	
	// Submit task to worker pool
	err := m.workerPool.Submit("market-data-processor", func() {
		for _, s := range strategies {
			strategyName := s.GetName()
			
			// Use circuit breaker to protect against strategy failures
			result := m.circuitBreaker.ExecuteWithFallback(
				"strategy-market-data-"+strategyName,
				func() (interface{}, error) {
					strategyStartTime := time.Now()
					err := s.OnMarketData(ctx, data)
					m.metrics.RecordStrategyExecution(strategyName, "market_data", time.Since(strategyStartTime))
					return nil, err
				},
				func(err error) (interface{}, error) {
					m.logger.Error("Circuit breaker triggered fallback for market data processing",
						zap.String("strategy", strategyName),
						zap.String("symbol", data.Symbol),
						zap.Error(err))
					return nil, nil
				},
			)
			
			if result.Error != nil {
				m.logger.Error("Failed to process market data",
					zap.Error(result.Error),
					zap.String("strategy", strategyName),
					zap.String("symbol", data.Symbol))
			}
		}
	})
	
	if err != nil {
		m.logger.Error("Failed to submit market data processing task",
			zap.Error(err),
			zap.String("symbol", data.Symbol))
	}
}

// ProcessOrderUpdate processes order updates for all running strategies
// using the worker pool and circuit breaker for protection
func (m *StrategyManager) ProcessOrderUpdate(ctx context.Context, order *orders.OrderResponse) {
	// Increment processed count
	m.processedOrders.Add(1)
	
	// Track order processing
	startTime := time.Now()
	defer m.metrics.RecordOrderProcessing(order.OrderId, time.Since(startTime))
	
	// Get prioritized strategies
	strategies := m.getPrioritizedStrategies()
	if len(strategies) == 0 {
		return
	}
	
	// Submit task to worker pool
	err := m.workerPool.Submit("order-processor", func() {
		for _, s := range strategies {
			strategyName := s.GetName()
			
			// Use circuit breaker to protect against strategy failures
			result := m.circuitBreaker.ExecuteWithFallback(
				"strategy-order-update-"+strategyName,
				func() (interface{}, error) {
					strategyStartTime := time.Now()
					err := s.OnOrderUpdate(ctx, order)
					m.metrics.RecordStrategyExecution(strategyName, "order_update", time.Since(strategyStartTime))
					return nil, err
				},
				func(err error) (interface{}, error) {
					m.logger.Error("Circuit breaker triggered fallback for order update processing",
						zap.String("strategy", strategyName),
						zap.String("order_id", order.OrderId),
						zap.Error(err))
					return nil, nil
				},
			)
			
			if result.Error != nil {
				m.logger.Error("Failed to process order update",
					zap.Error(result.Error),
					zap.String("strategy", strategyName),
					zap.String("order_id", order.OrderId))
			}
		}
	})
	
	if err != nil {
		m.logger.Error("Failed to submit order update processing task",
			zap.Error(err),
			zap.String("order_id", order.OrderId))
	}
}

// SetStrategyPriority sets the priority of a strategy
// Lower numbers indicate higher priority
func (m *StrategyManager) SetStrategyPriority(name string, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.strategies[name]; !exists {
		return ErrStrategyNotFound
	}
	
	m.strategyPriorities[name] = priority
	
	m.logger.Info("Strategy priority updated",
		zap.String("name", name),
		zap.Int("priority", priority))
	
	return nil
}

// GetStrategyPriority gets the priority of a strategy
func (m *StrategyManager) GetStrategyPriority(name string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if _, exists := m.strategies[name]; !exists {
		return 0, ErrStrategyNotFound
	}
	
	return m.strategyPriorities[name], nil
}

// GetProcessedCounts returns the number of processed market data and order updates
func (m *StrategyManager) GetProcessedCounts() (marketData, orders int64) {
	return m.processedMarketData.Load(), m.processedOrders.Load()
}

// GetAllStrategyStats returns statistics for all strategies
func (m *StrategyManager) GetAllStrategyStats() map[string]*StrategyStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := make(map[string]*StrategyStats)
	
	for name, strategy := range m.strategies {
		metrics := strategy.GetPerformanceMetrics()
		
		// Extract metrics with type assertions
		processedUpdates, _ := metrics["processed_updates"].(int64)
		executedTrades, _ := metrics["executed_trades"].(int64)
		pnl, _ := metrics["pnl"].(float64)
		
		stats[name] = &StrategyStats{
			Name:            name,
			ProcessedUpdates: processedUpdates,
			ExecutedTrades:   executedTrades,
			PnL:             pnl,
			LastUpdate:      time.Now(),
		}
	}
	
	return stats
}

// getPrioritizedStrategies returns a slice of running strategies sorted by priority
func (m *StrategyManager) getPrioritizedStrategies() []Strategy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a slice of strategy-priority pairs
	type strategyPriorityPair struct {
		strategy Strategy
		priority int
	}
	
	var pairs []strategyPriorityPair
	
	for name, strategy := range m.strategies {
		if m.running[name] {
			pairs = append(pairs, strategyPriorityPair{
				strategy: strategy,
				priority: m.strategyPriorities[name],
			})
		}
	}
	
	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].priority > pairs[j].priority {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	
	// Extract just the strategies
	strategies := make([]Strategy, len(pairs))
	for i, pair := range pairs {
		strategies[i] = pair.strategy
	}
	
	return strategies
}

