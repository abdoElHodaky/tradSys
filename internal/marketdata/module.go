package marketdata

import (
	"context"

	order_matching "github.com/abdoElHodaky/tradSys/internal/core/matching"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// TradingMarketDataModule provides the trading market data module for the fx application
var TradingMarketDataModule = fx.Options(
	fx.Provide(NewHandler),
)

// NewFxHandler creates a new market data handler for the fx application
func NewFxHandler(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
	orderEngine *order_matching.Engine,
) *Handler {
	handler := NewHandler(orderEngine, logger)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting market data handler")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping market data handler")
			handler.Stop()
			return nil
		},
	})

	return handler
}
