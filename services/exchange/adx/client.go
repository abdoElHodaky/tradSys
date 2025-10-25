// Package adx provides Abu Dhabi Exchange (ADX) client implementation
package adx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// Client implements the ExchangeInterface for Abu Dhabi Exchange
type Client struct {
	config         *Config
	connector      *Connector
	marketData     *MarketDataService
	orderManager   *OrderManager
	assetManager   *AssetManager
	compliance     *ComplianceService
	islamicService *IslamicFinanceService
	mu             sync.RWMutex
	connected      bool
}

// Config holds ADX-specific configuration
type Config struct {
	APIEndpoint       string
	APIKey            string
	APISecret         string
	Timeout           time.Duration
	RetryAttempts     int
	RateLimitRPS      int
	EnableIslamic     bool
	EnableShariaBoard bool
	TradingHours      *types.TradingHours
	SupportedAssets   []types.AssetType
	IslamicAssets     []types.AssetType
}

// NewClient creates a new ADX client
func NewClient(config *Config) *Client {
	return &Client{
		config:         config,
		connector:      NewConnector(config),
		marketData:     NewMarketDataService(config),
		orderManager:   NewOrderManager(config),
		assetManager:   NewAssetManager(config),
		compliance:     NewComplianceService(config),
		islamicService: NewIslamicFinanceService(config),
		connected:      false,
	}
}

// Connect establishes connection to ADX
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.connected {
		return nil
	}
	
	if err := c.connector.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to ADX: %w", err)
	}
	
	// Initialize Islamic finance services if enabled
	if c.config.EnableIslamic {
		if err := c.islamicService.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize Islamic finance services: %w", err)
		}
	}
	
	c.connected = true
	return nil
}

// Disconnect closes connection to ADX
func (c *Client) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.connected {
		return nil
	}
	
	if err := c.connector.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from ADX: %w", err)
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

// PlaceOrder places a trading order on ADX
func (c *Client) PlaceOrder(ctx context.Context, order *interfaces.Order) (*interfaces.OrderResponse, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to ADX")
	}
	
	// Validate order for ADX
	if err := c.validateOrder(order); err != nil {
		return nil, fmt.Errorf("order validation failed: %w", err)
	}
	
	// Check market hours
	if !c.IsMarketOpen() {
		return nil, fmt.Errorf("ADX market is closed")
	}
	
	// Additional Islamic finance validation if needed
	if order.AssetType.IsIslamic() && c.config.EnableIslamic {
		if err := c.islamicService.ValidateOrder(ctx, order); err != nil {
			return nil, fmt.Errorf("Islamic finance validation failed: %w", err)
		}
	}
	
	return c.orderManager.PlaceOrder(ctx, order)
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(ctx context.Context, orderID string) error {
	if !c.connected {
		return fmt.Errorf("not connected to ADX")
	}
	
	return c.orderManager.CancelOrder(ctx, orderID)
}

// GetOrderStatus retrieves order status
func (c *Client) GetOrderStatus(ctx context.Context, orderID string) (*interfaces.OrderStatus, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to ADX")
	}
	
	return c.orderManager.GetOrderStatus(ctx, orderID)
}

// GetMarketData retrieves market data for a symbol
func (c *Client) GetMarketData(ctx context.Context, symbol string) (*interfaces.MarketData, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to ADX")
	}
	
	return c.marketData.GetMarketData(ctx, symbol)
}

// SubscribeMarketData subscribes to real-time market data
func (c *Client) SubscribeMarketData(ctx context.Context, symbols []string) (<-chan *interfaces.MarketData, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to ADX")
	}
	
	return c.marketData.Subscribe(ctx, symbols)
}

// GetAssetInfo retrieves asset information
func (c *Client) GetAssetInfo(ctx context.Context, symbol string) (*interfaces.AssetInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to ADX")
	}
	
	return c.assetManager.GetAssetInfo(ctx, symbol)
}

// ValidateAsset validates an asset for trading
func (c *Client) ValidateAsset(ctx context.Context, asset *interfaces.Asset) error {
	if !c.connected {
		return fmt.Errorf("not connected to ADX")
	}
	
	return c.assetManager.ValidateAsset(ctx, asset)
}

// IsShariahCompliant checks if a symbol is Sharia-compliant
func (c *Client) IsShariahCompliant(ctx context.Context, symbol string) (bool, error) {
	if !c.config.EnableIslamic {
		return false, fmt.Errorf("Islamic finance not enabled for ADX")
	}
	
	return c.islamicService.IsShariahCompliant(ctx, symbol)
}

// GetHalalScreening retrieves Sharia compliance screening
func (c *Client) GetHalalScreening(ctx context.Context, symbol string) (*interfaces.HalalScreening, error) {
	if !c.config.EnableIslamic {
		return nil, fmt.Errorf("Islamic finance not enabled for ADX")
	}
	
	return c.islamicService.GetHalalScreening(ctx, symbol)
}

// GetExchangeType returns the exchange type
func (c *Client) GetExchangeType() types.ExchangeType {
	return types.ADX
}

// GetTradingHours returns ADX trading hours
func (c *Client) GetTradingHours() *types.TradingHours {
	return &types.TradingHours{
		Open:     "10:00",
		Close:    "15:00",
		Timezone: "GST",
	}
}

// IsMarketOpen checks if ADX market is currently open
func (c *Client) IsMarketOpen() bool {
	return types.ADX.IsMarketOpen()
}

// validateOrder validates an order for ADX-specific requirements
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
		return fmt.Errorf("asset type %s not supported on ADX", order.AssetType)
	}
	
	// Validate price and quantity
	if order.Price <= 0 {
		return fmt.Errorf("price must be positive")
	}
	
	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	
	// ADX-specific validations
	if order.AssetType == types.SUKUK {
		// Minimum order size for Sukuk
		if order.Quantity < 1000 {
			return fmt.Errorf("minimum Sukuk quantity is 1000")
		}
	}
	
	if order.AssetType == types.ISLAMIC_FUND {
		// Islamic fund specific validations
		if order.Quantity < 100 {
			return fmt.Errorf("minimum Islamic fund quantity is 100")
		}
	}
	
	return nil
}

// GetSukukInfo retrieves Sukuk-specific information
func (c *Client) GetSukukInfo(ctx context.Context, symbol string) (*SukukInfo, error) {
	if !c.config.EnableIslamic {
		return nil, fmt.Errorf("Islamic finance not enabled")
	}
	
	return c.islamicService.GetSukukInfo(ctx, symbol)
}

// CalculateZakat calculates Zakat for Islamic assets
func (c *Client) CalculateZakat(ctx context.Context, portfolio *IslamicPortfolio) (*ZakatCalculation, error) {
	if !c.config.EnableIslamic {
		return nil, fmt.Errorf("Islamic finance not enabled")
	}
	
	return c.islamicService.CalculateZakat(ctx, portfolio)
}

// GetShariaBoard returns information about the Sharia board
func (c *Client) GetShariaBoard(ctx context.Context) (*ShariaBoard, error) {
	if !c.config.EnableShariaBoard {
		return nil, fmt.Errorf("Sharia board information not enabled")
	}
	
	return c.islamicService.GetShariaBoard(ctx)
}

// GetDefaultConfig returns default ADX configuration
func GetDefaultConfig() *Config {
	return &Config{
		APIEndpoint:       "https://api.adx.ae",
		Timeout:           30 * time.Second,
		RetryAttempts:     3,
		RateLimitRPS:      100,
		EnableIslamic:     true,
		EnableShariaBoard: true,
		TradingHours: &types.TradingHours{
			Open:     "10:00",
			Close:    "15:00",
			Timezone: "GST",
		},
		SupportedAssets: []types.AssetType{
			types.STOCK, types.BOND, types.ETF, types.REIT,
			types.SUKUK, types.ISLAMIC_FUND, types.SHARIA_STOCK,
			types.ISLAMIC_ETF, types.ISLAMIC_REIT, types.TAKAFUL,
		},
		IslamicAssets: []types.AssetType{
			types.SUKUK, types.ISLAMIC_FUND, types.SHARIA_STOCK,
			types.ISLAMIC_ETF, types.ISLAMIC_REIT, types.TAKAFUL,
		},
	}
}

// SukukInfo represents Sukuk-specific information
type SukukInfo struct {
	Symbol          string    `json:"symbol"`
	IssuerName      string    `json:"issuer_name"`
	MaturityDate    time.Time `json:"maturity_date"`
	CouponRate      float64   `json:"coupon_rate"`
	FaceValue       float64   `json:"face_value"`
	IssuanceDate    time.Time `json:"issuance_date"`
	ShariaCompliant bool      `json:"sharia_compliant"`
	UnderlyingAsset string    `json:"underlying_asset"`
	Rating          string    `json:"rating"`
}

// IslamicPortfolio represents a portfolio of Islamic assets
type IslamicPortfolio struct {
	UserID     string                    `json:"user_id"`
	Assets     map[string]IslamicAsset   `json:"assets"`
	TotalValue float64                   `json:"total_value"`
	Currency   string                    `json:"currency"`
	AsOfDate   time.Time                 `json:"as_of_date"`
}

// IslamicAsset represents an Islamic financial asset
type IslamicAsset struct {
	Symbol       string          `json:"symbol"`
	AssetType    types.AssetType `json:"asset_type"`
	Quantity     float64         `json:"quantity"`
	CurrentPrice float64         `json:"current_price"`
	MarketValue  float64         `json:"market_value"`
	IsHalal      bool            `json:"is_halal"`
}

// ZakatCalculation represents Zakat calculation results
type ZakatCalculation struct {
	PortfolioValue    float64   `json:"portfolio_value"`
	ZakatableAmount   float64   `json:"zakatable_amount"`
	ZakatRate         float64   `json:"zakat_rate"`
	ZakatDue          float64   `json:"zakat_due"`
	Currency          string    `json:"currency"`
	CalculationDate   time.Time `json:"calculation_date"`
	NextDueDate       time.Time `json:"next_due_date"`
	ExemptAssets      []string  `json:"exempt_assets"`
}

// ShariaBoard represents Sharia board information
type ShariaBoard struct {
	Name        string              `json:"name"`
	Members     []ShariaBoardMember `json:"members"`
	Established time.Time           `json:"established"`
	Certifications []string         `json:"certifications"`
	ContactInfo ContactInfo         `json:"contact_info"`
}

// ShariaBoardMember represents a Sharia board member
type ShariaBoardMember struct {
	Name         string `json:"name"`
	Title        string `json:"title"`
	Qualifications []string `json:"qualifications"`
	Experience   int    `json:"experience_years"`
}

// ContactInfo represents contact information
type ContactInfo struct {
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Website string `json:"website"`
}
