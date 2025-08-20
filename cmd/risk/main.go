package main

import (
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/events"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/proto/risk"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		// Provide core components
		fx.Provide(func() (*zap.Logger, error) {
			return zap.NewProduction()
		}),

		// Include modules
		fx.Options(
			config.Module,
			micro.Module,
			micro.RegistryModule,
			events.BrokerModule,
			risk.Module,
		),

		// Configure service mesh
		fx.Invoke(func(service *micro.Service, config *config.Config, logger *zap.Logger) {
			meshOpts := micro.MeshOptions{
				EnableTracing:       config.Tracing.Enabled,
				EnableMetrics:       config.Metrics.Enabled,
				EnableCircuitBreaker: config.Resilience.CircuitBreakerEnabled,
				EnableRateLimiting:  config.Resilience.RateLimitingEnabled,
			}
			micro.ConfigureMesh(service.Service, meshOpts, logger)
		}),

		// Register service handlers
		fx.Invoke(func(service *micro.Service, handler *risk.Handler, logger *zap.Logger) {
			if err := risk.RegisterRiskServiceHandler(service.Server(), handler); err != nil {
				logger.Fatal("Failed to register handler", zap.Error(err))
			}
		}),
	)

	app.Run()
}

