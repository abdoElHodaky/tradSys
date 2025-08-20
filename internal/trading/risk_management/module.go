package risk_management

import (
	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_management"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the risk management module for the fx application
var Module = fx.Options(
	fx.Provide(NewService),
)

// NewFxService creates a new risk management service for the fx application
func NewFxService(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
	orderEngine *order_matching.Engine,
	orderService *order_management.Service,
) *Service {
	service := NewService(orderEngine, orderService, logger)
	
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx fx.Context) error {
			logger.Info("Starting risk management service")
			return nil
		},
		OnStop: func(ctx fx.Context) error {
			logger.Info("Stopping risk management service")
			service.Stop()
			return nil
		},
	})
	
	return service
}

