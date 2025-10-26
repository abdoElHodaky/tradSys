package services

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/errors"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// TradeServiceUnified implements the TradeService interface using the new architecture
type TradeServiceUnified struct {
	repository      interfaces.TradeRepository
	positionService interfaces.PositionService
	publisher       interfaces.EventPublisher
	logger          interfaces.Logger
	metrics         interfaces.MetricsCollector
}

// NewTradeServiceUnified creates a new unified trade service
func NewTradeServiceUnified(
	repository interfaces.TradeRepository,
	positionService interfaces.PositionService,
	publisher interfaces.EventPublisher,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *TradeServiceUnified {
	return &TradeServiceUnified{
		repository:      repository,
		positionService: positionService,
		publisher:       publisher,
		logger:          logger,
		metrics:         metrics,
	}
}

// CreateTrade creates a new trade
func (s *TradeServiceUnified) CreateTrade(ctx context.Context, trade *types.Trade) error {
	start := time.Now()
	defer func() {
		s.metrics.RecordTimer("trade_service.create_trade.duration", time.Since(start), map[string]string{
			"symbol": trade.Symbol,
		})
	}()

	s.logger.Info("Creating trade", 
		"trade_id", trade.ID, 
		"symbol", trade.Symbol, 
		"price", trade.Price, 
		"quantity", trade.Quantity)

	// Validate trade
	if err := s.validateTrade(trade); err != nil {
		s.metrics.IncrementCounter("trade_service.validation_failed", map[string]string{
			"symbol": trade.Symbol,
			"error":  errors.GetErrorCode(err).String(),
		})
		return errors.Wrap(err, errors.ErrValidationFailed, "trade validation failed")
	}

	// Set trade metadata
	trade.Timestamp = time.Now()
	trade.Value = trade.CalculateValue()

	// Create the trade in repository
	if err := s.repository.Create(ctx, trade); err != nil {
		s.metrics.IncrementCounter("trade_service.create_failed", map[string]string{
			"symbol": trade.Symbol,
			"error":  "repository_error",
		})
		return errors.Wrap(err, errors.ErrDatabaseConnection, "failed to create trade in repository")
	}

	// Update positions for both users
	if err := s.updatePositions(ctx, trade); err != nil {
		s.logger.Error("Failed to update positions after trade", "error", err, "trade_id", trade.ID)
		// Don't fail the trade creation, but log the error
	}

	// Publish trade event
	if s.publisher != nil {
		event := &interfaces.TradeEvent{
			Type:      interfaces.TradeEventExecuted,
			Trade:     trade,
			Timestamp: time.Now(),
		}
		if err := s.publisher.PublishTradeEvent(ctx, event); err != nil {
			s.logger.Error("Failed to publish trade event", "error", err, "trade_id", trade.ID)
		}
	}

	// Record metrics
	s.metrics.IncrementCounter("trade_service.created", map[string]string{
		"symbol":     trade.Symbol,
		"taker_side": string(trade.TakerSide),
	})
	s.metrics.RecordGauge("trade_service.trade_value", trade.Value, map[string]string{
		"symbol": trade.Symbol,
	})

	s.logger.Info("Trade created successfully", 
		"trade_id", trade.ID, 
		"symbol", trade.Symbol, 
		"value", trade.Value)
	
	return nil
}

// GetTrade retrieves a trade by ID
func (s *TradeServiceUnified) GetTrade(ctx context.Context, tradeID string) (*types.Trade, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordTimer("trade_service.get_trade.duration", time.Since(start), nil)
	}()

	if tradeID == "" {
		return nil, errors.New(errors.ErrInvalidInput, "trade ID cannot be empty")
	}

	trade, err := s.repository.GetByID(ctx, tradeID)
	if err != nil {
		s.metrics.IncrementCounter("trade_service.get_failed", map[string]string{
			"error": "repository_error",
		})
		return nil, errors.Wrap(err, errors.ErrOrderNotFound, "failed to get trade from repository")
	}

	if trade == nil {
		s.metrics.IncrementCounter("trade_service.get_failed", map[string]string{
			"error": "not_found",
		})
		return nil, errors.New(errors.ErrOrderNotFound, "trade not found")
	}

	s.metrics.IncrementCounter("trade_service.get_success", nil)
	return trade, nil
}

// ListTrades lists trades with optional filters
func (s *TradeServiceUnified) ListTrades(ctx context.Context, filters *interfaces.TradeFilters) ([]*types.Trade, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordTimer("trade_service.list_trades.duration", time.Since(start), nil)
	}()

	// Set default filters if not provided
	if filters == nil {
		filters = &interfaces.TradeFilters{
			Limit:  100,
			Offset: 0,
		}
	}

	// Validate and adjust filters
	if filters.Limit <= 0 {
		filters.Limit = 100
	}
	if filters.Limit > 1000 {
		filters.Limit = 1000 // Prevent excessive queries
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	var trades []*types.Trade
	var err error

	// Choose appropriate repository method based on filters
	if filters.Symbol != "" {
		trades, err = s.repository.ListBySymbol(ctx, filters.Symbol, filters.Limit, filters.Offset)
	} else if filters.UserID != "" {
		trades, err = s.repository.ListByUser(ctx, filters.UserID, filters.Limit, filters.Offset)
	} else if filters.StartTime != nil && filters.EndTime != nil {
		trades, err = s.repository.ListByTimeRange(ctx, *filters.StartTime, *filters.EndTime, filters.Limit, filters.Offset)
	} else {
		// Default to time-based query for recent trades
		endTime := time.Now()
		startTime := endTime.Add(-24 * time.Hour) // Last 24 hours
		trades, err = s.repository.ListByTimeRange(ctx, startTime, endTime, filters.Limit, filters.Offset)
	}

	if err != nil {
		s.metrics.IncrementCounter("trade_service.list_failed", map[string]string{
			"error": "repository_error",
		})
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to list trades")
	}

	// Apply additional filters
	filteredTrades := s.applyTradeFilters(trades, filters)

	s.metrics.IncrementCounter("trade_service.list_success", nil)
	s.metrics.RecordGauge("trade_service.trades_returned", float64(len(filteredTrades)), nil)

	return filteredTrades, nil
}

// GetTradesByOrder gets all trades for an order
func (s *TradeServiceUnified) GetTradesByOrder(ctx context.Context, orderID string) ([]*types.Trade, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordTimer("trade_service.get_trades_by_order.duration", time.Since(start), nil)
	}()

	if orderID == "" {
		return nil, errors.New(errors.ErrInvalidInput, "order ID cannot be empty")
	}

	// Get all trades and filter by order ID
	// This is a simplified implementation - in practice, you'd want a more efficient query
	filters := &interfaces.TradeFilters{
		Limit:  1000,
		Offset: 0,
	}
	
	allTrades, err := s.ListTrades(ctx, filters)
	if err != nil {
		return nil, err
	}

	var orderTrades []*types.Trade
	for _, trade := range allTrades {
		if trade.BuyOrderID == orderID || trade.SellOrderID == orderID {
			orderTrades = append(orderTrades, trade)
		}
	}

	s.metrics.RecordGauge("trade_service.order_trades_count", float64(len(orderTrades)), map[string]string{
		"order_id": orderID,
	})

	return orderTrades, nil
}

// GetTradesByUser gets all trades for a user
func (s *TradeServiceUnified) GetTradesByUser(ctx context.Context, userID string, limit, offset int) ([]*types.Trade, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordTimer("trade_service.get_trades_by_user.duration", time.Since(start), nil)
	}()

	if userID == "" {
		return nil, errors.New(errors.ErrInvalidInput, "user ID cannot be empty")
	}

	trades, err := s.repository.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		s.metrics.IncrementCounter("trade_service.get_by_user_failed", map[string]string{
			"error": "repository_error",
		})
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to get trades by user")
	}

	s.metrics.IncrementCounter("trade_service.get_by_user_success", map[string]string{
		"user_id": userID,
	})
	s.metrics.RecordGauge("trade_service.user_trades_count", float64(len(trades)), map[string]string{
		"user_id": userID,
	})

	return trades, nil
}

// GetTradeStatistics returns statistics about trades
func (s *TradeServiceUnified) GetTradeStatistics(ctx context.Context, filters *interfaces.TradeFilters) (*TradeStatistics, error) {
	trades, err := s.ListTrades(ctx, filters)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to get trades for statistics")
	}

	stats := &TradeStatistics{
		TotalTrades: len(trades),
		SymbolStats: make(map[string]*SymbolTradeStats),
	}

	var totalVolume, totalValue float64
	symbolCounts := make(map[string]int)
	sideCounts := make(map[types.OrderSide]int)

	for _, trade := range trades {
		// Aggregate totals
		totalVolume += trade.Quantity
		totalValue += trade.Value

		// Count by symbol
		symbolCounts[trade.Symbol]++

		// Count by taker side
		sideCounts[trade.TakerSide]++

		// Symbol-specific stats
		if _, exists := stats.SymbolStats[trade.Symbol]; !exists {
			stats.SymbolStats[trade.Symbol] = &SymbolTradeStats{
				Symbol: trade.Symbol,
			}
		}
		symbolStats := stats.SymbolStats[trade.Symbol]
		symbolStats.TradeCount++
		symbolStats.TotalVolume += trade.Quantity
		symbolStats.TotalValue += trade.Value

		// Track price range
		if symbolStats.HighPrice == 0 || trade.Price > symbolStats.HighPrice {
			symbolStats.HighPrice = trade.Price
		}
		if symbolStats.LowPrice == 0 || trade.Price < symbolStats.LowPrice {
			symbolStats.LowPrice = trade.Price
		}
		symbolStats.LastPrice = trade.Price
	}

	stats.TotalVolume = totalVolume
	stats.TotalValue = totalValue
	if len(trades) > 0 {
		stats.AverageTradeSize = totalVolume / float64(len(trades))
		stats.AverageTradeValue = totalValue / float64(len(trades))
	}

	// Calculate averages for symbol stats
	for _, symbolStats := range stats.SymbolStats {
		if symbolStats.TradeCount > 0 {
			symbolStats.AveragePrice = symbolStats.TotalValue / symbolStats.TotalVolume
			symbolStats.AverageSize = symbolStats.TotalVolume / float64(symbolStats.TradeCount)
		}
	}

	return stats, nil
}

// validateTrade validates a trade
func (s *TradeServiceUnified) validateTrade(trade *types.Trade) error {
	if trade == nil {
		return errors.New(errors.ErrInvalidInput, "trade cannot be nil")
	}

	if trade.ID == "" {
		return errors.New(errors.ErrMissingField, "trade ID is required")
	}

	if trade.Symbol == "" {
		return errors.New(errors.ErrMissingField, "symbol is required")
	}

	if trade.BuyOrderID == "" {
		return errors.New(errors.ErrMissingField, "buy order ID is required")
	}

	if trade.SellOrderID == "" {
		return errors.New(errors.ErrMissingField, "sell order ID is required")
	}

	if trade.Price <= 0 {
		return errors.New(errors.ErrInvalidPrice, "price must be positive")
	}

	if trade.Quantity <= 0 {
		return errors.New(errors.ErrInvalidQuantity, "quantity must be positive")
	}

	if trade.BuyUserID == "" {
		return errors.New(errors.ErrMissingField, "buy user ID is required")
	}

	if trade.SellUserID == "" {
		return errors.New(errors.ErrMissingField, "sell user ID is required")
	}

	if trade.TakerSide != types.OrderSideBuy && trade.TakerSide != types.OrderSideSell {
		return errors.New(errors.ErrInvalidInput, "invalid taker side")
	}

	return nil
}

// updatePositions updates positions for both users involved in the trade
func (s *TradeServiceUnified) updatePositions(ctx context.Context, trade *types.Trade) error {
	if s.positionService == nil {
		return nil // Position service not available
	}

	// Update buyer's position
	buyerPosition, err := s.positionService.GetPosition(ctx, trade.BuyUserID, trade.Symbol)
	if err != nil && !errors.Is(err, errors.ErrOrderNotFound) {
		return errors.Wrap(err, errors.ErrInternalError, "failed to get buyer position")
	}

	if buyerPosition == nil {
		// Create new position for buyer
		buyerPosition = &types.Position{
			UserID:        trade.BuyUserID,
			Symbol:        trade.Symbol,
			Quantity:      trade.Quantity,
			AveragePrice:  trade.Price,
			MarketValue:   trade.Value,
			LastUpdated:   time.Now(),
		}
	} else {
		// Update existing position
		totalQuantity := buyerPosition.Quantity + trade.Quantity
		totalValue := (buyerPosition.AveragePrice * buyerPosition.Quantity) + trade.Value
		buyerPosition.AveragePrice = totalValue / totalQuantity
		buyerPosition.Quantity = totalQuantity
		buyerPosition.MarketValue += trade.Value
		buyerPosition.LastUpdated = time.Now()
	}

	if err := s.positionService.UpdatePosition(ctx, buyerPosition); err != nil {
		return errors.Wrap(err, errors.ErrInternalError, "failed to update buyer position")
	}

	// Update seller's position
	sellerPosition, err := s.positionService.GetPosition(ctx, trade.SellUserID, trade.Symbol)
	if err != nil && !errors.Is(err, errors.ErrOrderNotFound) {
		return errors.Wrap(err, errors.ErrInternalError, "failed to get seller position")
	}

	if sellerPosition == nil {
		// Create new position for seller (negative quantity)
		sellerPosition = &types.Position{
			UserID:        trade.SellUserID,
			Symbol:        trade.Symbol,
			Quantity:      -trade.Quantity,
			AveragePrice:  trade.Price,
			MarketValue:   -trade.Value,
			LastUpdated:   time.Now(),
		}
	} else {
		// Update existing position
		sellerPosition.Quantity -= trade.Quantity
		sellerPosition.MarketValue -= trade.Value
		sellerPosition.LastUpdated = time.Now()
		
		// Recalculate average price if position is still open
		if sellerPosition.Quantity != 0 {
			sellerPosition.AveragePrice = sellerPosition.MarketValue / sellerPosition.Quantity
		}
	}

	if err := s.positionService.UpdatePosition(ctx, sellerPosition); err != nil {
		return errors.Wrap(err, errors.ErrInternalError, "failed to update seller position")
	}

	return nil
}

// applyTradeFilters applies additional filters to trades
func (s *TradeServiceUnified) applyTradeFilters(trades []*types.Trade, filters *interfaces.TradeFilters) []*types.Trade {
	if filters == nil {
		return trades
	}

	var filtered []*types.Trade

	for _, trade := range trades {
		// Filter by symbol (already handled in repository query, but double-check)
		if filters.Symbol != "" && trade.Symbol != filters.Symbol {
			continue
		}

		// Filter by user ID (already handled in repository query, but double-check)
		if filters.UserID != "" && trade.BuyUserID != filters.UserID && trade.SellUserID != filters.UserID {
			continue
		}

		// Filter by time range (already handled in repository query, but double-check)
		if filters.StartTime != nil && trade.Timestamp.Before(*filters.StartTime) {
			continue
		}
		if filters.EndTime != nil && trade.Timestamp.After(*filters.EndTime) {
			continue
		}

		filtered = append(filtered, trade)
	}

	return filtered
}

// TradeStatistics contains statistics about trades
type TradeStatistics struct {
	TotalTrades       int                           `json:"total_trades"`
	TotalVolume       float64                       `json:"total_volume"`
	TotalValue        float64                       `json:"total_value"`
	AverageTradeSize  float64                       `json:"average_trade_size"`
	AverageTradeValue float64                       `json:"average_trade_value"`
	SymbolStats       map[string]*SymbolTradeStats  `json:"symbol_stats"`
}

// SymbolTradeStats contains statistics for a specific symbol
type SymbolTradeStats struct {
	Symbol       string  `json:"symbol"`
	TradeCount   int     `json:"trade_count"`
	TotalVolume  float64 `json:"total_volume"`
	TotalValue   float64 `json:"total_value"`
	AveragePrice float64 `json:"average_price"`
	AverageSize  float64 `json:"average_size"`
	HighPrice    float64 `json:"high_price"`
	LowPrice     float64 `json:"low_price"`
	LastPrice    float64 `json:"last_price"`
}
