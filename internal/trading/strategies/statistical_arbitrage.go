package strategies

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/statistics"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StatisticalArbitrageStrategy implements a pairs trading strategy
type StatisticalArbitrageStrategy struct {
	*BaseStrategy

	// Strategy parameters
	pairID         string
	symbol1        string
	symbol2        string
	ratio          float64
	zScoreEntry    float64
	zScoreExit     float64
	positionSize   float64
	maxPositions   int
	lookbackPeriod int
	updateInterval time.Duration

	// Strategy state
	prices1       []float64
	prices2       []float64
	spread        []float64
	positions     map[string]*models.PairPosition
	spreadMean    float64
	spreadStdDev  float64
	currentZScore float64
	lastUpdate    time.Time

	// Services
	orderService orders.OrderService
	pairRepo     *repositories.PairRepository
	statsRepo    *repositories.PairStatisticsRepository
	positionRepo *repositories.PairPositionRepository
}

// StatisticalArbitrageParams contains parameters for the statistical arbitrage strategy
type StatisticalArbitrageParams struct {
	Name           string
	PairID         string
	Symbol1        string
	Symbol2        string
	Ratio          float64
	ZScoreEntry    float64
	ZScoreExit     float64
	PositionSize   float64
	MaxPositions   int
	LookbackPeriod int
	UpdateInterval time.Duration
}

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

// OnMarketData processes market data updates
func (s *StatisticalArbitrageStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
	if !s.IsRunning() {
		return nil
	}

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

	// Only proceed if we have enough data for both symbols
	if len(s.prices1) < s.lookbackPeriod || len(s.prices2) < s.lookbackPeriod {
		return nil
	}

	// Check if it's time to update statistics
	if time.Since(s.lastUpdate) >= s.updateInterval {
		if err := s.updateStatistics(ctx); err != nil {
			s.logger.Error("Failed to update statistics",
				zap.Error(err),
				zap.String("pair_id", s.pairID))
			return err
		}

		// Check for trading signals
		if err := s.checkForEntrySignals(ctx); err != nil {
			s.logger.Error("Failed to check for entry signals",
				zap.Error(err),
				zap.String("pair_id", s.pairID))
			return err
		}

		if err := s.checkForExitSignals(ctx); err != nil {
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

// checkForEntrySignals checks for entry signals
func (s *StatisticalArbitrageStrategy) checkForEntrySignals(ctx context.Context) error {
	// If we have reached max positions, don't enter new ones
	if len(s.positions) >= s.maxPositions {
		return nil
	}

	// Check for entry signals based on z-score
	if s.currentZScore <= -s.zScoreEntry {
		// Z-score is below negative threshold, go long pair (buy symbol1, sell symbol2)
		return s.enterLongPosition(ctx)
	} else if s.currentZScore >= s.zScoreEntry {
		// Z-score is above positive threshold, go short pair (sell symbol1, buy symbol2)
		return s.enterShortPosition(ctx)
	}

	return nil
}

// checkForExitSignals checks for exit signals
func (s *StatisticalArbitrageStrategy) checkForExitSignals(ctx context.Context) error {
	// Check each position for exit signals
	for id, position := range s.positions {
		if position.Status != "open" {
			continue
		}

		// Update position with current prices
		position.CurrentPrice1 = s.prices1[len(s.prices1)-1]
		position.CurrentPrice2 = s.prices2[len(s.prices2)-1]
		position.CurrentSpread = position.CurrentPrice1 - (s.ratio * position.CurrentPrice2)
		position.CurrentZScore = statistics.CalculateZScore(position.CurrentSpread, s.spreadMean, s.spreadStdDev)

		// Calculate current P&L
		pnl1 := position.Quantity1 * (position.CurrentPrice1 - position.EntryPrice1)
		pnl2 := position.Quantity2 * (position.EntryPrice2 - position.CurrentPrice2)
		position.PnL = pnl1 + pnl2

		// Update position in database
		if err := s.positionRepo.Update(ctx, position); err != nil {
			s.logger.Error("Failed to update position",
				zap.Error(err),
				zap.String("pair_id", s.pairID),
				zap.String("position_id", id))
		}

		// Check for exit signals
		if position.EntryZScore < 0 && position.CurrentZScore >= -s.zScoreExit {
			// Long position and z-score has mean-reverted
			return s.exitPosition(ctx, id, position)
		} else if position.EntryZScore > 0 && position.CurrentZScore <= s.zScoreExit {
			// Short position and z-score has mean-reverted
			return s.exitPosition(ctx, id, position)
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

// exitPosition exits a position
func (s *StatisticalArbitrageStrategy) exitPosition(ctx context.Context, id string, position *models.PairPosition) error {
	// Create orders to close the position
	var order1, order2 *models.Order

	if position.Quantity1 > 0 {
		// Long position in symbol1, need to sell
		order1 = &models.Order{
			OrderID:   uuid.New().String(),
			Symbol:    position.Symbol1,
			Side:      models.OrderSideSell,
			Type:      models.OrderTypeMarket,
			Quantity:  math.Abs(position.Quantity1),
			Price:     s.prices1[len(s.prices1)-1],
			Strategy:  s.name,
			Timestamp: time.Now(),
		}
	} else {
		// Short position in symbol1, need to buy
		order1 = &models.Order{
			OrderID:   uuid.New().String(),
			Symbol:    position.Symbol1,
			Side:      models.OrderSideBuy,
			Type:      models.OrderTypeMarket,
			Quantity:  math.Abs(position.Quantity1),
			Price:     s.prices1[len(s.prices1)-1],
			Strategy:  s.name,
			Timestamp: time.Now(),
		}
	}

	if position.Quantity2 > 0 {
		// Long position in symbol2, need to sell
		order2 = &models.Order{
			OrderID:   uuid.New().String(),
			Symbol:    position.Symbol2,
			Side:      models.OrderSideSell,
			Type:      models.OrderTypeMarket,
			Quantity:  math.Abs(position.Quantity2),
			Price:     s.prices2[len(s.prices2)-1],
			Strategy:  s.name,
			Timestamp: time.Now(),
		}
	} else {
		// Short position in symbol2, need to buy
		order2 = &models.Order{
			OrderID:   uuid.New().String(),
			Symbol:    position.Symbol2,
			Side:      models.OrderSideBuy,
			Type:      models.OrderTypeMarket,
			Quantity:  math.Abs(position.Quantity2),
			Price:     s.prices2[len(s.prices2)-1],
			Strategy:  s.name,
			Timestamp: time.Now(),
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
	pnl2 := position.Quantity2 * (position.EntryPrice2 - position.CurrentPrice2)
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

	s.logger.Info("Exited pair position",
		zap.String("pair_id", s.pairID),
		zap.String("position_id", id),
		zap.Float64("entry_z_score", position.EntryZScore),
		zap.Float64("exit_z_score", position.CurrentZScore),
		zap.Float64("pnl", position.PnL))

	return nil
}

// GetParameters returns the strategy parameters
func (s *StatisticalArbitrageStrategy) GetParameters() map[string]interface{} {
	params := s.BaseStrategy.GetParameters()
	params["pair_id"] = s.pairID
	params["symbol1"] = s.symbol1
	params["symbol2"] = s.symbol2
	params["ratio"] = s.ratio
	params["z_score_entry"] = s.zScoreEntry
	params["z_score_exit"] = s.zScoreExit
	params["position_size"] = s.positionSize
	params["max_positions"] = s.maxPositions
	params["lookback_period"] = s.lookbackPeriod
	params["update_interval"] = s.updateInterval.String()
	return params
}

// SetParameters sets the strategy parameters
func (s *StatisticalArbitrageStrategy) SetParameters(params map[string]interface{}) error {
	if err := s.BaseStrategy.SetParameters(params); err != nil {
		return err
	}

	if v, ok := params["z_score_entry"]; ok {
		if val, ok := v.(float64); ok {
			s.zScoreEntry = val
		}
	}

	if v, ok := params["z_score_exit"]; ok {
		if val, ok := v.(float64); ok {
			s.zScoreExit = val
		}
	}

	if v, ok := params["position_size"]; ok {
		if val, ok := v.(float64); ok {
			s.positionSize = val
		}
	}

	if v, ok := params["max_positions"]; ok {
		if val, ok := v.(int); ok {
			s.maxPositions = val
		}
	}

	if v, ok := params["lookback_period"]; ok {
		if val, ok := v.(int); ok {
			s.lookbackPeriod = val
		}
	}

	if v, ok := params["update_interval"]; ok {
		if val, ok := v.(string); ok {
			if d, err := time.ParseDuration(val); err == nil {
				s.updateInterval = d
			}
		}
	}

	s.logger.Info("Statistical arbitrage strategy parameters updated",
		zap.String("pair_id", s.pairID),
		zap.Float64("z_score_entry", s.zScoreEntry),
		zap.Float64("z_score_exit", s.zScoreExit),
		zap.Float64("position_size", s.positionSize),
		zap.Int("max_positions", s.maxPositions),
		zap.Int("lookback_period", s.lookbackPeriod),
		zap.Duration("update_interval", s.updateInterval))

	return nil
}
