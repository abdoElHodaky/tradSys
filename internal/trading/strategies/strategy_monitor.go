package strategies

import (
	"context"
	"math"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// GetMetrics returns current strategy engine metrics
func (e *UnifiedStrategyEngine) GetMetrics() *StrategyMetrics {
	return &StrategyMetrics{
		TotalOrders:      atomic.LoadInt64(&e.metrics.TotalOrders),
		SuccessfulTrades: atomic.LoadInt64(&e.metrics.SuccessfulTrades),
		TotalPnL:         e.metrics.TotalPnL,
		WinRate:          e.metrics.WinRate,
		AverageReturn:    e.metrics.AverageReturn,
		MaxDrawdown:      e.metrics.MaxDrawdown,
		SharpeRatio:      e.metrics.SharpeRatio,
		LastUpdateTime:   e.metrics.LastUpdateTime,
	}
}

// GetStrategyPerformance returns detailed performance metrics for a strategy
func (e *UnifiedStrategyEngine) GetStrategyPerformance(strategyID string) (*StrategyPerformance, error) {
	strategy, err := e.GetStrategy(strategyID)
	if err != nil {
		return nil, err
	}

	metrics := strategy.GetMetrics()

	// Calculate additional performance metrics
	winRate := float64(0)
	if metrics.TotalOrders > 0 {
		winRate = float64(metrics.SuccessfulTrades) / float64(metrics.TotalOrders) * 100
	}

	return &StrategyPerformance{
		StrategyID:     strategyID,
		Status:         StrategyStatusRunning, // Simplified
		TotalTrades:    metrics.TotalOrders,
		WinningTrades:  metrics.SuccessfulTrades,
		LosingTrades:   metrics.TotalOrders - metrics.SuccessfulTrades,
		WinRate:        winRate,
		TotalPnL:       metrics.TotalPnL,
		AverageWin:     metrics.AverageReturn,
		AverageLoss:    0, // Would calculate from trade history
		ProfitFactor:   0, // Would calculate from win/loss ratio
		MaxDrawdown:    metrics.MaxDrawdown,
		SharpeRatio:    metrics.SharpeRatio,
		CalmarRatio:    0,          // Would calculate Calmar ratio
		LastTradeTime:  time.Now(), // Would track from actual trades
		LastUpdateTime: metrics.LastUpdateTime,
	}, nil
}

// GetAllStrategyPerformance returns performance metrics for all strategies
func (e *UnifiedStrategyEngine) GetAllStrategyPerformance() map[string]*StrategyPerformance {
	strategies := e.GetAllStrategies()
	performance := make(map[string]*StrategyPerformance)

	for id := range strategies {
		if perf, err := e.GetStrategyPerformance(id); err == nil {
			performance[id] = perf
		}
	}

	return performance
}

// GetRiskMetrics returns risk-related metrics for the strategy engine
func (e *UnifiedStrategyEngine) GetRiskMetrics() *RiskMetrics {
	positions := e.monitor.GetAllPositions()

	var totalExposure float64
	var totalPnL float64
	var returns []float64

	for _, position := range positions {
		exposure := math.Abs(position.Quantity * position.CurrentPrice)
		totalExposure += exposure
		totalPnL += position.UnrealizedPnL + position.RealizedPnL

		// Calculate return for this position
		if position.AveragePrice > 0 {
			returnPct := (position.CurrentPrice - position.AveragePrice) / position.AveragePrice
			returns = append(returns, returnPct)
		}
	}

	// Calculate volatility
	volatility := calculateVolatility(returns)

	// Calculate VaR (simplified)
	var95 := calculateVaR(returns, 0.95)
	var99 := calculateVaR(returns, 0.99)

	return &RiskMetrics{
		CurrentExposure:   totalExposure,
		MaxExposure:       e.config.RiskLimits.MaxPositionSize,
		VaR95:             var95,
		VaR99:             var99,
		ExpectedShortfall: var99 * 1.2, // Simplified calculation
		Beta:              1.0,         // Would calculate against benchmark
		Volatility:        volatility,
	}
}

// GetExecutionMetrics returns execution-related metrics
func (e *UnifiedStrategyEngine) GetExecutionMetrics() *ExecutionMetrics {
	totalOrders := atomic.LoadInt64(&e.metrics.TotalOrders)
	successfulTrades := atomic.LoadInt64(&e.metrics.SuccessfulTrades)

	fillRate := float64(0)
	if totalOrders > 0 {
		fillRate = float64(successfulTrades) / float64(totalOrders) * 100
	}

	return &ExecutionMetrics{
		OrdersSubmitted: totalOrders,
		OrdersFilled:    successfulTrades,
		OrdersCancelled: 0, // Would track from order management
		OrdersRejected:  totalOrders - successfulTrades,
		FillRate:        fillRate,
		AverageSlippage: 0.001, // Would calculate from execution data
		AverageLatency:  50.0,  // Would measure actual latency
	}
}

// monitoringLoop runs the strategy monitoring loop
func (e *UnifiedStrategyEngine) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute) // Monitor every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.updateMetrics()
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// updateMetrics updates strategy metrics
func (e *UnifiedStrategyEngine) updateMetrics() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Update overall engine metrics
	var totalPnL float64
	var totalReturns []float64

	positions := e.monitor.GetAllPositions()
	for _, position := range positions {
		totalPnL += position.UnrealizedPnL + position.RealizedPnL

		if position.AveragePrice > 0 {
			returnPct := (position.CurrentPrice - position.AveragePrice) / position.AveragePrice
			totalReturns = append(totalReturns, returnPct)
		}
	}

	// Update metrics
	e.metrics.TotalPnL = totalPnL
	e.metrics.AverageReturn = calculateMean(totalReturns)
	e.metrics.MaxDrawdown = calculateMaxDrawdown(totalReturns)
	e.metrics.SharpeRatio = calculateSharpeRatio(totalReturns)
	e.metrics.LastUpdateTime = time.Now()

	// Calculate win rate
	totalOrders := atomic.LoadInt64(&e.metrics.TotalOrders)
	successfulTrades := atomic.LoadInt64(&e.metrics.SuccessfulTrades)
	if totalOrders > 0 {
		e.metrics.WinRate = float64(successfulTrades) / float64(totalOrders) * 100
	}

	e.logger.Debug("Updated strategy engine metrics",
		zap.Float64("total_pnl", e.metrics.TotalPnL),
		zap.Float64("win_rate", e.metrics.WinRate),
		zap.Float64("sharpe_ratio", e.metrics.SharpeRatio),
		zap.Int64("total_orders", totalOrders))
}

// LogPerformanceSummary logs a comprehensive performance summary
func (e *UnifiedStrategyEngine) LogPerformanceSummary() {
	metrics := e.GetMetrics()
	riskMetrics := e.GetRiskMetrics()
	executionMetrics := e.GetExecutionMetrics()

	e.logger.Info("Strategy Engine Performance Summary",
		zap.Int64("total_orders", metrics.TotalOrders),
		zap.Int64("successful_trades", metrics.SuccessfulTrades),
		zap.Float64("total_pnl", metrics.TotalPnL),
		zap.Float64("win_rate", metrics.WinRate),
		zap.Float64("average_return", metrics.AverageReturn),
		zap.Float64("max_drawdown", metrics.MaxDrawdown),
		zap.Float64("sharpe_ratio", metrics.SharpeRatio),
		zap.Float64("current_exposure", riskMetrics.CurrentExposure),
		zap.Float64("volatility", riskMetrics.Volatility),
		zap.Float64("fill_rate", executionMetrics.FillRate),
		zap.Int("active_strategies", len(e.GetEnabledStrategies())))
}

// MonitorRiskLimits monitors and enforces risk limits
func (e *UnifiedStrategyEngine) MonitorRiskLimits() error {
	riskMetrics := e.GetRiskMetrics()

	// Check position size limit
	if riskMetrics.CurrentExposure > e.config.RiskLimits.MaxPositionSize {
		e.logger.Warn("Position size limit exceeded",
			zap.Float64("current_exposure", riskMetrics.CurrentExposure),
			zap.Float64("max_position_size", e.config.RiskLimits.MaxPositionSize))

		// In a real implementation, would take corrective action
		return e.reduceExposure()
	}

	// Check daily loss limit
	if e.metrics.TotalPnL < -e.config.RiskLimits.MaxDailyLoss {
		e.logger.Error("Daily loss limit exceeded",
			zap.Float64("current_pnl", e.metrics.TotalPnL),
			zap.Float64("max_daily_loss", e.config.RiskLimits.MaxDailyLoss))

		// Stop all strategies
		return e.emergencyStop()
	}

	// Check drawdown limit
	if e.metrics.MaxDrawdown > e.config.RiskLimits.MaxDrawdown {
		e.logger.Warn("Drawdown limit exceeded",
			zap.Float64("current_drawdown", e.metrics.MaxDrawdown),
			zap.Float64("max_drawdown", e.config.RiskLimits.MaxDrawdown))

		// Reduce position sizes
		return e.reducePositionSizes()
	}

	return nil
}

// reduceExposure reduces overall exposure when limits are exceeded
func (e *UnifiedStrategyEngine) reduceExposure() error {
	e.logger.Info("Reducing exposure due to risk limit breach")

	// In a real implementation, would:
	// 1. Close largest positions first
	// 2. Reduce position sizes proportionally
	// 3. Temporarily disable high-risk strategies

	return nil
}

// emergencyStop stops all strategies in case of emergency
func (e *UnifiedStrategyEngine) emergencyStop() error {
	e.logger.Error("Emergency stop triggered - stopping all strategies")

	e.mu.RLock()
	for _, strategy := range e.strategies {
		if err := strategy.Stop(); err != nil {
			e.logger.Error("Failed to stop strategy during emergency",
				zap.String("strategy", strategy.GetID()),
				zap.Error(err))
		}
	}
	e.mu.RUnlock()

	return nil
}

// reducePositionSizes reduces position sizes when drawdown limits are exceeded
func (e *UnifiedStrategyEngine) reducePositionSizes() error {
	e.logger.Info("Reducing position sizes due to drawdown limit")

	// In a real implementation, would reduce position sizes by a percentage
	return nil
}

// Helper functions for statistical calculations

// calculateVolatility calculates the volatility of returns
func calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := calculateMean(returns)
	var sumSquaredDiffs float64

	for _, ret := range returns {
		diff := ret - mean
		sumSquaredDiffs += diff * diff
	}

	variance := sumSquaredDiffs / float64(len(returns)-1)
	return math.Sqrt(variance)
}

// calculateMean calculates the mean of a slice of float64
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, value := range values {
		sum += value
	}

	return sum / float64(len(values))
}

// calculateVaR calculates Value at Risk at given confidence level
func calculateVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Simplified VaR calculation using normal distribution assumption
	mean := calculateMean(returns)
	volatility := calculateVolatility(returns)

	// Z-score for confidence level (simplified)
	var zScore float64
	switch confidence {
	case 0.95:
		zScore = 1.645
	case 0.99:
		zScore = 2.326
	default:
		zScore = 1.645
	}

	return mean - zScore*volatility
}

// calculateMaxDrawdown calculates the maximum drawdown from returns
func calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	var peak float64 = 1.0
	var maxDrawdown float64
	var cumulative float64 = 1.0

	for _, ret := range returns {
		cumulative *= (1 + ret)
		if cumulative > peak {
			peak = cumulative
		}

		drawdown := (peak - cumulative) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateSharpeRatio calculates the Sharpe ratio
func calculateSharpeRatio(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	mean := calculateMean(returns)
	volatility := calculateVolatility(returns)

	if volatility == 0 {
		return 0
	}

	// Assuming risk-free rate of 0 for simplicity
	return mean / volatility
}
