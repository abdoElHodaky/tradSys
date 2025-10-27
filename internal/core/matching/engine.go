package order_matching

import (
	"go.uber.org/zap"
)

// Engine represents the order matching engine
type Engine struct {
	logger *zap.Logger
}

// NewEngine creates a new order matching engine
func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{
		logger: logger,
	}
}

// Start starts the engine
func (e *Engine) Start() error {
	e.logger.Info("Order matching engine started")
	return nil
}

// Stop stops the engine
func (e *Engine) Stop() error {
	e.logger.Info("Order matching engine stopped")
	return nil
}
