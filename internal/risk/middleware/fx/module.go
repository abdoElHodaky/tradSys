package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/internal/risk/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the risk middleware components
var Module = fx.Options(
	// Provide the order validation middleware
	fx.Provide(NewOrderValidationMiddleware),

	// Provide the exposure validation middleware
	fx.Provide(NewExposureValidationMiddleware),

	// Provide the circuit breaker middleware
	fx.Provide(NewCircuitBreakerMiddleware),

	// Register lifecycle hooks
	fx.Invoke(registerMiddlewareHooks),
)

// OrderValidationMiddlewareParams contains parameters for creating an OrderValidationMiddleware
type OrderValidationMiddlewareParams struct {
	fx.In

	Logger        *zap.Logger
	RiskValidator *risk.RiskValidator
	Next          middleware.OrderHandler `optional:"true"`
}

// NewOrderValidationMiddleware creates a new OrderValidationMiddleware
func NewOrderValidationMiddleware(params OrderValidationMiddlewareParams) *middleware.OrderValidationMiddleware {
	return middleware.NewOrderValidationMiddleware(
		params.Logger,
		params.RiskValidator,
		params.Next,
	)
}

// ExposureValidationMiddlewareParams contains parameters for creating an ExposureValidationMiddleware
type ExposureValidationMiddlewareParams struct {
	fx.In

	Logger        *zap.Logger
	RiskValidator *risk.RiskValidator
	RiskManager   *risk.RiskManager
	Next          middleware.OrderHandler `optional:"true"`
}

// NewExposureValidationMiddleware creates a new ExposureValidationMiddleware
func NewExposureValidationMiddleware(params ExposureValidationMiddlewareParams) *middleware.ExposureValidationMiddleware {
	return middleware.NewExposureValidationMiddleware(
		params.Logger,
		params.RiskValidator,
		params.RiskManager,
		params.Next,
	)
}

// CircuitBreakerMiddlewareParams contains parameters for creating a CircuitBreakerMiddleware
type CircuitBreakerMiddlewareParams struct {
	fx.In

	Logger      *zap.Logger
	RiskManager *risk.RiskManager
	Next        middleware.OrderHandler `optional:"true"`
}

// NewCircuitBreakerMiddleware creates a new CircuitBreakerMiddleware
func NewCircuitBreakerMiddleware(params CircuitBreakerMiddlewareParams) *middleware.CircuitBreakerMiddleware {
	return middleware.NewCircuitBreakerMiddleware(
		params.Logger,
		params.RiskManager,
		params.Next,
	)
}

// registerMiddlewareHooks registers lifecycle hooks for middleware components
func registerMiddlewareHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting risk middleware components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping risk middleware components")
			return nil
		},
	})
}

