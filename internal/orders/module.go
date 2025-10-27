package orders

import (
	"context"
	"github.com/abdoElHodaky/tradSys/internal/matching"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// OrderManagementModule provides the order management module for the fx application
var OrderManagementModule = fx.Options(
	fx.Provide(NewOrderService),
)

// NewFxService creates a new order management service for the fx application
func NewFxService(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
	orderEngine *matching.UnifiedMatchingEngine,
) *OrderService {
	service := NewOrderService(orderEngine, logger)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting order management service")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping order management service")
			service.Stop()
			return nil
		},
	})

	return service
}
