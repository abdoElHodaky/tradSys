package risk_management

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
	// Group operations by type for efficient processing
	updatePositionOps := make([]RiskOperation, 0)
	checkLimitOps := make([]RiskOperation, 0)
	addLimitOps := make([]RiskOperation, 0)

	for _, op := range batch {
		switch op.OpType {
		case OpTypeUpdatePosition:
			updatePositionOps = append(updatePositionOps, op)
		case OpTypeCheckLimit:
			checkLimitOps = append(checkLimitOps, op)
		case OpTypeAddLimit:
			addLimitOps = append(addLimitOps, op)
		default:
			// Send error for unknown operation type
			if op.ResultCh != nil {
				op.ResultCh <- RiskOperationResult{
					Success: false,
					Error:   fmt.Errorf("unknown operation type: %s", op.OpType),
				}
			}
		}
	}

	// Process each type of operation in batch
	if len(updatePositionOps) > 0 {
		s.processUpdatePositionBatch(updatePositionOps)
	}
	if len(checkLimitOps) > 0 {
		s.processCheckLimitBatch(checkLimitOps)
	}
	if len(addLimitOps) > 0 {
		s.processAddLimitBatch(addLimitOps)
	}
}

// processUpdatePositionBatch processes a batch of position update operations
func (s *Service) processUpdatePositionBatch(ops []RiskOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		data, ok := op.Data.(map[string]interface{})
		if !ok {
			if op.ResultCh != nil {
				op.ResultCh <- RiskOperationResult{
					Success: false,
					Error:   fmt.Errorf("invalid data format for position update"),
				}
			}
			continue
		}

		quantityDelta, ok := data["quantity_delta"].(float64)
		if !ok {
			if op.ResultCh != nil {
				op.ResultCh <- RiskOperationResult{
					Success: false,
					Error:   fmt.Errorf("missing or invalid quantity_delta"),
				}
			}
			continue
		}

		price, ok := data["price"].(float64)
		if !ok {
			if op.ResultCh != nil {
				op.ResultCh <- RiskOperationResult{
					Success: false,
					Error:   fmt.Errorf("missing or invalid price"),
				}
			}
			continue
		}

		// Update position
		s.updatePositionInternal(op.UserID, op.Symbol, quantityDelta, price)

		// Send success result
		if op.ResultCh != nil {
			op.ResultCh <- RiskOperationResult{
				Success: true,
			}
		}
	}
}

// processCheckLimitBatch processes a batch of limit check operations
func (s *Service) processCheckLimitBatch(ops []RiskOperation) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, op := range ops {
		data, ok := op.Data.(map[string]interface{})
		if !ok {
			if op.ResultCh != nil {
				op.ResultCh <- RiskOperationResult{
					Success: false,
					Error:   fmt.Errorf("invalid data format for limit check"),
				}
			}
			continue
		}

		orderSize, ok := data["order_size"].(float64)
		if !ok {
			if op.ResultCh != nil {
				op.ResultCh <- RiskOperationResult{
					Success: false,
					Error:   fmt.Errorf("missing or invalid order_size"),
				}
			}
			continue
		}

		currentPrice, ok := data["current_price"].(float64)
		if !ok {
			if op.ResultCh != nil {
				op.ResultCh <- RiskOperationResult{
					Success: false,
					Error:   fmt.Errorf("missing or invalid current_price"),
				}
			}
			continue
		}

		// Check risk limits
		result := s.checkRiskLimitsInternal(op.UserID, op.Symbol, orderSize, currentPrice)

		// Send result
		if op.ResultCh != nil {
			op.ResultCh <- RiskOperationResult{
				Success: true,
				Data:    result,
			}
		}
	}
}

// processAddLimitBatch processes a batch of add limit operations
func (s *Service) processAddLimitBatch(ops []RiskOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		limit, ok := op.Data.(*RiskLimit)
		if !ok {
			if op.ResultCh != nil {
				op.ResultCh <- RiskOperationResult{
					Success: false,
					Error:   fmt.Errorf("invalid data format for add limit"),
				}
			}
			continue
		}

		// Generate ID if not provided
		if limit.ID == "" {
			limit.ID = uuid.New().String()
		}

		// Set timestamps
		now := time.Now()
		limit.CreatedAt = now
		limit.UpdatedAt = now

		// Add to user's risk limits
		if s.RiskLimits[op.UserID] == nil {
			s.RiskLimits[op.UserID] = make([]*RiskLimit, 0)
		}
		s.RiskLimits[op.UserID] = append(s.RiskLimits[op.UserID], limit)

		// Cache the limit
		cacheKey := fmt.Sprintf("limit:%s:%s:%s", op.UserID, op.Symbol, string(limit.Type))
		s.RiskLimitCache.Set(cacheKey, limit, 5*time.Minute)

		s.logger.Info("Risk limit added",
			zap.String("user_id", op.UserID),
			zap.String("symbol", op.Symbol),
			zap.String("type", string(limit.Type)),
			zap.Float64("value", limit.Value))

		// Send success result
		if op.ResultCh != nil {
			op.ResultCh <- RiskOperationResult{
				Success: true,
				Data:    limit,
			}
		}
	}
}

// updatePositionInternal updates a position internally (must be called with lock held)
func (s *Service) updatePositionInternal(userID, symbol string, quantityDelta, price float64) {
	// Initialize user positions if not exists
	if s.Positions[userID] == nil {
		s.Positions[userID] = make(map[string]*Position)
	}

	position := s.Positions[userID][symbol]
	if position == nil {
		// Create new position
		position = &Position{
			ID:           uuid.New().String(),
			UserID:       userID,
			Symbol:       symbol,
			Quantity:     0,
			AveragePrice: 0,
			UnrealizedPnL: 0,
			RealizedPnL:  0,
			CreatedAt:    time.Now(),
		}
		s.Positions[userID][symbol] = position
	}

	// Calculate new average price and realized PnL
	if quantityDelta != 0 {
		oldQuantity := position.Quantity
		newQuantity := oldQuantity + quantityDelta

		// Calculate realized PnL for closing positions
		if (oldQuantity > 0 && quantityDelta < 0) || (oldQuantity < 0 && quantityDelta > 0) {
			closingQuantity := quantityDelta
			if abs(closingQuantity) > abs(oldQuantity) {
				closingQuantity = -oldQuantity
			}
			realizedPnL := closingQuantity * (price - position.AveragePrice)
			position.RealizedPnL += realizedPnL
		}

		// Update average price for opening positions
		if (oldQuantity >= 0 && quantityDelta > 0) || (oldQuantity <= 0 && quantityDelta < 0) {
			if newQuantity != 0 {
				position.AveragePrice = (position.AveragePrice*oldQuantity + price*quantityDelta) / newQuantity
			}
		} else if newQuantity == 0 {
			position.AveragePrice = 0
		}

		position.Quantity = newQuantity
	}

	position.LastUpdated = time.Now()

	// Cache the position
	cacheKey := fmt.Sprintf("position:%s:%s", userID, symbol)
	s.PositionCache.Set(cacheKey, position, 5*time.Minute)
}

// checkRiskLimitsInternal checks risk limits internally (must be called with read lock held)
func (s *Service) checkRiskLimitsInternal(userID, symbol string, orderSize, currentPrice float64) *RiskCheckResult {
	result := &RiskCheckResult{
		Passed:     true,
		RiskLevel:  RiskLevelLow,
		Violations: make([]string, 0),
		Warnings:   make([]string, 0),
		CheckedAt:  time.Now(),
	}

	// Get user's risk limits
	limits, exists := s.RiskLimits[userID]
	if !exists {
		return result
	}

	// Get current position
	var currentPosition *Position
	if userPositions, exists := s.Positions[userID]; exists {
		currentPosition = userPositions[symbol]
	}

	// Check each limit
	for _, limit := range limits {
		if !limit.Enabled {
			continue
		}

		// Skip if limit is for different symbol (unless it's a global limit)
		if limit.Symbol != "" && limit.Symbol != symbol {
			continue
		}

		violation := s.checkSingleLimit(limit, currentPosition, orderSize, currentPrice)
		if violation != "" {
			result.Violations = append(result.Violations, violation)
			result.Passed = false
			if result.RiskLevel < RiskLevelHigh {
				result.RiskLevel = RiskLevelHigh
			}
		}
	}

	return result
}

// checkSingleLimit checks a single risk limit
func (s *Service) checkSingleLimit(limit *RiskLimit, position *Position, orderSize, currentPrice float64) string {
	switch limit.Type {
	case RiskLimitTypeOrderSize:
		if orderSize > limit.Value {
			return fmt.Sprintf("Order size %.2f exceeds limit %.2f", orderSize, limit.Value)
		}

	case RiskLimitTypePosition:
		currentQuantity := float64(0)
		if position != nil {
			currentQuantity = position.Quantity
		}
		newQuantity := currentQuantity + orderSize
		if abs(newQuantity) > limit.Value {
			return fmt.Sprintf("Position size %.2f would exceed limit %.2f", abs(newQuantity), limit.Value)
		}

	case RiskLimitTypeExposure:
		currentExposure := float64(0)
		if position != nil {
			currentExposure = abs(position.Quantity * currentPrice)
		}
		newExposure := abs((position.Quantity + orderSize) * currentPrice)
		if newExposure > limit.Value {
			return fmt.Sprintf("Exposure %.2f would exceed limit %.2f", newExposure, limit.Value)
		}
	}

	return ""
}
