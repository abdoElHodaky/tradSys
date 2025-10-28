package engine

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Service represents a risk management service
type Service struct {
	// OrderEngine is the order matching engine
	OrderEngine *order_matching.Engine
	// OrderService is the order management service
	OrderService *orders.OrderService
	// Positions is a map of user ID and symbol to position
	Positions map[string]map[string]*Position
	// CircuitBreakers is a map of symbol to circuit breaker
	CircuitBreakers map[string]*RealtimeCircuitBreaker
	// PositionCache is a cache for frequently accessed positions
	PositionCache *cache.Cache
	// RiskLimitCache is a cache for frequently accessed risk limits
	RiskLimitCache *cache.Cache
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
	// Batch processing channel for risk operations
	riskBatchChan chan RiskOperation
	// Market data channel for price updates
	marketDataChan chan MarketDataUpdate
	
	// New components
	calculator     *RiskCalculator
	monitor        *RiskMonitor
	limitsManager  *LimitsManager
}

// NewService creates a new risk management service
func NewService(orderEngine *order_matching.Engine, orderService *orders.OrderService, logger *zap.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create new components
	calculator := NewRiskCalculator(logger)
	monitor := NewRiskMonitor(logger)
	limitsManager := NewLimitsManager(logger, calculator)

	service := &Service{
		OrderEngine:     orderEngine,
		OrderService:    orderService,
		Positions:       make(map[string]map[string]*Position),
		CircuitBreakers: make(map[string]*RealtimeCircuitBreaker),
		PositionCache:   cache.New(5*time.Minute, 10*time.Minute),
		RiskLimitCache:  cache.New(5*time.Minute, 10*time.Minute),
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		riskBatchChan:   make(chan RiskOperation, 1000),
		marketDataChan:  make(chan MarketDataUpdate, 1000),
		calculator:      calculator,
		monitor:         monitor,
		limitsManager:   limitsManager,
	}

	// Start background processes
	go service.processBatchOperations()
	go service.processMarketData()
	go service.subscribeToTrades()
	
	// Start monitor
	monitor.Start()

	logger.Info("Risk management service started")
	return service
}

// Stop stops the risk management service
func (s *Service) Stop() {
	s.cancel()
	s.monitor.Stop()
	s.logger.Info("Risk management service stopped")
}

// CheckRiskLimits checks risk limits for an order using the new components
func (s *Service) CheckRiskLimits(ctx context.Context, userID, symbol string, orderSize, currentPrice float64) (*RiskCheckResult, error) {
	// Check circuit breaker first
	s.mu.RLock()
	cb, exists := s.CircuitBreakers[symbol]
	if exists && cb.IsTripped() {
		s.mu.RUnlock()
		return &RiskCheckResult{
			Passed:     false,
			RiskLevel:  RiskLevelHigh,
			Violations: []string{"Circuit breaker triggered"},
			Warnings:   []string{},
			CheckedAt:  time.Now(),
		}, nil
	}
	s.mu.RUnlock()

	// Use the new limits manager for comprehensive checking
	return s.limitsManager.CheckAllLimits(ctx, userID, symbol, orderSize, currentPrice)
}

// SetRiskLimit sets a risk limit using the new limits manager
func (s *Service) SetRiskLimit(userID string, limit *RiskLimit) error {
	return s.limitsManager.SetLimit(userID, limit)
}

// GetRiskLimit gets a risk limit using the new limits manager
func (s *Service) GetRiskLimit(userID string, limitType RiskLimitType) (*RiskLimit, error) {
	return s.limitsManager.GetLimit(userID, limitType)
}

// UpdatePosition updates a position and triggers monitoring
func (s *Service) UpdatePosition(userID, symbol string, quantityDelta, price float64) {
	// Update position in memory
	s.mu.Lock()
	if s.Positions[userID] == nil {
		s.Positions[userID] = make(map[string]*Position)
	}
	
	position, exists := s.Positions[userID][symbol]
	if !exists {
		position = &Position{
			UserID:    userID,
			Symbol:    symbol,
			Quantity:  0,
			AvgPrice:  0,
			UpdatedAt: time.Now(),
		}
		s.Positions[userID][symbol] = position
	}
	
	// Update position
	oldQuantity := position.Quantity
	position.Quantity += quantityDelta
	
	// Update average price
	if quantityDelta > 0 {
		totalValue := (oldQuantity * position.AvgPrice) + (quantityDelta * price)
		position.AvgPrice = totalValue / position.Quantity
	}
	
	position.UpdatedAt = time.Now()
	s.mu.Unlock()
	
	// Trigger monitoring
	s.monitor.MonitorPosition(userID, symbol, position.Quantity, price)
}

// CalculatePortfolioRisk calculates portfolio risk using the new calculator
func (s *Service) CalculatePortfolioRisk(userID string) (*RiskCheckResult, error) {
	s.mu.RLock()
	userPositions, exists := s.Positions[userID]
	if !exists {
		s.mu.RUnlock()
		return &RiskCheckResult{
			Passed:     true,
			RiskLevel:  RiskLevelLow,
			Violations: []string{},
			Warnings:   []string{},
			CheckedAt:  time.Now(),
		}, nil
	}
	
	// Convert positions to value map
	positions := make(map[string]float64)
	for symbol, position := range userPositions {
		positions[symbol] = position.Quantity * position.AvgPrice
	}
	s.mu.RUnlock()
	
	return s.calculator.CalculatePortfolioRisk(userID, positions)
}

// GetPosition gets a position for a user and symbol
func (s *Service) GetPosition(userID, symbol string) (*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	userPositions, exists := s.Positions[userID]
	if !exists {
		return nil, ErrPositionNotFound
	}
	
	position, exists := userPositions[symbol]
	if !exists {
		return nil, ErrPositionNotFound
	}
	
	return position, nil
}

// GetAllPositions gets all positions for a user
func (s *Service) GetAllPositions(userID string) (map[string]*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	userPositions, exists := s.Positions[userID]
	if !exists {
		return make(map[string]*Position), nil
	}
	
	// Return a copy to prevent external modification
	result := make(map[string]*Position)
	for k, v := range userPositions {
		result[k] = v
	}
	
	return result, nil
}

// processBatchOperations processes batch operations (simplified)
func (s *Service) processBatchOperations() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	batch := make([]RiskOperation, 0, 100)
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if len(batch) > 0 {
				s.processBatch(batch)
				batch = batch[:0]
			}
		case op := <-s.riskBatchChan:
			batch = append(batch, op)
			if len(batch) >= 100 {
				s.processBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// processBatch processes a batch of operations (simplified)
func (s *Service) processBatch(batch []RiskOperation) {
	for _, op := range batch {
		switch op.OpType {
		case "update_position":
			s.processUpdatePosition(op)
		case "check_limit":
			s.processCheckLimit(op)
		default:
			s.logger.Warn("Unknown operation type", zap.String("op_type", op.OpType))
		}
	}
}

// processUpdatePosition processes a position update
func (s *Service) processUpdatePosition(op RiskOperation) {
	quantityDelta, ok := op.Data["quantity_delta"].(float64)
	if !ok {
		op.ResultCh <- RiskOperationResult{Success: false, Error: ErrInvalidData}
		return
	}
	
	price, ok := op.Data["price"].(float64)
	if !ok {
		op.ResultCh <- RiskOperationResult{Success: false, Error: ErrInvalidData}
		return
	}
	
	s.UpdatePosition(op.UserID, op.Symbol, quantityDelta, price)
	op.ResultCh <- RiskOperationResult{Success: true}
}

// processCheckLimit processes a limit check
func (s *Service) processCheckLimit(op RiskOperation) {
	orderSize, ok := op.Data["order_size"].(float64)
	if !ok {
		op.ResultCh <- RiskOperationResult{Success: false, Error: ErrInvalidData}
		return
	}
	
	currentPrice, ok := op.Data["current_price"].(float64)
	if !ok {
		op.ResultCh <- RiskOperationResult{Success: false, Error: ErrInvalidData}
		return
	}
	
	result, err := s.CheckRiskLimits(s.ctx, op.UserID, op.Symbol, orderSize, currentPrice)
	op.ResultCh <- RiskOperationResult{
		Success: err == nil,
		Error:   err,
		Data:    result,
	}
}

// processMarketData processes market data updates (simplified)
func (s *Service) processMarketData() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case update := <-s.marketDataChan:
			s.updateUnrealizedPnL(update.Symbol, update.Price)
			s.checkCircuitBreakers()
		}
	}
}

// updateUnrealizedPnL updates unrealized P&L for all positions (simplified)
func (s *Service) updateUnrealizedPnL(symbol string, price float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for userID, userPositions := range s.Positions {
		if position, exists := userPositions[symbol]; exists {
			// Monitor the position with new price
			s.monitor.MonitorPosition(userID, symbol, position.Quantity, price)
		}
	}
}

// checkCircuitBreakers checks circuit breakers (simplified)
func (s *Service) checkCircuitBreakers() {
	// Circuit breaker logic would go here
	// For now, this is a placeholder
}

// subscribeToTrades subscribes to trades from the order matching engine
func (s *Service) subscribeToTrades() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case trade := <-s.OrderEngine.TradeChannel:
			s.processTrade(trade)
		}
	}
}

// processTrade processes a trade for risk management
func (s *Service) processTrade(trade *order_matching.Trade) {
	// Get orders and update positions
	buyOrder, err := s.OrderEngine.GetOrder(trade.Symbol, trade.BuyOrderID)
	if err != nil {
		s.logger.Error("Failed to get buy order", zap.String("order_id", trade.BuyOrderID), zap.Error(err))
		return
	}
	
	sellOrder, err := s.OrderEngine.GetOrder(trade.Symbol, trade.SellOrderID)
	if err != nil {
		s.logger.Error("Failed to get sell order", zap.String("order_id", trade.SellOrderID), zap.Error(err))
		return
	}
	
	// Update positions
	s.UpdatePosition(buyOrder.UserID, trade.Symbol, trade.Quantity, trade.Price)
	s.UpdatePosition(sellOrder.UserID, trade.Symbol, -trade.Quantity, trade.Price)
	
	// Send market data update
	s.marketDataChan <- MarketDataUpdate{
		Symbol:    trade.Symbol,
		Price:     trade.Price,
		Timestamp: trade.Timestamp,
	}
}
