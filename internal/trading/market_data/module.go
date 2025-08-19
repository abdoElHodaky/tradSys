package market_data

import (
	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the market data module for the fx application
var Module = fx.Options(
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
		OnStart: func(ctx fx.Context) error {
			logger.Info("Starting market data handler")
			return nil
		},
		OnStop: func(ctx fx.Context) error {
			logger.Info("Stopping market data handler")
			handler.Stop()
			return nil
		},
	})
	
	return handler
}

