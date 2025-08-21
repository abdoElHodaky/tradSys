package optimized

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/workerpool"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/markcheno/go-talib"
	"go.uber.org/zap"
)

// MomentumStrategy implements a momentum-based trading strategy
// using MACD as the primary indicator
type MomentumStrategy struct {
	*BaseStrategy
	
	// Strategy parameters
	symbols        []string
	lookbackPeriod int
	updateInterval int // in seconds
	fastPeriod     int
	slowPeriod     int
	signalPeriod   int
	threshold      float64
	
	// Strategy state
	prices         map[string][]float64
	signals        map[string]float64
	positions      map[string]float64
	lastUpdate     map[string]time.Time
	
	// Concurrency control
	mu             sync.RWMutex
	
	// Dependencies
	workerPool     *workerpool.WorkerPoolFactory
	metrics        *StrategyMetrics
	
	// Performance metrics
	processedUpdates int64
	executedTrades   int64
	pnl              float64
}

// Initialize initializes the strategy
func (s *MomentumStrategy) Initialize(ctx context.Context) error {
	if err := s.BaseStrategy.Initialize(ctx); err != nil {
		return err
	}
	
	s.lastUpdate = make(map[string]time.Time)
	
	s.logger.Info("Momentum strategy initialized",
		zap.Strings("symbols", s.symbols),
		zap.Int("lookback_period", s.lookbackPeriod),
		zap.Int("update_interval", s.updateInterval),
		zap.Int("fast_period", s.fastPeriod),
		zap.Int("slow_period", s.slowPeriod),
		zap.Int("signal_period", s.signalPeriod),
		zap.Float64("threshold", s.threshold))
	
	return nil
}

// OnMarketData processes market data updates
func (s *MomentumStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
	if !s.IsRunning() {
		return nil
	}
	
	// Check if this data is for one of our symbols
	symbolFound := false
	for _, symbol := range s.symbols {
		if data.Symbol == symbol {
			symbolFound = true
			break
		}
	}
	
	if !symbolFound {
		return nil
	}
	
	// Update price series and check for signals
	err := s.workerPool.SubmitTask("momentum-strategy-"+s.name, func() error {
		return s.processMarketData(ctx, data)
	})
	
	if err != nil {
		s.logger.Error("Failed to submit market data processing task",
			zap.Error(err),
			zap.String("symbol", data.Symbol))
		return err
	}
	
	return nil
}

// processMarketData processes market data and generates trading signals
func (s *MomentumStrategy) processMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Increment processed updates counter
	s.processedUpdates++
	
	// Update price series
	if _, ok := s.prices[data.Symbol]; !ok {
		s.prices[data.Symbol] = make([]float64, 0, s.lookbackPeriod+100) // Extra capacity for safety
	}
	
	s.prices[data.Symbol] = append(s.prices[data.Symbol], data.Price)
	
	// Trim price series if it exceeds lookback period
	if len(s.prices[data.Symbol]) > s.lookbackPeriod {
		s.prices[data.Symbol] = s.prices[data.Symbol][len(s.prices[data.Symbol])-s.lookbackPeriod:]
	}
	
	// Check if it's time to update signals
	lastUpdate, ok := s.lastUpdate[data.Symbol]
	if !ok || time.Since(lastUpdate) >= time.Duration(s.updateInterval)*time.Second {
		// Calculate MACD
		if len(s.prices[data.Symbol]) >= s.slowPeriod {
			macd, signal, _ := talib.Macd(
				s.prices[data.Symbol],
				s.fastPeriod,
				s.slowPeriod,
				s.signalPeriod,
			)
			
			// Get the latest MACD and signal values
			if len(macd) > 0 && len(signal) > 0 {
				latestMACD := macd[len(macd)-1]
				latestSignal := signal[len(signal)-1]
				
				// Calculate the MACD histogram
				histogram := latestMACD - latestSignal
				
				// Store the signal
				s.signals[data.Symbol] = histogram
				
				// Generate trading signals
				currentPosition, hasPosition := s.positions[data.Symbol]
				if !hasPosition {
					currentPosition = 0
				}
				
				// Long signal: MACD crosses above signal line by threshold
				if histogram > s.threshold && currentPosition <= 0 {
					if err := s.enterLongPosition(ctx, data.Symbol, data.Price); err != nil {
						s.logger.Error("Failed to enter long position",
							zap.Error(err),
							zap.String("symbol", data.Symbol),
							zap.Float64("price", data.Price))
					}
				}
				// Short signal: MACD crosses below signal line by threshold
				else if histogram < -s.threshold && currentPosition >= 0 {
					if err := s.enterShortPosition(ctx, data.Symbol, data.Price); err != nil {
						s.logger.Error("Failed to enter short position",
							zap.Error(err),
							zap.String("symbol", data.Symbol),
							zap.Float64("price", data.Price))
					}
				}
				// Exit signal: MACD crosses back toward signal line
				else if (histogram < s.threshold/2 && currentPosition > 0) ||
					(histogram > -s.threshold/2 && currentPosition < 0) {
					if err := s.exitPosition(ctx, data.Symbol, data.Price); err != nil {
						s.logger.Error("Failed to exit position",
							zap.Error(err),
							zap.String("symbol", data.Symbol),
							zap.Float64("price", data.Price))
					}
				}
			}
		}
		
		s.lastUpdate[data.Symbol] = time.Now()
	}
	
	return nil
}

// OnOrderUpdate processes order updates
func (s *MomentumStrategy) OnOrderUpdate(ctx context.Context, order *orders.OrderResponse) error {
	if !s.IsRunning() {
		return nil
	}
	
	// Check if this order is for one of our symbols
	symbolFound := false
	for _, symbol := range s.symbols {
		if order.Symbol == symbol {
			symbolFound = true
			break
		}
	}
	
	if !symbolFound {
		return nil
	}
	
	// Process the order update
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Update position based on order status
	if order.Status == "filled" {
		// Update position
		if order.Side == "buy" {
			s.positions[order.Symbol] += order.Quantity
		} else if order.Side == "sell" {
			s.positions[order.Symbol] -= order.Quantity
		}
		
		s.logger.Info("Order filled",
			zap.String("order_id", order.OrderId),
			zap.String("symbol", order.Symbol),
			zap.String("side", order.Side),
			zap.Float64("quantity", order.Quantity),
			zap.Float64("price", order.Price),
			zap.Float64("current_position", s.positions[order.Symbol]))
	}
	
	return nil
}

// enterLongPosition enters a long position
func (s *MomentumStrategy) enterLongPosition(ctx context.Context, symbol string, price float64) error {
	// Create a buy order
	// In a real implementation, you would use an order service to submit this order
	
	// Update position
	s.positions[symbol] = 1.0 // Simplified position sizing
	
	// Increment executed trades counter
	s.executedTrades++
	
	s.logger.Info("Entered long position",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
		zap.Float64("signal", s.signals[symbol]))
	
	return nil
}

// enterShortPosition enters a short position
func (s *MomentumStrategy) enterShortPosition(ctx context.Context, symbol string, price float64) error {
	// Create a sell order
	// In a real implementation, you would use an order service to submit this order
	
	// Update position
	s.positions[symbol] = -1.0 // Simplified position sizing
	
	// Increment executed trades counter
	s.executedTrades++
	
	s.logger.Info("Entered short position",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
		zap.Float64("signal", s.signals[symbol]))
	
	return nil
}

// exitPosition exits a position
func (s *MomentumStrategy) exitPosition(ctx context.Context, symbol string, price float64) error {
	currentPosition := s.positions[symbol]
	if currentPosition == 0 {
		return nil
	}
	
	// Create an order to close the position
	// In a real implementation, you would use an order service to submit this order
	
	// Calculate P&L
	entryPrice := s.prices[symbol][len(s.prices[symbol])-2] // Simplified, should use actual entry price
	pnl := 0.0
	if currentPosition > 0 {
		pnl = (price - entryPrice) * currentPosition
	} else {
		pnl = (entryPrice - price) * -currentPosition
	}
	
	// Update P&L
	s.pnl += pnl
	
	// Reset position
	s.positions[symbol] = 0.0
	
	// Increment executed trades counter
	s.executedTrades++
	
	s.logger.Info("Exited position",
		zap.String("symbol", symbol),
		zap.Float64("price", price),
		zap.Float64("signal", s.signals[symbol]),
		zap.Float64("pnl", pnl),
		zap.Float64("total_pnl", s.pnl))
	
	return nil
}

// GetPerformanceMetrics returns performance metrics for the strategy
func (s *MomentumStrategy) GetPerformanceMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"processed_updates": s.processedUpdates,
		"executed_trades":   s.executedTrades,
		"pnl":              s.pnl,
		"positions":        s.positions,
		"signals":          s.signals,
	}
}

