package risk

import (
	"context"
	"errors"
	"time"

	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
)

// CheckRiskLimits checks risk limits for an order
func (s *Service) CheckRiskLimits(ctx context.Context, userID, symbol string, orderSize, currentPrice float64) (*RiskCheckResult, error) {
	// Check cache for circuit breaker
	s.mu.RLock()
	cb, exists := s.CircuitBreakers[symbol]
	if exists && cb.IsTripped() {
		s.mu.RUnlock()
		return &RiskCheckResult{
			Passed:     false,
			RiskLevel:  RiskLevelHigh,
			Violations: []string{"Circuit breaker triggered"},
			CheckedAt:  time.Now(),
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
			"order_size":    orderSize,
			"current_price": currentPrice,
			"trade_count":   10, // Example value, should be calculated based on user's recent trades
			"time_window":   5 * time.Minute,
			"drawdown":      0.05, // Example value, should be calculated based on user's account
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
		OpType:   "add_limit",
		UserID:   limit.UserID,
		Symbol:   limit.Symbol,
		Data:     limit,
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
func (s *Service) GetPosition(ctx context.Context, userID, symbol string) (*riskengine.Position, error) {
	// Check cache first
	if cachedPosition, found := s.PositionCache.Get(userID + ":" + symbol); found {
		return cachedPosition.(*riskengine.Position), nil
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
func (s *Service) GetPositions(ctx context.Context, userID string) ([]*riskengine.Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userPositions, exists := s.Positions[userID]
	if !exists {
		return []*riskengine.Position{}, nil
	}

	positions := make([]*riskengine.Position, 0, len(userPositions))
	for _, position := range userPositions {
		positions = append(positions, position)
	}

	return positions, nil
}

// AddCircuitBreaker adds a circuit breaker
func (s *Service) AddCircuitBreaker(ctx context.Context, symbol string, percentageThreshold float64, timeWindow, cooldownPeriod time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CircuitBreakers[symbol] = riskengine.NewCircuitBreaker(percentageThreshold, cooldownPeriod)

	return nil
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
					Passed:    true,
					RiskLevel: RiskLevelLow,
					Warnings:  []string{"No risk limits defined"},
					CheckedAt: time.Now(),
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
				RiskLevel:  RiskLevelLow,
				Violations: make([]string, 0),
				Warnings:   make([]string, 0),
				CheckedAt:  time.Now(),
			}

			switch limit.Type {
			case RiskLimitTypePosition:
				// Check position limit
				userPositions, exists := s.Positions[userID]
				if exists {
					position, exists := userPositions[symbol]
					if exists {
						currentValue := abs(position.Quantity)
						if currentValue > limit.Value {
							result.Passed = false
							result.RiskLevel = RiskLevelHigh
							result.Violations = append(result.Violations, "Position limit exceeded")
						}
					}
				}
			case RiskLimitTypeOrderSize:
				// Check order size limit
				orderSize, ok := data["order_size"].(float64)
				if ok {
					if orderSize > limit.Value {
						result.Passed = false
						result.RiskLevel = RiskLevelHigh
						result.Violations = append(result.Violations, "Order size limit exceeded")
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
					if totalExposure > limit.Value {
						result.Passed = false
						result.RiskLevel = RiskLevelHigh
						result.Violations = append(result.Violations, "Exposure limit exceeded")
					}
				}
			case RiskLimitTypeDrawdown:
				// Check drawdown limit
				drawdown, ok := data["drawdown"].(float64)
				if ok {
					if drawdown > limit.Value {
						result.Passed = false
						result.RiskLevel = RiskLevelHigh
						result.Violations = append(result.Violations, "Drawdown limit exceeded")
					}
				}
			case RiskLimitTypeTradeFrequency:
				// Check trade frequency limit
				tradeCount, ok := data["trade_count"].(int)
				if ok {
					timeWindow, ok := data["time_window"].(time.Duration)
					if ok {
						tradesPerSecond := float64(tradeCount) / timeWindow.Seconds()
						if tradesPerSecond > limit.Value {
							result.Passed = false
							result.RiskLevel = RiskLevelHigh
							result.Violations = append(result.Violations, "Trade frequency limit exceeded")
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
					Passed:    true,
					RiskLevel: RiskLevelLow,
					Warnings:  []string{"All risk checks passed"},
					CheckedAt: time.Now(),
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

