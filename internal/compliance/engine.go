package compliance

import (
	"context"

	"go.uber.org/zap"
)

// Engine represents the compliance engine
type Engine struct {
	logger *zap.Logger
}

// NewEngine creates a new compliance engine
func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{
		logger: logger,
	}
}

// Start starts the compliance engine
func (e *Engine) Start(ctx context.Context) error {
	e.logger.Info("Starting compliance engine")
	return nil
}

// Stop stops the compliance engine
func (e *Engine) Stop() error {
	e.logger.Info("Stopping compliance engine")
	return nil
}
