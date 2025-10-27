package engine

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// publishEvent publishes a risk event to the event channel
func (e *RealTimeRiskEngine) publishEvent(event *RiskEvent) {
	select {
	case e.eventChannel <- event:
	default:
		e.logger.Warn("Risk event channel full, dropping event",
			zap.String("event_type", string(event.Type)),
			zap.String("symbol", event.Symbol))
	}
}

// processEvents processes risk events
func (e *RealTimeRiskEngine) processEvents(ctx context.Context) {
	for {
		select {
		case event := <-e.eventChannel:
			e.handleRiskEvent(event)
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// handleRiskEvent handles a risk event
func (e *RealTimeRiskEngine) handleRiskEvent(event *RiskEvent) {
	switch event.Type {
	case EventLimitBreach:
		e.logger.Error("Risk limit breach",
			zap.String("symbol", event.Symbol),
			zap.String("message", event.Message))
	case EventCircuitBreaker:
		e.logger.Warn("Circuit breaker event",
			zap.String("symbol", event.Symbol),
			zap.String("message", event.Message))
	default:
		e.logger.Info("Risk event",
			zap.String("type", string(event.Type)),
			zap.String("symbol", event.Symbol))
	}
}

// calculateVaRPeriodically calculates VaR periodically
func (e *RealTimeRiskEngine) calculateVaRPeriodically(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5) // Calculate VaR every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.calculateVaR()
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// calculateVaR calculates Value at Risk
func (e *RealTimeRiskEngine) calculateVaR() {
	// This is a simplified VaR calculation
	// Real implementations would use more sophisticated models
	e.varCalculator.mu.Lock()
	defer e.varCalculator.mu.Unlock()

	// Calculate portfolio VaR using historical simulation
	// Get current positions
	positions := e.getPortfolioPositions()
	if len(positions) == 0 {
		e.logger.Debug("No positions for VaR calculation")
		return
	}

	// Calculate portfolio value
	portfolioValue := e.calculatePortfolioValue(positions)

	// Historical simulation VaR (simplified)
	// In production, this would use actual historical price data
	volatility := 0.02 // 2% daily volatility assumption

	// Calculate VaR using normal distribution approximation
	// VaR = Portfolio Value * Z-score * Volatility
	zScore := 1.645 // 95% confidence level z-score
	var95 := portfolioValue * zScore * volatility

	// Store VaR result
	e.varCalculator.currentVaR = var95
	e.varCalculator.lastCalculation = time.Now()

	e.logger.Info("VaR calculation completed",
		zap.Float64("portfolio_value", portfolioValue),
		zap.Float64("var_95", var95),
		zap.Float64("volatility", volatility),
		zap.Int("positions_count", len(positions)))
}

