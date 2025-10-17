package external

import (
	"context"
	"time"

	cache "github.com/patrickmn/go-cache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ManagerParams contains parameters for creating an external data manager
type ManagerParams struct {
	fx.In

	Logger *zap.Logger
}

// Manager handles external market data sources
type Manager struct {
	logger    *zap.Logger
	cache     *cache.Cache
	providers map[string]Provider
}

// NewManager creates a new external data manager
func NewManager(p ManagerParams) *Manager {
	return &Manager{
		logger:    p.Logger,
		cache:     cache.New(5*time.Minute, 10*time.Minute),
		providers: make(map[string]Provider),
	}
}

// RegisterProvider registers a provider
func (m *Manager) RegisterProvider(provider Provider) {
	m.providers[provider.Name()] = provider
}

// ConnectAll connects all providers
func (m *Manager) ConnectAll(ctx context.Context) error {
	for _, provider := range m.providers {
		if err := provider.Connect(ctx); err != nil {
			m.logger.Error("Failed to connect provider", zap.String("provider", provider.Name()), zap.Error(err))
			return err
		}
	}
	return nil
}

// DisconnectAll disconnects all providers
func (m *Manager) DisconnectAll(ctx context.Context) error {
	for _, provider := range m.providers {
		if err := provider.Disconnect(ctx); err != nil {
			m.logger.Error("Failed to disconnect provider", zap.String("provider", provider.Name()), zap.Error(err))
		}
	}
	return nil
}

// GetMarketData fetches market data from external sources
func (m *Manager) GetMarketData(symbol, interval string) (float64, float64, int64, error) {
	// In a real implementation, this would fetch data from external APIs
	// For now, just return placeholder values
	return 100.0, 1000.0, time.Now().Unix() * 1000, nil
}
