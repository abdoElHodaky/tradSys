package micro

import (
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"go-micro.dev/v4/registry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RegistryParams contains parameters for creating a registry
type RegistryParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

// NewRegistry creates a new service registry with fx dependency injection
func NewRegistry(p RegistryParams) registry.Registry {
	var reg registry.Registry

	// Create registry based on configuration
	switch p.Config.Registry.Type {
	case "etcd":
		reg = registry.NewRegistry(
			registry.Addrs(p.Config.Registry.Addresses...),
			registry.Timeout(time.Second*5),
		)
	case "consul":
		reg = registry.NewRegistry(
			registry.Addrs(p.Config.Registry.Addresses...),
			registry.Timeout(time.Second*5),
		)
	default:
		// Default to mdns for local development
		reg = registry.NewRegistry()
	}

	p.Logger.Info("Service registry initialized",
		zap.String("type", p.Config.Registry.Type))

	return reg
}

// RegistryModule provides the registry module for fx
var RegistryModule = fx.Options(
	fx.Provide(NewRegistry),
)

