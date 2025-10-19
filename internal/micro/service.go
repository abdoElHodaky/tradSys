package micro

import (
	"context"
	"time"

	unifiedconfig "github.com/abdoElHodaky/tradSys/internal/unified-config"
	gomicro "go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceParams contains the parameters for creating a micro service
type ServiceParams struct {
	fx.In

	Logger *zap.Logger
	Config *unifiedconfig.Config
	Lifecycle fx.Lifecycle
}

// Service represents a go-micro service with fx lifecycle management
type Service struct {
	gomicro.Service
	logger *zap.Logger
	config *unifiedconfig.Config
}

// NewService creates a new go-micro service with fx dependency injection
func NewService(p ServiceParams) (*Service, error) {
	// Create service options
	options := []gomicro.Option{
		gomicro.Name(p.Config.Service.Name),
		gomicro.Version(p.Config.Service.Version),
		gomicro.Address(p.Config.Service.Address),
		gomicro.RegisterTTL(time.Second * 30),
		gomicro.RegisterInterval(time.Second * 15),
	}

	// Add registry based on configuration
	if p.Config.Registry.Type == "etcd" {
		options = append(options, gomicro.Registry(
			registry.NewRegistry(
				registry.Addrs(p.Config.Registry.Addresses...),
			),
		))
	}

	// Create the service
	service := gomicro.NewService(options...)

	// Create our service wrapper
	s := &Service{
		Service: service,
		logger:  p.Logger,
		config:  p.Config,
	}

	// Add lifecycle hooks
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Initialize and start the service
			service.Init()

			// Start the service in a goroutine
			go func() {
				if err := service.Run(); err != nil {
					p.Logger.Error("Failed to run service", zap.Error(err))
				}
			}()

			p.Logger.Info("Service started",
				zap.String("name", p.Config.Service.Name),
				zap.String("address", p.Config.Service.Address))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("Stopping service", zap.String("name", p.Config.Service.Name))
			return nil
		},
	})

	return s, nil
}

// Module provides the go-micro module for fx
var Module = fx.Options(
	fx.Provide(NewService),
)
