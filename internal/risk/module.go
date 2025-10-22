package risk

import (
	"context"
	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RiskManagementModule provides the risk management module for the fx application
var RiskManagementModule = fx.Options(
	fx.Provide(NewService),
)

// NewFxService creates a new risk management service for the fx application
func NewFxService(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
	orderEngine *order_matching.Engine,
	orderService *orders.Service,
) *Service {
	service := NewService(orderEngine, orderService, logger)
	
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting risk management service")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping risk management service")
			service.Stop()
			return nil
		},
	})
	
	return service
}
