package strategies

import (
	"context"

	"go.uber.org/zap"
)

// Engine represents the strategies engine
type Engine struct {
	logger *zap.Logger
}

// NewEngine creates a new strategies engine
func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{
		logger: logger,
	}
}

// Start starts the strategies engine
func (e *Engine) Start(ctx context.Context) error {
	e.logger.Info("Starting strategies engine")
	return nil
}

// Stop stops the strategies engine
func (e *Engine) Stop() error {
	e.logger.Info("Stopping strategies engine")
	return nil
}
