package main

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/abdoElHodaky/tradSys/internal/strategy/plugin"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"github.com/abdoElHodaky/tradSys/internal/trading/orders"
	"go.uber.org/zap"
)

// PluginInfo is the exported plugin information
var PluginInfo = &plugin.PluginInfo{
	Name:        "ExampleStrategy",
	Version:     "1.0.0",
	Author:      "TradSys Team",
	Description: "An example strategy plugin",
	StrategyType: "example-strategy",
}

// CreateStrategy is the exported function to create a strategy
func CreateStrategy(config strategy.StrategyConfig, logger *zap.Logger) (strategy.Strategy, error) {
	return NewExampleStrategy(config, logger)
}

// ExampleStrategy implements the Strategy interface
type ExampleStrategy struct {
	name      string
	symbols   []string
	logger    *zap.Logger
	running   bool
	params    map[string]interface{}
	metrics   map[string]interface{}
}

// NewExampleStrategy creates a new example strategy
func NewExampleStrategy(config strategy.StrategyConfig, logger *zap.Logger) (*ExampleStrategy, error) {
	// Validate configuration
	if len(config.Symbols) == 0 {
		return nil, fmt.Errorf("no symbols specified")
	}

	// Create the strategy
	s := &ExampleStrategy{
		name:    config.Name,
		symbols: config.Symbols,
		logger:  logger,
		params:  config.Parameters,
		metrics: make(map[string]interface{}),
	}

	// Initialize metrics
	s.metrics["trades"] = 0
	s.metrics["profit"] = 0.0

	return s, nil
}

// Name returns the name of the strategy
func (s *ExampleStrategy) Name() string {
	return s.name
}

// Initialize initializes the strategy
func (s *ExampleStrategy) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing example strategy",
		zap.String("name", s.name),
		zap.Strings("symbols", s.symbols))
	
	// Perform any initialization here
	
	return nil
}

// Start starts the strategy
func (s *ExampleStrategy) Start(ctx context.Context) error {
	s.logger.Info("Starting example strategy", zap.String("name", s.name))
	s.running = true
	
	// Start any background processes here
	
	return nil
}

// Stop stops the strategy
func (s *ExampleStrategy) Stop(ctx context.Context) error {
	s.logger.Info("Stopping example strategy", zap.String("name", s.name))
	s.running = false
	
	// Stop any background processes here
	
	return nil
}

// IsRunning returns whether the strategy is running
func (s *ExampleStrategy) IsRunning() bool {
	return s.running
}

// OnMarketData processes market data updates
func (s *ExampleStrategy) OnMarketData(ctx context.Context, data *market_data.MarketDataResponse) error {
	// Process market data
	s.logger.Debug("Received market data",
		zap.String("symbol", data.Symbol),
		zap.Float64("price", data.Price),
		zap.Time("timestamp", data.Timestamp))
	
	// Implement strategy logic here
	
	return nil
}

// OnOrderUpdate processes order updates
func (s *ExampleStrategy) OnOrderUpdate(ctx context.Context, order *orders.OrderResponse) error {
	// Process order update
	s.logger.Debug("Received order update",
		zap.String("order_id", order.OrderID),
		zap.String("status", order.Status))
	
	// Update metrics
	if order.Status == "filled" {
		s.metrics["trades"] = s.metrics["trades"].(int) + 1
		s.metrics["profit"] = s.metrics["profit"].(float64) + order.PnL
	}
	
	return nil
}

// GetPerformanceMetrics returns performance metrics for the strategy
func (s *ExampleStrategy) GetPerformanceMetrics() map[string]interface{} {
	// Add timestamp to metrics
	s.metrics["last_updated"] = time.Now()
	
	return s.metrics
}

