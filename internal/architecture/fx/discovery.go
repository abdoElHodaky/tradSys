package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture/discovery"
	"go-micro.dev/v4/registry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DiscoveryModule provides service discovery components
var DiscoveryModule = fx.Options(
	// Provide the registry
	fx.Provide(NewRegistry),
	
	// Provide the service discovery
	fx.Provide(NewServiceDiscovery),
	
	// Provide the service selector
	fx.Provide(NewServiceSelector),
	
	// Register lifecycle hooks
	fx.Invoke(registerDiscoveryHooks),
)

// NewRegistry creates a new registry
func NewRegistry() registry.Registry {
	// Use the default registry from go-micro
	return registry.DefaultRegistry
}

// NewServiceDiscovery creates a new service discovery
func NewServiceDiscovery(reg registry.Registry, logger *zap.Logger) *discovery.ServiceDiscovery {
	return discovery.NewServiceDiscovery(reg, logger)
}

// NewServiceSelector creates a new service selector
func NewServiceSelector(discovery *discovery.ServiceDiscovery, logger *zap.Logger) *discovery.ServiceSelector {
	return discovery.NewServiceSelector(
		discovery,
		logger,
		discovery.NewRoundRobinStrategy(),
	)
}

// registerDiscoveryHooks registers lifecycle hooks for service discovery
func registerDiscoveryHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	registry registry.Registry,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting service discovery")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping service discovery")
			return nil
		},
	})
}

