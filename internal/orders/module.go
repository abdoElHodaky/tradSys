package order_management

import (
	"context"
	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// OrderManagementModule provides the order management module for the fx application
var OrderManagementModule = fx.Options(
	fx.Provide(NewService),
)

// NewFxService creates a new order management service for the fx application
func NewFxService(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
	orderEngine *order_matching.Engine,
) *Service {
	service := NewService(orderEngine, logger)
	
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
