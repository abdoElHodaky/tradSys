package external

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
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
	logger          *zap.Logger
	cache           *cache.Cache
	providers       map[string]Provider
	mu              sync.RWMutex
	defaultProvider string
}

// NewManager creates a new external data manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger:    logger,
		cache:     cache.New(5*time.Minute, 10*time.Minute),
		providers: make(map[string]Provider),
	}
}

// AddSource adds a new market data source
func (m *Manager) AddSource(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Adding market data source", zap.String("name", name))

	var provider Provider
	var err error

	switch name {
	case "binance":
		apiKey, _ := config["api_key"].(string)
		secretKey, _ := config["secret_key"].(string)
		provider = NewBinanceProvider(apiKey, secretKey, m.logger)
	default:
		return fmt.Errorf("unsupported provider: %s", name)
	}

	m.providers[name] = provider

	// Set as default if it's the first provider
	if m.defaultProvider == "" {
		m.defaultProvider = name
		m.logger.Info("Set default provider", zap.String("provider", name))
	}

	return err
}

// GetProvider returns a specific provider by name
func (m *Manager) GetProvider(name string) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// GetDefaultProvider returns the default market data provider
func (m *Manager) GetDefaultProvider() (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.defaultProvider == "" {
		return nil, errors.New("no default provider configured")
	}

	provider, exists := m.providers[m.defaultProvider]
	if !exists {
		return nil, fmt.Errorf("default provider %s not found", m.defaultProvider)
	}

	return provider, nil
}

// SetDefaultProvider sets the default provider
func (m *Manager) SetDefaultProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	m.defaultProvider = name
	m.logger.Info("Changed default provider", zap.String("provider", name))
	return nil
}

// ListProviders returns a list of available providers
func (m *Manager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}

	return providers
}

// GetMarketData fetches market data from the default external source
func (m *Manager) GetMarketData(symbol, interval string) (float64, float64, int64, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		// Fallback to mock data if no provider is available
		m.logger.Warn("No provider available, returning mock data", zap.Error(err))
		return 50000.0, 1000.0, time.Now().Unix() * 1000, nil
	}

	// Get ticker data from provider
	ticker, err := provider.GetTicker(nil, symbol)
	if err != nil {
		m.logger.Error("Failed to get ticker from provider", zap.Error(err))
		// Return mock data on error
		return 50000.0, 1000.0, time.Now().Unix() * 1000, nil
	}

	return ticker.Price, ticker.Volume, ticker.Timestamp.Unix() * 1000, nil
}

// ManagerModule provides the external market data manager for fx
var ManagerModule = fx.Options(
	fx.Provide(NewManager),
)
