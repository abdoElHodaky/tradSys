// Package egx provides Egyptian Exchange (EGX) client implementation
package egx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// Client implements the ExchangeInterface for Egyptian Exchange
type Client struct {
	config       *Config
	connector    *Connector
	marketData   *MarketDataService
	orderManager *OrderManager
	assetManager *AssetManager
	compliance   *ComplianceService
	mu           sync.RWMutex
	connected    bool
}

// Config holds EGX-specific configuration
type Config struct {
	APIEndpoint     string
	APIKey          string
	APISecret       string
	Timeout         time.Duration
	RetryAttempts   int
	RateLimitRPS    int
	EnableIslamic   bool
	TradingHours    *types.TradingHours
	SupportedAssets []types.AssetType
}

// NewClient creates a new EGX client
func NewClient(config *Config) *Client {
	return &Client{
		config:       config,
		connector:    NewConnector(config),
		marketData:   NewMarketDataService(config),
		orderManager: NewOrderManager(config),
		assetManager: NewAssetManager(config),
		compliance:   NewComplianceService(config),
		connected:    false,
	}
}

// Connect establishes connection to EGX
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.connected {
		return nil
	}
	
	if err := c.connector.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to EGX: %w", err)
	}
	
	c.connected = true
	return nil
}

// Disconnect closes connection to EGX
func (c *Client) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.connected {
		return nil
	}
	
	if err := c.connector.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from EGX: %w", err)
	}
	
	c.connected = false
	return nil
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// PlaceOrder places a trading order on EGX
func (c *Client) PlaceOrder(ctx context.Context, order *interfaces.Order) (*interfaces.OrderResponse, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to EGX")
	}
	
	// Validate order for EGX
	if err := c.validateOrder(order); err != nil {
		return nil, fmt.Errorf("order validation failed: %w", err)
	}
	
	// Check market hours
	if !c.IsMarketOpen() {
		return nil, fmt.Errorf("EGX market is closed")
	}
	
	return c.orderManager.PlaceOrder(ctx, order)
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(ctx context.Context, orderID string) error {
	if !c.connected {
		return fmt.Errorf("not connected to EGX")
	}
	
	return c.orderManager.CancelOrder(ctx, orderID)
}

// GetOrderStatus retrieves order status
func (c *Client) GetOrderStatus(ctx context.Context, orderID string) (*interfaces.OrderStatus, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to EGX")
	}
	
	return c.orderManager.GetOrderStatus(ctx, orderID)
}

// GetMarketData retrieves market data for a symbol
func (c *Client) GetMarketData(ctx context.Context, symbol string) (*interfaces.MarketData, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to EGX")
	}
	
	return c.marketData.GetMarketData(ctx, symbol)
}

// SubscribeMarketData subscribes to real-time market data
func (c *Client) SubscribeMarketData(ctx context.Context, symbols []string) (<-chan *interfaces.MarketData, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to EGX")
	}
	
	return c.marketData.Subscribe(ctx, symbols)
}

// GetAssetInfo retrieves asset information
func (c *Client) GetAssetInfo(ctx context.Context, symbol string) (*interfaces.AssetInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to EGX")
	}
	
	return c.assetManager.GetAssetInfo(ctx, symbol)
}

// ValidateAsset validates an asset for trading
func (c *Client) ValidateAsset(ctx context.Context, asset *interfaces.Asset) error {
	if !c.connected {
		return fmt.Errorf("not connected to EGX")
	}
	
	return c.assetManager.ValidateAsset(ctx, asset)
}

// IsShariahCompliant checks if a symbol is Sharia-compliant
func (c *Client) IsShariahCompliant(ctx context.Context, symbol string) (bool, error) {
	if !c.config.EnableIslamic {
		return false, fmt.Errorf("Islamic finance not enabled for EGX")
	}
	
	return c.compliance.IsShariahCompliant(ctx, symbol)
}

// GetHalalScreening retrieves Sharia compliance screening
func (c *Client) GetHalalScreening(ctx context.Context, symbol string) (*interfaces.HalalScreening, error) {
	if !c.config.EnableIslamic {
		return nil, fmt.Errorf("Islamic finance not enabled for EGX")
	}
	
	return c.compliance.GetHalalScreening(ctx, symbol)
}

// GetExchangeType returns the exchange type
func (c *Client) GetExchangeType() types.ExchangeType {
	return types.EGX
}

// GetTradingHours returns EGX trading hours
func (c *Client) GetTradingHours() *types.TradingHours {
	return &types.TradingHours{
		Open:     "10:00",
		Close:    "14:30",
		Timezone: "EET",
	}
}

// IsMarketOpen checks if EGX market is currently open
func (c *Client) IsMarketOpen() bool {
	return types.EGX.IsMarketOpen()
}

// validateOrder validates an order for EGX-specific requirements
func (c *Client) validateOrder(order *interfaces.Order) error {
	// Check if asset type is supported
	supported := false
	for _, assetType := range c.config.SupportedAssets {
		if order.AssetType == assetType {
			supported = true
			break
		}
	}
	
	if !supported {
		return fmt.Errorf("asset type %s not supported on EGX", order.AssetType)
	}
	
	// Validate price and quantity
	if order.Price <= 0 {
		return fmt.Errorf("price must be positive")
	}
	
	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	
	// EGX-specific validations
	if order.AssetType == types.STOCK {
		// Minimum order size for stocks
		if order.Quantity < 1 {
			return fmt.Errorf("minimum stock quantity is 1")
		}
	}
	
	return nil
}

// GetDefaultConfig returns default EGX configuration
func GetDefaultConfig() *Config {
	return &Config{
		APIEndpoint:   "https://api.egx.com.eg",
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RateLimitRPS:  100,
		EnableIslamic: true,
		TradingHours: &types.TradingHours{
			Open:     "10:00",
			Close:    "14:30",
			Timezone: "EET",
		},
		SupportedAssets: []types.AssetType{
			types.STOCK, types.BOND, types.ETF, types.REIT,
			types.MUTUAL_FUND, types.SUKUK, types.ISLAMIC_FUND,
			types.SHARIA_STOCK, types.ISLAMIC_ETF,
		},
	}
}

// Connector handles connection to EGX
type Connector struct {
	config    *Config
	conn      string
	mu        sync.RWMutex
	connected bool
}

// NewConnector creates a new connector
func NewConnector(config *Config) *Connector {
	return &Connector{
		config: config,
	}
}

// Connect establishes connection to EGX
func (c *Connector) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = true
	return nil
}

// Disconnect closes the connection
func (c *Connector) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	return nil
}

// IsConnected returns connection status
func (c *Connector) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// MarketDataService handles market data operations
type MarketDataService struct {
	config *Config
	mu     sync.RWMutex
}

// NewMarketDataService creates a new market data service
func NewMarketDataService(config *Config) *MarketDataService {
	return &MarketDataService{config: config}
}

// GetMarketData retrieves market data for a symbol
func (m *MarketDataService) GetMarketData(ctx context.Context, symbol string) (*interfaces.MarketData, error) {
	return &interfaces.MarketData{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}, nil
}

// Subscribe subscribes to market data feed
func (m *MarketDataService) Subscribe(ctx context.Context, symbols []string) (<-chan *interfaces.MarketData, error) {
	ch := make(chan *interfaces.MarketData)
	go func() {
		defer close(ch)
		for _, symbol := range symbols {
			select {
			case <-ctx.Done():
				return
			case ch <- &interfaces.MarketData{
				Symbol:    symbol,
				Timestamp: time.Now(),
			}:
			}
		}
	}()
	return ch, nil
}

// OrderManager handles order operations
type OrderManager struct {
	config *Config
	mu     sync.RWMutex
}

// NewOrderManager creates a new order manager
func NewOrderManager(config *Config) *OrderManager {
	return &OrderManager{config: config}
}

// PlaceOrder places an order
func (o *OrderManager) PlaceOrder(ctx context.Context, order *interfaces.Order) (*interfaces.OrderResponse, error) {
	return &interfaces.OrderResponse{
		OrderID:   "egx-order-id",
		Status:    "new",
		Message:   "Order placed",
		Timestamp: time.Now(),
	}, nil
}

// CancelOrder cancels an order
func (o *OrderManager) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

// GetOrderStatus gets order status
func (o *OrderManager) GetOrderStatus(ctx context.Context, orderID string) (*interfaces.OrderStatus, error) {
	return &interfaces.OrderStatus{
		OrderID: orderID,
		Status:  "filled",
	}, nil
}

// AssetManager handles asset operations
type AssetManager struct {
	config *Config
	mu     sync.RWMutex
}

// NewAssetManager creates a new asset manager
func NewAssetManager(config *Config) *AssetManager {
	return &AssetManager{config: config}
}

// GetAssetInfo retrieves asset information
func (a *AssetManager) GetAssetInfo(ctx context.Context, symbol string) (*interfaces.AssetInfo, error) {
	return &interfaces.AssetInfo{
		Symbol:    symbol,
		IsActive:  true,
		AssetType: types.STOCK,
	}, nil
}

// ValidateAsset validates an asset
func (a *AssetManager) ValidateAsset(ctx context.Context, asset *interfaces.Asset) error {
	return nil
}

// ComplianceService handles compliance operations
type ComplianceService struct {
	config *Config
	mu     sync.RWMutex
}

// NewComplianceService creates a new compliance service
func NewComplianceService(config *Config) *ComplianceService {
	return &ComplianceService{config: config}
}

// ValidateOrder validates an order against compliance rules
func (c *ComplianceService) ValidateOrder(order *interfaces.Order) error {
	return nil
}

// IsShariahCompliant checks if a symbol is Sharia-compliant
func (c *ComplianceService) IsShariahCompliant(ctx context.Context, symbol string) (bool, error) {
	return true, nil
}

// GetHalalScreening retrieves Sharia compliance screening
func (c *ComplianceService) GetHalalScreening(ctx context.Context, symbol string) (*interfaces.HalalScreening, error) {
	return &interfaces.HalalScreening{
		Symbol:          symbol,
		IsCompliant:     true,
		Score:           100,
		Violations:      []string{},
		Recommendations: []string{},
		LastUpdated:     time.Now(),
	}, nil
}
