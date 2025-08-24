package connectors

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
)

// ExchangeConfig contains configuration for an exchange connector
type ExchangeConfig struct {
	// APIKey is the API key for the exchange
	APIKey string
	
	// APISecret is the API secret for the exchange
	APISecret string
	
	// BaseURL is the base URL for the exchange API
	BaseURL string
	
	// WebSocketURL is the WebSocket URL for the exchange
	WebSocketURL string
	
	// Timeout is the timeout for API requests
	Timeout time.Duration
	
	// RateLimit is the rate limit for API requests
	RateLimit int
	
	// Testnet indicates whether to use the testnet
	Testnet bool
}

// ExchangeConnector defines the interface for an exchange connector
type ExchangeConnector interface {
	// Initialize initializes the exchange connector
	Initialize(ctx context.Context) error
	
	// GetName returns the name of the exchange
	GetName() string
	
	// GetMarketData gets market data for a symbol
	GetMarketData(ctx context.Context, symbol string) (*marketdata.MarketDataResponse, error)
	
	// SubscribeMarketData subscribes to market data for a symbol
	SubscribeMarketData(ctx context.Context, symbol string, callback func(*marketdata.MarketDataResponse)) error
	
	// UnsubscribeMarketData unsubscribes from market data for a symbol
	UnsubscribeMarketData(ctx context.Context, symbol string) error
	
	// PlaceOrder places an order
	PlaceOrder(ctx context.Context, order *orders.OrderRequest) (*orders.OrderResponse, error)
	
	// CancelOrder cancels an order
	CancelOrder(ctx context.Context, orderID string) error
	
	// GetOrder gets an order
	GetOrder(ctx context.Context, orderID string) (*orders.OrderResponse, error)
	
	// GetOpenOrders gets open orders
	GetOpenOrders(ctx context.Context, symbol string) ([]*orders.OrderResponse, error)
	
	// GetAccountInfo gets account information
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
	
	// Close closes the exchange connector
	Close() error
}

// AccountInfo contains account information
type AccountInfo struct {
	// Balances is a map of asset to balance
	Balances map[string]Balance
	
	// TotalEquity is the total equity
	TotalEquity float64
	
	// AvailableEquity is the available equity
	AvailableEquity float64
	
	// Margin is the margin used
	Margin float64
	
	// UnrealizedPnL is the unrealized P&L
	UnrealizedPnL float64
}

// Balance contains balance information
type Balance struct {
	// Free is the free balance
	Free float64
	
	// Locked is the locked balance
	Locked float64
	
	// Total is the total balance
	Total float64
}

