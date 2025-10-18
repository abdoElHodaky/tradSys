package external

import (
	"context"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the external market data module for the fx application
var Module = fx.Options(
	fx.Provide(NewManager),
	fx.Provide(NewBinanceProvider),
)

// NewFxManager creates a new market data provider manager for the fx application
func NewFxManager(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
	binanceProvider *BinanceProvider,
) *Manager {
	manager := NewManager(ManagerParams{Logger: logger})
	
	// Register lifecycle hooks
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting market data provider manager")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping market data provider manager")
			return nil
		},
	})
	
	return manager
}

// NewFxBinanceProvider creates a new Binance market data provider for the fx application
func NewFxBinanceProvider(
	logger *zap.Logger,
) *BinanceProvider {
	// In a real application, these would be loaded from environment variables or configuration
	apiKey := ""
	apiSecret := ""
	
	return NewBinanceProvider(apiKey, apiSecret, logger)
}
