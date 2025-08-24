package fx

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LazyModule provides lazily loaded strategy components
var LazyModule = fx.Options(
	// Provide the strategy factory (always loaded eagerly)
	fx.Provide(NewStrategyFactory),
	
	// Provide the strategy registry (always loaded eagerly)
	fx.Provide(NewStrategyRegistry),
	
	// Provide lazily loaded strategy components
	provideLazyStrategyManager,
	provideLazyMetricsCollector,
	
	// Register lifecycle hooks
	fx.Invoke(registerLazyHooks),
)

// provideLazyStrategyManager provides a lazily loaded strategy manager
func provideLazyStrategyManager(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"strategy-manager",
		func(registry *StrategyRegistry, logger *zap.Logger) (*StrategyManager, error) {
			logger.Info("Lazily initializing strategy manager")
			return NewStrategyManager(registry, logger)
		},
		logger,
		metrics,
	)
}

// provideLazyMetricsCollector provides a lazily loaded metrics collector
func provideLazyMetricsCollector(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"strategy-metrics-collector",
		func(registry *StrategyRegistry, logger *zap.Logger) (*StrategyMetricsCollector, error) {
			logger.Info("Lazily initializing strategy metrics collector")
			return NewStrategyMetricsCollector(registry, logger)
		},
		logger,
		metrics,
	)
}

// registerLazyHooks registers lifecycle hooks for the lazy strategy components
func registerLazyHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	strategyManagerProvider *lazy.LazyProvider,
	metricsCollectorProvider *lazy.LazyProvider,
) {
	logger.Info("Registering lazy strategy component hooks")
}

// GetStrategyManager gets the strategy manager, initializing it if necessary
func GetStrategyManager(provider *lazy.LazyProvider) (*StrategyManager, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*StrategyManager), nil
}

// GetStrategyMetricsCollector gets the strategy metrics collector, initializing it if necessary
func GetStrategyMetricsCollector(provider *lazy.LazyProvider) (*StrategyMetricsCollector, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*StrategyMetricsCollector), nil
}

