package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/workerpool"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/abdoElHodaky/tradSys/internal/strategy/optimization"
	strategyFx "github.com/abdoElHodaky/tradSys/internal/strategy/fx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the strategy optimization components
var Module = fx.Options(
	// Provide the backtester
	fx.Provide(NewBacktester),

	// Provide the strategy evaluator
	fx.Provide(NewStrategyEvaluator),

	// Provide the strategy optimizer
	fx.Provide(NewStrategyOptimizer),

	// Register lifecycle hooks
	fx.Invoke(registerOptimizationHooks),
)

// BacktesterParams contains parameters for creating a Backtester
type BacktesterParams struct {
	fx.In

	Logger *zap.Logger
}

// NewBacktester creates a new Backtester
func NewBacktester(params BacktesterParams) *optimization.Backtester {
	return optimization.NewBacktester(params.Logger)
}

// StrategyEvaluatorParams contains parameters for creating a StrategyEvaluator
type StrategyEvaluatorParams struct {
	fx.In

	Factory    *strategyFx.StrategyFactory
	Backtester *optimization.Backtester
	Logger     *zap.Logger
}

// NewStrategyEvaluator creates a new StrategyEvaluator
func NewStrategyEvaluator(params StrategyEvaluatorParams) *optimization.StrategyEvaluator {
	return optimization.NewStrategyEvaluator(
		params.Factory,
		params.Backtester,
		params.Logger,
	)
}

// StrategyOptimizerParams contains parameters for creating a StrategyOptimizer
type StrategyOptimizerParams struct {
	fx.In

	Factory       *strategyFx.StrategyFactory
	Evaluator     *optimization.StrategyEvaluator
	WorkerPool    *workerpool.WorkerPoolFactory
	Logger        *zap.Logger
}

// NewStrategyOptimizer creates a new StrategyOptimizer
func NewStrategyOptimizer(params StrategyOptimizerParams) *optimization.StrategyOptimizer {
	return optimization.NewStrategyOptimizer(
		params.Factory,
		params.Evaluator,
		params.WorkerPool,
		params.Logger,
	)
}

// registerOptimizationHooks registers lifecycle hooks for optimization components
func registerOptimizationHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting strategy optimization components")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping strategy optimization components")
			return nil
		},
	})
}

