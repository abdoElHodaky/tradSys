package strategies

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OptimizedStatisticalArbitrageStrategy implements a pairs trading strategy
// with optimizations for high-frequency trading
type OptimizedStatisticalArbitrageStrategy struct {
	*BaseStrategy
	
	// Strategy parameters
	pairID          string
	symbol1         string
	symbol2         string
	ratio           float64
	zScoreEntry     float64
	zScoreExit      float64
	positionSize    float64
	maxPositions    int
	lookbackPeriod  int
	updateInterval  time.Duration
	
	// Strategy state
	prices1         []float64
	prices2         []float64
	positions       map[string]*models.PairPosition
	lastUpdate      time.Time
	
	// Optimized statistics
	stats           *IncrementalStatistics
	correlation     *IncrementalCorrelation
	
	// Concurrency control
	mu              sync.RWMutex
	processingMu    sync.Mutex
	
	// Services
	orderService    orders.OrderService
	pairRepo        *repositories.PairRepository
	statsRepo       *repositories.PairStatisticsRepository
	positionRepo    *repositories.PairPositionRepository
	
	// Performance metrics
	processedUpdates uint64
	executedTrades   uint64
	
	// Pre-allocated buffers
	priceBuffer1    []float64
	priceBuffer2    []float64
}

// NewOptimizedStatisticalArbitrageStrategy creates a new optimized statistical arbitrage strategy
func NewOptimizedStatisticalArbitrageStrategy(
	logger *zap.Logger,
	params StatisticalArbitrageParams,
	orderService orders.OrderService,
	pairRepo *repositories.PairRepository,
	statsRepo *repositories.PairStatisticsRepository,
	positionRepo *repositories.PairPositionRepository,
) *OptimizedStatisticalArbitrageStrategy {
	return &OptimizedStatisticalArbitrageStrategy{
		BaseStrategy:   NewBaseStrategy(params.Name, logger),
		pairID:         params.PairID,
		symbol1:        params.Symbol1,
		symbol2:        params.Symbol2,
		ratio:          params.Ratio,
		zScoreEntry:    params.ZScoreEntry,
		zScoreExit:     params.ZScoreExit,
		positionSize:   params.PositionSize,
		maxPositions:   params.MaxPositions,
		lookbackPeriod: params.LookbackPeriod,
		updateInterval: params.UpdateInterval,
		positions:      make(map[string]*models.PairPosition),
		stats:          NewIncrementalStatistics(),
		correlation:    NewIncrementalCorrelation(),
		orderService:   orderService,
		pairRepo:       pairRepo,
		statsRepo:      statsRepo,
		positionRepo:   positionRepo,
		// Pre-allocate buffers with capacity
		priceBuffer1:   make([]float64, 0, params.LookbackPeriod+100), // Extra capacity for safety
		priceBuffer2:   make([]float64, 0, params.LookbackPeriod+100),
	}
}

// Initialize initializes the strategy
func (s *OptimizedStatisticalArbitrageStrategy) Initialize(ctx context.Context) error {
	if err := s.BaseStrategy.Initialize(ctx); err != nil {
		return err
	}
	
	// Pre-allocate price series with capacity to reduce reallocations
	s.prices1 = make([]float64, 0, s.lookbackPeriod)
	s.prices2 = make([]float64, 0, s.lookbackPeriod)
	
	// Load open positions
	positions, err := s.positionRepo.GetOpenPositions(ctx, s.pairID)
	if err != nil {
		s.logger.Error("Failed to load open positions",
			zap.Error(err),
			zap.String("pair_id", s.pairID))
	} else {
		for _, pos := range positions {
			s.positions[fmt.Sprintf("%d", pos.ID)] = pos
		}
		s.logger.Info("Loaded open positions",
			zap.Int("count", len(positions)),
			zap.String("pair_id", s.pairID))
	}
	
	// Load historical data for initial statistics
	// In a real implementation, this would load from a market data service
	// For now, we'll just initialize empty statistics
	
	s.logger.Info("Optimized statistical arbitrage strategy initialized",
		zap.String("pair_id", s.pairID),
		zap.String("symbol1", s.symbol1),
		zap.String("symbol2", s.symbol2),
		zap.Float64("ratio", s.ratio),
		zap.Float64("z_score_entry", s.zScoreEntry),
		zap.Float64("z_score_exit", s.zScoreExit))
	
	return nil
}

// OnMarketData processes market data updates
func (s *OptimizedStatisticalArbitrageStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
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
	
	// Only proceed if we have enough data for both symbols
	if len(s.prices1) < s.lookbackPeriod || len(s.prices2) < s.lookbackPeriod {
		return nil
	}
	
	// Check if it's time to update statistics and check for signals
	if time.Since(s.lastUpdate) >= s.updateInterval {
		// Update pair statistics in the database
		if err := s.updatePairStatistics(ctx); err != nil {
			s.logger.Error("Failed to update pair statistics",
				zap.Error(err),
				zap.String("pair_id", s.pairID))
		}
		
		// Check for trading signals
		if err := s.checkForEntrySignals(ctx); err != nil {
			s.logger.Error("Failed to check for entry signals",
				zap.Error(err),
				zap.String("pair_id", s.pairID))
		}
		
		if err := s.checkForExitSignals(ctx); err != nil {
			s.logger.Error("Failed to check for exit signals",
				zap.Error(err),
				zap.String("pair_id", s.pairID))
		}
		
		s.lastUpdate = time.Now()
	}
	
	// Increment processed updates counter
	s.processedUpdates++
	
	return nil
}

// updatePriceSeries1 updates the price series for symbol1
func (s *OptimizedStatisticalArbitrageStrategy) updatePriceSeries1(ctx context.Context, price float64) {
	// Add the new price
	s.prices1 = append(s.prices1, price)
	
	// If we have both prices, update the correlation and spread statistics
	if len(s.prices1) > 0 && len(s.prices2) > 0 {
		// Get the latest prices
		price1 := s.prices1[len(s.prices1)-1]
		price2 := s.prices2[len(s.prices2)-1]
		
		// Update correlation
		s.correlation.Add(price1, price2)
		
		// Calculate and update spread statistics
		spread := price1 - (s.ratio * price2)
		
		// If we have enough data, we can do an incremental update
		if len(s.prices1) > s.lookbackPeriod {
			// Get the oldest prices that will be removed
			oldPrice1 := s.prices1[0]
			oldPrice2 := s.prices2[0]
			oldSpread := oldPrice1 - (s.ratio * oldPrice2)
			
			// Update statistics incrementally
			s.stats.Update(oldSpread, spread)
			
			// Trim the series
			s.prices1 = s.prices1[1:]
		} else {
			// Just add the new spread
			s.stats.Add(spread)
		}
	}
}

// updatePriceSeries2 updates the price series for symbol2
func (s *OptimizedStatisticalArbitrageStrategy) updatePriceSeries2(ctx context.Context, price float64) {
	// Add the new price
	s.prices2 = append(s.prices2, price)
	
	// If we have both prices, update the correlation and spread statistics
	if len(s.prices1) > 0 && len(s.prices2) > 0 {
		// Get the latest prices
		price1 := s.prices1[len(s.prices1)-1]
		price2 := s.prices2[len(s.prices2)-1]
		
		// Update correlation
		s.correlation.Add(price1, price2)
		
		// Calculate and update spread statistics
		spread := price1 - (s.ratio * price2)
		
		// If we have enough data, we can do an incremental update
		if len(s.prices2) > s.lookbackPeriod {
			// Get the oldest prices that will be removed
			oldPrice1 := s.prices1[0]
			oldPrice2 := s.prices2[0]
			oldSpread := oldPrice1 - (s.ratio * oldPrice2)
			
			// Update statistics incrementally
			s.stats.Update(oldSpread, spread)
			
			// Trim the series
			s.prices2 = s.prices2[1:]
		} else {
			// Just add the new spread
			s.stats.Add(spread)
		}
	}
}

// updatePairStatistics updates the pair statistics in the database
func (s *OptimizedStatisticalArbitrageStrategy) updatePairStatistics(ctx context.Context) error {
	// Calculate current spread and z-score
	price1 := s.prices1[len(s.prices1)-1]
	price2 := s.prices2[len(s.prices2)-1]
	currentSpread := price1 - (s.ratio * price2)
	currentZScore := s.stats.ZScore(currentSpread)
	
	// Get correlation
	correlation := s.correlation.Correlation()
	
	// Save statistics to database
	stats := &models.PairStatistics{
		PairID:        s.pairID,
		Timestamp:     time.Now(),
		Correlation:   correlation,
		Cointegration: 0, // Would need to calculate this separately
		SpreadMean:    s.stats.Mean(),
		SpreadStdDev:  s.stats.StdDev(),
		CurrentZScore: currentZScore,
		SpreadValue:   currentSpread,
	}
	
	if err := s.statsRepo.Create(ctx, stats); err != nil {
		return fmt.Errorf("failed to save pair statistics: %w", err)
	}
	
	// Update pair in database with latest statistics
	pair, err := s.pairRepo.GetPair(ctx, s.pairID)
	if err != nil {
		return fmt.Errorf("failed to get pair: %w", err)
	}
	
	pair.Correlation = correlation
	// pair.Cointegration would be set here if calculated
	
	if err := s.pairRepo.Update(ctx, pair); err != nil {
		return fmt.Errorf("failed to update pair: %w", err)
	}
	
	return nil
}

// checkForEntrySignals checks for entry signals
func (s *OptimizedStatisticalArbitrageStrategy) checkForEntrySignals(ctx context.Context) error {
	// Get current z-score
	price1 := s.prices1[len(s.prices1)-1]
	price2 := s.prices2[len(s.prices2)-1]
	currentSpread := price1 - (s.ratio * price2)
	currentZScore := s.stats.ZScore(currentSpread)
	
	// Check if we have reached the maximum number of positions
	if len(s.positions) >= s.maxPositions {
		return nil
	}
	
	// Check for entry signals
	if currentZScore <= -s.zScoreEntry {
		// Spread is below the lower threshold, enter a long pair position
		if err := s.enterLongPosition(ctx); err != nil {
			return err
		}
	} else if currentZScore >= s.zScoreEntry {
		// Spread is above the upper threshold, enter a short pair position
		if err := s.enterShortPosition(ctx); err != nil {
			return err
		}
	}
	
	return nil
}

// checkForExitSignals checks for exit signals
func (s *OptimizedStatisticalArbitrageStrategy) checkForExitSignals(ctx context.Context) error {
	// Get current z-score
	price1 := s.prices1[len(s.prices1)-1]
	price2 := s.prices2[len(s.prices2)-1]
	currentSpread := price1 - (s.ratio * price2)
	currentZScore := s.stats.ZScore(currentSpread)
	
	// Check for exit signals
	for id, position := range s.positions {
		// Update position with current prices
		position.CurrentPrice1 = price1
		position.CurrentPrice2 = price2
		position.CurrentSpread = currentSpread
		position.CurrentZScore = currentZScore
		
		// Check if the position should be exited
		if (position.Quantity1 > 0 && currentZScore >= -s.zScoreExit) ||
			(position.Quantity1 < 0 && currentZScore <= s.zScoreExit) {
			// Exit the position
			if err := s.exitPosition(ctx, id, position); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// enterLongPosition enters a long pair position
func (s *OptimizedStatisticalArbitrageStrategy) enterLongPosition(ctx context.Context) error {
	// Calculate position sizes
	qty1 := s.positionSize
	qty2 := s.positionSize * s.ratio
	
	// Create buy order for symbol1
	buyOrder := &models.Order{
		OrderID:    uuid.New().String(),
		Symbol:     s.symbol1,
		Side:       models.OrderSideBuy,
		Type:       models.OrderTypeMarket,
		Quantity:   qty1,
		Price:      s.prices1[len(s.prices1)-1],
		Strategy:   s.name,
		Timestamp:  time.Now(),
	}
	
	// Create sell order for symbol2
	sellOrder := &models.Order{
		OrderID:    uuid.New().String(),
		Symbol:     s.symbol2,
		Side:       models.OrderSideSell,
		Type:       models.OrderTypeMarket,
		Quantity:   qty2,
		Price:      s.prices2[len(s.prices2)-1],
		Strategy:   s.name,
		Timestamp:  time.Now(),
	}
	
	// Submit orders
	// In a real implementation, you would use the order service to submit these orders
	// and handle the responses. For simplicity, we'll assume they're executed immediately.
	
	// Create and store position
	position := &models.PairPosition{
		PairID:         s.pairID,
		EntryTimestamp: time.Now(),
		Symbol1:        s.symbol1,
		Symbol2:        s.symbol2,
		Quantity1:      qty1,
		Quantity2:      -qty2, // Negative quantity indicates short position
		EntryPrice1:    s.prices1[len(s.prices1)-1],
		EntryPrice2:    s.prices2[len(s.prices2)-1],
		CurrentPrice1:  s.prices1[len(s.prices1)-1],
		CurrentPrice2:  s.prices2[len(s.prices2)-1],
		EntrySpread:    s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1]),
		CurrentSpread:  s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1]),
		EntryZScore:    s.stats.ZScore(s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1])),
		CurrentZScore:  s.stats.ZScore(s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1])),
		Status:         "open",
	}
	
	// Save position to database
	if err := s.positionRepo.Create(ctx, position); err != nil {
		s.logger.Error("Failed to create position",
			zap.Error(err),
			zap.String("pair_id", s.pairID))
		return err
	}
	
	// Add to local positions map
	s.positions[fmt.Sprintf("%d", position.ID)] = position
	
	// Increment executed trades counter
	s.executedTrades++
	
	s.logger.Info("Entered long pair position",
		zap.String("pair_id", s.pairID),
		zap.String("symbol1", s.symbol1),
		zap.String("symbol2", s.symbol2),
		zap.Float64("quantity1", qty1),
		zap.Float64("quantity2", -qty2),
		zap.Float64("entry_price1", position.EntryPrice1),
		zap.Float64("entry_price2", position.EntryPrice2),
		zap.Float64("entry_z_score", position.EntryZScore))
	
	return nil
}

// enterShortPosition enters a short pair position
func (s *OptimizedStatisticalArbitrageStrategy) enterShortPosition(ctx context.Context) error {
	// Calculate position sizes
	qty1 := s.positionSize
	qty2 := s.positionSize * s.ratio
	
	// Create sell order for symbol1
	sellOrder := &models.Order{
		OrderID:    uuid.New().String(),
		Symbol:     s.symbol1,
		Side:       models.OrderSideSell,
		Type:       models.OrderTypeMarket,
		Quantity:   qty1,
		Price:      s.prices1[len(s.prices1)-1],
		Strategy:   s.name,
		Timestamp:  time.Now(),
	}
	
	// Create buy order for symbol2
	buyOrder := &models.Order{
		OrderID:    uuid.New().String(),
		Symbol:     s.symbol2,
		Side:       models.OrderSideBuy,
		Type:       models.OrderTypeMarket,
		Quantity:   qty2,
		Price:      s.prices2[len(s.prices2)-1],
		Strategy:   s.name,
		Timestamp:  time.Now(),
	}
	
	// Submit orders
	// In a real implementation, you would use the order service to submit these orders
	// and handle the responses. For simplicity, we'll assume they're executed immediately.
	
	// Create and store position
	position := &models.PairPosition{
		PairID:         s.pairID,
		EntryTimestamp: time.Now(),
		Symbol1:        s.symbol1,
		Symbol2:        s.symbol2,
		Quantity1:      -qty1, // Negative quantity indicates short position
		Quantity2:      qty2,
		EntryPrice1:    s.prices1[len(s.prices1)-1],
		EntryPrice2:    s.prices2[len(s.prices2)-1],
		CurrentPrice1:  s.prices1[len(s.prices1)-1],
		CurrentPrice2:  s.prices2[len(s.prices2)-1],
		EntrySpread:    s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1]),
		CurrentSpread:  s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1]),
		EntryZScore:    s.stats.ZScore(s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1])),
		CurrentZScore:  s.stats.ZScore(s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1])),
		Status:         "open",
	}
	
	// Save position to database
	if err := s.positionRepo.Create(ctx, position); err != nil {
		s.logger.Error("Failed to create position",
			zap.Error(err),
			zap.String("pair_id", s.pairID))
		return err
	}
	
	// Add to local positions map
	s.positions[fmt.Sprintf("%d", position.ID)] = position
	
	// Increment executed trades counter
	s.executedTrades++
	
	s.logger.Info("Entered short pair position",
		zap.String("pair_id", s.pairID),
		zap.String("symbol1", s.symbol1),
		zap.String("symbol2", s.symbol2),
		zap.Float64("quantity1", -qty1),
		zap.Float64("quantity2", qty2),
		zap.Float64("entry_price1", position.EntryPrice1),
		zap.Float64("entry_price2", position.EntryPrice2),
		zap.Float64("entry_z_score", position.EntryZScore))
	
	return nil
}

// exitPosition exits a position
func (s *OptimizedStatisticalArbitrageStrategy) exitPosition(ctx context.Context, id string, position *models.PairPosition) error {
	// Create orders to close the position
	var order1, order2 *models.Order
	
	if position.Quantity1 > 0 {
		// Long position in symbol1, need to sell
		order1 = &models.Order{
			OrderID:    uuid.New().String(),
			Symbol:     position.Symbol1,
			Side:       models.OrderSideSell,
			Type:       models.OrderTypeMarket,
			Quantity:   math.Abs(position.Quantity1),
			Price:      s.prices1[len(s.prices1)-1],
			Strategy:   s.name,
			Timestamp:  time.Now(),
		}
	} else {
		// Short position in symbol1, need to buy
		order1 = &models.Order{
			OrderID:    uuid.New().String(),
			Symbol:     position.Symbol1,
			Side:       models.OrderSideBuy,
			Type:       models.OrderTypeMarket,
			Quantity:   math.Abs(position.Quantity1),
			Price:      s.prices1[len(s.prices1)-1],
			Strategy:   s.name,
			Timestamp:  time.Now(),
		}
	}
	
	if position.Quantity2 > 0 {
		// Long position in symbol2, need to sell
		order2 = &models.Order{
			OrderID:    uuid.New().String(),
			Symbol:     position.Symbol2,
			Side:       models.OrderSideSell,
			Type:       models.OrderTypeMarket,
			Quantity:   math.Abs(position.Quantity2),
			Price:      s.prices2[len(s.prices2)-1],
			Strategy:   s.name,
			Timestamp:  time.Now(),
		}
	} else {
		// Short position in symbol2, need to buy
		order2 = &models.Order{
			OrderID:    uuid.New().String(),
			Symbol:     position.Symbol2,
			Side:       models.OrderSideBuy,
			Type:       models.OrderTypeMarket,
			Quantity:   math.Abs(position.Quantity2),
			Price:      s.prices2[len(s.prices2)-1],
			Strategy:   s.name,
			Timestamp:  time.Now(),
		}
	}
	
	// Submit orders
	// In a real implementation, you would use the order service to submit these orders
	// and handle the responses. For simplicity, we'll assume they're executed immediately.
	
	// Update position
	position.Status = "closed"
	position.ExitTimestamp = time.Now()
	
	// Calculate final P&L
	pnl1 := position.Quantity1 * (position.CurrentPrice1 - position.EntryPrice1)
	pnl2 := position.Quantity2 * (position.CurrentPrice2 - position.EntryPrice2)
	position.PnL = pnl1 + pnl2
	
	// Update position in database
	if err := s.positionRepo.Update(ctx, position); err != nil {
		s.logger.Error("Failed to update position",
			zap.Error(err),
			zap.String("pair_id", s.pairID),
			zap.String("position_id", id))
		return err
	}
	
	// Remove from local positions map
	delete(s.positions, id)
	
	// Increment executed trades counter
	s.executedTrades++
	
	s.logger.Info("Exited pair position",
		zap.String("pair_id", s.pairID),
		zap.String("position_id", id),
		zap.Float64("entry_z_score", position.EntryZScore),
		zap.Float64("exit_z_score", position.CurrentZScore),
		zap.Float64("pnl", position.PnL))
	
	return nil
}

// GetPerformanceMetrics returns performance metrics for the strategy
func (s *OptimizedStatisticalArbitrageStrategy) GetPerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"processed_updates": s.processedUpdates,
		"executed_trades":   s.executedTrades,
		"open_positions":    len(s.positions),
		"correlation":       s.correlation.Correlation(),
		"spread_mean":       s.stats.Mean(),
		"spread_stddev":     s.stats.StdDev(),
		"current_z_score":   s.stats.ZScore(s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1])),
	}
}

