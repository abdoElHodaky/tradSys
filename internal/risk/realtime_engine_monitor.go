package risk

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// GetMetrics returns current risk engine metrics
func (e *RealTimeRiskEngine) GetMetrics() *RiskMetrics {
	return &RiskMetrics{
		ChecksPerSecond:     e.calculateChecksPerSecond(),
		AverageLatency:      e.metrics.AverageLatency,
		MaxLatency:          e.metrics.MaxLatency,
		TotalChecks:         atomic.LoadInt64(&e.metrics.TotalChecks),
		RejectedOrders:      atomic.LoadInt64(&e.metrics.RejectedOrders),
		CircuitBreakerTrips: atomic.LoadInt64(&e.metrics.CircuitBreakerTrips),
		LastUpdateTime:      e.metrics.LastUpdateTime,
	}
}

// GetPortfolioRisk returns comprehensive portfolio risk metrics
func (e *RealTimeRiskEngine) GetPortfolioRisk() *PortfolioRisk {
	var totalVaR float64
	componentVaR := make(map[string]float64)
	marginalVaR := make(map[string]float64)
	
	var totalExposure float64
	var totalValue float64
	
	// Calculate risk metrics for each position
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		symbol := key.(string)
		position := value.(*RealtimePosition)
		
		// Calculate position value and exposure
		positionValue := math.Abs(position.Quantity * position.MarketPrice)
		totalExposure += positionValue
		totalValue += position.Quantity * position.MarketPrice
		
		// Calculate component VaR (simplified)
		if e.config.EnableVaRCalculation {
			componentVaR[symbol] = e.calculatePositionVaR(position)
			totalVaR += componentVaR[symbol]
			
			// Calculate marginal VaR (simplified)
			marginalVaR[symbol] = componentVaR[symbol] / positionValue
		}
		
		return true
	})
	
	// Calculate concentration risk (Herfindahl index)
	concentrationRisk := e.calculateConcentrationRisk()
	
	// Calculate leverage ratio
	leverageRatio := float64(0)
	if totalValue != 0 {
		leverageRatio = totalExposure / math.Abs(totalValue)
	}
	
	return &PortfolioRisk{
		TotalVaR:          totalVaR,
		ComponentVaR:      componentVaR,
		MarginalVaR:       marginalVaR,
		ConcentrationRisk: concentrationRisk,
		LeverageRatio:     leverageRatio,
		BetaToMarket:      1.0, // Simplified - would calculate against market index
		Timestamp:         time.Now(),
	}
}

// GetAllPositions returns all current positions
func (e *RealTimeRiskEngine) GetAllPositions() map[string]*RealtimePosition {
	positions := make(map[string]*RealtimePosition)
	
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		symbol := key.(string)
		position := value.(*RealtimePosition)
		
		// Create a copy to avoid race conditions
		positions[symbol] = &RealtimePosition{
			Symbol:         position.Symbol,
			Quantity:       position.Quantity,
			AveragePrice:   position.AveragePrice,
			MarketPrice:    position.MarketPrice,
			UnrealizedPnL:  position.UnrealizedPnL,
			RealizedPnL:    position.RealizedPnL,
			Delta:          position.Delta,
			Gamma:          position.Gamma,
			Vega:           position.Vega,
			Theta:          position.Theta,
			LastUpdateTime: position.LastUpdateTime,
		}
		
		return true
	})
	
	return positions
}

// GetRiskAlerts returns current risk alerts
func (e *RealTimeRiskEngine) GetRiskAlerts() []*RiskAlert {
	var alerts []*RiskAlert
	
	// Check position limit alerts
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		symbol := key.(string)
		position := value.(*RealtimePosition)
		
		limit := e.getPositionLimit(symbol)
		if math.Abs(position.Quantity) > limit*0.9 { // Alert at 90% of limit
			alerts = append(alerts, &RiskAlert{
				ID:           generateAlertID(),
				Type:         "position_limit",
				Symbol:       symbol,
				Message:      "Position approaching limit",
				Severity:     SeverityWarning,
				Threshold:    limit,
				CurrentValue: math.Abs(position.Quantity),
				Timestamp:    time.Now(),
				Acknowledged: false,
			})
		}
		
		return true
	})
	
	// Check daily loss alert
	e.limitManager.mu.RLock()
	currentLoss := e.limitManager.currentDailyLoss
	lossLimit := e.limitManager.dailyLossLimit
	e.limitManager.mu.RUnlock()
	
	if currentLoss > lossLimit*0.8 { // Alert at 80% of limit
		alerts = append(alerts, &RiskAlert{
			ID:           generateAlertID(),
			Type:         "daily_loss",
			Symbol:       "PORTFOLIO",
			Message:      "Daily loss approaching limit",
			Severity:     SeverityWarning,
			Threshold:    lossLimit,
			CurrentValue: currentLoss,
			Timestamp:    time.Now(),
			Acknowledged: false,
		})
	}
	
	return alerts
}

// CalculateVaR calculates Value at Risk for a specific symbol
func (e *RealTimeRiskEngine) CalculateVaR(symbol string) (*VaRResult, error) {
	if !e.config.EnableVaRCalculation {
		return nil, fmt.Errorf("VaR calculation is disabled")
	}
	
	position := e.getPosition(symbol)
	if position.Quantity == 0 {
		return &VaRResult{
			Symbol:          symbol,
			VaR:             0,
			ConfidenceLevel: e.config.VaRConfidenceLevel,
			TimeHorizon:     e.config.VaRTimeHorizon.String(),
			Timestamp:       time.Now(),
		}, nil
	}
	
	var95 := e.calculatePositionVaR(position)
	
	return &VaRResult{
		Symbol:          symbol,
		VaR:             var95,
		ConfidenceLevel: e.config.VaRConfidenceLevel,
		TimeHorizon:     e.config.VaRTimeHorizon.String(),
		Timestamp:       time.Now(),
	}, nil
}

// RunStressTest runs stress test scenarios
func (e *RealTimeRiskEngine) RunStressTest(scenarios []*StressTestScenario) []*StressTestResult {
	if !e.config.StressTestEnabled {
		return nil
	}
	
	var results []*StressTestResult
	
	for _, scenario := range scenarios {
		if !scenario.Enabled {
			continue
		}
		
		result := e.runSingleStressTest(scenario)
		results = append(results, result)
	}
	
	return results
}

// runSingleStressTest runs a single stress test scenario
func (e *RealTimeRiskEngine) runSingleStressTest(scenario *StressTestScenario) *StressTestResult {
	var totalPnL float64
	var worstPnL float64
	var worstPosition string
	
	// Apply shocks to each position
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		symbol := key.(string)
		position := value.(*RealtimePosition)
		
		// Get shock for this symbol
		shock, exists := scenario.Shocks[symbol]
		if !exists {
			shock = scenario.Shocks["DEFAULT"] // Use default shock if symbol-specific not found
		}
		
		// Calculate stressed price
		stressedPrice := position.MarketPrice * (1 + shock)
		
		// Calculate P&L under stress
		pnl := position.Quantity * (stressedPrice - position.AveragePrice)
		totalPnL += pnl
		
		// Track worst position
		if pnl < worstPnL {
			worstPnL = pnl
			worstPosition = symbol
		}
		
		return true
	})
	
	return &StressTestResult{
		Scenario:      scenario.Name,
		TotalPnL:      totalPnL,
		WorstPosition: worstPosition,
		WorstPnL:      worstPnL,
		Timestamp:     time.Now(),
	}
}

// LogRiskSummary logs comprehensive risk summary
func (e *RealTimeRiskEngine) LogRiskSummary() {
	metrics := e.GetMetrics()
	portfolioRisk := e.GetPortfolioRisk()
	alerts := e.GetRiskAlerts()
	
	e.logger.Info("Risk Engine Summary",
		zap.Float64("checks_per_second", metrics.ChecksPerSecond),
		zap.Duration("avg_latency", metrics.AverageLatency),
		zap.Duration("max_latency", metrics.MaxLatency),
		zap.Int64("total_checks", metrics.TotalChecks),
		zap.Int64("rejected_orders", metrics.RejectedOrders),
		zap.Float64("total_var", portfolioRisk.TotalVaR),
		zap.Float64("concentration_risk", portfolioRisk.ConcentrationRisk),
		zap.Float64("leverage_ratio", portfolioRisk.LeverageRatio),
		zap.Int("active_alerts", len(alerts)))
}

// MonitorRiskLimits continuously monitors risk limits
func (e *RealTimeRiskEngine) MonitorRiskLimits(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30) // Check every 30 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		case <-ticker.C:
			e.checkAllRiskLimits()
		}
	}
}

// checkAllRiskLimits checks all risk limits and generates alerts
func (e *RealTimeRiskEngine) checkAllRiskLimits() {
	// Check position limits
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		symbol := key.(string)
		position := value.(*RealtimePosition)
		
		limit := e.getPositionLimit(symbol)
		if math.Abs(position.Quantity) > limit {
			e.publishEvent(&RiskEvent{
				Type:      EventLimitBreach,
				Symbol:    symbol,
				Severity:  SeverityError,
				Message:   fmt.Sprintf("Position limit breached: %f > %f", math.Abs(position.Quantity), limit),
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"position_size": position.Quantity,
					"limit":         limit,
				},
			})
		}
		
		return true
	})
	
	// Check daily loss limit
	e.limitManager.mu.RLock()
	currentLoss := e.limitManager.currentDailyLoss
	lossLimit := e.limitManager.dailyLossLimit
	e.limitManager.mu.RUnlock()
	
	if currentLoss > lossLimit {
		e.publishEvent(&RiskEvent{
			Type:      EventLimitBreach,
			Symbol:    "PORTFOLIO",
			Severity:  SeverityCritical,
			Message:   fmt.Sprintf("Daily loss limit breached: %f > %f", currentLoss, lossLimit),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"current_loss": currentLoss,
				"limit":        lossLimit,
			},
		})
	}
}

// UpdateMarketData updates market data for risk calculations
func (e *RealTimeRiskEngine) UpdateMarketData(symbol string, price float64) {
	if positionInterface, exists := e.positionManager.positions.Load(symbol); exists {
		position := positionInterface.(*RealtimePosition)
		
		// Update market price
		oldPrice := position.MarketPrice
		position.MarketPrice = price
		
		// Recalculate unrealized P&L
		if position.AveragePrice > 0 {
			position.UnrealizedPnL = position.Quantity * (price - position.AveragePrice)
		}
		
		// Update Greeks (simplified)
		if oldPrice > 0 {
			priceChange := (price - oldPrice) / oldPrice
			position.Delta = priceChange // Simplified delta calculation
		}
		
		position.LastUpdateTime = time.Now()
		e.positionManager.positions.Store(symbol, position)
		
		// Publish position update event
		e.publishEvent(&RiskEvent{
			Type:      EventPositionUpdate,
			Symbol:    symbol,
			Severity:  SeverityInfo,
			Message:   fmt.Sprintf("Position updated for %s", symbol),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"old_price":       oldPrice,
				"new_price":       price,
				"unrealized_pnl":  position.UnrealizedPnL,
			},
		})
	}
}

// Helper functions for calculations

// calculateChecksPerSecond calculates checks per second
func (e *RealTimeRiskEngine) calculateChecksPerSecond() float64 {
	totalChecks := atomic.LoadInt64(&e.metrics.TotalChecks)
	if totalChecks == 0 {
		return 0
	}
	
	// Simplified calculation - in production would use time windows
	elapsed := time.Since(e.metrics.LastUpdateTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	
	return float64(totalChecks) / elapsed
}

// calculatePositionVaR calculates VaR for a single position
func (e *RealTimeRiskEngine) calculatePositionVaR(position *RealtimePosition) float64 {
	// Simplified VaR calculation
	// In production, would use historical returns and proper statistical models
	
	positionValue := math.Abs(position.Quantity * position.MarketPrice)
	volatility := 0.02 // Assume 2% daily volatility
	
	// 95% VaR using normal distribution
	zScore := 1.645 // 95% confidence level
	var95 := positionValue * volatility * zScore
	
	return var95
}

// calculateConcentrationRisk calculates portfolio concentration risk
func (e *RealTimeRiskEngine) calculateConcentrationRisk() float64 {
	var totalValue float64
	var sumSquares float64
	
	// Calculate total portfolio value and sum of squares
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		position := value.(*RealtimePosition)
		positionValue := math.Abs(position.Quantity * position.MarketPrice)
		totalValue += positionValue
		
		return true
	})
	
	if totalValue == 0 {
		return 0
	}
	
	// Calculate sum of squared weights
	e.positionManager.positions.Range(func(key, value interface{}) bool {
		position := value.(*RealtimePosition)
		positionValue := math.Abs(position.Quantity * position.MarketPrice)
		weight := positionValue / totalValue
		sumSquares += weight * weight
		
		return true
	})
	
	// Herfindahl index
	return sumSquares
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// GetCircuitBreakerStatus returns circuit breaker status
func (e *RealTimeRiskEngine) GetCircuitBreakerStatus() map[string]interface{} {
	e.circuitBreaker.mu.RLock()
	defer e.circuitBreaker.mu.RUnlock()
	
	return map[string]interface{}{
		"enabled":           e.circuitBreaker.enabled,
		"state":             e.circuitBreaker.state,
		"failure_count":     e.circuitBreaker.failureCount,
		"success_count":     e.circuitBreaker.successCount,
		"last_failure_time": e.circuitBreaker.lastFailureTime,
		"threshold":         e.circuitBreaker.threshold,
		"timeout":           e.circuitBreaker.timeout,
	}
}

// ResetMetrics resets all risk engine metrics
func (e *RealTimeRiskEngine) ResetMetrics() {
	atomic.StoreInt64(&e.metrics.TotalChecks, 0)
	atomic.StoreInt64(&e.metrics.RejectedOrders, 0)
	atomic.StoreInt64(&e.metrics.CircuitBreakerTrips, 0)
	e.metrics.AverageLatency = 0
	e.metrics.MaxLatency = 0
	e.metrics.LastUpdateTime = time.Now()
	
	e.logger.Info("Risk engine metrics reset")
}

// GetHealthStatus returns overall health status of risk engine
func (e *RealTimeRiskEngine) GetHealthStatus() map[string]interface{} {
	metrics := e.GetMetrics()
	
	isHealthy := true
	issues := []string{}
	
	// Check latency
	if metrics.AverageLatency > e.config.MaxLatency {
		isHealthy = false
		issues = append(issues, "High latency")
	}
	
	// Check rejection rate
	rejectionRate := float64(0)
	if metrics.TotalChecks > 0 {
		rejectionRate = float64(metrics.RejectedOrders) / float64(metrics.TotalChecks) * 100
	}
	
	if rejectionRate > 10 { // More than 10% rejection rate
		isHealthy = false
		issues = append(issues, "High rejection rate")
	}
	
	// Check circuit breaker
	if e.isCircuitBreakerOpen() {
		isHealthy = false
		issues = append(issues, "Circuit breaker open")
	}
	
	return map[string]interface{}{
		"healthy":         isHealthy,
		"issues":          issues,
		"avg_latency":     metrics.AverageLatency,
		"max_latency":     metrics.MaxLatency,
		"rejection_rate":  rejectionRate,
		"total_checks":    metrics.TotalChecks,
		"last_update":     metrics.LastUpdateTime,
	}
}
