package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture/discovery"
	"github.com/micro/go-micro/v4/registry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DiscoveryModule provides the service discovery components
var DiscoveryModule = fx.Options(
	// Provide the registry
	fx.Provide(NewRegistry),
	
	// Provide the service discovery
	fx.Provide(NewServiceDiscovery),
	
	// Provide the round-robin strategy
	fx.Provide(NewRoundRobinStrategy),
	
	// Provide the random strategy
	fx.Provide(NewRandomStrategy),
	
	// Provide the service selector with round-robin strategy
	fx.Provide(NewServiceSelector),
	
	// Register lifecycle hooks
	fx.Invoke(registerDiscoveryHooks),
)

// NewRegistry creates a new registry
func NewRegistry() registry.Registry {
	return registry.NewRegistry()
}

// NewServiceDiscovery creates a new service discovery
func NewServiceDiscovery(registry registry.Registry, logger *zap.Logger) *discovery.ServiceDiscovery {
	return discovery.NewServiceDiscovery(registry, logger)
}

// NewRoundRobinStrategy creates a new round-robin strategy
func NewRoundRobinStrategy() *discovery.RoundRobinStrategy {
	return discovery.NewRoundRobinStrategy()
}

// NewRandomStrategy creates a new random strategy
func NewRandomStrategy() *discovery.RandomStrategy {
	return discovery.NewRandomStrategy()
}

// NewServiceSelector creates a new service selector with round-robin strategy
func NewServiceSelector(
	discovery *discovery.ServiceDiscovery,
	logger *zap.Logger,
	strategy *discovery.RoundRobinStrategy,
) *discovery.ServiceSelector {
	return discovery.NewServiceSelector(discovery, logger, strategy)
}

// registerDiscoveryHooks registers lifecycle hooks for the discovery components
func registerDiscoveryHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	discovery *discovery.ServiceDiscovery,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting discovery components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping discovery components")
			return nil
		},
	})
}
