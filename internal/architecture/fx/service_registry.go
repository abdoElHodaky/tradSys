package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceRegistryModule provides the service registry components
var ServiceRegistryModule = fx.Options(
	// Provide the service registry
	fx.Provide(NewServiceRegistry),

	// Register lifecycle hooks
	fx.Invoke(registerServiceRegistryHooks),
)

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(logger *zap.Logger) *architecture.ServiceRegistry {
	registry := architecture.NewServiceRegistry()
	logger.Info("Created service registry")
	return registry
}

// registerServiceRegistryHooks registers lifecycle hooks for the service registry
func registerServiceRegistryHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	registry *architecture.ServiceRegistry,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting service registry")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping service registry")
			return nil
		},
	})
}
