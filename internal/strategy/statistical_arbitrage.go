package strategy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/models"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"go.uber.org/zap"
)

// StatisticalArbitrageStrategy implements a statistical arbitrage trading strategy
type StatisticalArbitrageStrategy struct {
	BaseStrategy
	params StatisticalArbitrageParams
	
	// Pair symbols
	symbol1 string
	symbol2 string
	
	// Price history
	priceHistory1 []float64
	priceHistory2 []float64
	
	// Spread history
	spreadHistory []float64
	
	// Statistics
	spreadMean   float64
	spreadStdDev float64
	
	// Position tracking
	position struct {
		InPosition bool
		Quantity1  float64
		Quantity2  float64
		EntrySpread float64
		EntryTime   time.Time
	}
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
}

// StatisticalArbitrageParams contains parameters for the statistical arbitrage strategy
type StatisticalArbitrageParams struct {
	// Pair symbols
	Symbol1 string
	Symbol2 string
	
	// Entry/exit thresholds in standard deviations
	EntryThreshold float64
	ExitThreshold  float64
	
	// Position sizing
	MaxPosition float64
	
	// Risk management
	StopLoss       float64
	TakeProfit     float64
	MaxHoldingTime time.Duration
	
	// Lookback period for calculating statistics
	LookbackPeriod int
	
	// Minimum number of data points required before trading
	MinDataPoints int
	
	// Rebalancing frequency
	RebalanceInterval time.Duration
	
	// Execution parameters
	ExecutionDelay time.Duration
}

// NewStatisticalArbitrageStrategy creates a new statistical arbitrage strategy
func NewStatisticalArbitrageStrategy(params StatisticalArbitrageParams, logger *zap.Logger) *StatisticalArbitrageStrategy {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	return &StatisticalArbitrageStrategy{
		BaseStrategy: BaseStrategy{
			name:        "StatisticalArbitrage",
			description: "Statistical arbitrage strategy for trading correlated pairs",
			symbols:     map[string]bool{params.Symbol1: true, params.Symbol2: true},
			active:      false,
			logger:      logger,
		},
		params:         params,
		symbol1:        params.Symbol1,
		symbol2:        params.Symbol2,
		priceHistory1:  make([]float64, 0, params.LookbackPeriod),
		priceHistory2:  make([]float64, 0, params.LookbackPeriod),
		spreadHistory:  make([]float64, 0, params.LookbackPeriod),
		logger:         logger,
	}
}

// Initialize prepares the strategy for trading
func (s *StatisticalArbitrageStrategy) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing statistical arbitrage strategy",
		zap.String("symbol1", s.symbol1),
		zap.String("symbol2", s.symbol2),
		zap.Float64("entryThreshold", s.params.EntryThreshold),
		zap.Float64("exitThreshold", s.params.ExitThreshold))
	
	// Initialize price histories
	s.mu.Lock()
	s.priceHistory1 = make([]float64, 0, s.params.LookbackPeriod)
	s.priceHistory2 = make([]float64, 0, s.params.LookbackPeriod)
	s.spreadHistory = make([]float64, 0, s.params.LookbackPeriod)
	s.mu.Unlock()
	
	s.active = true
	return nil
}

// OnMarketData processes new market data
func (s *StatisticalArbitrageStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
	if !s.active {
		return nil
	}
	
	symbol := data.Symbol
	price := data.LastPrice
	
	// Update price history for the relevant symbol
	s.mu.Lock()
	defer s.mu.Unlock()
	
	switch symbol {
	case s.symbol1:
		s.updatePriceHistory1(price)
	case s.symbol2:
		s.updatePriceHistory2(price)
	default:
		// Ignore data for other symbols
		return nil
	}
	
	// Check if we have enough data for both symbols
	if len(s.priceHistory1) == 0 || len(s.priceHistory2) == 0 {
		return nil
	}
	
	// Calculate current spread
	currentSpread := s.calculateSpread(s.priceHistory1[len(s.priceHistory1)-1], s.priceHistory2[len(s.priceHistory2)-1])
	
	// Update spread history
	s.updateSpreadHistory(currentSpread)
	
	// Check if we have enough data points to calculate statistics
	if len(s.spreadHistory) < s.params.MinDataPoints {
		return nil
	}
	
	// Calculate spread statistics
	s.calculateSpreadStatistics()
	
	// Check for trading signals
	if s.position.InPosition {
		s.checkExitSignal(ctx, currentSpread)
	} else {
		s.checkEntrySignal(ctx, currentSpread)
	}
	
	return nil
}

// updatePriceHistory1 updates the price history for symbol1
func (s *StatisticalArbitrageStrategy) updatePriceHistory1(price float64) {
	s.priceHistory1 = append(s.priceHistory1, price)
	if len(s.priceHistory1) > s.params.LookbackPeriod {
		s.priceHistory1 = s.priceHistory1[1:]
	}
}

// updatePriceHistory2 updates the price history for symbol2
func (s *StatisticalArbitrageStrategy) updatePriceHistory2(price float64) {
	s.priceHistory2 = append(s.priceHistory2, price)
	if len(s.priceHistory2) > s.params.LookbackPeriod {
		s.priceHistory2 = s.priceHistory2[1:]
	}
}

// updateSpreadHistory updates the spread history
func (s *StatisticalArbitrageStrategy) updateSpreadHistory(spread float64) {
	s.spreadHistory = append(s.spreadHistory, spread)
	if len(s.spreadHistory) > s.params.LookbackPeriod {
		s.spreadHistory = s.spreadHistory[1:]
	}
}

// calculateSpread calculates the spread between two prices
func (s *StatisticalArbitrageStrategy) calculateSpread(price1, price2 float64) float64 {
	return price1 - price2
}

// calculateSpreadStatistics calculates the mean and standard deviation of the spread
func (s *StatisticalArbitrageStrategy) calculateSpreadStatistics() {
	var sum float64
	for _, spread := range s.spreadHistory {
		sum += spread
	}
	s.spreadMean = sum / float64(len(s.spreadHistory))
	
	var sumSquaredDiff float64
	for _, spread := range s.spreadHistory {
		diff := spread - s.spreadMean
		sumSquaredDiff += diff * diff
	}
	s.spreadStdDev = math.Sqrt(sumSquaredDiff / float64(len(s.spreadHistory)))
}

// checkEntrySignal checks for entry signals
func (s *StatisticalArbitrageStrategy) checkEntrySignal(ctx context.Context, currentSpread float64) {
	// Calculate z-score
	zScore := (currentSpread - s.spreadMean) / s.spreadStdDev
	
	// Check for entry conditions
	if zScore > s.params.EntryThreshold {
		// Spread is too high, expect it to decrease
		// Short symbol1, long symbol2
		s.enterPosition(ctx, -s.params.MaxPosition, s.params.MaxPosition, currentSpread)
	} else if zScore < -s.params.EntryThreshold {
		// Spread is too low, expect it to increase
		// Long symbol1, short symbol2
		s.enterPosition(ctx, s.params.MaxPosition, -s.params.MaxPosition, currentSpread)
	}
}

// checkExitSignal checks for exit signals
func (s *StatisticalArbitrageStrategy) checkExitSignal(ctx context.Context, currentSpread float64) {
	// Calculate z-score
	zScore := (currentSpread - s.spreadMean) / s.spreadStdDev
	
	// Check for exit conditions
	if s.position.Quantity1 > 0 {
		// We are long symbol1, short symbol2
		// Exit when spread has increased enough (z-score close to 0 or positive)
		if zScore > -s.params.ExitThreshold {
			s.exitPosition(ctx)
		}
	} else if s.position.Quantity1 < 0 {
		// We are short symbol1, long symbol2
		// Exit when spread has decreased enough (z-score close to 0 or negative)
		if zScore < s.params.ExitThreshold {
			s.exitPosition(ctx)
		}
	}
	
	// Check stop loss
	spreadChange := currentSpread - s.position.EntrySpread
	if (s.position.Quantity1 > 0 && spreadChange < -s.params.StopLoss) ||
		(s.position.Quantity1 < 0 && spreadChange > s.params.StopLoss) {
		s.logger.Info("Exiting position due to stop loss",
			zap.Float64("entrySpread", s.position.EntrySpread),
			zap.Float64("currentSpread", currentSpread),
			zap.Float64("spreadChange", spreadChange),
			zap.Float64("stopLoss", s.params.StopLoss))
		s.exitPosition(ctx)
	}
	
	// Check take profit
	if (s.position.Quantity1 > 0 && spreadChange > s.params.TakeProfit) ||
		(s.position.Quantity1 < 0 && spreadChange < -s.params.TakeProfit) {
		s.logger.Info("Exiting position due to take profit",
			zap.Float64("entrySpread", s.position.EntrySpread),
			zap.Float64("currentSpread", currentSpread),
			zap.Float64("spreadChange", spreadChange),
			zap.Float64("takeProfit", s.params.TakeProfit))
		s.exitPosition(ctx)
	}
	
	// Check max holding time
	if time.Since(s.position.EntryTime) > s.params.MaxHoldingTime {
		s.logger.Info("Exiting position due to max holding time",
			zap.Time("entryTime", s.position.EntryTime),
			zap.Duration("holdingTime", time.Since(s.position.EntryTime)),
			zap.Duration("maxHoldingTime", s.params.MaxHoldingTime))
		s.exitPosition(ctx)
	}
}

// enterPosition enters a new position
func (s *StatisticalArbitrageStrategy) enterPosition(ctx context.Context, quantity1, quantity2, entrySpread float64) {
	s.logger.Info("Entering position",
		zap.Float64("quantity1", quantity1),
		zap.Float64("quantity2", quantity2),
		zap.Float64("entrySpread", entrySpread))
	
	// Create orders for both symbols
	if quantity1 > 0 {
		// Buy order for symbol1
		buyOrder := &models.Order{
			Symbol:   s.symbol1,
			Side:     "buy",
			Type:     "market",
			Quantity: quantity1,
		}
		
		// Sell order for symbol2
		sellOrder := &models.Order{
			Symbol:   s.symbol2,
			Side:     "sell",
			Type:     "market",
			Quantity: math.Abs(quantity2),
		}
		
		// Submit orders
		// Note: In a real implementation, these would be submitted to the broker
		// s.SubmitOrder(ctx, buyOrder)
		// s.SubmitOrder(ctx, sellOrder)
	} else {
		// Sell order for symbol1
		sellOrder := &models.Order{
			Symbol:   s.symbol1,
			Side:     "sell",
			Type:     "market",
			Quantity: math.Abs(quantity1),
		}
		
		// Buy order for symbol2
		buyOrder := &models.Order{
			Symbol:   s.symbol2,
			Side:     "buy",
			Type:     "market",
			Quantity: quantity2,
		}
		
		// Submit orders
		// Note: In a real implementation, these would be submitted to the broker
		// s.SubmitOrder(ctx, sellOrder)
		// s.SubmitOrder(ctx, buyOrder)
	}
	
	// Update position state
	s.position.InPosition = true
	s.position.Quantity1 = quantity1
	s.position.Quantity2 = quantity2
	s.position.EntrySpread = entrySpread
	s.position.EntryTime = time.Now()
}

// exitPosition exits the current position
func (s *StatisticalArbitrageStrategy) exitPosition(ctx context.Context) {
	s.logger.Info("Exiting position",
		zap.Float64("quantity1", s.position.Quantity1),
		zap.Float64("quantity2", s.position.Quantity2))
	
	// Create orders to close positions
	var order1, order2 *models.Order
	
	if s.position.Quantity1 > 0 {
		// Sell order to close long position in symbol1
		order1 = &models.Order{
			Symbol:   s.symbol1,
			Side:     "sell",
			Type:     "market",
			Quantity: math.Abs(s.position.Quantity1),
		}
	} else {
		// Buy order to close short position in symbol1
		order1 = &models.Order{
			Symbol:   s.symbol1,
			Side:     "buy",
			Type:     "market",
			Quantity: math.Abs(s.position.Quantity1),
		}
	}
	
	if s.position.Quantity2 > 0 {
		// Sell order to close long position in symbol2
		order2 = &models.Order{
			Symbol:   s.symbol2,
			Side:     "sell",
			Type:     "market",
			Quantity: math.Abs(s.position.Quantity2),
		}
	} else {
		// Buy order to close short position in symbol2
		order2 = &models.Order{
			Symbol:   s.symbol2,
			Side:     "buy",
			Type:     "market",
			Quantity: math.Abs(s.position.Quantity2),
		}
	}
	
	// Submit orders
	// Note: In a real implementation, these would be submitted to the broker
	// s.SubmitOrder(ctx, order1)
	// s.SubmitOrder(ctx, order2)
	
	// Reset position state
	s.position.InPosition = false
	s.position.Quantity1 = 0
	s.position.Quantity2 = 0
}

// Shutdown cleans up resources
func (s *StatisticalArbitrageStrategy) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down statistical arbitrage strategy")
	
	// Exit any open positions
	if s.position.InPosition {
		s.exitPosition(ctx)
	}
	
	s.active = false
	return nil
}

