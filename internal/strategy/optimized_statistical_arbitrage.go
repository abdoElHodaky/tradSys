package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"github.com/abdoElHodaky/tradSys/internal/trading/order"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"
)

// OptimizedStatisticalArbitrageStrategy is an optimized implementation of a statistical arbitrage strategy
type OptimizedStatisticalArbitrageStrategy struct {
	name          string
	symbol1       string
	symbol2       string
	windowSize    int
	zThreshold    float64
	prices1       []float64
	prices2       []float64
	ratios        []float64
	positions     map[string]float64
	running       bool
	processingMu  sync.Mutex
	positionMu    sync.RWMutex
	logger        *zap.Logger
}

// NewOptimizedStatisticalArbitrageStrategy creates a new optimized statistical arbitrage strategy
func NewOptimizedStatisticalArbitrageStrategy(
	name string,
	symbol1 string,
	symbol2 string,
	windowSize int,
	zThreshold float64,
	logger *zap.Logger,
) *OptimizedStatisticalArbitrageStrategy {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &OptimizedStatisticalArbitrageStrategy{
		name:       name,
		symbol1:    symbol1,
		symbol2:    symbol2,
		windowSize: windowSize,
		zThreshold: zThreshold,
		prices1:    make([]float64, 0, windowSize),
		prices2:    make([]float64, 0, windowSize),
		ratios:     make([]float64, 0, windowSize),
		positions:  make(map[string]float64),
		logger:     logger,
	}
}

// GetName returns the name of the strategy
func (s *OptimizedStatisticalArbitrageStrategy) GetName() string {
	return s.name
}

// Initialize initializes the strategy
func (s *OptimizedStatisticalArbitrageStrategy) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing strategy",
		zap.String("name", s.name),
		zap.String("symbol1", s.symbol1),
		zap.String("symbol2", s.symbol2),
		zap.Int("windowSize", s.windowSize),
		zap.Float64("zThreshold", s.zThreshold),
	)

	s.running = true
	return nil
}

// Shutdown shuts down the strategy
func (s *OptimizedStatisticalArbitrageStrategy) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down strategy",
		zap.String("name", s.name),
	)

	s.running = false
	return nil
}

// IsRunning returns whether the strategy is running
func (s *OptimizedStatisticalArbitrageStrategy) IsRunning() bool {
	return s.running
}

// ProcessMarketData processes market data
func (s *OptimizedStatisticalArbitrageStrategy) ProcessMarketData(ctx context.Context, data *market_data.MarketData) error {
	if !s.IsRunning() {
		return nil
	}

	// Check if this data is for one of our symbols
	if data.Symbol != s.symbol1 && data.Symbol != s.symbol2 {
		return nil
	}

	// Use a mutex to ensure only one update is processed at a time
	// This prevents race conditions when updating the price series and statistics
	s.processingMu.Lock()
	defer s.processingMu.Unlock()

	// Update price series and statistics
	if data.Symbol == s.symbol1 {
		s.updatePriceSeries1(ctx, data.Price)
	} else if data.Symbol == s.symbol2 {
		s.updatePriceSeries2(ctx, data.Price)
	}

	// Check if we have enough data to calculate statistics
	if len(s.prices1) < s.windowSize || len(s.prices2) < s.windowSize {
		return nil
	}

	// Calculate the price ratio
	ratio := s.prices1[len(s.prices1)-1] / s.prices2[len(s.prices2)-1]
	s.ratios = append(s.ratios, ratio)
	if len(s.ratios) > s.windowSize {
		s.ratios = s.ratios[1:]
	}

	// Calculate z-score
	mean, std := stat.MeanStdDev(s.ratios, nil)
	if std == 0 {
		return nil
	}
	zScore := (ratio - mean) / std

	s.logger.Debug("Calculated z-score",
		zap.Float64("zScore", zScore),
		zap.Float64("mean", mean),
		zap.Float64("std", std),
		zap.Float64("ratio", ratio),
	)

	// Generate trading signals based on z-score
	if zScore > s.zThreshold {
		// Ratio is too high, sell symbol1 and buy symbol2
		s.generateSellSignal(ctx, s.symbol1, s.prices1[len(s.prices1)-1])
		s.generateBuySignal(ctx, s.symbol2, s.prices2[len(s.prices2)-1])
	} else if zScore < -s.zThreshold {
		// Ratio is too low, buy symbol1 and sell symbol2
		s.generateBuySignal(ctx, s.symbol1, s.prices1[len(s.prices1)-1])
		s.generateSellSignal(ctx, s.symbol2, s.prices2[len(s.prices2)-1])
	} else if zScore > -0.5 && zScore < 0.5 {
		// Ratio is close to mean, close positions
		s.closePositions(ctx)
	}

	return nil
}

// ProcessOrder processes an order
func (s *OptimizedStatisticalArbitrageStrategy) ProcessOrder(ctx context.Context, order *order.Order) error {
	if !s.IsRunning() {
		return nil
	}

	// Check if this order is for one of our symbols
	if order.Symbol != s.symbol1 && order.Symbol != s.symbol2 {
		return nil
	}

	// Update positions based on order
	s.positionMu.Lock()
	defer s.positionMu.Unlock()

	// Update position based on order side
	if order.Side == "buy" {
		s.positions[order.Symbol] += order.Size
	} else if order.Side == "sell" {
		s.positions[order.Symbol] -= order.Size
	}

	s.logger.Debug("Updated position",
		zap.String("symbol", order.Symbol),
		zap.Float64("position", s.positions[order.Symbol]),
		zap.String("orderId", order.ID),
		zap.String("side", order.Side),
		zap.Float64("size", order.Size),
	)

	return nil
}

// updatePriceSeries1 updates the price series for symbol1
func (s *OptimizedStatisticalArbitrageStrategy) updatePriceSeries1(ctx context.Context, price float64) {
	s.prices1 = append(s.prices1, price)
	if len(s.prices1) > s.windowSize {
		s.prices1 = s.prices1[1:]
	}
}

// updatePriceSeries2 updates the price series for symbol2
func (s *OptimizedStatisticalArbitrageStrategy) updatePriceSeries2(ctx context.Context, price float64) {
	s.prices2 = append(s.prices2, price)
	if len(s.prices2) > s.windowSize {
		s.prices2 = s.prices2[1:]
	}
}

// generateBuySignal generates a buy signal
func (s *OptimizedStatisticalArbitrageStrategy) generateBuySignal(ctx context.Context, symbol string, price float64) {
	s.positionMu.RLock()
	position := s.positions[symbol]
	s.positionMu.RUnlock()

	// Check if we already have a long position
	if position >= 1.0 {
		return
	}

	// Generate a buy signal
	s.logger.Info("Generating buy signal",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
	)

	// In a real implementation, this would submit an order to a trading system
}

// generateSellSignal generates a sell signal
func (s *OptimizedStatisticalArbitrageStrategy) generateSellSignal(ctx context.Context, symbol string, price float64) {
	s.positionMu.RLock()
	position := s.positions[symbol]
	s.positionMu.RUnlock()

	// Check if we already have a short position
	if position <= -1.0 {
		return
	}

	// Generate a sell signal
	s.logger.Info("Generating sell signal",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
	)

	// In a real implementation, this would submit an order to a trading system
}

// closePositions closes all positions
func (s *OptimizedStatisticalArbitrageStrategy) closePositions(ctx context.Context) {
	s.positionMu.RLock()
	position1 := s.positions[s.symbol1]
	position2 := s.positions[s.symbol2]
	s.positionMu.RUnlock()

	// Close positions if they exist
	if position1 != 0 {
		s.logger.Info("Closing position",
			zap.String("symbol", s.symbol1),
			zap.Float64("position", position1),
		)
		// In a real implementation, this would submit an order to a trading system
	}

	if position2 != 0 {
		s.logger.Info("Closing position",
			zap.String("symbol", s.symbol2),
			zap.Float64("position", position2),
		)
		// In a real implementation, this would submit an order to a trading system
	}
}

// GetStats gets the strategy statistics
func (s *OptimizedStatisticalArbitrageStrategy) GetStats() map[string]interface{} {
	s.processingMu.Lock()
	defer s.processingMu.Unlock()

	s.positionMu.RLock()
	positions := make(map[string]float64)
	for symbol, position := range s.positions {
		positions[symbol] = position
	}
	s.positionMu.RUnlock()

	stats := map[string]interface{}{
		"name":        s.name,
		"symbol1":     s.symbol1,
		"symbol2":     s.symbol2,
		"windowSize":  s.windowSize,
		"zThreshold":  s.zThreshold,
		"running":     s.running,
		"positions":   positions,
		"priceCount1": len(s.prices1),
		"priceCount2": len(s.prices2),
		"ratioCount":  len(s.ratios),
	}

	return stats
}

