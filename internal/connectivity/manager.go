package connectivity

import (
	"context"

	"go.uber.org/zap"
)

// Manager represents the connectivity manager
type Manager struct {
	logger *zap.Logger
}

// NewManager creates a new connectivity manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// Start starts the connectivity manager
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting connectivity manager")
	return nil
}

// Stop stops the connectivity manager
func (m *Manager) Stop() error {
	m.logger.Info("Stopping connectivity manager")
	return nil
}
