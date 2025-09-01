package risk_management

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/math"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_management"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// RiskLimitType represents the type of risk limit
type RiskLimitType string

const (
	// RiskLimitTypePosition represents a position limit
	RiskLimitTypePosition RiskLimitType = "position"
	// RiskLimitTypeOrderSize represents an order size limit
	RiskLimitTypeOrderSize RiskLimitType = "order_size"
	// RiskLimitTypeExposure represents an exposure limit
	RiskLimitTypeExposure RiskLimitType = "exposure"
	// RiskLimitTypeDrawdown represents a drawdown limit
	RiskLimitTypeDrawdown RiskLimitType = "drawdown"
	// RiskLimitTypeTradeFrequency represents a trade frequency limit
	RiskLimitTypeTradeFrequency RiskLimitType = "trade_frequency"
)

// RiskCheckResult represents the result of a risk check
type RiskCheckResult struct {
	// Passed indicates whether the risk check passed
	Passed bool
	// Message is the message for the risk check
	Message string
	// LimitType is the type of risk limit
	LimitType RiskLimitType
	// CurrentValue is the current value
	CurrentValue float64
	// LimitValue is the limit value
	LimitValue float64
}

// RiskLimit represents a risk limit
type RiskLimit struct {
	// ID is the unique identifier for the risk limit
	ID string
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Type is the type of risk limit
	Type RiskLimitType
	// Value is the limit value
	Value float64
	// CreatedAt is the time the risk limit was created
	CreatedAt time.Time
	// UpdatedAt is the time the risk limit was last updated
	UpdatedAt time.Time
	// Enabled indicates whether the risk limit is enabled
	Enabled bool
}

// Position represents a trading position
type Position struct {
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Quantity is the position quantity (positive for long, negative for short)
	Quantity float64
	// AverageEntryPrice is the average entry price
	AverageEntryPrice float64
	// UnrealizedPnL is the unrealized profit and loss
	UnrealizedPnL float64
	// RealizedPnL is the realized profit and loss
	RealizedPnL float64
	// LastUpdated is the time the position was last updated
	LastUpdated time.Time
}

// CircuitBreaker represents a circuit breaker
type CircuitBreaker struct {
	// Symbol is the trading symbol
	Symbol string
	// PercentageThreshold is the percentage threshold for triggering the circuit breaker
	PercentageThreshold float64
	// TimeWindow is the time window for the circuit breaker
	TimeWindow time.Duration
	// CooldownPeriod is the cooldown period after triggering
	CooldownPeriod time.Duration
	// LastTriggered is the time the circuit breaker was last triggered
	LastTriggered time.Time
	// Triggered indicates whether the circuit breaker is currently triggered
	Triggered bool
	// ReferencePrice is the reference price for calculating the percentage change
	ReferencePrice float64
}

// RiskOperation represents a batch operation on risk data
type RiskOperation struct {
	// OpType is the operation type
	OpType string
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Data is the operation data
	Data interface{}
	// ResultCh is the result channel
	ResultCh chan RiskOperationResult
}

// RiskOperationResult represents the result of a risk operation
type RiskOperationResult struct {
	// Success indicates whether the operation was successful
	Success bool
	// Error is the error if the operation failed
	Error error
	// Data is the result data
	Data interface{}
}

// Service represents a risk management service
type Service struct {
	// OrderEngine is the order matching engine
	OrderEngine *order_matching.Engine
	// OrderService is the order management service
	OrderService *order_management.Service
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

// MarketDataUpdate represents a market data update
type MarketDataUpdate struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the current price
	Price float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// NewService creates a new risk management service
func NewService(orderEngine *order_matching.Engine, orderService *order_management.Service, logger *zap.Logger) *Service {
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

// processBatchOperations processes batch operations for risk data
func (s *Service) processBatchOperations() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	batch := make([]RiskOperation, 0, 100)
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case op := <-s.riskBatchChan:
			batch = append(batch, op)
			
			// Process batch if it's full
			if len(batch) >= 100 {
				s.processBatch(batch)
				batch = make([]RiskOperation, 0, 100)
			}
		case <-ticker.C:
			// Process remaining operations in batch
			if len(batch) > 0 {
				s.processBatch(batch)
				batch = make([]RiskOperation, 0, 100)
			}
		}
	}
}

// processBatch processes a batch of risk operations
func (s *Service) processBatch(batch []RiskOperation) {
	// Group operations by type
	updatePositionOps := make([]RiskOperation, 0)
	checkLimitOps := make([]RiskOperation, 0)
	addLimitOps := make([]RiskOperation, 0)
	
	for _, op := range batch {
		switch op.OpType {
		case "update_position":
			updatePositionOps = append(updatePositionOps, op)
		case "check_limit":
			checkLimitOps = append(checkLimitOps, op)
		case "add_limit":
			addLimitOps = append(addLimitOps, op)
		}
	}
	
	// Process update position operations
	if len(updatePositionOps) > 0 {
		s.processUpdatePositionBatch(updatePositionOps)
	}
	
	// Process check limit operations
	if len(checkLimitOps) > 0 {
		s.processCheckLimitBatch(checkLimitOps)
	}
	
	// Process add limit operations
	if len(addLimitOps) > 0 {
		s.processAddLimitBatch(addLimitOps)
	}
}

// processUpdatePositionBatch processes a batch of update position operations
func (s *Service) processUpdatePositionBatch(ops []RiskOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, op := range ops {
		userID := op.UserID
		symbol := op.Symbol
		data := op.Data.(map[string]interface{})
		
		// Get position
		var position *Position
		userPositions, exists := s.Positions[userID]
		if !exists {
			userPositions = make(map[string]*Position)
			s.Positions[userID] = userPositions
		}
		
		position, exists = userPositions[symbol]
		if !exists {
			position = &Position{
				UserID:            userID,
				Symbol:            symbol,
				Quantity:          0,
				AverageEntryPrice: 0,
				UnrealizedPnL:     0,
				RealizedPnL:       0,
				LastUpdated:       time.Now(),
			}
			userPositions[symbol] = position
		}
		
		// Update position
		quantityDelta, ok := data["quantity_delta"].(float64)
		if ok {
			price, ok := data["price"].(float64)
			if ok {
				// Calculate realized PnL for reducing positions
				var realizedPnL float64
				if (position.Quantity > 0 && quantityDelta < 0) || (position.Quantity < 0 && quantityDelta > 0) {
					// Reducing position
					reduceQuantity := min(abs(position.Quantity), abs(quantityDelta))
					if position.Quantity > 0 {
						reduceQuantity = -reduceQuantity
					}
					realizedPnL = reduceQuantity * (price - position.AverageEntryPrice)
					position.RealizedPnL += realizedPnL
				}
				
				// Update average entry price for increasing positions
				if (position.Quantity >= 0 && quantityDelta > 0) || (position.Quantity <= 0 && quantityDelta < 0) {
					// Increasing position
					oldValue := position.Quantity * position.AverageEntryPrice
					newValue := quantityDelta * price
					position.AverageEntryPrice = (oldValue + newValue) / (position.Quantity + quantityDelta)
				}
				
				// Update quantity
				position.Quantity += quantityDelta
				
				// If position is flat (zero), reset average entry price
				if position.Quantity == 0 {
					position.AverageEntryPrice = 0
				}
				
				// Update last updated time
				position.LastUpdated = time.Now()
				
				// Update position in cache
				s.PositionCache.Set(userID+":"+symbol, position, cache.DefaultExpiration)
			}
		}
		
		// Send result
		op.ResultCh <- RiskOperationResult{
			Success: true,
			Data:    position,
		}
	}
}

// processCheckLimitBatch processes a batch of check limit operations
func (s *Service) processCheckLimitBatch(ops []RiskOperation) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, op := range ops {
		userID := op.UserID
		symbol := op.Symbol
		data := op.Data.(map[string]interface{})
		
		// Get risk limits for user
		limits, exists := s.RiskLimits[userID]
		if !exists {
			op.ResultCh <- RiskOperationResult{
				Success: true,
				Data: &RiskCheckResult{
					Passed:  true,
					Message: "No risk limits defined",
				},
			}
			continue
		}
		
		// Check each limit
		results := make([]*RiskCheckResult, 0)
		
		for _, limit := range limits {
			if !limit.Enabled {
				continue
			}
			
			// Skip limits for other symbols
			if limit.Symbol != "" && limit.Symbol != symbol {
				continue
			}
			
			result := &RiskCheckResult{
				Passed:     true,
				LimitType:  limit.Type,
				LimitValue: limit.Value,
			}
			
			switch limit.Type {
			case RiskLimitTypePosition:
				// Check position limit
				userPositions, exists := s.Positions[userID]
				if exists {
					position, exists := userPositions[symbol]
					if exists {
						result.CurrentValue = abs(position.Quantity)
						if result.CurrentValue > limit.Value {
							result.Passed = false
							result.Message = "Position limit exceeded"
						}
					}
				}
			case RiskLimitTypeOrderSize:
				// Check order size limit
				orderSize, ok := data["order_size"].(float64)
				if ok {
					result.CurrentValue = orderSize
					if result.CurrentValue > limit.Value {
						result.Passed = false
						result.Message = "Order size limit exceeded"
					}
				}
			case RiskLimitTypeExposure:
				// Check exposure limit
				userPositions, exists := s.Positions[userID]
				if exists {
					var totalExposure float64
					for _, pos := range userPositions {
						// For exposure, we use absolute value of position * current price
						price, ok := data["current_price"].(float64)
						if ok {
							totalExposure += abs(pos.Quantity) * price
						}
					}
					result.CurrentValue = totalExposure
					if result.CurrentValue > limit.Value {
						result.Passed = false
						result.Message = "Exposure limit exceeded"
					}
				}
			case RiskLimitTypeDrawdown:
				// Check drawdown limit
				drawdown, ok := data["drawdown"].(float64)
				if ok {
					result.CurrentValue = drawdown
					if result.CurrentValue > limit.Value {
						result.Passed = false
						result.Message = "Drawdown limit exceeded"
					}
				}
			case RiskLimitTypeTradeFrequency:
				// Check trade frequency limit
				tradeCount, ok := data["trade_count"].(int)
				if ok {
					timeWindow, ok := data["time_window"].(time.Duration)
					if ok {
						tradesPerSecond := float64(tradeCount) / timeWindow.Seconds()
						result.CurrentValue = tradesPerSecond
						if result.CurrentValue > limit.Value {
							result.Passed = false
							result.Message = "Trade frequency limit exceeded"
						}
					}
				}
			}
			
			results = append(results, result)
		}
		
		// Check if any limit failed
		allPassed := true
		var failedResult *RiskCheckResult
		
		for _, result := range results {
			if !result.Passed {
				allPassed = false
				failedResult = result
				break
			}
		}
		
		// Send result
		if allPassed {
			op.ResultCh <- RiskOperationResult{
				Success: true,
				Data: &RiskCheckResult{
					Passed:  true,
					Message: "All risk checks passed",
				},
			}
		} else {
			op.ResultCh <- RiskOperationResult{
				Success: true,
				Data:    failedResult,
			}
		}
	}
}

// processAddLimitBatch processes a batch of add limit operations
func (s *Service) processAddLimitBatch(ops []RiskOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, op := range ops {
		userID := op.UserID
		limit := op.Data.(*RiskLimit)
		
		// Generate ID if not provided
		if limit.ID == "" {
			limit.ID = uuid.New().String()
		}
		
		// Set timestamps
		now := time.Now()
		if limit.CreatedAt.IsZero() {
			limit.CreatedAt = now
		}
		limit.UpdatedAt = now
		
		// Add to risk limits
		if _, exists := s.RiskLimits[userID]; !exists {
			s.RiskLimits[userID] = make([]*RiskLimit, 0)
		}
		s.RiskLimits[userID] = append(s.RiskLimits[userID], limit)
		
		// Add to cache
		s.RiskLimitCache.Set(userID+":"+limit.ID, limit, cache.DefaultExpiration)
		
		// Send result
		op.ResultCh <- RiskOperationResult{
			Success: true,
			Data:    limit,
		}
	}
}

// processMarketData processes market data updates
func (s *Service) processMarketData() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case update := <-s.marketDataChan:
			// Update unrealized PnL for all positions in this symbol
			s.updateUnrealizedPnL(update.Symbol, update.Price)
			
			// Check circuit breakers
			s.checkCircuitBreaker(update.Symbol, update.Price, update.Timestamp)
		}
	}
}

// updateUnrealizedPnL updates the unrealized PnL for all positions in a symbol
func (s *Service) updateUnrealizedPnL(symbol string, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for userID, userPositions := range s.Positions {
		position, exists := userPositions[symbol]
		if exists && position.Quantity != 0 {
			// Calculate unrealized PnL
			position.UnrealizedPnL = position.Quantity * (price - position.AverageEntryPrice)
			position.LastUpdated = time.Now()
			
			// Update position in cache
			s.PositionCache.Set(userID+":"+symbol, position, cache.DefaultExpiration)
		}
	}
}

// checkCircuitBreakers periodically checks circuit breakers
func (s *Service) checkCircuitBreakers() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			
			// Check if any triggered circuit breakers should be reset
			for symbol, cb := range s.CircuitBreakers {
				if cb.Triggered && now.Sub(cb.LastTriggered) > cb.CooldownPeriod {
					cb.Triggered = false
					s.logger.Info("Circuit breaker reset",
						zap.String("symbol", symbol),
						zap.Float64("reference_price", cb.ReferencePrice))
				}
			}
			
			s.mu.Unlock()
		}
	}
}

// checkCircuitBreaker checks if a circuit breaker should be triggered
func (s *Service) checkCircuitBreaker(symbol string, price float64, timestamp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cb, exists := s.CircuitBreakers[symbol]
	if !exists {
		return
	}
	
	// Skip if already triggered
	if cb.Triggered {
		return
	}
	
	// Calculate percentage change
	if cb.ReferencePrice == 0 {
		cb.ReferencePrice = price
		return
	}
	
	percentageChange := math.Abs((price - cb.ReferencePrice) / cb.ReferencePrice * 100)
	
	// Check if circuit breaker should be triggered
	if percentageChange >= cb.PercentageThreshold {
		cb.Triggered = true
		cb.LastTriggered = timestamp
		
		s.logger.Warn("Circuit breaker triggered",
			zap.String("symbol", symbol),
			zap.Float64("reference_price", cb.ReferencePrice),
			zap.Float64("current_price", price),
			zap.Float64("percentage_change", percentageChange),
			zap.Float64("threshold", cb.PercentageThreshold))
	}
}

// subscribeToTrades subscribes to trades from the order matching engine
func (s *Service) subscribeToTrades() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case trade := <-s.OrderEngine.TradeChannel:
			// Process trade for risk management
			s.processTrade(trade)
		}
	}
}

// processTrade processes a trade for risk management
func (s *Service) processTrade(trade *order_matching.Trade) {
	// Get buy order
	buyOrder, err := s.OrderEngine.GetOrder(trade.Symbol, trade.BuyOrderID)
	if err != nil {
		s.logger.Error("Failed to get buy order",
			zap.String("order_id", trade.BuyOrderID),
			zap.Error(err))
		return
	}
	
	// Get sell order
	sellOrder, err := s.OrderEngine.GetOrder(trade.Symbol, trade.SellOrderID)
	if err != nil {
		s.logger.Error("Failed to get sell order",
			zap.String("order_id", trade.SellOrderID),
			zap.Error(err))
		return
	}
	
	// Update buy position
	s.updatePosition(buyOrder.UserID, trade.Symbol, trade.Quantity, trade.Price)
	
	// Update sell position
	s.updatePosition(sellOrder.UserID, trade.Symbol, -trade.Quantity, trade.Price)
	
	// Update market data
	s.marketDataChan <- MarketDataUpdate{
		Symbol:    trade.Symbol,
		Price:     trade.Price,
		Timestamp: trade.Timestamp,
	}
}

// updatePosition updates a position
func (s *Service) updatePosition(userID, symbol string, quantityDelta, price float64) {
	// Use batch processing for better performance
	resultCh := make(chan RiskOperationResult, 1)
	s.riskBatchChan <- RiskOperation{
		OpType: "update_position",
		UserID: userID,
		Symbol: symbol,
		Data: map[string]interface{}{
			"quantity_delta": quantityDelta,
			"price":          price,
		},
		ResultCh: resultCh,
	}
	
	// Wait for result
	<-resultCh
}

// CheckRiskLimits checks risk limits for an order
func (s *Service) CheckRiskLimits(ctx context.Context, userID, symbol string, orderSize, currentPrice float64) (*RiskCheckResult, error) {
	// Check cache for circuit breaker
	s.mu.RLock()
	cb, exists := s.CircuitBreakers[symbol]
	if exists && cb.Triggered {
		s.mu.RUnlock()
		return &RiskCheckResult{
			Passed:  false,
			Message: "Circuit breaker triggered",
		}, nil
	}
	s.mu.RUnlock()
	
	// Use batch processing for better performance
	resultCh := make(chan RiskOperationResult, 1)
	s.riskBatchChan <- RiskOperation{
		OpType: "check_limit",
		UserID: userID,
		Symbol: symbol,
		Data: map[string]interface{}{
			"order_size":     orderSize,
			"current_price":  currentPrice,
			"trade_count":    10, // Example value, should be calculated based on user's recent trades
			"time_window":    5 * time.Minute,
			"drawdown":       0.05, // Example value, should be calculated based on user's account
		},
		ResultCh: resultCh,
	}
	
	// Wait for result
	result := <-resultCh
	if !result.Success {
		return nil, errors.New("failed to check risk limits")
	}
	
	return result.Data.(*RiskCheckResult), nil
}

// AddRiskLimit adds a risk limit
func (s *Service) AddRiskLimit(ctx context.Context, limit *RiskLimit) (*RiskLimit, error) {
	// Use batch processing for better performance
	resultCh := make(chan RiskOperationResult, 1)
	s.riskBatchChan <- RiskOperation{
		OpType: "add_limit",
		UserID: limit.UserID,
		Symbol: limit.Symbol,
		Data:   limit,
		ResultCh: resultCh,
	}
	
	// Wait for result
	result := <-resultCh
	if !result.Success {
		return nil, errors.New("failed to add risk limit")
	}
	
	return result.Data.(*RiskLimit), nil
}

// GetPosition gets a position
func (s *Service) GetPosition(ctx context.Context, userID, symbol string) (*Position, error) {
	// Check cache first
	if cachedPosition, found := s.PositionCache.Get(userID + ":" + symbol); found {
		return cachedPosition.(*Position), nil
	}
	
	// If not in cache, check the map
	s.mu.RLock()
	userPositions, exists := s.Positions[userID]
	if !exists {
		s.mu.RUnlock()
		return nil, errors.New("position not found")
	}
	
	position, exists := userPositions[symbol]
	s.mu.RUnlock()
	
	if !exists {
		return nil, errors.New("position not found")
	}
	
	// Add to cache for future requests
	s.PositionCache.Set(userID+":"+symbol, position, cache.DefaultExpiration)
	
	return position, nil
}

// GetPositions gets all positions for a user
func (s *Service) GetPositions(ctx context.Context, userID string) ([]*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	userPositions, exists := s.Positions[userID]
	if !exists {
		return []*Position{}, nil
	}
	
	positions := make([]*Position, 0, len(userPositions))
	for _, position := range userPositions {
		positions = append(positions, position)
	}
	
	return positions, nil
}

// AddCircuitBreaker adds a circuit breaker
func (s *Service) AddCircuitBreaker(ctx context.Context, symbol string, percentageThreshold float64, timeWindow, cooldownPeriod time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.CircuitBreakers[symbol] = &CircuitBreaker{
		Symbol:              symbol,
		PercentageThreshold: percentageThreshold,
		TimeWindow:          timeWindow,
		CooldownPeriod:      cooldownPeriod,
		LastTriggered:       time.Time{},
		Triggered:           false,
		ReferencePrice:      0,
	}
	
	return nil
}

// Stop stops the service
func (s *Service) Stop() {
	s.cancel()
}

// Use math.Abs and math.Min from internal/math package instead
