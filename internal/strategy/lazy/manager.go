package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// LazyStrategyManager is a lazy-loaded manager for trading strategies
type LazyStrategyManager struct {
	// Component coordinator
	coordinator *coordination.ComponentCoordinator
	
	// Component name prefix
	componentNamePrefix string
	
	// Logger
	logger *zap.Logger
	
	// Lock manager for thread safety
	lockManager *coordination.LockManager
	
	// Strategy factory
	factory strategy.StrategyFactory
	
	// Active strategies
	activeStrategies map[string]bool
	activeStrategiesMu sync.RWMutex
}

// NewLazyStrategyManager creates a new lazy-loaded strategy manager
func NewLazyStrategyManager(
	coordinator *coordination.ComponentCoordinator,
	lockManager *coordination.LockManager,
	factory strategy.StrategyFactory,
	logger *zap.Logger,
) (*LazyStrategyManager, error) {
	componentNamePrefix := "strategy-"
	
	// Register the lock for strategy operations
	lockManager.RegisterLock("strategies", &sync.Mutex{})
	
	return &LazyStrategyManager{
		coordinator:         coordinator,
		componentNamePrefix: componentNamePrefix,
		logger:              logger,
		lockManager:         lockManager,
		factory:             factory,
		activeStrategies:    make(map[string]bool),
	}, nil
}

// GetStrategy gets a strategy by name
func (m *LazyStrategyManager) GetStrategy(
	ctx context.Context,
	strategyName string,
	config strategy.StrategyConfig,
) (strategy.Strategy, error) {
	componentName := m.componentNamePrefix + strategyName
	
	// Check if the component is already registered
	_, err := m.coordinator.GetComponentInfo(componentName)
	if err != nil {
		// Component not registered, register it
		err = m.registerStrategy(ctx, strategyName, componentName, config)
		if err != nil {
			return nil, err
		}
	}
	
	// Get the component
	strategyInterface, err := m.coordinator.GetComponent(ctx, componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual strategy type
	strat, ok := strategyInterface.(strategy.Strategy)
	if !ok {
		return nil, fmt.Errorf("invalid strategy type for strategy %s", strategyName)
	}
	
	// Update active strategies
	m.activeStrategiesMu.Lock()
	m.activeStrategies[strategyName] = true
	m.activeStrategiesMu.Unlock()
	
	return strat, nil
}

// registerStrategy registers a strategy with the coordinator
func (m *LazyStrategyManager) registerStrategy(
	ctx context.Context,
	strategyName string,
	componentName string,
	config strategy.StrategyConfig,
) error {
	// Acquire the lock to prevent concurrent strategy creation
	err := m.lockManager.AcquireLock("strategies", "strategy-manager")
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer m.lockManager.ReleaseLock("strategies", "strategy-manager")
	
	// Create the provider function
	providerFn := func(log *zap.Logger) (interface{}, error) {
		// Create the strategy
		strat, err := m.factory.CreateStrategy(config, log)
		if err != nil {
			return nil, err
		}
		
		// Initialize the strategy
		err = strat.Initialize(ctx)
		if err != nil {
			return nil, err
		}
		
		return strat, nil
	}
	
	// Estimate memory usage based on strategy type
	memoryEstimate := int64(50 * 1024 * 1024) // Default 50MB
	
	switch config.Type {
	case "market_making":
		memoryEstimate = 100 * 1024 * 1024 // 100MB
	case "statistical_arbitrage":
		memoryEstimate = 200 * 1024 * 1024 // 200MB
	case "trend_following":
		memoryEstimate = 75 * 1024 * 1024 // 75MB
	}
	
	// Create the lazy provider
	provider := lazy.NewEnhancedLazyProvider(
		componentName,
		providerFn,
		m.logger,
		nil, // Metrics will be handled by the coordinator
		lazy.WithMemoryEstimate(memoryEstimate),
		lazy.WithTimeout(30*time.Second),
		lazy.WithPriority(30), // Medium priority
	)
	
	// Register with the coordinator
	return m.coordinator.RegisterComponent(
		componentName,
		"strategy",
		provider,
		[]string{}, // No dependencies
	)
}

// StartStrategy starts a strategy
func (m *LazyStrategyManager) StartStrategy(
	ctx context.Context,
	strategyName string,
	config strategy.StrategyConfig,
) error {
	// Get the strategy
	strat, err := m.GetStrategy(ctx, strategyName, config)
	if err != nil {
		return err
	}
	
	// Start the strategy
	return strat.Start(ctx)
}

// StopStrategy stops a strategy
func (m *LazyStrategyManager) StopStrategy(
	ctx context.Context,
	strategyName string,
) error {
	componentName := m.componentNamePrefix + strategyName
	
	// Get the component
	strategyInterface, err := m.coordinator.GetComponent(ctx, componentName)
	if err != nil {
		return err
	}
	
	// Cast to the actual strategy type
	strat, ok := strategyInterface.(strategy.Strategy)
	if !ok {
		return fmt.Errorf("invalid strategy type for strategy %s", strategyName)
	}
	
	// Stop the strategy
	return strat.Stop(ctx)
}

// ReleaseStrategy releases a strategy
func (m *LazyStrategyManager) ReleaseStrategy(
	ctx context.Context,
	strategyName string,
) error {
	componentName := m.componentNamePrefix + strategyName
	
	// Get the component
	strategyInterface, err := m.coordinator.GetComponent(ctx, componentName)
	if err != nil {
		return err
	}
	
	// Cast to the actual strategy type
	strat, ok := strategyInterface.(strategy.Strategy)
	if !ok {
		return fmt.Errorf("invalid strategy type for strategy %s", strategyName)
	}
	
	// Stop the strategy if it's running
	if strat.IsRunning() {
		err = strat.Stop(ctx)
		if err != nil {
			return err
		}
	}
	
	// Update active strategies
	m.activeStrategiesMu.Lock()
	delete(m.activeStrategies, strategyName)
	m.activeStrategiesMu.Unlock()
	
	// Shutdown the component
	return m.coordinator.ShutdownComponent(ctx, componentName)
}

// OnMarketData processes market data for all active strategies
func (m *LazyStrategyManager) OnMarketData(
	ctx context.Context,
	data *marketdata.MarketDataResponse,
) error {
	m.activeStrategiesMu.RLock()
	activeStrategies := make([]string, 0, len(m.activeStrategies))
	for strategyName := range m.activeStrategies {
		activeStrategies = append(activeStrategies, strategyName)
	}
	m.activeStrategiesMu.RUnlock()
	
	var lastErr error
	for _, strategyName := range activeStrategies {
		componentName := m.componentNamePrefix + strategyName
		
		// Get the component
		strategyInterface, err := m.coordinator.GetComponent(ctx, componentName)
		if err != nil {
			lastErr = err
			m.logger.Error("Failed to get strategy",
				zap.String("strategy", strategyName),
				zap.Error(err),
			)
			continue
		}
		
		// Cast to the actual strategy type
		strat, ok := strategyInterface.(strategy.Strategy)
		if !ok {
			lastErr = fmt.Errorf("invalid strategy type for strategy %s", strategyName)
			m.logger.Error("Invalid strategy type",
				zap.String("strategy", strategyName),
				zap.Error(lastErr),
			)
			continue
		}
		
		// Process market data
		err = strat.OnMarketData(ctx, data)
		if err != nil {
			lastErr = err
			m.logger.Error("Failed to process market data",
				zap.String("strategy", strategyName),
				zap.Error(err),
			)
		}
	}
	
	return lastErr
}

// OnOrderUpdate processes order updates for all active strategies
func (m *LazyStrategyManager) OnOrderUpdate(
	ctx context.Context,
	order *orders.OrderResponse,
) error {
	m.activeStrategiesMu.RLock()
	activeStrategies := make([]string, 0, len(m.activeStrategies))
	for strategyName := range m.activeStrategies {
		activeStrategies = append(activeStrategies, strategyName)
	}
	m.activeStrategiesMu.RUnlock()
	
	var lastErr error
	for _, strategyName := range activeStrategies {
		componentName := m.componentNamePrefix + strategyName
		
		// Get the component
		strategyInterface, err := m.coordinator.GetComponent(ctx, componentName)
		if err != nil {
			lastErr = err
			m.logger.Error("Failed to get strategy",
				zap.String("strategy", strategyName),
				zap.Error(err),
			)
			continue
		}
		
		// Cast to the actual strategy type
		strat, ok := strategyInterface.(strategy.Strategy)
		if !ok {
			lastErr = fmt.Errorf("invalid strategy type for strategy %s", strategyName)
			m.logger.Error("Invalid strategy type",
				zap.String("strategy", strategyName),
				zap.Error(lastErr),
			)
			continue
		}
		
		// Process order update
		err = strat.OnOrderUpdate(ctx, order)
		if err != nil {
			lastErr = err
			m.logger.Error("Failed to process order update",
				zap.String("strategy", strategyName),
				zap.Error(err),
			)
		}
	}
	
	return lastErr
}

// ListActiveStrategies lists active strategies
func (m *LazyStrategyManager) ListActiveStrategies() []string {
	m.activeStrategiesMu.RLock()
	defer m.activeStrategiesMu.RUnlock()
	
	strategies := make([]string, 0, len(m.activeStrategies))
	for strategyName := range m.activeStrategies {
		strategies = append(strategies, strategyName)
	}
	
	return strategies
}

// GetStrategyMetrics gets metrics for a strategy
func (m *LazyStrategyManager) GetStrategyMetrics(
	ctx context.Context,
	strategyName string,
) (map[string]interface{}, error) {
	componentName := m.componentNamePrefix + strategyName
	
	// Get the component
	strategyInterface, err := m.coordinator.GetComponent(ctx, componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual strategy type
	strat, ok := strategyInterface.(strategy.Strategy)
	if !ok {
		return nil, fmt.Errorf("invalid strategy type for strategy %s", strategyName)
	}
	
	// Get metrics
	return strat.GetPerformanceMetrics(), nil
}

// ShutdownAll shuts down all strategies
func (m *LazyStrategyManager) ShutdownAll(ctx context.Context) error {
	m.activeStrategiesMu.RLock()
	activeStrategies := make([]string, 0, len(m.activeStrategies))
	for strategyName := range m.activeStrategies {
		activeStrategies = append(activeStrategies, strategyName)
	}
	m.activeStrategiesMu.RUnlock()
	
	var lastErr error
	for _, strategyName := range activeStrategies {
		err := m.ReleaseStrategy(ctx, strategyName)
		if err != nil {
			lastErr = err
			m.logger.Error("Failed to release strategy",
				zap.String("strategy", strategyName),
				zap.Error(err),
			)
		}
	}
	
	return lastErr
}

