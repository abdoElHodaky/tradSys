package strategy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"go.uber.org/zap"
)

// MeanReversionStrategy implements a mean reversion trading strategy
type MeanReversionStrategy struct {
	BaseStrategy
	lookbackPeriod    int
	deviationThreshold float64
	bollingerPeriod   int
	bollingerStdDev   float64
	positionSize      float64
	stopLossPercent   float64
	takeProfitPercent float64
	prices            map[string][]float64
	means             map[string]float64
	stdDevs           map[string]float64
	mutex             sync.RWMutex
}

// MeanReversionStrategyParams contains parameters for the mean reversion strategy
type MeanReversionStrategyParams struct {
	LookbackPeriod     int
	DeviationThreshold float64
	BollingerPeriod    int
	BollingerStdDev    float64
	PositionSize       float64
	StopLossPercent    float64
	TakeProfitPercent  float64
}

// NewMeanReversionStrategy creates a new mean reversion strategy
func NewMeanReversionStrategy(logger *zap.Logger, params MeanReversionStrategyParams) *MeanReversionStrategy {
	// Set default values if not provided
	if params.LookbackPeriod == 0 {
		params.LookbackPeriod = 20
	}
	if params.DeviationThreshold == 0 {
		params.DeviationThreshold = 2.0
	}
	if params.BollingerPeriod == 0 {
		params.BollingerPeriod = 20
	}
	if params.BollingerStdDev == 0 {
		params.BollingerStdDev = 2.0
	}
	if params.PositionSize == 0 {
		params.PositionSize = 0.1
	}
	if params.StopLossPercent == 0 {
		params.StopLossPercent = 0.02
	}
	if params.TakeProfitPercent == 0 {
		params.TakeProfitPercent = 0.03
	}

	return &MeanReversionStrategy{
		BaseStrategy: BaseStrategy{
			logger:      logger,
			name:        "MeanReversion",
			description: "Mean reversion trading strategy using Bollinger Bands",
			symbols:     make(map[string]bool),
			active:      false,
		},
		lookbackPeriod:     params.LookbackPeriod,
		deviationThreshold: params.DeviationThreshold,
		bollingerPeriod:    params.BollingerPeriod,
		bollingerStdDev:    params.BollingerStdDev,
		positionSize:       params.PositionSize,
		stopLossPercent:    params.StopLossPercent,
		takeProfitPercent:  params.TakeProfitPercent,
		prices:             make(map[string][]float64),
		means:              make(map[string]float64),
		stdDevs:            make(map[string]float64),
	}
}

// Initialize initializes the strategy
func (s *MeanReversionStrategy) Initialize(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("Initializing mean reversion strategy",
		zap.Int("lookback_period", s.lookbackPeriod),
		zap.Float64("deviation_threshold", s.deviationThreshold),
		zap.Int("bollinger_period", s.bollingerPeriod),
		zap.Float64("bollinger_std_dev", s.bollingerStdDev),
		zap.Float64("position_size", s.positionSize),
		zap.Float64("stop_loss_percent", s.stopLossPercent),
		zap.Float64("take_profit_percent", s.takeProfitPercent),
	)

	// Initialize price arrays for each symbol
	for symbol := range s.symbols {
		s.prices[symbol] = make([]float64, 0, s.lookbackPeriod*2)
		s.means[symbol] = 0.0
		s.stdDevs[symbol] = 0.0
	}

	s.active = true
	return nil
}

// ProcessTick processes a market data tick
func (s *MeanReversionStrategy) ProcessTick(ctx context.Context, tick *models.MarketDataTick) error {
	if !s.active {
		return fmt.Errorf("strategy not active")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	symbol := tick.Symbol
	price := tick.Price

	// Check if we're tracking this symbol
	if _, ok := s.symbols[symbol]; !ok {
		return nil
	}

	// Add price to history
	s.prices[symbol] = append(s.prices[symbol], price)

	// Trim array if it exceeds twice the lookback period
	if len(s.prices[symbol]) > s.lookbackPeriod*2 {
		s.prices[symbol] = s.prices[symbol][len(s.prices[symbol])-s.lookbackPeriod*2:]
	}

	// Calculate Bollinger Bands if we have enough data
	if len(s.prices[symbol]) >= s.bollingerPeriod {
		s.calculateBollingerBands(symbol)

		// Get current price and Bollinger Bands
		currentPrice := s.prices[symbol][len(s.prices[symbol])-1]
		mean := s.means[symbol]
		stdDev := s.stdDevs[symbol]
		upperBand := mean + s.bollingerStdDev*stdDev
		lowerBand := mean - s.bollingerStdDev*stdDev

		// Calculate z-score (number of standard deviations from mean)
		zScore := (currentPrice - mean) / stdDev

		s.logger.Debug("Mean reversion calculation",
			zap.String("symbol", symbol),
			zap.Float64("price", currentPrice),
			zap.Float64("mean", mean),
			zap.Float64("std_dev", stdDev),
			zap.Float64("upper_band", upperBand),
			zap.Float64("lower_band", lowerBand),
			zap.Float64("z_score", zScore),
		)

		// Generate buy signal if price is below lower Bollinger Band
		if zScore < -s.deviationThreshold {
			s.generateBuySignal(ctx, symbol, price, zScore)
		}

		// Generate sell signal if price is above upper Bollinger Band
		if zScore > s.deviationThreshold {
			s.generateSellSignal(ctx, symbol, price, zScore)
		}
	}

	return nil
}

// calculateBollingerBands calculates the Bollinger Bands for a symbol
func (s *MeanReversionStrategy) calculateBollingerBands(symbol string) {
	// Get the most recent prices
	prices := s.prices[symbol]
	if len(prices) < s.bollingerPeriod {
		return
	}

	recentPrices := prices[len(prices)-s.bollingerPeriod:]

	// Calculate mean
	var sum float64
	for _, price := range recentPrices {
		sum += price
	}
	mean := sum / float64(s.bollingerPeriod)

	// Calculate standard deviation
	var variance float64
	for _, price := range recentPrices {
		variance += math.Pow(price-mean, 2)
	}
	variance /= float64(s.bollingerPeriod)
	stdDev := math.Sqrt(variance)

	// Update mean and standard deviation
	s.means[symbol] = mean
	s.stdDevs[symbol] = stdDev
}

// generateBuySignal generates a buy signal
func (s *MeanReversionStrategy) generateBuySignal(ctx context.Context, symbol string, price float64, zScore float64) {
	// Check if we already have a position
	position, err := s.GetPosition(ctx, symbol)
	if err == nil && position.Size > 0 {
		// Already long, do nothing
		return
	}

	// Calculate position size based on z-score
	adjustedPositionSize := s.positionSize
	if math.Abs(zScore) > s.deviationThreshold*1.5 {
		// Increase position size for stronger signals
		adjustedPositionSize *= 1.5
	}

	// Calculate stop loss and take profit levels
	stopLoss := price * (1 - s.stopLossPercent)
	takeProfit := price * (1 + s.takeProfitPercent)

	// Create order
	order := &models.Order{
		Symbol:     symbol,
//<<<<<<< codegen-bot/fix-order-model-syntax
		Side:       models.OrderSideBuy,
		Type:       models.OrderTypeLimit,
//=======
//<<<<<<< codegen-bot/pairs-management-implementation
		Side:       models.OrderSideBuy,
		Type:       models.OrderTypeLimit,
//=======
		Side:       "buy",
		OrderType:  "limit",
//>>>>>>> main
//>>>>>>> main
		Quantity:   adjustedPositionSize,
		Price:      price,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		Strategy:   s.name,
		Timestamp:  time.Now(),
	}

	// Submit order
	if err := s.SubmitOrder(ctx, order); err != nil {
		s.logger.Error("Failed to submit buy order",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.Float64("price", price),
			zap.Float64("z_score", zScore),
		)
		return
	}

	s.logger.Info("Generated buy signal",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
		zap.Float64("z_score", zScore),
		zap.Float64("position_size", adjustedPositionSize),
		zap.Float64("stop_loss", stopLoss),
		zap.Float64("take_profit", takeProfit),
	)
}

// generateSellSignal generates a sell signal
func (s *MeanReversionStrategy) generateSellSignal(ctx context.Context, symbol string, price float64, zScore float64) {
	// Check if we already have a position
	position, err := s.GetPosition(ctx, symbol)
	if err == nil && position.Size < 0 {
		// Already short, do nothing
		return
	}

	// Calculate position size based on z-score
	adjustedPositionSize := s.positionSize
	if math.Abs(zScore) > s.deviationThreshold*1.5 {
		// Increase position size for stronger signals
		adjustedPositionSize *= 1.5
	}

	// Calculate stop loss and take profit levels
	stopLoss := price * (1 + s.stopLossPercent)
	takeProfit := price * (1 - s.takeProfitPercent)

	// Create order
	order := &models.Order{
		Symbol:     symbol,
//<<<<<<< codegen-bot/fix-order-model-syntax
		Side:       models.OrderSideSell,
		Type:       models.OrderTypeLimit,
//=======
//<<<<<<< codegen-bot/pairs-management-implementation
		Side:       models.OrderSideSell,
		Type:       models.OrderTypeLimit,
//=======
		Side:       "sell",
		OrderType:  "limit",
//>>>>>>> main
//>>>>>>> main
		Quantity:   adjustedPositionSize,
		Price:      price,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		Strategy:   s.name,
		Timestamp:  time.Now(),
	}

	// Submit order
	if err := s.SubmitOrder(ctx, order); err != nil {
		s.logger.Error("Failed to submit sell order",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.Float64("price", price),
			zap.Float64("z_score", zScore),
		)
		return
	}

	s.logger.Info("Generated sell signal",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
		zap.Float64("z_score", zScore),
		zap.Float64("position_size", adjustedPositionSize),
		zap.Float64("stop_loss", stopLoss),
		zap.Float64("take_profit", takeProfit),
	)
}

// Stop stops the strategy
func (s *MeanReversionStrategy) Stop(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.active = false
	s.logger.Info("Mean reversion strategy stopped")
	return nil
}

// GetParameters returns the strategy parameters
func (s *MeanReversionStrategy) GetParameters() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return map[string]interface{}{
		"lookback_period":     s.lookbackPeriod,
		"deviation_threshold": s.deviationThreshold,
		"bollinger_period":    s.bollingerPeriod,
		"bollinger_std_dev":   s.bollingerStdDev,
		"position_size":       s.positionSize,
		"stop_loss_percent":   s.stopLossPercent,
		"take_profit_percent": s.takeProfitPercent,
	}
}

// SetParameters sets the strategy parameters
func (s *MeanReversionStrategy) SetParameters(params map[string]interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update parameters if provided
	if val, ok := params["lookback_period"]; ok {
		if period, ok := val.(int); ok && period > 0 {
			s.lookbackPeriod = period
		} else {
			return fmt.Errorf("invalid lookback_period: %v", val)
		}
	}

	if val, ok := params["deviation_threshold"]; ok {
		if threshold, ok := val.(float64); ok && threshold > 0 {
			s.deviationThreshold = threshold
		} else {
			return fmt.Errorf("invalid deviation_threshold: %v", val)
		}
	}

	if val, ok := params["bollinger_period"]; ok {
		if period, ok := val.(int); ok && period > 0 {
			s.bollingerPeriod = period
		} else {
			return fmt.Errorf("invalid bollinger_period: %v", val)
		}
	}

	if val, ok := params["bollinger_std_dev"]; ok {
		if stdDev, ok := val.(float64); ok && stdDev > 0 {
			s.bollingerStdDev = stdDev
		} else {
			return fmt.Errorf("invalid bollinger_std_dev: %v", val)
		}
	}

	if val, ok := params["position_size"]; ok {
		if size, ok := val.(float64); ok && size > 0 {
			s.positionSize = size
		} else {
			return fmt.Errorf("invalid position_size: %v", val)
		}
	}

	if val, ok := params["stop_loss_percent"]; ok {
		if percent, ok := val.(float64); ok && percent > 0 {
			s.stopLossPercent = percent
		} else {
			return fmt.Errorf("invalid stop_loss_percent: %v", val)
		}
	}

	if val, ok := params["take_profit_percent"]; ok {
		if percent, ok := val.(float64); ok && percent > 0 {
			s.takeProfitPercent = percent
		} else {
			return fmt.Errorf("invalid take_profit_percent: %v", val)
		}
	}

	s.logger.Info("Mean reversion strategy parameters updated",
		zap.Int("lookback_period", s.lookbackPeriod),
		zap.Float64("deviation_threshold", s.deviationThreshold),
		zap.Int("bollinger_period", s.bollingerPeriod),
		zap.Float64("bollinger_std_dev", s.bollingerStdDev),
		zap.Float64("position_size", s.positionSize),
		zap.Float64("stop_loss_percent", s.stopLossPercent),
		zap.Float64("take_profit_percent", s.takeProfitPercent),
	)

	return nil
}
