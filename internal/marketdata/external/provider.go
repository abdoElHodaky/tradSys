package external

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MarketDataType represents the type of market data
type MarketDataType string

const (
	// MarketDataTypeOrderBook represents order book data
	MarketDataTypeOrderBook MarketDataType = "order_book"
	// MarketDataTypeTrade represents trade data
	MarketDataTypeTrade MarketDataType = "trade"
	// MarketDataTypeTicker represents ticker data
	MarketDataTypeTicker MarketDataType = "ticker"
	// MarketDataTypeOHLCV represents OHLCV data
	MarketDataTypeOHLCV MarketDataType = "ohlcv"
)

// OrderBookData represents order book data
type OrderBookData struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the bids
	Bids [][]float64
	// Asks is the asks
	Asks [][]float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// TradeData represents trade data
type TradeData struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the price of the trade
	Price float64
	// Quantity is the quantity of the trade
	Quantity float64
	// Side is the side of the trade
	Side string
	// Timestamp is the time of the trade
	Timestamp time.Time
	// TradeID is the trade ID
	TradeID string
}

// TickerData represents ticker data
type TickerData struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the current price
	Price float64
	// Volume is the 24-hour volume
	Volume float64
	// Change is the 24-hour price change
	Change float64
	// ChangePercent is the 24-hour price change percentage
	ChangePercent float64
	// High is the 24-hour high price
	High float64
	// Low is the 24-hour low price
	Low float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// OHLCVData represents OHLCV data
type OHLCVData struct {
	// Symbol is the trading symbol
	Symbol string
	// Interval is the interval
	Interval string
	// Open is the open price
	Open float64
	// High is the high price
	High float64
	// Low is the low price
	Low float64
	// Close is the close price
	Close float64
	// Volume is the volume
	Volume float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// MarketDataCallback is a callback function for market data
type MarketDataCallback func(interface{})

// Provider represents a market data provider
type Provider interface {
	// Name returns the name of the provider
	Name() string
	// Connect connects to the provider
	Connect(ctx context.Context) error
	// Disconnect disconnects from the provider
	Disconnect(ctx context.Context) error
	// SubscribeOrderBook subscribes to order book updates
	SubscribeOrderBook(ctx context.Context, symbol string, callback MarketDataCallback) error
	// UnsubscribeOrderBook unsubscribes from order book updates
	UnsubscribeOrderBook(ctx context.Context, symbol string) error
	// SubscribeTrades subscribes to trade updates
	SubscribeTrades(ctx context.Context, symbol string, callback MarketDataCallback) error
	// UnsubscribeTrades unsubscribes from trade updates
	UnsubscribeTrades(ctx context.Context, symbol string) error
	// SubscribeTicker subscribes to ticker updates
	SubscribeTicker(ctx context.Context, symbol string, callback MarketDataCallback) error
	// UnsubscribeTicker unsubscribes from ticker updates
	UnsubscribeTicker(ctx context.Context, symbol string) error
	// SubscribeOHLCV subscribes to OHLCV updates
	SubscribeOHLCV(ctx context.Context, symbol, interval string, callback MarketDataCallback) error
	// UnsubscribeOHLCV unsubscribes from OHLCV updates
	UnsubscribeOHLCV(ctx context.Context, symbol, interval string) error
	// GetOrderBook gets the order book
	GetOrderBook(ctx context.Context, symbol string) (*OrderBookData, error)
	// GetTrades gets trades
	GetTrades(ctx context.Context, symbol string, limit int) ([]TradeData, error)
	// GetTicker gets the ticker
	GetTicker(ctx context.Context, symbol string) (*TickerData, error)
	// GetOHLCV gets OHLCV data
	GetOHLCV(ctx context.Context, symbol, interval string, limit int) ([]OHLCVData, error)
}

// ProviderManager represents a market data provider manager (deprecated - use Manager from manager.go)
type ProviderManager struct {
	// Providers is a map of provider name to provider
	Providers map[string]Provider
	// DefaultProvider is the default provider
	DefaultProvider string
	// Logger
	logger *zap.Logger
	// Mutex for thread safety
	mu sync.RWMutex
}

// NewProviderManager creates a new market data provider manager (deprecated - use NewManager from manager.go)
func NewProviderManager(logger *zap.Logger) *ProviderManager {
	return &ProviderManager{
		Providers: make(map[string]Provider),
		logger:    logger,
	}
}

// RegisterProvider registers a provider
func (m *ProviderManager) RegisterProvider(provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Providers[provider.Name()] = provider
	
	// Set as default if no default provider is set
	if m.DefaultProvider == "" {
		m.DefaultProvider = provider.Name()
	}
}

// SetDefaultProvider sets the default provider
func (m *ProviderManager) SetDefaultProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.Providers[name]; !exists {
		return errors.New("provider not found")
	}
	
	m.DefaultProvider = name
	return nil
}

// GetProvider gets a provider by name
func (m *ProviderManager) GetProvider(name string) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	provider, exists := m.Providers[name]
	if !exists {
		return nil, errors.New("provider not found")
	}
	
	return provider, nil
}

// GetDefaultProvider gets the default provider
func (m *ProviderManager) GetDefaultProvider() (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.DefaultProvider == "" {
		return nil, errors.New("no default provider set")
	}
	
	provider, exists := m.Providers[m.DefaultProvider]
	if !exists {
		return nil, errors.New("default provider not found")
	}
	
	return provider, nil
}

// ConnectAll connects to all providers
func (m *ProviderManager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for name, provider := range m.Providers {
		if err := provider.Connect(ctx); err != nil {
			m.logger.Error("Failed to connect to provider", 
				zap.Error(err), 
				zap.String("provider", name))
			return err
		}
	}
	
	return nil
}

// DisconnectAll disconnects from all providers
func (m *ProviderManager) DisconnectAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for name, provider := range m.Providers {
		if err := provider.Disconnect(ctx); err != nil {
			m.logger.Error("Failed to disconnect from provider", 
				zap.Error(err), 
				zap.String("provider", name))
			return err
		}
	}
	
	return nil
}
