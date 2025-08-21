package fx

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/resilience"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/workerpool"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the strategy components
var Module = fx.Options(
	// Provide the strategy factory
	fx.Provide(NewStrategyFactory),

	// Provide the strategy registry
	fx.Provide(NewStrategyRegistry),

	// Provide the strategy manager
	fx.Provide(NewStrategyManager),

	// Provide the strategy metrics collector
	fx.Provide(NewStrategyMetricsCollector),

	// Register lifecycle hooks
	fx.Invoke(registerStrategyHooks),
)

// StrategyFactoryParams contains parameters for creating a StrategyFactory
type StrategyFactoryParams struct {
	fx.In

	Logger              *zap.Logger
	WorkerPoolFactory   *workerpool.WorkerPoolFactory
	CircuitBreakerFactory *resilience.CircuitBreakerFactory
}

// StrategyRegistry manages registered strategies
type StrategyRegistry struct {
	strategies map[string]strategy.Strategy
	logger     *zap.Logger
}

// NewStrategyRegistry creates a new StrategyRegistry
func NewStrategyRegistry(logger *zap.Logger) *StrategyRegistry {
	return &StrategyRegistry{
		strategies: make(map[string]strategy.Strategy),
		logger:     logger,
	}
}

// RegisterStrategy registers a strategy
func (r *StrategyRegistry) RegisterStrategy(s strategy.Strategy) {
	r.strategies[s.Name()] = s
	r.logger.Info("Registered strategy", zap.String("name", s.Name()))
}

// GetStrategy gets a strategy by name
func (r *StrategyRegistry) GetStrategy(name string) (strategy.Strategy, bool) {
	s, exists := r.strategies[name]
	return s, exists
}

// GetAllStrategies gets all registered strategies
func (r *StrategyRegistry) GetAllStrategies() map[string]strategy.Strategy {
	return r.strategies
}

// StrategyManager manages strategy lifecycle
type StrategyManager struct {
	registry *StrategyRegistry
	logger   *zap.Logger
}

// NewStrategyManager creates a new StrategyManager
func NewStrategyManager(registry *StrategyRegistry, logger *zap.Logger) *StrategyManager {
	return &StrategyManager{
		registry: registry,
		logger:   logger,
	}
}

// StartStrategy starts a strategy
func (m *StrategyManager) StartStrategy(ctx context.Context, name string) error {
	s, exists := m.registry.GetStrategy(name)
	if !exists {
		return fmt.Errorf("strategy not found: %s", name)
	}

	if s.IsRunning() {
		return nil
	}

	m.logger.Info("Starting strategy", zap.String("name", name))
	return s.Start(ctx)
}

// StopStrategy stops a strategy
func (m *StrategyManager) StopStrategy(ctx context.Context, name string) error {
	s, exists := m.registry.GetStrategy(name)
	if !exists {
		return fmt.Errorf("strategy not found: %s", name)
	}

	if !s.IsRunning() {
		return nil
	}

	m.logger.Info("Stopping strategy", zap.String("name", name))
	return s.Stop(ctx)
}

// StrategyMetricsCollector collects metrics from strategies
type StrategyMetricsCollector struct {
	registry *StrategyRegistry
	logger   *zap.Logger
}

// NewStrategyMetricsCollector creates a new StrategyMetricsCollector
func NewStrategyMetricsCollector(registry *StrategyRegistry, logger *zap.Logger) *StrategyMetricsCollector {
	return &StrategyMetricsCollector{
		registry: registry,
		logger:   logger,
	}
}

// CollectMetrics collects metrics from all strategies
func (c *StrategyMetricsCollector) CollectMetrics() map[string]map[string]interface{} {
	metrics := make(map[string]map[string]interface{})
	
	for name, s := range c.registry.GetAllStrategies() {
		metrics[name] = s.GetPerformanceMetrics()
	}
	
	return metrics
}

// registerStrategyHooks registers lifecycle hooks for strategy components
func registerStrategyHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	registry *StrategyRegistry,
	manager *StrategyManager,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting strategy components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping all strategies")
			
			// Stop all running strategies
			for name, s := range registry.GetAllStrategies() {
				if s.IsRunning() {
					if err := manager.StopStrategy(ctx, name); err != nil {
						logger.Error("Failed to stop strategy", 
							zap.String("name", name),
							zap.Error(err))
					}
				}
			}
			
			return nil
		},
	})
}

