package external

import (
	"context"
	"errors"
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

// GetDefaultProvider returns the default provider
func (m *Manager) GetDefaultProvider() (Provider, error) {
	// Return the first available provider as default
	for _, provider := range m.providers {
		return provider, nil
	}
	
	// If no providers are registered, create a mock provider
	mockProvider := &MockProvider{
		name: "mock",
	}
	return mockProvider, nil
}

// MockProvider is a mock provider for testing
type MockProvider struct {
	name string
}

// Name returns the provider name
func (m *MockProvider) Name() string {
	return m.name
}

// Connect connects to the provider
func (m *MockProvider) Connect(ctx context.Context) error {
	return nil
}

// Disconnect disconnects from the provider
func (m *MockProvider) Disconnect(ctx context.Context) error {
	return nil
}

// GetTicker gets ticker data
func (m *MockProvider) GetTicker(ctx context.Context, symbol string) (*TickerData, error) {
	return &TickerData{
		Symbol: symbol,
		Price:  100.0,
		Volume: 1000.0,
	}, nil
}

// GetOrderBook gets order book data
func (m *MockProvider) GetOrderBook(ctx context.Context, symbol string) (*OrderBookData, error) {
	return &OrderBookData{
		Symbol: symbol,
		Bids:   []PriceLevel{{Price: 99.0, Quantity: 100}},
		Asks:   []PriceLevel{{Price: 101.0, Quantity: 100}},
	}, nil
}

// GetTrades gets trade data
func (m *MockProvider) GetTrades(ctx context.Context, symbol string, limit int) ([]TradeData, error) {
	return []TradeData{
		{
			Symbol:    symbol,
			Price:     100.0,
			Quantity:  10.0,
			Timestamp: time.Now(),
		},
	}, nil
}

// GetOHLCV gets OHLCV data
func (m *MockProvider) GetOHLCV(ctx context.Context, symbol, interval string, limit int) ([]OHLCVData, error) {
	return []OHLCVData{
		{
			Symbol:    symbol,
			Timestamp: time.Now(),
			Open:      100.0,
			High:      105.0,
			Low:       95.0,
			Close:     102.0,
			Volume:    1000.0,
		},
	}, nil
}

// GetSymbols gets available symbols
func (m *MockProvider) GetSymbols(ctx context.Context) ([]string, error) {
	return []string{"BTCUSDT", "ETHUSDT", "ADAUSDT"}, nil
}

// SubscribeOrderBook subscribes to order book updates
func (m *MockProvider) SubscribeOrderBook(ctx context.Context, symbol string, callback MarketDataCallback) error {
	return errors.New("subscription not supported in mock provider")
}

// UnsubscribeOrderBook unsubscribes from order book updates
func (m *MockProvider) UnsubscribeOrderBook(ctx context.Context, symbol string) error {
	return errors.New("unsubscription not supported in mock provider")
}

// SubscribeTrades subscribes to trade updates
func (m *MockProvider) SubscribeTrades(ctx context.Context, symbol string, callback MarketDataCallback) error {
	return errors.New("subscription not supported in mock provider")
}

// UnsubscribeTrades unsubscribes from trade updates
func (m *MockProvider) UnsubscribeTrades(ctx context.Context, symbol string) error {
	return errors.New("unsubscription not supported in mock provider")
}

// SubscribeTicker subscribes to ticker updates
func (m *MockProvider) SubscribeTicker(ctx context.Context, symbol string, callback MarketDataCallback) error {
	return errors.New("subscription not supported in mock provider")
}

// UnsubscribeTicker unsubscribes from ticker updates
func (m *MockProvider) UnsubscribeTicker(ctx context.Context, symbol string) error {
	return errors.New("unsubscription not supported in mock provider")
}

// SubscribeOHLCV subscribes to OHLCV updates
func (m *MockProvider) SubscribeOHLCV(ctx context.Context, symbol, interval string, callback MarketDataCallback) error {
	return errors.New("subscription not supported in mock provider")
}

// UnsubscribeOHLCV unsubscribes from OHLCV updates
func (m *MockProvider) UnsubscribeOHLCV(ctx context.Context, symbol, interval string) error {
	return errors.New("unsubscription not supported in mock provider")
}
