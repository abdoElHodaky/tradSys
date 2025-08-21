package strategy

import (
	"context"

	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
)

// Strategy defines the interface for all trading strategies
type Strategy interface {
	// Name returns the name of the strategy
	Name() string

	// Initialize initializes the strategy
	Initialize(ctx context.Context) error

	// Start starts the strategy
	Start(ctx context.Context) error

	// Stop stops the strategy
	Stop(ctx context.Context) error

	// IsRunning returns whether the strategy is running
	IsRunning() bool

	// OnMarketData processes market data updates
	OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error

	// OnOrderUpdate processes order updates
	OnOrderUpdate(ctx context.Context, order *orders.OrderResponse) error

	// GetPerformanceMetrics returns performance metrics for the strategy
	GetPerformanceMetrics() map[string]interface{}
}

// StrategyConfig defines the configuration for a strategy
type StrategyConfig struct {
	// Name is the name of the strategy
	Name string `json:"name"`

	// Type is the type of the strategy
	Type string `json:"type"`

	// Symbols are the trading symbols for the strategy
	Symbols []string `json:"symbols"`

	// Parameters are the strategy parameters
	Parameters map[string]interface{} `json:"parameters"`

	// RiskLimits are the risk limits for the strategy
	RiskLimits map[string]interface{} `json:"risk_limits"`
}

// StrategyFactory defines the interface for creating strategies
type StrategyFactory interface {
	// CreateStrategy creates a strategy from a configuration
	CreateStrategy(config StrategyConfig) (Strategy, error)

	// GetAvailableStrategyTypes returns the available strategy types
	GetAvailableStrategyTypes() []string
}

