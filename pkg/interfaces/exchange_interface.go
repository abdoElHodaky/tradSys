// Package interfaces provides unified interfaces for TradSys v3
package interfaces

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// ExchangeInterface defines the standard interface for all exchange implementations
type ExchangeInterface interface {
	// Core trading operations
	PlaceOrder(ctx context.Context, order *Order) (*OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error)
	
	// Market data operations
	GetMarketData(ctx context.Context, symbol string) (*MarketData, error)
	SubscribeMarketData(ctx context.Context, symbols []string) (<-chan *MarketData, error)
	
	// Asset-specific operations
	GetAssetInfo(ctx context.Context, symbol string) (*AssetInfo, error)
	ValidateAsset(ctx context.Context, asset *Asset) error
	
	// Islamic finance operations
	IsShariahCompliant(ctx context.Context, symbol string) (bool, error)
	GetHalalScreening(ctx context.Context, symbol string) (*HalalScreening, error)
	
	// Exchange information
	GetExchangeType() types.ExchangeType
	GetTradingHours() *types.TradingHours
	IsMarketOpen() bool
	
	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
}

// Order represents a trading order
type Order struct {
	ID          string           `json:"id"`
	Symbol      string           `json:"symbol"`
	AssetType   types.AssetType  `json:"asset_type"`
	Exchange    types.ExchangeType `json:"exchange"`
	Side        OrderSide        `json:"side"`
	Type        OrderType        `json:"type"`
	Quantity    float64          `json:"quantity"`
	Price       float64          `json:"price"`
	TimeInForce TimeInForce      `json:"time_in_force"`
	UserID      string           `json:"user_id"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// OrderResponse represents the response from placing an order
type OrderResponse struct {
	OrderID     string    `json:"order_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}

// OrderStatus represents the current status of an order
type OrderStatus struct {
	OrderID       string    `json:"order_id"`
	Status        string    `json:"status"`
	FilledQty     float64   `json:"filled_qty"`
	RemainingQty  float64   `json:"remaining_qty"`
	AvgPrice      float64   `json:"avg_price"`
	LastUpdated   time.Time `json:"last_updated"`
}

// MarketData represents real-time market data
type MarketData struct {
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Volume      float64   `json:"volume"`
	High        float64   `json:"high"`
	Low         float64   `json:"low"`
	Open        float64   `json:"open"`
	Close       float64   `json:"close"`
	Change      float64   `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Timestamp   time.Time `json:"timestamp"`
}

// AssetInfo represents information about a financial asset
type AssetInfo struct {
	Symbol        string           `json:"symbol"`
	Name          string           `json:"name"`
	AssetType     types.AssetType  `json:"asset_type"`
	Exchange      types.ExchangeType `json:"exchange"`
	Currency      string           `json:"currency"`
	IsActive      bool             `json:"is_active"`
	IsShariahCompliant bool        `json:"is_shariah_compliant"`
	MinOrderSize  float64          `json:"min_order_size"`
	MaxOrderSize  float64          `json:"max_order_size"`
	PriceStep     float64          `json:"price_step"`
}

// Asset represents a financial asset
type Asset struct {
	Symbol      string           `json:"symbol"`
	Name        string           `json:"name"`
	AssetType   types.AssetType  `json:"asset_type"`
	Exchange    types.ExchangeType `json:"exchange"`
	Currency    string           `json:"currency"`
	IsActive    bool             `json:"is_active"`
}

// HalalScreening represents Sharia compliance screening results
type HalalScreening struct {
	Symbol        string    `json:"symbol"`
	IsCompliant   bool      `json:"is_compliant"`
	Score         float64   `json:"score"`
	Violations    []string  `json:"violations"`
	Recommendations []string `json:"recommendations"`
	LastUpdated   time.Time `json:"last_updated"`
}

// OrderSide represents the side of an order (buy/sell)
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeStop   OrderType = "STOP"
)

// TimeInForce represents how long an order remains active
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancelled
	TimeInForceIOC TimeInForce = "IOC" // Immediate Or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill Or Kill
	TimeInForceDAY TimeInForce = "DAY" // Day order
)
