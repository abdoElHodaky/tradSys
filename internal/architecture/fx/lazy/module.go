package lazy

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the lazy loading components
var Module = fx.Options(
	// Provide the lazy loading metrics
	fx.Provide(NewLazyLoadingMetrics),
	
	// Register lifecycle hooks
	fx.Invoke(registerHooks),
)

// registerHooks registers lifecycle hooks for the lazy loading components
func registerHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	metrics *LazyLoadingMetrics,
) {
	logger.Info("Initializing lazy loading components")
}

// ProvideLazy creates a lazy provider for a component
func ProvideLazy(name string, constructor interface{}) fx.Option {
	return fx.Provide(func(logger *zap.Logger, metrics *LazyLoadingMetrics) *LazyProvider {
		return NewLazyProvider(name, constructor, logger, metrics)
	})
}

// ProvideProxyModule creates a proxy module for a component
func ProvideProxyModule(name string, constructor interface{}) fx.Option {
	return fx.Provide(func(logger *zap.Logger, metrics *LazyLoadingMetrics) *ProxyModule {
		return NewProxyModule(name, constructor, logger, metrics)
	})
}

