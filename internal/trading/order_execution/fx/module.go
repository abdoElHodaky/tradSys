package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/resilience"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_execution"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the order execution components
var Module = fx.Options(
	// Provide the order execution service
	fx.Provide(NewOrderExecutionService),

	// Register lifecycle hooks
	fx.Invoke(registerOrderExecutionHooks),
)

// OrderExecutionServiceParams contains parameters for creating an OrderExecutionService
type OrderExecutionServiceParams struct {
	fx.In

	Engine              *order_matching.Engine
	CircuitBreakerFactory *resilience.CircuitBreakerFactory
	Logger              *zap.Logger
}

// NewOrderExecutionService creates a new OrderExecutionService
func NewOrderExecutionService(params OrderExecutionServiceParams) *order_execution.OrderExecutionService {
	return order_execution.NewOrderExecutionService(
		params.Engine,
		params.CircuitBreakerFactory,
		params.Logger,
	)
}

// registerOrderExecutionHooks registers lifecycle hooks for order execution components
func registerOrderExecutionHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	service *order_execution.OrderExecutionService,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting order execution service")
			return service.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping order execution service")
			return service.Stop(ctx)
		},
	})
}

