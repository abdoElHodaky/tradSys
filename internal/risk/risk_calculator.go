package risk

import (
	"context"
	"math"
	"time"

	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"go.uber.org/zap"
)

// RiskCalculator handles risk calculations and assessments
type RiskCalculator struct {
	logger *zap.Logger
}

// NewRiskCalculator creates a new risk calculator
func NewRiskCalculator(logger *zap.Logger) *RiskCalculator {
	return &RiskCalculator{
		logger: logger,
	}
}

// CheckOrderRisk performs comprehensive risk checks on an order
func (rc *RiskCalculator) CheckOrderRisk(ctx context.Context, userID, symbol string, quantity, price float64, side string, positions map[string]*riskengine.Position, limits []*RiskLimit) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Passed:    true,
		RiskLevel: RiskLevelLow,
		Warnings:  make([]string, 0),
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check position limits
	if err := rc.checkPositionLimits(userID, symbol, quantity, side, positions, limits, result); err != nil {
		rc.logger.Error("Position limit check failed", zap.Error(err))
		return result, err
	}

	// Check order size limits
	if err := rc.checkOrderSizeLimits(userID, symbol, quantity, price, limits, result); err != nil {
		rc.logger.Error("Order size limit check failed", zap.Error(err))
		return result, err
	}

	// Check exposure limits
	if err := rc.checkExposureLimits(userID, symbol, quantity, price, side, positions, limits, result); err != nil {
		rc.logger.Error("Exposure limit check failed", zap.Error(err))
		return result, err
	}

	// Check concentration limits
	if err := rc.checkConcentrationLimits(userID, symbol, quantity, price, side, positions, limits, result); err != nil {
		rc.logger.Error("Concentration limit check failed", zap.Error(err))
		return result, err
	}

	// Calculate overall risk level
	rc.calculateOverallRiskLevel(result)

	return result, nil
}

// checkPositionLimits checks position-based risk limits
func (rc *RiskCalculator) checkPositionLimits(userID, symbol string, quantity float64, side string, positions map[string]*riskengine.Position, limits []*RiskLimit, result *RiskCheckResult) error {
	// Get current position
	currentPosition := 0.0
	if pos, exists := positions[symbol]; exists {
		currentPosition = pos.Quantity
	}

	// Calculate new position after order
	newPosition := currentPosition
	if side == "buy" {
		newPosition += quantity
	} else {
		newPosition -= quantity
	}

	// Check position limits
	for _, limit := range limits {
		if !limit.IsEnabled() || limit.Type != RiskLimitTypePosition {
			continue
		}

		if limit.Symbol != "" && limit.Symbol != symbol {
			continue
		}

		if math.Abs(newPosition) > limit.Value {
			result.Passed = false
			result.RiskLevel = RiskLevelHigh
			result.AddWarning("Position limit exceeded")
			result.SetDetail("position_limit_exceeded", map[string]interface{}{
				"current_position": currentPosition,
				"new_position":     newPosition,
				"limit":            limit.Value,
				"limit_id":         limit.ID,
			})
		}
	}

	return nil
}

// checkOrderSizeLimits checks order size limits
func (rc *RiskCalculator) checkOrderSizeLimits(userID, symbol string, quantity, price float64, limits []*RiskLimit, result *RiskCheckResult) error {
	orderValue := quantity * price

	for _, limit := range limits {
		if !limit.IsEnabled() || limit.Type != RiskLimitTypeOrderSize {
			continue
		}

		if limit.Symbol != "" && limit.Symbol != symbol {
			continue
		}

		if orderValue > limit.Value {
			result.Passed = false
			result.RiskLevel = RiskLevelHigh
			result.AddWarning("Order size limit exceeded")
			result.SetDetail("order_size_limit_exceeded", map[string]interface{}{
				"order_value": orderValue,
				"limit":       limit.Value,
				"limit_id":    limit.ID,
			})
		}
	}

	return nil
}

// checkExposureLimits checks exposure limits
func (rc *RiskCalculator) checkExposureLimits(userID, symbol string, quantity, price float64, side string, positions map[string]*riskengine.Position, limits []*RiskLimit, result *RiskCheckResult) error {
	// Calculate current total exposure
	totalExposure := 0.0
	for _, pos := range positions {
		totalExposure += math.Abs(pos.Quantity * pos.AveragePrice)
	}

	// Add new order exposure
	orderExposure := quantity * price
	newTotalExposure := totalExposure + orderExposure

	for _, limit := range limits {
		if !limit.IsEnabled() || limit.Type != RiskLimitTypeExposure {
			continue
		}

		if newTotalExposure > limit.Value {
			result.Passed = false
			result.RiskLevel = RiskLevelHigh
			result.AddWarning("Exposure limit exceeded")
			result.SetDetail("exposure_limit_exceeded", map[string]interface{}{
				"current_exposure": totalExposure,
				"new_exposure":     newTotalExposure,
				"limit":            limit.Value,
				"limit_id":         limit.ID,
			})
		}
	}

	return nil
}

// checkConcentrationLimits checks concentration limits
func (rc *RiskCalculator) checkConcentrationLimits(userID, symbol string, quantity, price float64, side string, positions map[string]*riskengine.Position, limits []*RiskLimit, result *RiskCheckResult) error {
	// Calculate total portfolio value
	totalPortfolioValue := 0.0
	for _, pos := range positions {
		totalPortfolioValue += math.Abs(pos.Quantity * pos.AveragePrice)
	}

	// Calculate symbol concentration after order
	symbolValue := 0.0
	if pos, exists := positions[symbol]; exists {
		symbolValue = math.Abs(pos.Quantity * pos.AveragePrice)
	}

	// Add new order value
	orderValue := quantity * price
	newSymbolValue := symbolValue + orderValue
	newTotalValue := totalPortfolioValue + orderValue

	concentration := 0.0
	if newTotalValue > 0 {
		concentration = newSymbolValue / newTotalValue
	}

	for _, limit := range limits {
		if !limit.IsEnabled() || limit.Type != RiskLimitTypeConcentration {
			continue
		}

		if limit.Symbol != "" && limit.Symbol != symbol {
			continue
		}

		if concentration > limit.Value {
			result.Passed = false
			result.RiskLevel = RiskLevelMedium
			result.AddWarning("Concentration limit exceeded")
			result.SetDetail("concentration_limit_exceeded", map[string]interface{}{
				"concentration": concentration,
				"limit":         limit.Value,
				"limit_id":      limit.ID,
			})
		}
	}

	return nil
}

// CalculatePositionRisk calculates risk metrics for a position
func (rc *RiskCalculator) CalculatePositionRisk(ctx context.Context, position *riskengine.Position, currentPrice float64) (*PositionRiskMetrics, error) {
	metrics := &PositionRiskMetrics{
		Symbol:       position.Symbol,
		UserID:       position.UserID,
		Quantity:     position.Quantity,
		AveragePrice: position.AveragePrice,
		CurrentPrice: currentPrice,
		CalculatedAt: time.Now(),
	}

	// Calculate market value
	metrics.MarketValue = math.Abs(position.Quantity) * currentPrice

	// Calculate unrealized PnL
	metrics.UnrealizedPnL = position.Quantity * (currentPrice - position.AveragePrice)
	if position.AveragePrice != 0 {
		metrics.UnrealizedPnLPercent = metrics.UnrealizedPnL / (math.Abs(position.Quantity) * position.AveragePrice) * 100
	}

	// Calculate VaR (simplified calculation)
	volatility := rc.estimateVolatility(position.Symbol)     // This would come from market data
	metrics.VaR95 = metrics.MarketValue * volatility * 1.645 // 95% confidence
	metrics.VaR99 = metrics.MarketValue * volatility * 2.326 // 99% confidence

	// Calculate expected shortfall (simplified)
	metrics.ExpectedShortfall = metrics.VaR95 * 1.3

	// Determine risk level
	metrics.RiskLevel = rc.determinePositionRiskLevel(metrics)

	return metrics, nil
}

// CalculateAccountRisk calculates risk metrics for an entire account
func (rc *RiskCalculator) CalculateAccountRisk(ctx context.Context, userID string, positions map[string]*riskengine.Position, currentPrices map[string]float64) (*AccountRiskMetrics, error) {
	metrics := &AccountRiskMetrics{
		UserID:       userID,
		Positions:    make([]*PositionRiskMetrics, 0),
		CalculatedAt: time.Now(),
	}

	totalMarketValue := 0.0
	totalUnrealizedPnL := 0.0
	portfolioVaR95 := 0.0
	portfolioVaR99 := 0.0

	// Calculate metrics for each position
	for symbol, position := range positions {
		currentPrice, exists := currentPrices[symbol]
		if !exists {
			rc.logger.Warn("Current price not available for symbol", zap.String("symbol", symbol))
			continue
		}

		posMetrics, err := rc.CalculatePositionRisk(ctx, position, currentPrice)
		if err != nil {
			rc.logger.Error("Failed to calculate position risk", zap.String("symbol", symbol), zap.Error(err))
			continue
		}

		metrics.Positions = append(metrics.Positions, posMetrics)
		totalMarketValue += posMetrics.MarketValue
		totalUnrealizedPnL += posMetrics.UnrealizedPnL
		portfolioVaR95 += posMetrics.VaR95 * posMetrics.VaR95 // Simplified portfolio VaR
		portfolioVaR99 += posMetrics.VaR99 * posMetrics.VaR99
	}

	metrics.TotalMarketValue = totalMarketValue
	metrics.TotalUnrealizedPnL = totalUnrealizedPnL
	if totalMarketValue != 0 {
		metrics.TotalUnrealizedPnLPercent = totalUnrealizedPnL / totalMarketValue * 100
	}

	// Portfolio VaR (simplified - assumes independence)
	metrics.PortfolioVaR95 = math.Sqrt(portfolioVaR95)
	metrics.PortfolioVaR99 = math.Sqrt(portfolioVaR99)

	// Calculate concentration risk
	metrics.ConcentrationRisk = rc.calculateConcentrationRisk(metrics.Positions, totalMarketValue)

	// Determine overall risk level
	metrics.RiskLevel = rc.determineAccountRiskLevel(metrics)

	return metrics, nil
}

// calculateOverallRiskLevel determines the overall risk level based on all checks
func (rc *RiskCalculator) calculateOverallRiskLevel(result *RiskCheckResult) {
	if !result.Passed {
		// Already set to high risk if any check failed
		return
	}

	warningCount := len(result.Warnings)
	if warningCount == 0 {
		result.RiskLevel = RiskLevelLow
	} else if warningCount <= 2 {
		result.RiskLevel = RiskLevelMedium
	} else {
		result.RiskLevel = RiskLevelHigh
	}
}

// determinePositionRiskLevel determines risk level for a position
func (rc *RiskCalculator) determinePositionRiskLevel(metrics *PositionRiskMetrics) RiskLevel {
	// Risk level based on unrealized PnL percentage
	if math.Abs(metrics.UnrealizedPnLPercent) > 20 {
		return RiskLevelCritical
	} else if math.Abs(metrics.UnrealizedPnLPercent) > 10 {
		return RiskLevelHigh
	} else if math.Abs(metrics.UnrealizedPnLPercent) > 5 {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// determineAccountRiskLevel determines risk level for an account
func (rc *RiskCalculator) determineAccountRiskLevel(metrics *AccountRiskMetrics) RiskLevel {
	// Risk level based on total unrealized PnL percentage and concentration
	if math.Abs(metrics.TotalUnrealizedPnLPercent) > 15 || metrics.ConcentrationRisk > 0.5 {
		return RiskLevelCritical
	} else if math.Abs(metrics.TotalUnrealizedPnLPercent) > 8 || metrics.ConcentrationRisk > 0.3 {
		return RiskLevelHigh
	} else if math.Abs(metrics.TotalUnrealizedPnLPercent) > 3 || metrics.ConcentrationRisk > 0.2 {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// calculateConcentrationRisk calculates concentration risk for a portfolio
func (rc *RiskCalculator) calculateConcentrationRisk(positions []*PositionRiskMetrics, totalValue float64) float64 {
	if totalValue == 0 || len(positions) == 0 {
		return 0
	}

	// Find the largest position as percentage of total
	maxConcentration := 0.0
	for _, pos := range positions {
		concentration := pos.MarketValue / totalValue
		if concentration > maxConcentration {
			maxConcentration = concentration
		}
	}

	return maxConcentration
}

// estimateVolatility estimates volatility for a symbol (simplified)
// In a real implementation, this would use historical price data
func (rc *RiskCalculator) estimateVolatility(symbol string) float64 {
	// Default volatility estimates by asset type
	// This is a simplified approach - real implementation would use historical data
	volatilityMap := map[string]float64{
		"BTC":   0.04, // 4% daily volatility for crypto
		"ETH":   0.035,
		"AAPL":  0.02, // 2% daily volatility for stocks
		"GOOGL": 0.025,
		"TSLA":  0.035,
	}

	if vol, exists := volatilityMap[symbol]; exists {
		return vol
	}

	// Default volatility for unknown symbols
	return 0.025
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
