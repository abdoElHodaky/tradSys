package strategies

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/statistics"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// updateStatistics updates the statistical measures
func (s *StatisticalArbitrageStrategy) updateStatistics(ctx context.Context) error {
	// Ensure we have enough data
	if len(s.prices1) < s.lookbackPeriod || len(s.prices2) < s.lookbackPeriod {
		return fmt.Errorf("insufficient data for statistical analysis")
	}

	// Calculate correlation
	correlation, err := statistics.CalculateCorrelation(s.prices1, s.prices2)
	if err != nil {
		return fmt.Errorf("failed to calculate correlation: %w", err)
	}

	// Calculate cointegration
	cointegration, isCointegrated, err := statistics.EngleGrangerTest(s.prices1, s.prices2)
	if err != nil {
		return fmt.Errorf("failed to perform cointegration test: %w", err)
	}

	// Calculate spread
	spread, err := statistics.CalculateSpread(s.prices1, s.prices2, s.ratio)
	if err != nil {
		return fmt.Errorf("failed to calculate spread: %w", err)
	}
	s.spread = spread

	// Calculate spread statistics
	spreadMean, err := statistics.CalculateMean(s.spread)
	if err != nil {
		return fmt.Errorf("failed to calculate spread mean: %w", err)
	}
	s.spreadMean = spreadMean

	spreadStdDev, err := statistics.CalculateStdDev(s.spread, s.spreadMean)
	if err != nil {
		return fmt.Errorf("failed to calculate spread standard deviation: %w", err)
	}
	s.spreadStdDev = spreadStdDev

	// Calculate current z-score
	currentSpread := s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1])
	s.currentZScore = statistics.CalculateZScore(currentSpread, s.spreadMean, s.spreadStdDev)

	// Save statistics to database
	stats := &models.PairStatistics{
		PairID:        s.pairID,
		Timestamp:     time.Now(),
		Correlation:   correlation,
		Cointegration: cointegration,
		SpreadMean:    s.spreadMean,
		SpreadStdDev:  s.spreadStdDev,
		CurrentZScore: s.currentZScore,
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
	pair.Cointegration = cointegration

	if err := s.pairRepo.UpdatePair(ctx, pair); err != nil {
		return fmt.Errorf("failed to update pair: %w", err)
	}

	s.logger.Debug("Updated pair statistics",
		zap.String("pair_id", s.pairID),
		zap.Float64("correlation", correlation),
		zap.Float64("cointegration", cointegration),
		zap.Bool("is_cointegrated", isCointegrated),
		zap.Float64("spread_mean", s.spreadMean),
		zap.Float64("spread_std_dev", s.spreadStdDev),
		zap.Float64("current_z_score", s.currentZScore))

	return nil
}

// checkEntrySignals checks for entry signals
func (s *StatisticalArbitrageStrategy) checkEntrySignals(ctx context.Context) error {
	// Check if we have reached the maximum number of positions
	if len(s.positions) >= s.maxPositions {
		return nil
	}

	// Check if we have sufficient data
	if len(s.prices1) < s.lookbackPeriod || len(s.prices2) < s.lookbackPeriod {
		return nil
	}

	// Check for long entry signal (z-score is significantly negative)
	if s.currentZScore <= -s.zScoreEntry {
		s.logger.Info("Long entry signal detected",
			zap.String("pair_id", s.pairID),
			zap.Float64("z_score", s.currentZScore),
			zap.Float64("threshold", -s.zScoreEntry))

		if err := s.enterLongPosition(ctx); err != nil {
			return fmt.Errorf("failed to enter long position: %w", err)
		}
	}

	// Check for short entry signal (z-score is significantly positive)
	if s.currentZScore >= s.zScoreEntry {
		s.logger.Info("Short entry signal detected",
			zap.String("pair_id", s.pairID),
			zap.Float64("z_score", s.currentZScore),
			zap.Float64("threshold", s.zScoreEntry))

		if err := s.enterShortPosition(ctx); err != nil {
			return fmt.Errorf("failed to enter short position: %w", err)
		}
	}

	return nil
}

// checkExitSignals checks for exit signals
func (s *StatisticalArbitrageStrategy) checkExitSignals(ctx context.Context) error {
	for positionID, position := range s.positions {
		// Check if position should be closed based on z-score
		shouldClose := false

		// For long positions (positive quantity1), close when z-score approaches zero
		if position.Quantity1 > 0 && s.currentZScore >= -s.zScoreExit {
			shouldClose = true
		}

		// For short positions (negative quantity1), close when z-score approaches zero
		if position.Quantity1 < 0 && s.currentZScore <= s.zScoreExit {
			shouldClose = true
		}

		if shouldClose {
			s.logger.Info("Exit signal detected",
				zap.String("pair_id", s.pairID),
				zap.String("position_id", positionID),
				zap.Float64("z_score", s.currentZScore),
				zap.Float64("exit_threshold", s.zScoreExit))

			if err := s.closePosition(ctx, position); err != nil {
				return fmt.Errorf("failed to close position %s: %w", positionID, err)
			}
		}
	}

	return nil
}

// enterLongPosition enters a long pair position
func (s *StatisticalArbitrageStrategy) enterLongPosition(ctx context.Context) error {
	// Calculate position sizes
	qty1 := s.positionSize
	qty2 := s.positionSize * s.ratio

	// Create buy order for symbol1
	buyOrder := &models.Order{
		OrderID:   uuid.New().String(),
		Symbol:    s.symbol1,
		Side:      models.OrderSideBuy,
		Type:      models.OrderTypeMarket,
		Quantity:  qty1,
		Price:     s.prices1[len(s.prices1)-1],
		Strategy:  s.name,
		Timestamp: time.Now(),
	}

	// Create sell order for symbol2
	sellOrder := &models.Order{
		OrderID:   uuid.New().String(),
		Symbol:    s.symbol2,
		Side:      models.OrderSideSell,
		Type:      models.OrderTypeMarket,
		Quantity:  qty2,
		Price:     s.prices2[len(s.prices2)-1],
		Strategy:  s.name,
		Timestamp: time.Now(),
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
		Quantity2:      qty2,
		EntryPrice1:    s.prices1[len(s.prices1)-1],
		EntryPrice2:    s.prices2[len(s.prices2)-1],
		CurrentPrice1:  s.prices1[len(s.prices1)-1],
		CurrentPrice2:  s.prices2[len(s.prices2)-1],
		EntrySpread:    s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1]),
		CurrentSpread:  s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1]),
		EntryZScore:    s.currentZScore,
		CurrentZScore:  s.currentZScore,
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

	s.logger.Info("Entered long pair position",
		zap.String("pair_id", s.pairID),
		zap.String("symbol1", s.symbol1),
		zap.String("symbol2", s.symbol2),
		zap.Float64("quantity1", qty1),
		zap.Float64("quantity2", qty2),
		zap.Float64("entry_price1", position.EntryPrice1),
		zap.Float64("entry_price2", position.EntryPrice2),
		zap.Float64("entry_z_score", position.EntryZScore))

	return nil
}

// enterShortPosition enters a short pair position
func (s *StatisticalArbitrageStrategy) enterShortPosition(ctx context.Context) error {
	// Calculate position sizes
	qty1 := s.positionSize
	qty2 := s.positionSize * s.ratio

	// Create sell order for symbol1
	sellOrder := &models.Order{
		OrderID:   uuid.New().String(),
		Symbol:    s.symbol1,
		Side:      models.OrderSideSell,
		Type:      models.OrderTypeMarket,
		Quantity:  qty1,
		Price:     s.prices1[len(s.prices1)-1],
		Strategy:  s.name,
		Timestamp: time.Now(),
	}

	// Create buy order for symbol2
	buyOrder := &models.Order{
		OrderID:   uuid.New().String(),
		Symbol:    s.symbol2,
		Side:      models.OrderSideBuy,
		Type:      models.OrderTypeMarket,
		Quantity:  qty2,
		Price:     s.prices2[len(s.prices2)-1],
		Strategy:  s.name,
		Timestamp: time.Now(),
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
		EntryZScore:    s.currentZScore,
		CurrentZScore:  s.currentZScore,
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

// closePosition closes a pair position
func (s *StatisticalArbitrageStrategy) closePosition(ctx context.Context, position *models.PairPosition) error {
	// Calculate current PnL
	currentPrice1 := s.prices1[len(s.prices1)-1]
	currentPrice2 := s.prices2[len(s.prices2)-1]

	// Update position with current prices
	position.CurrentPrice1 = currentPrice1
	position.CurrentPrice2 = currentPrice2
	position.CurrentSpread = currentPrice1 - (s.ratio * currentPrice2)
	position.CurrentZScore = s.currentZScore

	// Calculate PnL
	pnl1 := position.Quantity1 * (currentPrice1 - position.EntryPrice1)
	pnl2 := position.Quantity2 * (currentPrice2 - position.EntryPrice2)
	totalPnL := pnl1 + pnl2

	position.RealizedPnL = totalPnL
	position.ExitTimestamp = time.Now()
	position.Status = "closed"

	// Create closing orders
	// Close symbol1 position
	closeOrder1 := &models.Order{
		OrderID:   uuid.New().String(),
		Symbol:    s.symbol1,
		Side:      getOppositeSide(getOrderSide(position.Quantity1)),
		Type:      models.OrderTypeMarket,
		Quantity:  math.Abs(position.Quantity1),
		Price:     currentPrice1,
		Strategy:  s.name,
		Timestamp: time.Now(),
	}

	// Close symbol2 position
	closeOrder2 := &models.Order{
		OrderID:   uuid.New().String(),
		Symbol:    s.symbol2,
		Side:      getOppositeSide(getOrderSide(position.Quantity2)),
		Type:      models.OrderTypeMarket,
		Quantity:  math.Abs(position.Quantity2),
		Price:     currentPrice2,
		Strategy:  s.name,
		Timestamp: time.Now(),
	}

	// Submit closing orders
	// In a real implementation, you would use the order service to submit these orders

	// Update position in database
	if err := s.positionRepo.Update(ctx, position); err != nil {
		s.logger.Error("Failed to update position",
			zap.Error(err),
			zap.String("pair_id", s.pairID),
			zap.String("position_id", fmt.Sprintf("%d", position.ID)))
		return err
	}

	// Remove from local positions map
	delete(s.positions, fmt.Sprintf("%d", position.ID))

	s.logger.Info("Closed pair position",
		zap.String("pair_id", s.pairID),
		zap.String("position_id", fmt.Sprintf("%d", position.ID)),
		zap.Float64("realized_pnl", totalPnL),
		zap.Float64("exit_z_score", s.currentZScore))

	return nil
}

// Helper functions for risk calculations

// calculateVaR calculates Value at Risk at given confidence level
func (s *StatisticalArbitrageStrategy) calculateVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Sort returns in ascending order
	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	// Calculate VaR at given confidence level
	index := int((1.0 - confidence) * float64(len(sorted)))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return -sorted[index] // Return as positive value
}

// calculateExpectedShortfall calculates Expected Shortfall (Conditional VaR)
func (s *StatisticalArbitrageStrategy) calculateExpectedShortfall(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Sort returns in ascending order
	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	// Calculate threshold for given confidence level
	index := int((1.0 - confidence) * float64(len(sorted)))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	// Calculate average of returns below threshold
	var sum float64
	count := 0
	for i := 0; i <= index; i++ {
		sum += sorted[i]
		count++
	}

	if count == 0 {
		return 0
	}

	return -sum / float64(count) // Return as positive value
}

// calculateVolatility calculates annualized volatility
func (s *StatisticalArbitrageStrategy) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// Calculate mean
	var sum float64
	for _, ret := range returns {
		sum += ret
	}
	mean := sum / float64(len(returns))

	// Calculate variance
	var variance float64
	for _, ret := range returns {
		diff := ret - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)

	// Calculate standard deviation and annualize (assuming daily returns)
	stdDev := math.Sqrt(variance)
	annualizedVol := stdDev * math.Sqrt(252) // 252 trading days per year

	return annualizedVol
}

// generateTradingSignal generates a trading signal based on current conditions
func (s *StatisticalArbitrageStrategy) generateTradingSignal() *TradingSignal {
	signal := &TradingSignal{
		Symbol1:   s.symbol1,
		Symbol2:   s.symbol2,
		ZScore:    s.currentZScore,
		Timestamp: time.Now(),
	}

	// Calculate current spread
	if len(s.prices1) > 0 && len(s.prices2) > 0 {
		signal.Spread = s.prices1[len(s.prices1)-1] - (s.ratio * s.prices2[len(s.prices2)-1])
	}

	// Determine signal type and strength
	absZScore := math.Abs(s.currentZScore)

	if absZScore >= s.zScoreEntry {
		if s.currentZScore > 0 {
			signal.Type = SignalShort
		} else {
			signal.Type = SignalLong
		}
		// Signal strength increases with z-score magnitude
		signal.Strength = math.Min(1.0, absZScore/MaxZScoreThreshold)
	} else if absZScore <= s.zScoreExit {
		signal.Type = SignalClose
		signal.Strength = 1.0 - (absZScore / s.zScoreExit)
	} else {
		signal.Type = SignalNone
		signal.Strength = 0.0
	}

	return signal
}

// Helper functions

// getOrderSide returns the order side based on quantity
func getOrderSide(quantity float64) models.OrderSide {
	if quantity > 0 {
		return models.OrderSideBuy
	}
	return models.OrderSideSell
}

// getOppositeSide returns the opposite order side
func getOppositeSide(side models.OrderSide) models.OrderSide {
	if side == models.OrderSideBuy {
		return models.OrderSideSell
	}
	return models.OrderSideBuy
}
