package risk_management

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	order_matching "github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Service represents the risk management service
type Service struct {
	// OrderEngine is the order matching engine
	OrderEngine *order_matching.Engine
	// OrderService is the order management service
	OrderService *orders.Service
	// Positions is a map of user ID and symbol to position
	Positions map[string]map[string]*Position
	// RiskLimits is a map of user ID to risk limits
	RiskLimits map[string][]*RiskLimit
	// CircuitBreakers is a map of symbol to circuit breaker
	CircuitBreakers map[string]*CircuitBreaker
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
}

// NewService creates a new risk management service
func NewService(orderEngine *order_matching.Engine, orderService *orders.Service, logger *zap.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		OrderEngine:     orderEngine,
		OrderService:    orderService,
		Positions:       make(map[string]map[string]*Position),
		RiskLimits:      make(map[string][]*RiskLimit),
		CircuitBreakers: make(map[string]*CircuitBreaker),
		PositionCache:   cache.New(5*time.Minute, 10*time.Minute),
		RiskLimitCache:  cache.New(5*time.Minute, 10*time.Minute),
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		riskBatchChan:   make(chan RiskOperation, 1000),
		marketDataChan:  make(chan MarketDataUpdate, 1000),
	}

	// Start batch processor
	go service.processBatchOperations()

	// Start market data processor
	go service.processMarketData()

	// Start circuit breaker checker
	go service.checkCircuitBreakers()

	// Subscribe to trades from the order matching engine
	go service.subscribeToTrades()

	return service
}

// CheckRiskLimits checks risk limits for a user and symbol using early return pattern
func (s *Service) CheckRiskLimits(ctx context.Context, userID, symbol string, orderSize, currentPrice float64) (*RiskCheckResult, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	if symbol == "" {
		return nil, errors.New("symbol cannot be empty")
	}

	if orderSize <= 0 {
		return nil, errors.New("order size must be positive")
	}

	if currentPrice <= 0 {
		return nil, errors.New("current price must be positive")
	}

	// Check if circuit breaker is triggered
	if s.IsCircuitBreakerTriggered(symbol) {
		return &RiskCheckResult{
			Passed:     false,
			RiskLevel:  RiskLevelCritical,
			Violations: []string{fmt.Sprintf("Circuit breaker triggered for symbol %s", symbol)},
			Warnings:   []string{},
			CheckedAt:  time.Now(),
		}, nil
	}

	// Use batch processing for high-frequency checks
	resultCh := make(chan RiskOperationResult, 1)
	operation := RiskOperation{
		OpType: OpTypeCheckLimit,
		UserID: userID,
		Symbol: symbol,
		Data: map[string]interface{}{
			"order_size":    orderSize,
			"current_price": currentPrice,
		},
		ResultCh: resultCh,
	}

	// Send to batch processor
	select {
	case s.riskBatchChan <- operation:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil, errors.New("risk check timeout")
	}

	// Wait for result
	select {
	case result := <-resultCh:
		if !result.Success {
			return nil, result.Error
		}
		return result.Data.(*RiskCheckResult), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil, errors.New("risk check result timeout")
	}
}

// AddRiskLimit adds a risk limit for a user using early return pattern
func (s *Service) AddRiskLimit(ctx context.Context, limit *RiskLimit) (*RiskLimit, error) {
	if limit == nil {
		return nil, errors.New("risk limit cannot be nil")
	}

	if limit.UserID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	if limit.Value <= 0 {
		return nil, errors.New("limit value must be positive")
	}

	// Use batch processing for consistency
	resultCh := make(chan RiskOperationResult, 1)
	operation := RiskOperation{
		OpType:   OpTypeAddLimit,
		UserID:   limit.UserID,
		Symbol:   limit.Symbol,
		Data:     limit,
		ResultCh: resultCh,
	}

	// Send to batch processor
	select {
	case s.riskBatchChan <- operation:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil, errors.New("add limit timeout")
	}

	// Wait for result
	select {
	case result := <-resultCh:
		if !result.Success {
			return nil, result.Error
		}
		return result.Data.(*RiskLimit), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil, errors.New("add limit result timeout")
	}
}

// GetPosition gets a position for a user and symbol using early return pattern
func (s *Service) GetPosition(ctx context.Context, userID, symbol string) (*Position, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	if symbol == "" {
		return nil, errors.New("symbol cannot be empty")
	}

	// Try cache first
	cacheKey := fmt.Sprintf("position:%s:%s", userID, symbol)
	if cached, found := s.PositionCache.Get(cacheKey); found {
		return cached.(*Position), nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get from memory
	if userPositions, exists := s.Positions[userID]; exists {
		if position, exists := userPositions[symbol]; exists {
			// Update cache
			s.PositionCache.Set(cacheKey, position, 5*time.Minute)
			return position, nil
		}
	}

	return nil, fmt.Errorf("position not found for user %s and symbol %s", userID, symbol)
}

// GetPositions gets all positions for a user using early return pattern
func (s *Service) GetPositions(ctx context.Context, userID string) ([]*Position, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	userPositions, exists := s.Positions[userID]
	if !exists {
		return []*Position{}, nil // Return empty slice instead of nil
	}

	positions := make([]*Position, 0, len(userPositions))
	for _, position := range userPositions {
		positions = append(positions, position)
	}

	return positions, nil
}

// GetRiskLimits gets all risk limits for a user using early return pattern
func (s *Service) GetRiskLimits(ctx context.Context, userID string) ([]*RiskLimit, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	limits, exists := s.RiskLimits[userID]
	if !exists {
		return []*RiskLimit{}, nil // Return empty slice instead of nil
	}

	// Return a copy to avoid race conditions
	result := make([]*RiskLimit, len(limits))
	copy(result, limits)
	return result, nil
}

// UpdateMarketData updates market data for risk calculations
func (s *Service) UpdateMarketData(symbol string, price float64) error {
	if symbol == "" {
		return errors.New("symbol cannot be empty")
	}

	if price <= 0 {
		return errors.New("price must be positive")
	}

	update := MarketDataUpdate{
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now(),
	}

	select {
	case s.marketDataChan <- update:
		return nil
	default:
		return errors.New("market data channel full")
	}
}

// Stop stops the risk management service
func (s *Service) Stop() {
	s.cancel()
}

// Helper functions

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
