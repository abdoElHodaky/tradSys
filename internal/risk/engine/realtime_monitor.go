package engine

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// monitorCircuitBreaker monitors circuit breaker conditions
func (e *RealTimeRiskEngine) monitorCircuitBreaker(ctx context.Context) {
	ticker := time.NewTicker(time.Second) // Check every second
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.checkCircuitBreakerConditions()
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// checkCircuitBreakerConditions checks if circuit breaker should be triggered
func (e *RealTimeRiskEngine) checkCircuitBreakerConditions() {
	// This is a simplified implementation
	// Real systems would monitor market conditions and trigger based on volatility, etc.

	e.circuitBreaker.mu.Lock()
	defer e.circuitBreaker.mu.Unlock()

	// Check if circuit breaker should be reset
	if e.circuitBreaker.isTripped &&
		time.Since(e.circuitBreaker.tripTime) > e.circuitBreaker.cooldownPeriod {
		e.circuitBreaker.isTripped = false
		e.logger.Info("Circuit breaker reset after cooldown period")
	}
}

// IsTripped returns whether the circuit breaker is currently tripped
func (cb *CircuitBreaker) IsTripped() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.isTripped
}

// GetReferencePrice returns the reference price
func (cb *CircuitBreaker) GetReferencePrice() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.referencePrice
}

// SetReferencePrice sets the reference price
func (cb *CircuitBreaker) SetReferencePrice(price float64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.referencePrice = price
}

// Trip triggers the circuit breaker
func (cb *CircuitBreaker) Trip() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.isTripped = true
	cb.tripTime = time.Now()
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.isTripped = false
}

// GetLastTriggered returns the last time the circuit breaker was triggered
func (cb *CircuitBreaker) GetLastTriggered() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.tripTime
}

// GetCooldownPeriod returns the cooldown period
func (cb *CircuitBreaker) GetCooldownPeriod() time.Duration {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.cooldownPeriod
}

// GetPriceChangeThreshold returns the price change threshold
func (cb *CircuitBreaker) GetPriceChangeThreshold() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.priceChangeThreshold
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(priceChangeThreshold float64, cooldownPeriod time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		enabled:              true,
		priceChangeThreshold: priceChangeThreshold,
		cooldownPeriod:       cooldownPeriod,
		isTripped:            false,
		referencePrice:       0,
	}
}

// GetMetrics returns current risk metrics
func (e *RealTimeRiskEngine) GetMetrics() *RiskMetrics {
	return e.metrics
}

// GetPosition returns the current position for a symbol
func (e *RealTimeRiskEngine) GetPosition(symbol string) *Position {
	return e.getPosition(symbol)
}

// Trade represents a trade (imported from order matching)
type Trade struct {
	ID        string
	Symbol    string
	Price     float64
	Quantity  float64
	TakerSide types.OrderSide
	Timestamp time.Time
}

// getPortfolioPositions returns all current positions
func (e *RealTimeRiskEngine) getPortfolioPositions() map[string]*Position {
	e.positionManager.mu.RLock()
	defer e.positionManager.mu.RUnlock()

	positions := make(map[string]*Position)
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		symbol := key.(string)
		position := value.(*Position)
		if position.Quantity != 0 {
			positions[symbol] = &Position{
				Symbol:         position.Symbol,
				Quantity:       position.Quantity,
				AveragePrice:   position.AveragePrice,
				LastUpdateTime: position.LastUpdateTime,
			}
		}
		return true
	})

	return positions
}

// calculatePortfolioValue calculates the total portfolio value
func (e *RealTimeRiskEngine) calculatePortfolioValue(positions map[string]*Position) float64 {
	totalValue := 0.0

	for symbol, position := range positions {
		// In production, you would get current market price
		// For now, use the average price as an approximation
		positionValue := position.Quantity * position.AveragePrice
		totalValue += positionValue

		e.logger.Debug("Position value calculated",
			zap.String("symbol", symbol),
			zap.Float64("quantity", position.Quantity),
			zap.Float64("avg_price", position.AveragePrice),
			zap.Float64("position_value", positionValue))
	}

	return totalValue
}
