package lazy

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ProxyModule creates a proxy for a module that defers initialization
type ProxyModule struct {
	name     string
	provider *LazyProvider
	logger   *zap.Logger
	metrics  *LazyLoadingMetrics
}

// NewProxyModule creates a new proxy module
func NewProxyModule(name string, constructor interface{}, logger *zap.Logger, metrics *LazyLoadingMetrics) *ProxyModule {
	return &ProxyModule{
		name:     name,
		provider: NewLazyProvider(name, constructor, logger, metrics),
		logger:   logger,
		metrics:  metrics,
	}
}

// AsOption returns an fx.Option that registers the proxy module
func (m *ProxyModule) AsOption() fx.Option {
	return fx.Options(
		m.provider.AsOption(),
		fx.Invoke(func(lifecycle fx.Lifecycle) {
			lifecycle.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					m.logger.Debug("Registered lazy module", zap.String("name", m.name))
					return nil
				},
				OnStop: func(ctx context.Context) error {
					// If the module was initialized, call its stop method
					if m.provider.IsInitialized() {
						instance, err := m.provider.Get()
						if err != nil {
							return err
						}

						// Check if the instance has a Stop method
						if stopper, ok := instance.(interface {
							Stop(context.Context) error
						}); ok {
							m.logger.Debug("Stopping lazy module", zap.String("name", m.name))
							return stopper.Stop(ctx)
						}
					}
					return nil
				},
			})
		}),
	)
}

// GetProvider returns the lazy provider
func (m *ProxyModule) GetProvider() *LazyProvider {
	return m.provider
}

// Get returns the lazily initialized component
func (m *ProxyModule) Get() (interface{}, error) {
	return m.provider.Get()
}

