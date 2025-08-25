package micro

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/micro/go-micro/v4/registry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RegistryParams contains the parameters for creating a registry
type RegistryParams struct {
	fx.In

	Logger    *zap.Logger
	Config    *config.Config
	Lifecycle fx.Lifecycle
}

// NewRegistry creates a new service registry with fx dependency injection
// It uses configuration values for registry type and addresses
func NewRegistry(p RegistryParams) registry.Registry {
	var reg registry.Registry

	// Create registry based on configuration
	switch p.Config.Registry.Type {
	case "etcd":
		reg = registry.NewRegistry(
			registry.Addrs(p.Config.Registry.Addresses...),
		)
	case "consul":
		reg = registry.NewRegistry(
			registry.Addrs(p.Config.Registry.Addresses...),
		)
	case "kubernetes":
		reg = registry.NewRegistry()
	default:
		// Default to mdns for local development
		reg = registry.NewRegistry()
	}

	// Add lifecycle hooks
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.Logger.Info("Service registry initialized",
				zap.String("type", p.Config.Registry.Type))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("Service registry stopped")
			return nil
		},
	})

	return reg
}

// RegistryModule provides the registry module for fx
var RegistryModule = fx.Options(
	fx.Provide(NewRegistry),
)
