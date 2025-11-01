// 🎯 **Risk Engine Processors**
// Generated using TradSys Code Splitting Standards
//
// This file contains business logic processing and type-specific handlers
// for the RealTime Risk Engine component. It implements the processor pattern
// with polymorphism to avoid complex switch statements.
//
// File size limit: 350 lines

package risk_management

import (
	"fmt"
	"math"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// performPreTradeCheck performs pre-trade risk checks
func (e *RealTimeRiskEngine) performPreTradeCheck(req *RiskCheckRequest, response *RiskCheckResponse) (*RiskCheckResponse, error) {
	// Early return for nil order
	if req.Order == nil {
		response.Status = RiskCheckStatusError
		response.Message = "order is required for pre-trade check"
		response.Passed = false
		return response, nil
	}

	order := req.Order

	// Check order size limit
	if order.Quantity > e.config.MaxOrderSize {
		response.Status = RiskCheckStatusRejected
		response.CurrentValue = order.Quantity
		response.LimitValue = e.config.MaxOrderSize
		response.Message = fmt.Sprintf("order size %.2f exceeds limit %.2f", order.Quantity, e.config.MaxOrderSize)
		response.Passed = false
		e.metrics.RejectedOrders++
		return response, nil
	}

	// Check position limit
	if err := e.checkPositionLimit(order, response); err != nil {
		return response, err
	}

	// Check daily loss limit
	if err := e.checkDailyLossLimit(order, response); err != nil {
		return response, err
	}

	// Check circuit breaker
	if e.circuitBreaker.IsTriggeredFlag {
		response.Status = RiskCheckStatusRejected
		response.Message = "circuit breaker is tripped"
		response.Passed = false
		return response, nil
	}

	// All checks passed
	response.Status = RiskCheckStatusPassed
	response.Message = "pre-trade checks passed"
	response.Passed = true

	return response, nil
}

// performPostTradeCheck performs post-trade risk checks
func (e *RealTimeRiskEngine) performPostTradeCheck(req *RiskCheckRequest, response *RiskCheckResponse) (*RiskCheckResponse, error) {
	// Early return for nil order
	if req.Order == nil {
		response.Status = RiskCheckStatusError
		response.Message = "order is required for post-trade check"
		response.Passed = false
		return response, nil
	}

	order := req.Order

	// Update position after trade
	if err := e.updatePositionAfterTrade(order); err != nil {
		response.Status = RiskCheckStatusError
		response.Message = fmt.Sprintf("failed to update position: %v", err)
		response.Passed = false
		return response, err
	}

	// Check if position is within limits after trade
	position, exists := e.positionManager.positions.Load(order.Symbol)
	if exists {
		pos := position.(*Position)
		if math.Abs(pos.Quantity) > e.config.MaxPositionSize {
			response.Status = RiskCheckStatusRejected
			response.CurrentValue = math.Abs(pos.Quantity)
			response.LimitValue = e.config.MaxPositionSize
			response.Message = fmt.Sprintf("position size %.2f exceeds limit %.2f after trade", math.Abs(pos.Quantity), e.config.MaxPositionSize)
			response.Passed = false
			return response, nil
		}
	}

	// All checks passed
	response.Status = RiskCheckStatusPassed
	response.Message = "post-trade checks passed"
	response.Passed = true

	return response, nil
}

// performPositionRiskCheck performs position risk checks
func (e *RealTimeRiskEngine) performPositionRiskCheck(req *RiskCheckRequest, response *RiskCheckResponse) (*RiskCheckResponse, error) {
	// Early return for nil position
	if req.Position == nil {
		response.Status = RiskCheckStatusError
		response.Message = "position is required for position risk check"
		response.Passed = false
		return response, nil
	}

	position := req.Position

	// Check position size
	if math.Abs(position.Quantity) > e.config.MaxPositionSize {
		response.Status = RiskCheckStatusRejected
		response.CurrentValue = math.Abs(position.Quantity)
		response.LimitValue = e.config.MaxPositionSize
		response.Message = fmt.Sprintf("position size %.2f exceeds limit %.2f", math.Abs(position.Quantity), e.config.MaxPositionSize)
		response.Passed = false
		return response, nil
	}

	// Check unrealized PnL
	if position.UnrealizedPnL < -e.config.MaxDailyLoss {
		response.Status = RiskCheckStatusRejected
		response.CurrentValue = -position.UnrealizedPnL
		response.LimitValue = e.config.MaxDailyLoss
		response.Message = fmt.Sprintf("unrealized loss %.2f exceeds daily limit %.2f", -position.UnrealizedPnL, e.config.MaxDailyLoss)
		response.Passed = false
		return response, nil
	}

	// All checks passed
	response.Status = RiskCheckStatusPassed
	response.Message = "position risk checks passed"
	response.Passed = true

	return response, nil
}

// performVaRCheck performs Value at Risk checks
func (e *RealTimeRiskEngine) performVaRCheck(req *RiskCheckRequest, response *RiskCheckResponse) (*RiskCheckResponse, error) {
	// Calculate current VaR
	currentVaR := e.calculateCurrentVaR()

	// Check if VaR exceeds limit (e.g., 2x daily loss limit)
	varLimit := e.config.MaxDailyLoss * 2

	if currentVaR > varLimit {
		response.Status = RiskCheckStatusRejected
		response.CurrentValue = currentVaR
		response.LimitValue = varLimit
		response.Message = fmt.Sprintf("VaR %.2f exceeds limit %.2f", currentVaR, varLimit)
		response.Passed = false
		return response, nil
	}

	// All checks passed
	response.Status = RiskCheckStatusPassed
	response.CurrentValue = currentVaR
	response.LimitValue = varLimit
	response.Message = "VaR checks passed"
	response.Passed = true

	return response, nil
}

// checkPositionLimit checks if order would exceed position limits
func (e *RealTimeRiskEngine) checkPositionLimit(order *types.Order, response *RiskCheckResponse) error {
	// Get current position
	position, exists := e.positionManager.positions.Load(order.Symbol)
	currentQuantity := 0.0
	if exists {
		pos := position.(*Position)
		currentQuantity = pos.Quantity
	}

	// Calculate new position after order
	var newQuantity float64
	if order.Side == types.OrderSideBuy {
		newQuantity = currentQuantity + order.Quantity
	} else {
		newQuantity = currentQuantity - order.Quantity
	}

	// Check if new position would exceed limit
	if math.Abs(newQuantity) > e.config.MaxPositionSize {
		response.Status = RiskCheckStatusRejected
		response.CurrentValue = math.Abs(newQuantity)
		response.LimitValue = e.config.MaxPositionSize
		response.Message = fmt.Sprintf("order would result in position %.2f exceeding limit %.2f", math.Abs(newQuantity), e.config.MaxPositionSize)
		response.Passed = false
		e.metrics.RejectedOrders++
		return nil
	}

	return nil
}

// checkDailyLossLimit checks if order would exceed daily loss limits
func (e *RealTimeRiskEngine) checkDailyLossLimit(order *types.Order, response *RiskCheckResponse) error {
	e.limitManager.mu.RLock()
	currentDailyLoss := e.limitManager.currentDailyLoss
	e.limitManager.mu.RUnlock()

	// Estimate potential loss from this order (simplified)
	potentialLoss := order.Quantity * order.Price * 0.01 // Assume 1% potential loss

	if currentDailyLoss+potentialLoss > e.config.MaxDailyLoss {
		response.Status = RiskCheckStatusRejected
		response.CurrentValue = currentDailyLoss + potentialLoss
		response.LimitValue = e.config.MaxDailyLoss
		response.Message = fmt.Sprintf("order would result in daily loss %.2f exceeding limit %.2f", currentDailyLoss+potentialLoss, e.config.MaxDailyLoss)
		response.Passed = false
		e.metrics.RejectedOrders++
		return nil
	}

	return nil
}

// updatePositionAfterTrade updates position after a trade
func (e *RealTimeRiskEngine) updatePositionAfterTrade(order *types.Order) error {
	// Get or create position
	var position *Position
	if pos, exists := e.positionManager.positions.Load(order.Symbol); exists {
		position = pos.(*Position)
	} else {
		position = &Position{
			Symbol:       order.Symbol,
			Quantity:     0,
			AveragePrice: 0,
			LastUpdated:  time.Now(),
		}
	}

	// Update position based on order
	if order.Side == types.OrderSideBuy {
		// Calculate new average price
		totalValue := position.Quantity*position.AveragePrice + order.Quantity*order.Price
		position.Quantity += order.Quantity
		if position.Quantity != 0 {
			position.AveragePrice = totalValue / position.Quantity
		}
	} else {
		// Sell order
		position.Quantity -= order.Quantity
		// For sells, we don't update average price
	}

	position.LastUpdated = time.Now()

	// Store updated position
	e.positionManager.positions.Store(order.Symbol, position)

	return nil
}

// calculateCurrentVaR calculates the current Value at Risk
func (e *RealTimeRiskEngine) calculateCurrentVaR() float64 {
	e.varCalculator.mu.RLock()
	defer e.varCalculator.mu.RUnlock()

	// If VaR calculation is disabled, return 0
	if !e.varCalculator.enabled {
		return 0
	}

	// Return cached VaR if recent
	if time.Since(e.varCalculator.lastCalculation) < time.Hour {
		return e.varCalculator.currentVaR
	}

	// For now, return a simple calculation
	// In a full implementation, this would use historical data and correlation matrices
	return e.varCalculator.currentVaR
}

// calculateVaR calculates Value at Risk using historical simulation
func (e *RealTimeRiskEngine) calculateVaR() {
	e.varCalculator.mu.Lock()
	defer e.varCalculator.mu.Unlock()

	// Simple VaR calculation (in a real implementation, this would be more sophisticated)
	totalPortfolioValue := 0.0

	// Calculate total portfolio value
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		position := value.(*Position)
		totalPortfolioValue += position.Quantity * position.MarketPrice
		return true
	})

	// Simple VaR calculation: assume 2% daily volatility at 95% confidence
	e.varCalculator.currentVaR = totalPortfolioValue * 0.02 * 1.645 // 95% confidence level
	e.varCalculator.lastCalculation = time.Now()

	e.logger.Debug("VaR calculated",
		zap.Float64("var", e.varCalculator.currentVaR),
		zap.Float64("portfolio_value", totalPortfolioValue))
}

// checkCircuitBreakerConditions checks if circuit breaker should be triggered
func (e *RealTimeRiskEngine) checkCircuitBreakerConditions() {
	// Simple circuit breaker logic
	// In a real implementation, this would check price movements, volatility, etc.

	// For now, just check if daily loss exceeds threshold
	e.limitManager.mu.RLock()
	currentDailyLoss := e.limitManager.currentDailyLoss
	e.limitManager.mu.RUnlock()

	lossThreshold := e.config.MaxDailyLoss * 0.8 // Trip at 80% of daily loss limit

	if currentDailyLoss > lossThreshold && !e.circuitBreaker.IsTriggeredFlag {
		e.circuitBreaker.IsTriggeredFlag = true
		e.circuitBreaker.LastTriggeredTime = time.Now()
		e.metrics.CircuitBreakerTrips++

		e.logger.Warn("Circuit breaker tripped",
			zap.Float64("daily_loss", currentDailyLoss),
			zap.Float64("threshold", lossThreshold))

		// Send circuit breaker event
		event := &RiskEvent{
			Type:      RiskEventTypeCircuitBreak,
			Symbol:    "ALL",
			Timestamp: time.Now(),
		}

		select {
		case e.eventChannel <- event:
		default:
			e.logger.Warn("Event channel full, dropping circuit breaker event")
		}
	}

	// Check if circuit breaker should be reset
	if e.circuitBreaker.IsTriggeredFlag && time.Since(e.circuitBreaker.LastTriggeredTime) > e.circuitBreaker.CooldownPeriod {
		e.circuitBreaker.IsTriggeredFlag = false
		e.logger.Info("Circuit breaker reset")
	}
}

// handleCircuitBreakerEvent handles circuit breaker events
func (e *RealTimeRiskEngine) handleCircuitBreakerEvent(event *RiskEvent) {
	e.logger.Warn("Handling circuit breaker event",
		zap.String("symbol", event.Symbol),
		zap.Time("timestamp", event.Timestamp))

	// In a real implementation, this would:
	// - Cancel all pending orders
	// - Notify risk managers
	// - Update risk limits
	// - Send alerts
}

// handleLimitBreachEvent handles limit breach events
func (e *RealTimeRiskEngine) handleLimitBreachEvent(event *RiskEvent) {
	e.logger.Warn("Handling limit breach event",
		zap.String("symbol", event.Symbol),
		zap.Time("timestamp", event.Timestamp))

	// In a real implementation, this would:
	// - Send alerts to risk managers
	// - Adjust position limits
	// - Log the breach for compliance
}

// handleVaRUpdateEvent handles VaR update events
func (e *RealTimeRiskEngine) handleVaRUpdateEvent(event *RiskEvent) {
	e.logger.Debug("Handling VaR update event",
		zap.String("symbol", event.Symbol),
		zap.Time("timestamp", event.Timestamp))

	// Recalculate VaR
	e.calculateVaR()
}
