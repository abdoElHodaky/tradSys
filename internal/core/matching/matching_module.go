package order_matching

import (
	"context"
	"go.uber.org/fx"
	"go.uber.org/zap"
	
	"github.com/abdoElHodaky/tradSys/pkg/matching"
)

// OrderMatchingModule provides the order matching module for the fx application
var OrderMatchingModule = fx.Options(
	fx.Provide(matching.NewEngine),
)

// NewFxEngine creates a new order matching engine for the fx application
func NewFxEngine(
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) *matching.Engine {
	engine := matching.NewEngine(logger)

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
