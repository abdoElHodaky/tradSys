package order_management

import (
	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the order management module for the fx application
var Module = fx.Options(
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
		OnStart: func(ctx fx.Context) error {
			logger.Info("Starting order management service")
			return nil
		},
		OnStop: func(ctx fx.Context) error {
			logger.Info("Stopping order management service")
			service.Stop()
			return nil
		},
	})
	
	return service
}

