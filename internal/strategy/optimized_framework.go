package strategy

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"github.com/abdoElHodaky/tradSys/internal/trading/order"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

// OptimizedStrategyManager is a high-performance strategy manager
type OptimizedStrategyManager struct {
	strategies          map[string]Strategy
	strategyPriorities  map[string]int
	processedMarketData uint64
	processedOrders     uint64
	workerPool          chan struct{}
	marketDataPool      sync.Pool
	orderPool           sync.Pool
	logger              *zap.Logger
	mu                  sync.RWMutex
	maxWorkers          int
}

// NewOptimizedStrategyManager creates a new optimized strategy manager
func NewOptimizedStrategyManager(maxWorkers int, logger *zap.Logger) *OptimizedStrategyManager {
	if logger == nil {
		logger = zap.NewNop()
	}

	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	return &OptimizedStrategyManager{
		strategies:         make(map[string]Strategy),
		strategyPriorities: make(map[string]int),
		workerPool:         make(chan struct{}, maxWorkers),
		marketDataPool: sync.Pool{
			New: func() interface{} {
				return &market_data.MarketData{}
			},
		},
		orderPool: sync.Pool{
			New: func() interface{} {
				return &order.Order{}
			},
		},
		logger:     logger,
		maxWorkers: maxWorkers,
	}
}

// RegisterStrategy registers a strategy with the manager
func (m *OptimizedStrategyManager) RegisterStrategy(ctx context.Context, strategy Strategy, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := strategy.Name()
	if _, exists := m.strategies[name]; exists {
		return fmt.Errorf("strategy already registered: %s", name)
	}

	m.strategies[name] = strategy
	m.strategyPriorities[name] = priority

	m.logger.Info("Registered strategy",
		zap.String("strategy", name),
		zap.Int("priority", priority),
	)

	return nil
}

// UnregisterStrategy unregisters a strategy from the manager
func (m *OptimizedStrategyManager) UnregisterStrategy(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return fmt.Errorf("strategy not registered: %s", name)
	}

	// Shutdown the strategy
	if err := strategy.Shutdown(ctx); err != nil {
		m.logger.Error("Failed to shutdown strategy",
			zap.String("strategy", name),
			zap.Error(err),
		)
	}

	delete(m.strategies, name)
	delete(m.strategyPriorities, name)

	m.logger.Info("Unregistered strategy",
		zap.String("strategy", name),
	)

	return nil
}

// GetStrategy gets a strategy by name
func (m *OptimizedStrategyManager) GetStrategy(name string) (Strategy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy not registered: %s", name)
	}

	return strategy, nil
}

// GetRegisteredStrategies gets all registered strategies
func (m *OptimizedStrategyManager) GetRegisteredStrategies() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	strategies := make([]string, 0, len(m.strategies))
	for name := range m.strategies {
		strategies = append(strategies, name)
	}

	return strategies
}

// ProcessMarketData processes market data through all registered strategies
func (m *OptimizedStrategyManager) ProcessMarketData(ctx context.Context, data *market_data.MarketData) {
	// Increment processed count
	atomic.AddUint64(&m.processedMarketData, 1)
	
	// Get prioritized strategies
	strategies := m.getPrioritizedStrategies()
	if len(strategies) == 0 {
		return
	}
	
	// Create a copy of the market data to avoid race conditions
	dataCopy := m.marketDataPool.Get().(*market_data.MarketData)
	*dataCopy = *data
	
	// Try to get a worker from the pool
	select {
	case m.workerPool <- struct{}{}:
		go func() {
			defer func() {
				<-m.workerPool
				m.marketDataPool.Put(dataCopy)
			}()
			
			// Process the market data through each strategy
			for _, s := range strategies {
				strategy := s
				if !strategy.IsRunning() {
					continue
				}
				
				// Process the market data
				if err := strategy.ProcessMarketData(ctx, dataCopy); err != nil {
					m.logger.Error("Failed to process market data",
						zap.String("strategy", strategy.Name()),
						zap.Error(err),
					)
				}
			}
		}()
	default:
		// Worker pool is full, process synchronously
		m.logger.Debug("Worker pool full, processing market data synchronously")
		
		// Process the market data through each strategy
		for _, s := range strategies {
			strategy := s
			if !strategy.IsRunning() {
				continue
			}
			
			// Process the market data
			if err := strategy.ProcessMarketData(ctx, dataCopy); err != nil {
				m.logger.Error("Failed to process market data",
					zap.String("strategy", strategy.Name()),
					zap.Error(err),
				)
			}
		}
		
		// Return the copy to the pool
		m.marketDataPool.Put(dataCopy)
	}
}

// ProcessOrder processes an order through all registered strategies
func (m *OptimizedStrategyManager) ProcessOrder(ctx context.Context, order *order.Order) {
	// Increment processed count
	atomic.AddUint64(&m.processedOrders, 1)
	
	// Get prioritized strategies
	strategies := m.getPrioritizedStrategies()
	if len(strategies) == 0 {
		return
	}
	
	// Create a copy of the order to avoid race conditions
	orderCopy := m.orderPool.Get().(*order.Order)
	*orderCopy = *order
	
	// Try to get a worker from the pool
	select {
	case m.workerPool <- struct{}{}:
		go func() {
			defer func() {
				<-m.workerPool
				m.orderPool.Put(orderCopy)
			}()
			
			// Process the order through each strategy
			for _, s := range strategies {
				strategy := s
				if !strategy.IsRunning() {
					continue
				}
				
				// Process the order
				if err := strategy.ProcessOrder(ctx, orderCopy); err != nil {
					m.logger.Error("Failed to process order",
						zap.String("strategy", strategy.Name()),
						zap.Error(err),
					)
				}
			}
		}()
	default:
		// Worker pool is full, process synchronously
		m.logger.Debug("Worker pool full, processing order synchronously")
		
		// Process the order through each strategy
		for _, s := range strategies {
			strategy := s
			if !strategy.IsRunning() {
				continue
			}
			
			// Process the order
			if err := strategy.ProcessOrder(ctx, orderCopy); err != nil {
				m.logger.Error("Failed to process order",
					zap.String("strategy", strategy.Name()),
					zap.Error(err),
				)
			}
		}
		
		// Return the copy to the pool
		m.orderPool.Put(orderCopy)
	}
}

// GetStats gets the strategy manager statistics
func (m *OptimizedStrategyManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"registered_strategies":  len(m.strategies),
		"processed_market_data":  atomic.LoadUint64(&m.processedMarketData),
		"processed_orders":       atomic.LoadUint64(&m.processedOrders),
		"max_workers":            m.maxWorkers,
		"strategy_stats":         make(map[string]interface{}),
	}

	// Get stats for each strategy
	for name, strategy := range m.strategies {
		stats["strategy_stats"].(map[string]interface{})[name] = map[string]interface{}{
			"priority": m.strategyPriorities[name],
			"running":  strategy.IsRunning(),
		}
	}

	return stats
}

// Shutdown shuts down the strategy manager
func (m *OptimizedStrategyManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Shutting down strategy manager")

	// Shutdown all strategies
	for name, strategy := range m.strategies {
		if err := strategy.Shutdown(ctx); err != nil {
			m.logger.Error("Failed to shutdown strategy",
				zap.String("strategy", name),
				zap.Error(err),
			)
		}
	}

	// Clear the strategies
	m.strategies = make(map[string]Strategy)
	m.strategyPriorities = make(map[string]int)

	return nil
}

// getPrioritizedStrategies gets strategies sorted by priority
func (m *OptimizedStrategyManager) getPrioritizedStrategies() []Strategy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a slice of strategy names
	names := make([]string, 0, len(m.strategies))
	for name := range m.strategies {
		names = append(names, name)
	}

	// Sort by priority (higher priority first)
	sort.Slice(names, func(i, j int) bool {
		return m.strategyPriorities[names[i]] > m.strategyPriorities[names[j]]
	})

	// Create a slice of strategies
	strategies := make([]Strategy, 0, len(names))
	for _, name := range names {
		strategies = append(strategies, m.strategies[name])
	}

	return strategies
}

// ParallelStrategyManager is a strategy manager that processes data in parallel
type ParallelStrategyManager struct {
	strategies          map[string]Strategy
	strategyPriorities  map[string]int
	processedMarketData uint64
	processedOrders     uint64
	pool                *ants.Pool
	logger              *zap.Logger
	mu                  sync.RWMutex
}

// NewParallelStrategyManager creates a new parallel strategy manager
func NewParallelStrategyManager(maxWorkers int, logger *zap.Logger) (*ParallelStrategyManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	// Create a worker pool
	pool, err := ants.NewPool(maxWorkers)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	return &ParallelStrategyManager{
		strategies:         make(map[string]Strategy),
		strategyPriorities: make(map[string]int),
		pool:               pool,
		logger:             logger,
	}, nil
}

// RegisterStrategy registers a strategy with the manager
func (m *ParallelStrategyManager) RegisterStrategy(ctx context.Context, strategy Strategy, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := strategy.Name()
	if _, exists := m.strategies[name]; exists {
		return fmt.Errorf("strategy already registered: %s", name)
	}

	m.strategies[name] = strategy
	m.strategyPriorities[name] = priority

	m.logger.Info("Registered strategy",
		zap.String("strategy", name),
		zap.Int("priority", priority),
	)

	return nil
}

// UnregisterStrategy unregisters a strategy from the manager
func (m *ParallelStrategyManager) UnregisterStrategy(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return fmt.Errorf("strategy not registered: %s", name)
	}

	// Shutdown the strategy
	if err := strategy.Shutdown(ctx); err != nil {
		m.logger.Error("Failed to shutdown strategy",
			zap.String("strategy", name),
			zap.Error(err),
		)
	}

	delete(m.strategies, name)
	delete(m.strategyPriorities, name)

	m.logger.Info("Unregistered strategy",
		zap.String("strategy", name),
	)

	return nil
}

// GetStrategy gets a strategy by name
func (m *ParallelStrategyManager) GetStrategy(name string) (Strategy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy not registered: %s", name)
	}

	return strategy, nil
}

// GetRegisteredStrategies gets all registered strategies
func (m *ParallelStrategyManager) GetRegisteredStrategies() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	strategies := make([]string, 0, len(m.strategies))
	for name := range m.strategies {
		strategies = append(strategies, name)
	}

	return strategies
}

// ProcessMarketData processes market data through all registered strategies
func (m *ParallelStrategyManager) ProcessMarketData(ctx context.Context, data *market_data.MarketData) {
	// Increment processed count
	atomic.AddUint64(&m.processedMarketData, 1)

	// Get strategies
	m.mu.RLock()
	strategies := make([]Strategy, 0, len(m.strategies))
	for _, strategy := range m.strategies {
		if strategy.IsRunning() {
			strategies = append(strategies, strategy)
		}
	}
	m.mu.RUnlock()

	if len(strategies) == 0 {
		return
	}

	// Create a wait group to wait for all strategies to finish
	var wg sync.WaitGroup
	wg.Add(len(strategies))

	// Process the market data through each strategy in parallel
	for _, s := range strategies {
		strategy := s
		dataCopy := *data // Create a copy to avoid race conditions

		// Submit the task to the worker pool
		err := m.pool.Submit(func() {
			defer wg.Done()

			// Process the market data
			if err := strategy.ProcessMarketData(ctx, &dataCopy); err != nil {
				m.logger.Error("Failed to process market data",
					zap.String("strategy", strategy.Name()),
					zap.Error(err),
				)
			}
		})

		if err != nil {
			m.logger.Error("Failed to submit task to worker pool",
				zap.Error(err),
			)
			wg.Done()
		}
	}

	// Wait for all strategies to finish
	wg.Wait()
}

// ProcessOrder processes an order through all registered strategies
func (m *ParallelStrategyManager) ProcessOrder(ctx context.Context, order *order.Order) {
	// Increment processed count
	atomic.AddUint64(&m.processedOrders, 1)

	// Get strategies
	m.mu.RLock()
	strategies := make([]Strategy, 0, len(m.strategies))
	for _, strategy := range m.strategies {
		if strategy.IsRunning() {
			strategies = append(strategies, strategy)
		}
	}
	m.mu.RUnlock()

	if len(strategies) == 0 {
		return
	}

	// Create a wait group to wait for all strategies to finish
	var wg sync.WaitGroup
	wg.Add(len(strategies))

	// Process the order through each strategy in parallel
	for _, s := range strategies {
		strategy := s
		orderCopy := *order // Create a copy to avoid race conditions

		// Submit the task to the worker pool
		err := m.pool.Submit(func() {
			defer wg.Done()

			// Process the order
			if err := strategy.ProcessOrder(ctx, &orderCopy); err != nil {
				m.logger.Error("Failed to process order",
					zap.String("strategy", strategy.Name()),
					zap.Error(err),
				)
			}
		})

		if err != nil {
			m.logger.Error("Failed to submit task to worker pool",
				zap.Error(err),
			)
			wg.Done()
		}
	}

	// Wait for all strategies to finish
	wg.Wait()
}

// GetStats gets the strategy manager statistics
func (m *ParallelStrategyManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"registered_strategies":  len(m.strategies),
		"processed_market_data":  atomic.LoadUint64(&m.processedMarketData),
		"processed_orders":       atomic.LoadUint64(&m.processedOrders),
		"worker_pool_running":    m.pool.Running(),
		"worker_pool_capacity":   m.pool.Cap(),
		"strategy_stats":         make(map[string]interface{}),
	}

	// Get stats for each strategy
	for name, strategy := range m.strategies {
		stats["strategy_stats"].(map[string]interface{})[name] = map[string]interface{}{
			"priority": m.strategyPriorities[name],
			"running":  strategy.IsRunning(),
		}
	}

	return stats
}

// Shutdown shuts down the strategy manager
func (m *ParallelStrategyManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Shutting down strategy manager")

	// Shutdown all strategies
	for name, strategy := range m.strategies {
		if err := strategy.Shutdown(ctx); err != nil {
			m.logger.Error("Failed to shutdown strategy",
				zap.String("strategy", name),
				zap.Error(err),
			)
		}
	}

	// Clear the strategies
	m.strategies = make(map[string]Strategy)
	m.strategyPriorities = make(map[string]int)

	// Release the worker pool
	m.pool.Release()

	return nil
}

