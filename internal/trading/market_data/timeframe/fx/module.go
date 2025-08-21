package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/timeframe"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the timeframe components
var Module = fx.Options(
	// Provide the timeframe aggregator
	fx.Provide(NewTimeframeAggregator),

	// Provide the indicator calculator
	fx.Provide(NewIndicatorCalculator),

	// Register lifecycle hooks
	fx.Invoke(registerTimeframeHooks),
)

// TimeframeAggregatorParams contains parameters for creating a TimeframeAggregator
type TimeframeAggregatorParams struct {
	fx.In

	Logger *zap.Logger
}

// NewTimeframeAggregator creates a new TimeframeAggregator
func NewTimeframeAggregator(params TimeframeAggregatorParams) *timeframe.TimeframeAggregator {
	return timeframe.NewTimeframeAggregator(params.Logger, 1000)
}

// NewIndicatorCalculator creates a new IndicatorCalculator
func NewIndicatorCalculator(aggregator *timeframe.TimeframeAggregator) *timeframe.IndicatorCalculator {
	return timeframe.NewIndicatorCalculator(aggregator)
}

// registerTimeframeHooks registers lifecycle hooks for timeframe components
func registerTimeframeHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting timeframe components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping timeframe components")
			return nil
		},
	})
}

