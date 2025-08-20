package order_matching

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the order matching module for the fx application
var Module = fx.Options(
	fx.Provide(NewEngine),
)

// NewFxEngine creates a new order matching engine for the fx application
func NewFxEngine(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) *Engine {
	engine := NewEngine(logger)
	
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx fx.Context) error {
			logger.Info("Starting order matching engine")
			return nil
		},
		OnStop: func(ctx fx.Context) error {
			logger.Info("Stopping order matching engine")
			return nil
		},
	})
	
	return engine
}

