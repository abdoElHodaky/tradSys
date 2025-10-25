// Package exchange provides unified exchange factory and management
package exchange

import (
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
	"github.com/abdoElHodaky/tradSys/services/exchange/egx"
	"github.com/abdoElHodaky/tradSys/services/exchange/adx"
)

// Factory manages exchange client instances
type Factory struct {
	exchanges map[types.ExchangeType]interfaces.ExchangeInterface
	configs   map[types.ExchangeType]interface{}
	mu        sync.RWMutex
}

// NewFactory creates a new exchange factory
func NewFactory() *Factory {
	return &Factory{
		exchanges: make(map[types.ExchangeType]interfaces.ExchangeInterface),
		configs:   make(map[types.ExchangeType]interface{}),
	}
}

// RegisterExchange registers an exchange client with the factory
func (f *Factory) RegisterExchange(exchangeType types.ExchangeType, exchange interfaces.ExchangeInterface) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exchanges[exchangeType] = exchange
}

// GetExchange retrieves an exchange client by type
func (f *Factory) GetExchange(exchangeType types.ExchangeType) (interfaces.ExchangeInterface, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	exchange, exists := f.exchanges[exchangeType]
	if !exists {
		return nil, fmt.Errorf("exchange %s not registered", exchangeType)
	}
	
	return exchange, nil
}

// GetAllExchanges returns all registered exchanges
func (f *Factory) GetAllExchanges() map[types.ExchangeType]interfaces.ExchangeInterface {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	result := make(map[types.ExchangeType]interfaces.ExchangeInterface)
	for k, v := range f.exchanges {
		result[k] = v
	}
	
	return result
}

// IsExchangeRegistered checks if an exchange is registered
func (f *Factory) IsExchangeRegistered(exchangeType types.ExchangeType) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	_, exists := f.exchanges[exchangeType]
	return exists
}

// GetSupportedExchanges returns list of supported exchange types
func (f *Factory) GetSupportedExchanges() []types.ExchangeType {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	var exchanges []types.ExchangeType
	for exchangeType := range f.exchanges {
		exchanges = append(exchanges, exchangeType)
	}
	
	return exchanges
}

// CreateEGXClient creates and registers an EGX client
func (f *Factory) CreateEGXClient(config *egx.Config) error {
	if config == nil {
		config = egx.GetDefaultConfig()
	}
	
	client := egx.NewClient(config)
	f.RegisterExchange(types.EGX, client)
	
	f.mu.Lock()
	f.configs[types.EGX] = config
	f.mu.Unlock()
	
	return nil
}

// CreateADXClient creates and registers an ADX client
func (f *Factory) CreateADXClient(config *adx.Config) error {
	if config == nil {
		config = adx.GetDefaultConfig()
	}
	
	client := adx.NewClient(config)
	f.RegisterExchange(types.ADX, client)
	
	f.mu.Lock()
	f.configs[types.ADX] = config
	f.mu.Unlock()
	
	return nil
}

// GetExchangeConfig retrieves configuration for an exchange
func (f *Factory) GetExchangeConfig(exchangeType types.ExchangeType) (interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	config, exists := f.configs[exchangeType]
	if !exists {
		return nil, fmt.Errorf("config for exchange %s not found", exchangeType)
	}
	
	return config, nil
}

// UpdateExchangeConfig updates configuration for an exchange
func (f *Factory) UpdateExchangeConfig(exchangeType types.ExchangeType, config interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, exists := f.exchanges[exchangeType]; !exists {
		return fmt.Errorf("exchange %s not registered", exchangeType)
	}
	
	f.configs[exchangeType] = config
	return nil
}

// RemoveExchange removes an exchange from the factory
func (f *Factory) RemoveExchange(exchangeType types.ExchangeType) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, exists := f.exchanges[exchangeType]; !exists {
		return fmt.Errorf("exchange %s not registered", exchangeType)
	}
	
	delete(f.exchanges, exchangeType)
	delete(f.configs, exchangeType)
	
	return nil
}

// ValidateExchangeSupport validates if an exchange supports a specific asset type
func (f *Factory) ValidateExchangeSupport(exchangeType types.ExchangeType, assetType types.AssetType) error {
	exchange, err := f.GetExchange(exchangeType)
	if err != nil {
		return err
	}
	
	// For now, we'll use a simple validation based on exchange type
	// This can be enhanced with more sophisticated logic
	switch exchangeType {
	case types.EGX:
		return f.validateEGXAssetSupport(assetType)
	case types.ADX:
		return f.validateADXAssetSupport(assetType)
	default:
		return fmt.Errorf("unsupported exchange: %s", exchangeType)
	}
}

// validateEGXAssetSupport validates EGX asset support
func (f *Factory) validateEGXAssetSupport(assetType types.AssetType) error {
	supportedAssets := []types.AssetType{
		types.STOCK, types.BOND, types.ETF, types.REIT,
		types.MUTUAL_FUND, types.SUKUK, types.ISLAMIC_FUND,
		types.SHARIA_STOCK, types.ISLAMIC_ETF,
	}
	
	for _, supported := range supportedAssets {
		if assetType == supported {
			return nil
		}
	}
	
	return fmt.Errorf("asset type %s not supported on EGX", assetType)
}

// validateADXAssetSupport validates ADX asset support
func (f *Factory) validateADXAssetSupport(assetType types.AssetType) error {
	supportedAssets := []types.AssetType{
		types.STOCK, types.BOND, types.ETF, types.REIT,
		types.SUKUK, types.ISLAMIC_FUND, types.SHARIA_STOCK,
		types.ISLAMIC_ETF, types.ISLAMIC_REIT, types.TAKAFUL,
	}
	
	for _, supported := range supportedAssets {
		if assetType == supported {
			return nil
		}
	}
	
	return fmt.Errorf("asset type %s not supported on ADX", assetType)
}

// GetExchangeForAsset returns the best exchange for a given asset type
func (f *Factory) GetExchangeForAsset(assetType types.AssetType) (interfaces.ExchangeInterface, error) {
	// Simple logic: prefer EGX for traditional assets, ADX for Islamic assets
	if assetType.IsIslamic() {
		if f.IsExchangeRegistered(types.ADX) {
			return f.GetExchange(types.ADX)
		}
	}
	
	// Default to EGX if available
	if f.IsExchangeRegistered(types.EGX) {
		return f.GetExchange(types.EGX)
	}
	
	return nil, fmt.Errorf("no suitable exchange found for asset type %s", assetType)
}

// DefaultFactory is the global exchange factory instance
var DefaultFactory = NewFactory()

// GetDefaultFactory returns the default factory instance
func GetDefaultFactory() *Factory {
	return DefaultFactory
}

// InitializeDefaultExchanges initializes default exchange clients
func InitializeDefaultExchanges() error {
	// Initialize EGX with default config
	if err := DefaultFactory.CreateEGXClient(nil); err != nil {
		return fmt.Errorf("failed to initialize EGX client: %w", err)
	}
	
	// Initialize ADX with default config
	if err := DefaultFactory.CreateADXClient(nil); err != nil {
		return fmt.Errorf("failed to initialize ADX client: %w", err)
	}
	
	return nil
}
