package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/command"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/query"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the core architecture components
var Module = fx.Options(
	// Provide the command bus
	fx.Provide(NewCommandBus),
	
	// Provide the query bus
	fx.Provide(NewQueryBus),
	
	// Include the discovery module
	fx.Options(DiscoveryModule),
	
	// Include the gateway module
	fx.Options(GatewayModule),
	
	// Include the CQRS module
	fx.Options(CQRSModule),
	
	// Include the circuit breaker module
	fx.Options(CircuitBreakerModule),
	
	// Include the architecture circuit breaker module
	fx.Options(ArchitectureCircuitBreakerModule),
	
	// Include the bulkhead module
	fx.Options(BulkheadModule),
	
	// Include the worker pool module
	fx.Options(WorkerPoolModule),
	
	// Include the service registry module
	fx.Options(ServiceRegistryModule),
	
	// Include the tracing module
	fx.Options(TracingModule),
	
	// Include the sharding module
	fx.Options(ShardingModule),
	
	// Register lifecycle hooks
	fx.Invoke(registerArchitectureHooks),
)

// NewCommandBus creates a new command bus
func NewCommandBus() *command.CommandBus {
	return command.NewCommandBus()
}

// NewQueryBus creates a new query bus
func NewQueryBus() *query.QueryBus {
	return query.NewQueryBus()
}

// registerArchitectureHooks registers lifecycle hooks for the architecture components
func registerArchitectureHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	commandBus *command.CommandBus,
	queryBus *query.QueryBus,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting architecture components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping architecture components")
			return nil
		},
	})
}

// NewMarketDataModule creates a new market data module
func NewMarketDataModule() fx.Option {
	return fx.Module("marketdata",
		// Provide market data components
		fx.Provide(func() interface{} {
			// This is a placeholder for the actual market data module
			return nil
		}),
	)
}

// NewOrdersModule creates a new orders module
func NewOrdersModule() fx.Option {
	return fx.Module("orders",
		// Provide orders components
		fx.Provide(func() interface{} {
			// This is a placeholder for the actual orders module
			return nil
		}),
	)
}

// NewRiskModule creates a new risk module
func NewRiskModule() fx.Option {
	return fx.Module("risk",
		// Provide risk components
		fx.Provide(func() interface{} {
			// This is a placeholder for the actual risk module
			return nil
		}),
	)
}

// NewTradingModule creates a new trading module
func NewTradingModule() fx.Option {
	return fx.Module("trading",
		// Include the market data module
		fx.Options(NewMarketDataModule()),
		
		// Include the orders module
		fx.Options(NewOrdersModule()),
		
		// Include the risk module
		fx.Options(NewRiskModule()),
	)
}

