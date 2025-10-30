package strategies

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"go.uber.org/zap"
)

// NewStatisticalArbitrageStrategy creates a new statistical arbitrage strategy
func NewStatisticalArbitrageStrategy(
	logger *zap.Logger,
	params StatisticalArbitrageParams,
	orderService orders.OrderService,
	pairRepo *repositories.PairRepository,
	statsRepo *repositories.PairStatisticsRepository,
	positionRepo *repositories.PairPositionRepository,
) *StatisticalArbitrageStrategy {
	return &StatisticalArbitrageStrategy{
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
		orderService:   orderService,
		pairRepo:       pairRepo,
		statsRepo:      statsRepo,
		positionRepo:   positionRepo,
	}
}

// Initialize initializes the strategy
func (s *StatisticalArbitrageStrategy) Initialize(ctx context.Context) error {
	if err := s.BaseStrategy.Initialize(ctx); err != nil {
		return err
	}

	// Load historical data for both symbols
	// This would typically come from a market data service
	// For now, we'll just initialize empty slices
	s.prices1 = make([]float64, 0, s.lookbackPeriod)
	s.prices2 = make([]float64, 0, s.lookbackPeriod)
	s.spread = make([]float64, 0, s.lookbackPeriod)

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

	s.logger.Info("Statistical arbitrage strategy initialized",
		zap.String("pair_id", s.pairID),
		zap.String("symbol1", s.symbol1),
		zap.String("symbol2", s.symbol2),
		zap.Float64("ratio", s.ratio),
		zap.Float64("z_score_entry", s.zScoreEntry),
		zap.Float64("z_score_exit", s.zScoreExit))

	return nil
}

// Start starts the strategy
func (s *StatisticalArbitrageStrategy) Start(ctx context.Context) error {
	if err := s.BaseStrategy.Start(ctx); err != nil {
		return err
	}

	s.logger.Info("Statistical arbitrage strategy started",
		zap.String("pair_id", s.pairID),
		zap.String("symbol1", s.symbol1),
		zap.String("symbol2", s.symbol2))

	return nil
}

// Stop stops the strategy
func (s *StatisticalArbitrageStrategy) Stop(ctx context.Context) error {
	if err := s.BaseStrategy.Stop(ctx); err != nil {
		return err
	}

	s.logger.Info("Statistical arbitrage strategy stopped",
		zap.String("pair_id", s.pairID))

	return nil
}

// OnMarketData handles market data updates
func (s *StatisticalArbitrageStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketData) error {
	// Check if this data is for one of our symbols
	if data.Symbol != s.symbol1 && data.Symbol != s.symbol2 {
		return nil
	}

	// Update price series
	if data.Symbol == s.symbol1 {
		s.updatePriceSeries(ctx, &s.prices1, data.Price)
	} else if data.Symbol == s.symbol2 {
		s.updatePriceSeries(ctx, &s.prices2, data.Price)
	}

	// Check if we should update statistics and generate signals
	if time.Since(s.lastUpdate) >= s.updateInterval {
		// Update statistics
		if err := s.updateStatistics(ctx); err != nil {
			s.logger.Error("Failed to update statistics",
				zap.Error(err),
				zap.String("pair_id", s.pairID))
			return err
		}

		// Check for entry signals
		if err := s.checkEntrySignals(ctx); err != nil {
			s.logger.Error("Failed to check for entry signals",
				zap.Error(err),
				zap.String("pair_id", s.pairID))
			return err
		}

		// Check for exit signals
		if err := s.checkExitSignals(ctx); err != nil {
			s.logger.Error("Failed to check for exit signals",
				zap.Error(err),
				zap.String("pair_id", s.pairID))
			return err
		}

		s.lastUpdate = time.Now()
	}

	return nil
}

// updatePriceSeries updates a price series with a new price
func (s *StatisticalArbitrageStrategy) updatePriceSeries(ctx context.Context, prices *[]float64, price float64) {
	// Add the new price
	*prices = append(*prices, price)

	// Trim the series if it exceeds the lookback period
	if len(*prices) > s.lookbackPeriod {
		*prices = (*prices)[len(*prices)-s.lookbackPeriod:]
	}
}

// GetCurrentPositions returns the current positions
func (s *StatisticalArbitrageStrategy) GetCurrentPositions() map[string]*models.PairPosition {
	return s.positions
}

// GetPairMetrics returns current pair metrics
func (s *StatisticalArbitrageStrategy) GetPairMetrics() *PairMetrics {
	return &PairMetrics{
		PairID:         s.pairID,
		Symbol1:        s.symbol1,
		Symbol2:        s.symbol2,
		SpreadMean:     s.spreadMean,
		SpreadStdDev:   s.spreadStdDev,
		CurrentZScore:  s.currentZScore,
		LastUpdate:     s.lastUpdate,
		SampleSize:     len(s.spread),
	}
}

// GetPerformanceMetrics calculates and returns performance metrics
func (s *StatisticalArbitrageStrategy) GetPerformanceMetrics(ctx context.Context) (*PerformanceMetrics, error) {
	// Get all closed positions for this strategy
	closedPositions, err := s.positionRepo.GetClosedPositions(ctx, s.pairID)
	if err != nil {
		return nil, fmt.Errorf("failed to get closed positions: %w", err)
	}

	metrics := &PerformanceMetrics{
		LastUpdated: time.Now(),
	}

	if len(closedPositions) == 0 {
		return metrics, nil
	}

	// Calculate basic metrics
	metrics.TotalTrades = len(closedPositions)
	
	var totalPnL float64
	var wins, losses []float64
	var consecutiveWins, consecutiveLosses int
	var maxConsecutiveWins, maxConsecutiveLosses int
	var maxDrawdown float64
	var runningPnL float64
	var peak float64

	for _, pos := range closedPositions {
		pnl := pos.RealizedPnL
		totalPnL += pnl
		runningPnL += pnl

		// Track peak and drawdown
		if runningPnL > peak {
			peak = runningPnL
		}
		drawdown := (peak - runningPnL) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}

		// Categorize wins and losses
		if pnl > 0 {
			wins = append(wins, pnl)
			metrics.WinningTrades++
			consecutiveWins++
			consecutiveLosses = 0
			if consecutiveWins > maxConsecutiveWins {
				maxConsecutiveWins = consecutiveWins
			}
		} else {
			losses = append(losses, -pnl) // Store as positive for calculation
			metrics.LosingTrades++
			consecutiveLosses++
			consecutiveWins = 0
			if consecutiveLosses > maxConsecutiveLosses {
				maxConsecutiveLosses = consecutiveLosses
			}
		}
	}

	metrics.TotalPnL = totalPnL
	metrics.MaxDrawdown = maxDrawdown
	metrics.MaxConsecutiveWins = maxConsecutiveWins
	metrics.MaxConsecutiveLosses = maxConsecutiveLosses

	// Calculate win rate
	if metrics.TotalTrades > 0 {
		metrics.WinRate = float64(metrics.WinningTrades) / float64(metrics.TotalTrades)
	}

	// Calculate average win/loss
	if len(wins) > 0 {
		var sumWins float64
		for _, win := range wins {
			sumWins += win
		}
		metrics.AverageWin = sumWins / float64(len(wins))
	}

	if len(losses) > 0 {
		var sumLosses float64
		for _, loss := range losses {
			sumLosses += loss
		}
		metrics.AverageLoss = sumLosses / float64(len(losses))
	}

	// Calculate profit factor
	if metrics.AverageLoss > 0 {
		grossProfit := metrics.AverageWin * float64(len(wins))
		grossLoss := metrics.AverageLoss * float64(len(losses))
		if grossLoss > 0 {
			metrics.ProfitFactor = grossProfit / grossLoss
		}
	}

	return metrics, nil
}

// GetRiskMetrics calculates and returns risk metrics
func (s *StatisticalArbitrageStrategy) GetRiskMetrics(ctx context.Context) (*RiskMetrics, error) {
	// Get recent PnL data for risk calculations
	positions, err := s.positionRepo.GetRecentPositions(ctx, s.pairID, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent positions: %w", err)
	}

	metrics := &RiskMetrics{}

	if len(positions) == 0 {
		return metrics, nil
	}

	// Extract PnL series
	pnlSeries := make([]float64, len(positions))
	var currentExposure float64
	var maxLeverage float64

	for i, pos := range positions {
		pnlSeries[i] = pos.RealizedPnL
		
		// Calculate current exposure (sum of absolute position values)
		if pos.Status == "OPEN" {
			exposure := pos.Quantity1*pos.EntryPrice1 + pos.Quantity2*pos.EntryPrice2
			currentExposure += exposure
			
			// Calculate leverage (simplified)
			if pos.InitialCapital > 0 {
				leverage := exposure / pos.InitialCapital
				if leverage > maxLeverage {
					maxLeverage = leverage
				}
			}
		}
	}

	metrics.CurrentExposure = currentExposure
	metrics.MaxLeverage = maxLeverage

	// Calculate VaR and other risk metrics using the PnL series
	if len(pnlSeries) >= 20 { // Need sufficient data
		metrics.VaR95 = s.calculateVaR(pnlSeries, 0.95)
		metrics.VaR99 = s.calculateVaR(pnlSeries, 0.99)
		metrics.ExpectedShortfall = s.calculateExpectedShortfall(pnlSeries, 0.95)
		metrics.Volatility = s.calculateVolatility(pnlSeries)
	}

	return metrics, nil
}

// ValidateParameters validates strategy parameters
func (s *StatisticalArbitrageStrategy) ValidateParameters() error {
	if s.pairID == "" {
		return fmt.Errorf("pair ID cannot be empty")
	}
	if s.symbol1 == "" || s.symbol2 == "" {
		return fmt.Errorf("symbols cannot be empty")
	}
	if s.symbol1 == s.symbol2 {
		return fmt.Errorf("symbols must be different")
	}
	if s.ratio <= 0 {
		return fmt.Errorf("ratio must be positive")
	}
	if s.zScoreEntry <= 0 {
		return fmt.Errorf("z-score entry threshold must be positive")
	}
	if s.zScoreExit < 0 {
		return fmt.Errorf("z-score exit threshold must be non-negative")
	}
	if s.zScoreExit >= s.zScoreEntry {
		return fmt.Errorf("z-score exit must be less than entry threshold")
	}
	if s.positionSize <= 0 {
		return fmt.Errorf("position size must be positive")
	}
	if s.maxPositions <= 0 {
		return fmt.Errorf("max positions must be positive")
	}
	if s.lookbackPeriod < MinSampleSize {
		return fmt.Errorf("lookback period must be at least %d", MinSampleSize)
	}
	if s.updateInterval <= 0 {
		return fmt.Errorf("update interval must be positive")
	}

	return nil
}

// GetStrategyInfo returns basic information about the strategy
func (s *StatisticalArbitrageStrategy) GetStrategyInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":             s.Name(),
		"type":             "statistical_arbitrage",
		"pair_id":          s.pairID,
		"symbol1":          s.symbol1,
		"symbol2":          s.symbol2,
		"ratio":            s.ratio,
		"z_score_entry":    s.zScoreEntry,
		"z_score_exit":     s.zScoreExit,
		"position_size":    s.positionSize,
		"max_positions":    s.maxPositions,
		"lookback_period":  s.lookbackPeriod,
		"update_interval":  s.updateInterval.String(),
		"current_z_score":  s.currentZScore,
		"spread_mean":      s.spreadMean,
		"spread_std_dev":   s.spreadStdDev,
		"active_positions": len(s.positions),
		"last_update":      s.lastUpdate,
		"status":           s.Status().String(),
	}
}
