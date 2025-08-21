package optimized

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the optimized strategy components
var Module = fx.Options(
	// Provide the strategy manager
	fx.Provide(NewStrategyManager),
	
	// Provide the strategy factory
	fx.Provide(NewStrategyFactory),
	
	// Provide the strategy metrics collector
	fx.Provide(NewStrategyMetrics),
	
	// Register lifecycle hooks
	fx.Invoke(registerStrategyHooks),
)

// registerStrategyHooks registers lifecycle hooks for the strategy components
func registerStrategyHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	manager *StrategyManager,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx fx.Context) error {
			logger.Info("Starting optimized strategy components")
			return nil
		},
		OnStop: func(ctx fx.Context) error {
			logger.Info("Stopping optimized strategy components")
			
			// Log statistics for all strategies
			stats := manager.GetAllStrategyStats()
			for name, stat := range stats {
				logger.Info("Strategy statistics",
					zap.String("name", name),
					zap.Int64("processed_updates", stat.ProcessedUpdates),
					zap.Int64("executed_trades", stat.ExecutedTrades),
					zap.Float64("pnl", stat.PnL))
			}
			
			return nil
		},
	})
}

