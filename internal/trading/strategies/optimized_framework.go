package strategies

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool/performance"
	"github.com/abdoElHodaky/tradSys/internal/performance/latency"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// OptimizedStrategyManager is an enhanced version of StrategyManager
// optimized for high-frequency trading scenarios
type OptimizedStrategyManager struct {
	logger             *zap.Logger
	strategies         map[string]Strategy
	strategyPriorities map[string]int
	running            map[string]bool
	mu                 sync.RWMutex

	// Worker pool for strategy execution
	workerPool chan struct{}

	// Object pools to reduce GC pressure
	marketDataPool *pools.MarketDataPool
	orderPool      *pools.OrderPool

	// Latency tracking
	latencyTracker *latency.LatencyTracker

	// Statistics
	processedMarketData uint64
	processedOrders     uint64

	// Circuit breaker
	circuitBreakerEnabled bool
	circuitBreakerTripped int32 // atomic
}

// NewOptimizedStrategyManager creates a new optimized strategy manager
func NewOptimizedStrategyManager(logger *zap.Logger, workerCount int) *OptimizedStrategyManager {
	// Default to number of CPUs if workerCount is not specified
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	return &OptimizedStrategyManager{
		logger:                logger,
		strategies:            make(map[string]Strategy),
		strategyPriorities:    make(map[string]int),
		running:               make(map[string]bool),
		workerPool:            make(chan struct{}, workerCount),
		marketDataPool:        pools.NewMarketDataPool(),
		orderPool:             pools.NewOrderPool(),
		latencyTracker:        latency.NewLatencyTracker(logger),
		circuitBreakerEnabled: true,
	}
}

// RegisterStrategy registers a strategy with optional priority
// Higher priority (lower number) strategies are executed first
func (m *OptimizedStrategyManager) RegisterStrategy(strategy Strategy, priority int) error {
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
func (m *OptimizedStrategyManager) UnregisterStrategy(name string) error {
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

// StartStrategy starts a strategy
func (m *OptimizedStrategyManager) StartStrategy(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return ErrStrategyNotFound
	}

	if m.running[name] {
		return ErrStrategyAlreadyRunning
	}

	startTime := time.Now()
	if err := strategy.Start(ctx); err != nil {
		return err
	}
	m.latencyTracker.TrackStrategyExecution(name+"_start", startTime)

	m.running[name] = true

	m.logger.Info("Strategy started", zap.String("name", name))

	return nil
}

// StopStrategy stops a strategy
func (m *OptimizedStrategyManager) StopStrategy(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return ErrStrategyNotFound
	}

	if !m.running[name] {
		return ErrStrategyNotRunning
	}

	startTime := time.Now()
	if err := strategy.Stop(ctx); err != nil {
		return err
	}
	m.latencyTracker.TrackStrategyExecution(name+"_stop", startTime)

	m.running[name] = false

	m.logger.Info("Strategy stopped", zap.String("name", name))

	return nil
}

// GetStrategy returns a strategy
func (m *OptimizedStrategyManager) GetStrategy(name string) (Strategy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return nil, ErrStrategyNotFound
	}

	return strategy, nil
}

// ListStrategies returns a list of registered strategies
func (m *OptimizedStrategyManager) ListStrategies() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var strategies []string
	for name := range m.strategies {
		strategies = append(strategies, name)
	}

	return strategies
}

// IsStrategyRunning checks if a strategy is running
func (m *OptimizedStrategyManager) IsStrategyRunning(name string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.strategies[name]; !exists {
		return false, ErrStrategyNotFound
	}

	return m.running[name], nil
}

// ProcessMarketData processes market data updates for all running strategies
// with optimized worker pool and priority-based execution
func (m *OptimizedStrategyManager) ProcessMarketData(ctx context.Context, data *marketdata.MarketDataResponse) {
	// Check circuit breaker
	if m.circuitBreakerEnabled && atomic.LoadInt32(&m.circuitBreakerTripped) == 1 {
		m.logger.Warn("Circuit breaker tripped, skipping market data processing",
			zap.String("symbol", data.Symbol))
		return
	}

	// Track market data processing
	startTime := time.Now()
	defer m.latencyTracker.TrackMarketDataProcessing(data.Symbol, startTime)

	// Increment processed count
	atomic.AddUint64(&m.processedMarketData, 1)

	// Get prioritized strategies
	strategies := m.getPrioritizedStrategies()
	if len(strategies) == 0 {
		return
	}

	// Create a copy of the market data to avoid race conditions
	dataCopy := m.marketDataPool.Get()
	*dataCopy = *data

	// Try to get a worker from the pool
	select {
	case m.workerPool <- struct{}{}:
		go func() {
			defer func() {
				<-m.workerPool
				m.marketDataPool.Put(dataCopy)
			}()

			for _, s := range strategies {
				strategyStartTime := time.Now()
				if err := s.OnMarketData(ctx, dataCopy); err != nil {
					m.logger.Error("Failed to process market data",
						zap.Error(err),
						zap.String("strategy", s.GetName()),
						zap.String("symbol", dataCopy.Symbol))
				}
				m.latencyTracker.TrackStrategyExecution(s.GetName()+"_market_data", strategyStartTime)
			}
		}()
	default:
		// Worker pool is full, process in current goroutine
		m.logger.Warn("Worker pool full, processing market data in current goroutine",
			zap.String("symbol", data.Symbol))

		for _, s := range strategies {
			strategyStartTime := time.Now()
			if err := s.OnMarketData(ctx, dataCopy); err != nil {
				m.logger.Error("Failed to process market data",
					zap.Error(err),
					zap.String("strategy", s.GetName()),
					zap.String("symbol", dataCopy.Symbol))
			}
			m.latencyTracker.TrackStrategyExecution(s.GetName()+"_market_data", strategyStartTime)
		}

		m.marketDataPool.Put(dataCopy)
	}
}

// ProcessOrderUpdate processes order updates for all running strategies
// with optimized worker pool and priority-based execution
func (m *OptimizedStrategyManager) ProcessOrderUpdate(ctx context.Context, order *orders.OrderResponse) {
	// Check circuit breaker
	if m.circuitBreakerEnabled && atomic.LoadInt32(&m.circuitBreakerTripped) == 1 {
		m.logger.Warn("Circuit breaker tripped, skipping order update processing",
			zap.String("order_id", order.OrderId))
		return
	}

	// Track order processing
	startTime := time.Now()
	defer m.latencyTracker.TrackOrderProcessing(order.OrderId, startTime)

	// Increment processed count
	atomic.AddUint64(&m.processedOrders, 1)

	// Get prioritized strategies
	strategies := m.getPrioritizedStrategies()
	if len(strategies) == 0 {
		return
	}

	// Create a copy of the order to avoid race conditions
	orderCopy := m.orderPool.Get()
	*orderCopy = *order

	// Try to get a worker from the pool
	select {
	case m.workerPool <- struct{}{}:
		go func() {
			defer func() {
				<-m.workerPool
				m.orderPool.Put(orderCopy)
			}()

			for _, s := range strategies {
				strategyStartTime := time.Now()
				if err := s.OnOrderUpdate(ctx, orderCopy); err != nil {
					m.logger.Error("Failed to process order update",
						zap.Error(err),
						zap.String("strategy", s.GetName()),
						zap.String("order_id", orderCopy.OrderId))
				}
				m.latencyTracker.TrackStrategyExecution(s.GetName()+"_order_update", strategyStartTime)
			}
		}()
	default:
		// Worker pool is full, process in current goroutine
		m.logger.Warn("Worker pool full, processing order update in current goroutine",
			zap.String("order_id", order.OrderId))

		for _, s := range strategies {
			strategyStartTime := time.Now()
			if err := s.OnOrderUpdate(ctx, orderCopy); err != nil {
				m.logger.Error("Failed to process order update",
					zap.Error(err),
					zap.String("strategy", s.GetName()),
					zap.String("order_id", orderCopy.OrderId))
			}
			m.latencyTracker.TrackStrategyExecution(s.GetName()+"_order_update", strategyStartTime)
		}

		m.orderPool.Put(orderCopy)
	}
}

// SetStrategyPriority sets the priority of a strategy
// Lower numbers indicate higher priority
func (m *OptimizedStrategyManager) SetStrategyPriority(name string, priority int) error {
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
func (m *OptimizedStrategyManager) GetStrategyPriority(name string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.strategies[name]; !exists {
		return 0, ErrStrategyNotFound
	}

	return m.strategyPriorities[name], nil
}

// EnableCircuitBreaker enables the circuit breaker
func (m *OptimizedStrategyManager) EnableCircuitBreaker() {
	m.circuitBreakerEnabled = true
	atomic.StoreInt32(&m.circuitBreakerTripped, 0)
	m.logger.Info("Circuit breaker enabled")
}

// DisableCircuitBreaker disables the circuit breaker
func (m *OptimizedStrategyManager) DisableCircuitBreaker() {
	m.circuitBreakerEnabled = false
	atomic.StoreInt32(&m.circuitBreakerTripped, 0)
	m.logger.Info("Circuit breaker disabled")
}

// TripCircuitBreaker trips the circuit breaker
func (m *OptimizedStrategyManager) TripCircuitBreaker() {
	if m.circuitBreakerEnabled {
		atomic.StoreInt32(&m.circuitBreakerTripped, 1)
		m.logger.Warn("Circuit breaker tripped")
	}
}

// ResetCircuitBreaker resets the circuit breaker
func (m *OptimizedStrategyManager) ResetCircuitBreaker() {
	atomic.StoreInt32(&m.circuitBreakerTripped, 0)
	m.logger.Info("Circuit breaker reset")
}

// IsCircuitBreakerTripped checks if the circuit breaker is tripped
func (m *OptimizedStrategyManager) IsCircuitBreakerTripped() bool {
	return atomic.LoadInt32(&m.circuitBreakerTripped) == 1
}

// GetLatencyTracker returns the latency tracker
func (m *OptimizedStrategyManager) GetLatencyTracker() *latency.LatencyTracker {
	return m.latencyTracker
}

// GetProcessedCounts returns the number of processed market data and order updates
func (m *OptimizedStrategyManager) GetProcessedCounts() (marketData, orders uint64) {
	return atomic.LoadUint64(&m.processedMarketData), atomic.LoadUint64(&m.processedOrders)
}

// getPrioritizedStrategies returns a slice of running strategies sorted by priority
func (m *OptimizedStrategyManager) getPrioritizedStrategies() []Strategy {
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
	// Using a simple bubble sort for clarity, could use sort.Slice in production
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
