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

// MomentumStrategy implements a momentum-based trading strategy
type MomentumStrategy struct {
	BaseStrategy
	lookbackPeriod    int
	momentumThreshold float64
	volatilityWindow  int
	positionSize      float64
	stopLossPercent   float64
	takeProfitPercent float64
	prices            map[string][]float64
	returns           map[string][]float64
	volatility        map[string]float64
	mutex             sync.RWMutex
}

// MomentumStrategyParams contains parameters for the momentum strategy
type MomentumStrategyParams struct {
	LookbackPeriod    int
	MomentumThreshold float64
	VolatilityWindow  int
	PositionSize      float64
	StopLossPercent   float64
	TakeProfitPercent float64
}

// NewMomentumStrategy creates a new momentum strategy
func NewMomentumStrategy(logger *zap.Logger, params MomentumStrategyParams) *MomentumStrategy {
	// Set default values if not provided
	if params.LookbackPeriod == 0 {
		params.LookbackPeriod = 20
	}
	if params.MomentumThreshold == 0 {
		params.MomentumThreshold = 0.02
	}
	if params.VolatilityWindow == 0 {
		params.VolatilityWindow = 20
	}
	if params.PositionSize == 0 {
		params.PositionSize = 0.1
	}
	if params.StopLossPercent == 0 {
		params.StopLossPercent = 0.02
	}
	if params.TakeProfitPercent == 0 {
		params.TakeProfitPercent = 0.05
	}

	return &MomentumStrategy{
		BaseStrategy: BaseStrategy{
			logger:      logger,
			name:        "Momentum",
			description: "Momentum-based trading strategy",
			symbols:     make(map[string]bool),
			active:      false,
		},
		lookbackPeriod:    params.LookbackPeriod,
		momentumThreshold: params.MomentumThreshold,
		volatilityWindow:  params.VolatilityWindow,
		positionSize:      params.PositionSize,
		stopLossPercent:   params.StopLossPercent,
		takeProfitPercent: params.TakeProfitPercent,
		prices:            make(map[string][]float64),
		returns:           make(map[string][]float64),
		volatility:        make(map[string]float64),
	}
}

// Initialize initializes the strategy
func (s *MomentumStrategy) Initialize(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("Initializing momentum strategy",
		zap.Int("lookback_period", s.lookbackPeriod),
		zap.Float64("momentum_threshold", s.momentumThreshold),
		zap.Int("volatility_window", s.volatilityWindow),
		zap.Float64("position_size", s.positionSize),
		zap.Float64("stop_loss_percent", s.stopLossPercent),
		zap.Float64("take_profit_percent", s.takeProfitPercent),
	)

	// Initialize price and return arrays for each symbol
	for symbol := range s.symbols {
		s.prices[symbol] = make([]float64, 0, s.lookbackPeriod*2)
		s.returns[symbol] = make([]float64, 0, s.lookbackPeriod*2)
		s.volatility[symbol] = 0.0
	}

	s.active = true
	return nil
}

// ProcessTick processes a market data tick
func (s *MomentumStrategy) ProcessTick(ctx context.Context, tick *models.MarketDataTick) error {
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

	// Calculate return if we have at least 2 prices
	if len(s.prices[symbol]) >= 2 {
		prevPrice := s.prices[symbol][len(s.prices[symbol])-2]
		ret := (price - prevPrice) / prevPrice
		s.returns[symbol] = append(s.returns[symbol], ret)
	}

	// Trim arrays if they exceed twice the lookback period
	if len(s.prices[symbol]) > s.lookbackPeriod*2 {
		s.prices[symbol] = s.prices[symbol][len(s.prices[symbol])-s.lookbackPeriod*2:]
	}
	if len(s.returns[symbol]) > s.lookbackPeriod*2 {
		s.returns[symbol] = s.returns[symbol][len(s.returns[symbol])-s.lookbackPeriod*2:]
	}

	// Calculate volatility if we have enough returns
	if len(s.returns[symbol]) >= s.volatilityWindow {
		s.volatility[symbol] = s.calculateVolatility(s.returns[symbol], s.volatilityWindow)
	}

	// Generate signals if we have enough data
	if len(s.prices[symbol]) >= s.lookbackPeriod {
		momentum := s.calculateMomentum(s.prices[symbol], s.lookbackPeriod)
		s.logger.Debug("Momentum calculation",
			zap.String("symbol", symbol),
			zap.Float64("momentum", momentum),
			zap.Float64("volatility", s.volatility[symbol]),
		)

		// Generate buy signal if momentum exceeds threshold
		if momentum > s.momentumThreshold {
			s.generateBuySignal(ctx, symbol, price, momentum)
		}

		// Generate sell signal if momentum is negative and exceeds threshold
		if momentum < -s.momentumThreshold {
			s.generateSellSignal(ctx, symbol, price, momentum)
		}
	}

	return nil
}

// calculateMomentum calculates the momentum indicator
func (s *MomentumStrategy) calculateMomentum(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	currentPrice := prices[len(prices)-1]
	pastPrice := prices[len(prices)-period]

	return (currentPrice - pastPrice) / pastPrice
}

// calculateVolatility calculates the volatility (standard deviation of returns)
func (s *MomentumStrategy) calculateVolatility(returns []float64, period int) float64 {
	if len(returns) < period {
		return 0
	}

	// Get the most recent returns
	recentReturns := returns[len(returns)-period:]

	// Calculate mean
	var sum float64
	for _, ret := range recentReturns {
		sum += ret
	}
	mean := sum / float64(period)

	// Calculate variance
	var variance float64
	for _, ret := range recentReturns {
		variance += math.Pow(ret-mean, 2)
	}
	variance /= float64(period)

	// Return standard deviation
	return math.Sqrt(variance)
}

// generateBuySignal generates a buy signal
func (s *MomentumStrategy) generateBuySignal(ctx context.Context, symbol string, price float64, momentum float64) {
	// Check if we already have a position
	position, err := s.GetPosition(ctx, symbol)
	if err == nil && position.Size > 0 {
		// Already long, do nothing
		return
	}

	// Calculate position size based on volatility
	adjustedPositionSize := s.positionSize
	if s.volatility[symbol] > 0 {
		// Reduce position size for high volatility
		volatilityFactor := 0.02 / s.volatility[symbol]
		if volatilityFactor < 1 {
			adjustedPositionSize *= volatilityFactor
		}
	}

	// Calculate stop loss and take profit levels
	stopLoss := price * (1 - s.stopLossPercent)
	takeProfit := price * (1 + s.takeProfitPercent)

	// Create order
	order := &models.Order{
		Symbol:     symbol,
		Side:       models.OrderSideBuy,
		Type:       models.OrderTypeMarket,
		Side:       models.OrderSideBuy,
		Type:       models.OrderTypeMarket,
		Side:       "buy",
		OrderType:  "market",
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
			zap.Float64("momentum", momentum),
		)
		return
	}

	s.logger.Info("Generated buy signal",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
		zap.Float64("momentum", momentum),
		zap.Float64("position_size", adjustedPositionSize),
		zap.Float64("stop_loss", stopLoss),
		zap.Float64("take_profit", takeProfit),
	)
}

// generateSellSignal generates a sell signal
func (s *MomentumStrategy) generateSellSignal(ctx context.Context, symbol string, price float64, momentum float64) {
	// Check if we already have a position
	position, err := s.GetPosition(ctx, symbol)
	if err == nil && position.Size < 0 {
		// Already short, do nothing
		return
	}

	// Calculate position size based on volatility
	adjustedPositionSize := s.positionSize
	if s.volatility[symbol] > 0 {
		// Reduce position size for high volatility
		volatilityFactor := 0.02 / s.volatility[symbol]
		if volatilityFactor < 1 {
			adjustedPositionSize *= volatilityFactor
		}
	}

	// Calculate stop loss and take profit levels
	stopLoss := price * (1 + s.stopLossPercent)
	takeProfit := price * (1 - s.takeProfitPercent)

	// Create order
	order := &models.Order{
		Symbol:     symbol,
		Side:       models.OrderSideSell,
		Type:       models.OrderTypeMarket,
		Side:       models.OrderSideSell,
		Type:       models.OrderTypeMarket,
		Side:       "sell",
		OrderType:  "market",
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
			zap.Float64("momentum", momentum),
		)
		return
	}

	s.logger.Info("Generated sell signal",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
		zap.Float64("momentum", momentum),
		zap.Float64("position_size", adjustedPositionSize),
		zap.Float64("stop_loss", stopLoss),
		zap.Float64("take_profit", takeProfit),
	)
}

// Stop stops the strategy
func (s *MomentumStrategy) Stop(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.active = false
	s.logger.Info("Momentum strategy stopped")
	return nil
}

// GetParameters returns the strategy parameters
func (s *MomentumStrategy) GetParameters() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return map[string]interface{}{
		"lookback_period":     s.lookbackPeriod,
		"momentum_threshold":  s.momentumThreshold,
		"volatility_window":   s.volatilityWindow,
		"position_size":       s.positionSize,
		"stop_loss_percent":   s.stopLossPercent,
		"take_profit_percent": s.takeProfitPercent,
	}
}

// SetParameters sets the strategy parameters
func (s *MomentumStrategy) SetParameters(params map[string]interface{}) error {
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

	if val, ok := params["momentum_threshold"]; ok {
		if threshold, ok := val.(float64); ok && threshold > 0 {
			s.momentumThreshold = threshold
		} else {
			return fmt.Errorf("invalid momentum_threshold: %v", val)
		}
	}

	if val, ok := params["volatility_window"]; ok {
		if window, ok := val.(int); ok && window > 0 {
			s.volatilityWindow = window
		} else {
			return fmt.Errorf("invalid volatility_window: %v", val)
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

	s.logger.Info("Momentum strategy parameters updated",
		zap.Int("lookback_period", s.lookbackPeriod),
		zap.Float64("momentum_threshold", s.momentumThreshold),
		zap.Int("volatility_window", s.volatilityWindow),
		zap.Float64("position_size", s.positionSize),
		zap.Float64("stop_loss_percent", s.stopLossPercent),
		zap.Float64("take_profit_percent", s.takeProfitPercent),
	)

	return nil
}
