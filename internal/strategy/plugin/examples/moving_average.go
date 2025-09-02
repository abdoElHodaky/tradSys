package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/abdoElHodaky/tradSys/internal/strategy/plugin"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// PluginInfo contains information about the plugin
var PluginInfo = &plugin.PluginInfo{
	Name:        "Moving Average Strategy",
	Version:     "1.0.0",
	Author:      "TradSys Team",
	Description: "A simple moving average crossover strategy",
	StrategyType: "moving_average_crossover",
}

// MovingAverageStrategy implements a moving average crossover strategy
type MovingAverageStrategy struct {
	config        strategy.StrategyConfig
	logger        *zap.Logger
	shortPeriod   int
	longPeriod    int
	shortMA       map[string]float64
	longMA        map[string]float64
	prices        map[string][]float64
	lastSignal    map[string]string
	initialized   bool
}

// CreateStrategy creates a new moving average strategy
func CreateStrategy(config strategy.StrategyConfig, logger *zap.Logger) (strategy.Strategy, error) {
	// Get strategy-specific configuration
	shortPeriod, ok := config.Params["short_period"].(int)
	if !ok {
		shortPeriod = 10 // Default value
	}
	
	longPeriod, ok := config.Params["long_period"].(int)
	if !ok {
		longPeriod = 30 // Default value
	}
	
	return &MovingAverageStrategy{
		config:      config,
		logger:      logger,
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
		shortMA:     make(map[string]float64),
		longMA:      make(map[string]float64),
		prices:      make(map[string][]float64),
		lastSignal:  make(map[string]string),
		initialized: false,
	}, nil
}

// Initialize initializes the strategy
func (s *MovingAverageStrategy) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing moving average strategy",
		zap.Int("short_period", s.shortPeriod),
		zap.Int("long_period", s.longPeriod),
		zap.Strings("symbols", s.config.Symbols),
	)
	
	// Initialize price history for each symbol
	for _, symbol := range s.config.Symbols {
		s.prices[symbol] = make([]float64, 0, s.longPeriod*2)
		s.lastSignal[symbol] = "NONE"
	}
	
	s.initialized = true
	return nil
}

// ProcessMarketData processes market data updates
func (s *MovingAverageStrategy) ProcessMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
	if !s.initialized {
		return fmt.Errorf("strategy not initialized")
	}
	
	symbol := data.Symbol
	price := data.Price
	
	// Update price history
	prices, ok := s.prices[symbol]
	if !ok {
		return fmt.Errorf("symbol not found: %s", symbol)
	}
	
	// Add the new price
	prices = append(prices, price)
	
	// Keep only the necessary history
	if len(prices) > s.longPeriod*2 {
		prices = prices[len(prices)-s.longPeriod*2:]
	}
	
	s.prices[symbol] = prices
	
	// Calculate moving averages if we have enough data
	if len(prices) >= s.longPeriod {
		// Calculate short MA
		shortMA := 0.0
		for i := len(prices) - s.shortPeriod; i < len(prices); i++ {
			shortMA += prices[i]
		}
		shortMA /= float64(s.shortPeriod)
		s.shortMA[symbol] = shortMA
		
		// Calculate long MA
		longMA := 0.0
		for i := len(prices) - s.longPeriod; i < len(prices); i++ {
			longMA += prices[i]
		}
		longMA /= float64(s.longPeriod)
		s.longMA[symbol] = longMA
		
		s.logger.Debug("Calculated moving averages",
			zap.String("symbol", symbol),
			zap.Float64("price", price),
			zap.Float64("short_ma", shortMA),
			zap.Float64("long_ma", longMA),
		)
	}
	
	return nil
}

// GenerateSignals generates trading signals
func (s *MovingAverageStrategy) GenerateSignals(ctx context.Context) ([]*strategy.Signal, error) {
	if !s.initialized {
		return nil, fmt.Errorf("strategy not initialized")
	}
	
	signals := make([]*strategy.Signal, 0)
	
	for _, symbol := range s.config.Symbols {
		shortMA, shortOK := s.shortMA[symbol]
		longMA, longOK := s.longMA[symbol]
		
		if !shortOK || !longOK {
			continue // Not enough data yet
		}
		
		var signal string
		
		// Generate signal based on moving average crossover
		if shortMA > longMA {
			// Short MA above Long MA - bullish
			signal = "BUY"
		} else if shortMA < longMA {
			// Short MA below Long MA - bearish
			signal = "SELL"
		} else {
			// No clear signal
			signal = "NONE"
		}
		
		// Only generate a new signal if it's different from the last one
		if signal != s.lastSignal[symbol] && signal != "NONE" {
			s.logger.Info("Generated signal",
				zap.String("symbol", symbol),
				zap.String("signal", signal),
				zap.Float64("short_ma", shortMA),
				zap.Float64("long_ma", longMA),
			)
			
			signals = append(signals, &strategy.Signal{
				Symbol:    symbol,
				Direction: signal,
				Strength:  1.0,
				Reason:    fmt.Sprintf("Moving average crossover: short(%.2f) %s long(%.2f)", shortMA, getComparisonSymbol(signal), longMA),
				Timestamp: time.Now().Unix(),
			})
			
			s.lastSignal[symbol] = signal
		}
	}
	
	return signals, nil
}
// getComparisonSymbol returns the comparison symbol based on signal direction
func getComparisonSymbol(signal string) string {
	if signal == "BUY" {
		return ">"
	}
	return "<"
}

// GenerateOrders generates orders based on signals
func (s *MovingAverageStrategy) GenerateOrders(ctx context.Context, signals []*strategy.Signal) ([]*orders.OrderRequest, error) {
	if !s.initialized {
		return nil, fmt.Errorf("strategy not initialized")
	}
	
	orders := make([]*orders.OrderRequest, 0, len(signals))
	
	for _, signal := range signals {
		// Get the current price
		prices, ok := s.prices[signal.Symbol]
		if !ok || len(prices) == 0 {
			continue
		}
		
		currentPrice := prices[len(prices)-1]
		
		// Create an order based on the signal
		order := &orders.OrderRequest{
			Symbol:    signal.Symbol,
			Side:      signal.Direction,
			Type:      "MARKET",
			Quantity:  1.0, // Fixed quantity for simplicity
			Price:     currentPrice,
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"strategy": "moving_average_crossover",
				"reason":   signal.Reason,
			},
		}
		
		orders = append(orders, order)
		
		s.logger.Info("Generated order",
			zap.String("symbol", order.Symbol),
			zap.String("side", order.Side),
			zap.Float64("price", order.Price),
			zap.Float64("quantity", order.Quantity),
		)
	}
	
	return orders, nil
}

// Name returns the name of the strategy
func (s *MovingAverageStrategy) Name() string {
	return "MovingAverageCrossover"
}

// Description returns the description of the strategy
func (s *MovingAverageStrategy) Description() string {
	return fmt.Sprintf("Moving average crossover strategy (short: %d, long: %d)", s.shortPeriod, s.longPeriod)
}

// Type returns the type of the strategy
func (s *MovingAverageStrategy) Type() string {
	return "moving_average_crossover"
}

// Cleanup cleans up the strategy
func (s *MovingAverageStrategy) Cleanup() error {
	s.logger.Info("Cleaning up moving average strategy")
	s.initialized = false
	return nil
}
