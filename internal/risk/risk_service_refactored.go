package risk

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"github.com/abdoElHodaky/tradSys/pkg/common"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Service represents a refactored risk management service
type Service struct {
	*common.BaseService
	
	// Core components
	calculator     *RiskCalculator
	monitor        *RiskMonitor
	limitsManager  *RiskLimitsManager
	
	// External dependencies
	orderEngine    *order_matching.Engine
	orderService   *orders.Service
	
	// Position management
	positions      map[string]map[string]*riskengine.Position
	positionCache  *cache.Cache
	
	// Thread safety
	mu             sync.RWMutex
	
	// Market data processing
	marketDataChan chan MarketDataUpdate
}

// NewService creates a new refactored risk management service
func NewService(orderEngine *order_matching.Engine, orderService *orders.Service, logger *zap.Logger) *Service {
	baseService := common.NewBaseService("risk-service", "1.0.0", logger)
	
	service := &Service{
		BaseService:    baseService,
		calculator:     NewRiskCalculator(logger),
		monitor:        NewRiskMonitor(logger),
		limitsManager:  NewRiskLimitsManager(logger),
		orderEngine:    orderEngine,
		orderService:   orderService,
		positions:      make(map[string]map[string]*riskengine.Position),
		positionCache:  cache.New(5*time.Minute, 10*time.Minute),
		marketDataChan: make(chan MarketDataUpdate, 1000),
	}
	
	// Set up alert callback
	service.monitor.AddAlertCallback(service.handleRiskAlert)
	
	return service
}

// Start starts the risk service
func (s *Service) Start(ctx context.Context) error {
	if err := s.BaseService.Start(ctx); err != nil {
		return err
	}
	
	s.Logger.Info("Starting risk service components")
	
	// Start monitor
	if err := s.monitor.Start(); err != nil {
		s.Logger.Error("Failed to start risk monitor", zap.Error(err))
		return err
	}
	
	// Start market data processor
	go s.processMarketData()
	
	// Subscribe to trades from the order matching engine
	go s.subscribeToTrades()
	
	s.Logger.Info("Risk service started successfully")
	return nil
}

// Stop stops the risk service
func (s *Service) Stop(ctx context.Context) error {
	s.Logger.Info("Stopping risk service")
	
	// Stop monitor
	if err := s.monitor.Stop(); err != nil {
		s.Logger.Error("Failed to stop risk monitor", zap.Error(err))
	}
	
	// Stop limits manager
	s.limitsManager.Stop()
	
	return s.BaseService.Stop(ctx)
}

// Health returns the health status of the risk service
func (s *Service) Health() common.HealthStatus {
	baseHealth := s.BaseService.Health()
	
	// Add risk-specific health checks
	details := baseHealth.Details
	if details == nil {
		details = make(map[string]interface{})
	}
	
	// Check position cache
	details["position_cache_items"] = s.positionCache.ItemCount()
	
	// Check limits manager stats
	if stats, err := s.limitsManager.GetRiskLimitStats(context.Background()); err == nil {
		details["risk_limits"] = stats
	}
	
	// Check market data flow
	details["market_data_channel_size"] = len(s.marketDataChan)
	
	return common.HealthStatus{
		Status:    baseHealth.Status,
		Timestamp: time.Now(),
		Details:   details,
	}
}

// CheckOrderRisk performs comprehensive risk checks on an order
func (s *Service) CheckOrderRisk(ctx context.Context, userID, symbol string, quantity, price float64, side string) (*RiskCheckResult, error) {
	// Get user positions
	userPositions := s.getUserPositions(userID)
	
	// Get applicable risk limits
	limits, err := s.limitsManager.GetRiskLimits(ctx, userID)
	if err != nil {
		s.Logger.Error("Failed to get risk limits", zap.Error(err))
		return nil, err
	}
	
	// Perform risk calculation
	result, err := s.calculator.CheckOrderRisk(ctx, userID, symbol, quantity, price, side, userPositions, limits)
	if err != nil {
		s.Logger.Error("Risk check failed", zap.Error(err))
		return nil, err
	}
	
	// Log risk check result
	s.Logger.Info("Risk check completed",
		zap.String("user_id", userID),
		zap.String("symbol", symbol),
		zap.Bool("passed", result.Passed),
		zap.String("risk_level", string(result.RiskLevel)))
	
	return result, nil
}

// AddRiskLimit adds a risk limit
func (s *Service) AddRiskLimit(ctx context.Context, limit *RiskLimit) (*RiskLimit, error) {
	return s.limitsManager.AddRiskLimit(ctx, limit)
}

// UpdateRiskLimit updates a risk limit
func (s *Service) UpdateRiskLimit(ctx context.Context, limit *RiskLimit) (*RiskLimit, error) {
	return s.limitsManager.UpdateRiskLimit(ctx, limit)
}

// DeleteRiskLimit deletes a risk limit
func (s *Service) DeleteRiskLimit(ctx context.Context, userID, limitID string) error {
	return s.limitsManager.DeleteRiskLimit(ctx, userID, limitID)
}

// GetRiskLimits gets all risk limits for a user
func (s *Service) GetRiskLimits(ctx context.Context, userID string) ([]*RiskLimit, error) {
	return s.limitsManager.GetRiskLimits(ctx, userID)
}

// UpdatePosition updates a position after a trade
func (s *Service) UpdatePosition(ctx context.Context, userID, symbol string, quantity, price float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Get or create user positions map
	if _, exists := s.positions[userID]; !exists {
		s.positions[userID] = make(map[string]*riskengine.Position)
	}
	
	// Get or create position
	position, exists := s.positions[userID][symbol]
	if !exists {
		position = &riskengine.Position{
			UserID:         userID,
			Symbol:         symbol,
			Quantity:       0,
			AveragePrice:   0,
			UnrealizedPnL:  0,
			RealizedPnL:    0,
			LastUpdateTime: time.Now(),
		}
		s.positions[userID][symbol] = position
	}
	
	// Update position
	if position.Quantity == 0 {
		// New position
		position.Quantity = quantity
		position.AveragePrice = price
	} else {
		// Update existing position
		totalValue := position.Quantity*position.AveragePrice + quantity*price
		totalQuantity := position.Quantity + quantity
		
		if totalQuantity != 0 {
			position.AveragePrice = totalValue / totalQuantity
		}
		position.Quantity = totalQuantity
	}
	
	position.LastUpdateTime = time.Now()
	
	// Update cache
	cacheKey := userID + ":" + symbol
	s.positionCache.Set(cacheKey, position, cache.DefaultExpiration)
	
	// Monitor the position
	limits, err := s.limitsManager.GetRiskLimits(ctx, userID)
	if err != nil {
		s.Logger.Error("Failed to get limits for position monitoring", zap.Error(err))
	} else {
		s.monitor.MonitorPosition(userID, symbol, position, limits)
	}
	
	s.Logger.Debug("Position updated",
		zap.String("user_id", userID),
		zap.String("symbol", symbol),
		zap.Float64("quantity", position.Quantity),
		zap.Float64("average_price", position.AveragePrice))
	
	return nil
}

// GetPosition gets a position for a user and symbol
func (s *Service) GetPosition(ctx context.Context, userID, symbol string) (*riskengine.Position, error) {
	// Check cache first
	cacheKey := userID + ":" + symbol
	if cached, found := s.positionCache.Get(cacheKey); found {
		return cached.(*riskengine.Position), nil
	}
	
	s.mu.RLock()
	userPositions, exists := s.positions[userID]
	if !exists {
		s.mu.RUnlock()
		return nil, ErrPositionNotFound
	}
	
	position, exists := userPositions[symbol]
	s.mu.RUnlock()
	
	if !exists {
		return nil, ErrPositionNotFound
	}
	
	// Add to cache for future requests
	s.positionCache.Set(cacheKey, position, cache.DefaultExpiration)
	
	return position, nil
}

// GetPositions gets all positions for a user
func (s *Service) GetPositions(ctx context.Context, userID string) ([]*riskengine.Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	userPositions, exists := s.positions[userID]
	if !exists {
		return []*riskengine.Position{}, nil
	}
	
	positions := make([]*riskengine.Position, 0, len(userPositions))
	for _, position := range userPositions {
		positions = append(positions, position)
	}
	
	return positions, nil
}

// CalculatePositionRisk calculates risk metrics for a position
func (s *Service) CalculatePositionRisk(ctx context.Context, userID, symbol string, currentPrice float64) (*PositionRiskMetrics, error) {
	position, err := s.GetPosition(ctx, userID, symbol)
	if err != nil {
		return nil, err
	}
	
	return s.calculator.CalculatePositionRisk(ctx, position, currentPrice)
}

// CalculateAccountRisk calculates risk metrics for an entire account
func (s *Service) CalculateAccountRisk(ctx context.Context, userID string, currentPrices map[string]float64) (*AccountRiskMetrics, error) {
	userPositions := s.getUserPositions(userID)
	return s.calculator.CalculateAccountRisk(ctx, userID, userPositions, currentPrices)
}

// AddCircuitBreaker adds a circuit breaker for a symbol
func (s *Service) AddCircuitBreaker(ctx context.Context, symbol string, percentageThreshold float64, timeWindow, cooldownPeriod time.Duration) error {
	s.monitor.AddCircuitBreaker(symbol, percentageThreshold, cooldownPeriod)
	return nil
}

// UpdateMarketData updates market data for risk monitoring
func (s *Service) UpdateMarketData(update MarketDataUpdate) {
	select {
	case s.marketDataChan <- update:
		// Also send to monitor
		s.monitor.UpdateMarketData(update)
	default:
		s.Logger.Warn("Market data channel full, dropping update",
			zap.String("symbol", update.Symbol))
	}
}

// processMarketData processes market data updates
func (s *Service) processMarketData() {
	for {
		select {
		case <-s.Context().Done():
			return
		case update := <-s.marketDataChan:
			s.handleMarketDataUpdate(update)
		}
	}
}

// handleMarketDataUpdate handles a single market data update
func (s *Service) handleMarketDataUpdate(update MarketDataUpdate) {
	// Update unrealized PnL for all positions in this symbol
	s.updateUnrealizedPnL(update.Symbol, update.Price)
}

// updateUnrealizedPnL updates the unrealized PnL for all positions in a symbol
func (s *Service) updateUnrealizedPnL(symbol string, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for userID, userPositions := range s.positions {
		position, exists := userPositions[symbol]
		if exists && position.Quantity != 0 {
			// Calculate unrealized PnL
			position.UnrealizedPnL = position.Quantity * (price - position.AveragePrice)
			position.LastUpdateTime = time.Now()
			
			// Update position in cache
			cacheKey := userID + ":" + symbol
			s.positionCache.Set(cacheKey, position, cache.DefaultExpiration)
		}
	}
}

// subscribeToTrades subscribes to trades from the order matching engine
func (s *Service) subscribeToTrades() {
	// This would be implemented based on the order matching engine's interface
	// For now, this is a placeholder
	s.Logger.Info("Subscribed to trades from order matching engine")
}

// handleRiskAlert handles risk alerts from the monitor
func (s *Service) handleRiskAlert(alert *RiskAlert) {
	s.Logger.Warn("Risk alert received",
		zap.String("type", string(alert.Type)),
		zap.String("severity", string(alert.Severity)),
		zap.String("message", alert.Message),
		zap.String("user_id", alert.UserID),
		zap.String("symbol", alert.Symbol))
	
	// Here you could implement additional alert handling logic
	// such as sending notifications, triggering automated responses, etc.
}

// getUserPositions gets all positions for a user (internal helper)
func (s *Service) getUserPositions(userID string) map[string]*riskengine.Position {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	userPositions, exists := s.positions[userID]
	if !exists {
		return make(map[string]*riskengine.Position)
	}
	
	// Return a copy to avoid race conditions
	result := make(map[string]*riskengine.Position)
	for symbol, position := range userPositions {
		result[symbol] = position
	}
	
	return result
}

// GetRiskLimitStats gets statistics about risk limits
func (s *Service) GetRiskLimitStats(ctx context.Context) (map[string]interface{}, error) {
	return s.limitsManager.GetRiskLimitStats(ctx)
}

// EnableRiskLimit enables a risk limit
func (s *Service) EnableRiskLimit(ctx context.Context, userID, limitID string) error {
	return s.limitsManager.EnableRiskLimit(ctx, userID, limitID)
}

// DisableRiskLimit disables a risk limit
func (s *Service) DisableRiskLimit(ctx context.Context, userID, limitID string) error {
	return s.limitsManager.DisableRiskLimit(ctx, userID, limitID)
}
