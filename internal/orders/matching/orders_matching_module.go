package order_matching

import (
	"context"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"github.com/abdoElHodaky/tradSys/internal/matching"
)

// Engine is an alias for the unified matching engine
type Engine = matching.UnifiedMatchingEngine

// NewEngine creates a new order matching engine
func NewEngine(logger *zap.Logger) *Engine {
	return matching.NewUnifiedMatchingEngine(logger, nil)
}

// OrderMatchingModule provides the order matching module for the fx application
var OrderMatchingModule = fx.Options(
	fx.Provide(NewEngine),
)

// NewFxEngine creates a new order matching engine for the fx application
func NewFxEngine(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) *Engine {
	engine := NewEngine(logger)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting order matching engine")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping order matching engine")
			return nil
		},
	})

	return engine
}
