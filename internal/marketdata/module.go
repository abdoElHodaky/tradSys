package marketdata

import (
	"context"
	
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata/external"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the market data module for the fx application
var Module = fx.Options(
	fx.Provide(NewService),
	fx.Options(external.Module),
)

// NewFxService creates a new market data service for the fx application
func NewFxService(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
	marketDataRepository *repositories.MarketDataRepository,
	externalManager *external.Manager,
) *Service {
	service := NewService(marketDataRepository, externalManager, logger)
	
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting market data service")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping market data service")
			service.Stop()
			return nil
		},
	})
	
	return service
}
