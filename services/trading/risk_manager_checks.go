package trading

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// checkOrderValue validates order value against limits
func (rm *RiskManager) checkOrderValue(orderValue float64, result *RiskCheckResult) error {
	if orderValue > rm.config.MaxOrderValue {
		violation := RiskViolation{
			Rule:        "MAX_ORDER_VALUE",
			Description: "Order value exceeds maximum allowed",
			Severity:    "CRITICAL",
			Value:       orderValue,
			Limit:       rm.config.MaxOrderValue,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Reduce order size to stay within $%.2f limit", rm.config.MaxOrderValue))
	}

	return nil
}

// checkPositionSize validates position size limits
func (rm *RiskManager) checkPositionSize(ctx context.Context, order *interfaces.Order, profile *UserRiskProfile, result *RiskCheckResult) error {
	// Get current positions
	positions, err := rm.riskStore.GetPositions(ctx, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user positions: %w", err)
	}

	// Calculate new position size
	var currentPosition float64
	for _, pos := range positions {
		if pos.Symbol == order.Symbol {
			currentPosition = pos.Quantity
			break
		}
	}

	var newPosition float64
	if order.Side == interfaces.OrderSideBuy {
		newPosition = currentPosition + order.Quantity
	} else {
		newPosition = currentPosition - order.Quantity
	}

	newPositionValue := newPosition * order.Price
	maxPositionSize := profile.MaxPositionSize
	if maxPositionSize == 0 {
		maxPositionSize = rm.config.MaxPositionSize
	}

	if newPositionValue > maxPositionSize {
		violation := RiskViolation{
			Rule:        "MAX_POSITION_SIZE",
			Description: "Position size exceeds maximum allowed",
			Severity:    "HIGH",
			Value:       newPositionValue,
			Limit:       maxPositionSize,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Reduce position size to stay within $%.2f limit", maxPositionSize))
	}

	return nil
}

// checkDailyVolume validates daily trading volume limits
func (rm *RiskManager) checkDailyVolume(ctx context.Context, order *interfaces.Order, profile *UserRiskProfile, result *RiskCheckResult) error {
	today := time.Now().Truncate(24 * time.Hour)
	currentVolume, err := rm.riskStore.GetDailyVolume(ctx, order.UserID, today)
	if err != nil {
		return fmt.Errorf("failed to get daily volume: %w", err)
	}

	orderValue := order.Price * order.Quantity
	newDailyVolume := currentVolume + orderValue

	maxDailyVolume := profile.MaxDailyVolume
	if maxDailyVolume == 0 {
		maxDailyVolume = rm.config.MaxDailyVolume
	}

	if newDailyVolume > maxDailyVolume {
		violation := RiskViolation{
			Rule:        "MAX_DAILY_VOLUME",
			Description: "Daily trading volume exceeds maximum allowed",
			Severity:    "HIGH",
			Value:       newDailyVolume,
			Limit:       maxDailyVolume,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Daily volume limit of $%.2f would be exceeded", maxDailyVolume))
	}

	return nil
}

// checkConcentration validates portfolio concentration limits
func (rm *RiskManager) checkConcentration(ctx context.Context, order *interfaces.Order, profile *UserRiskProfile, result *RiskCheckResult) error {
	// Get all positions
	positions, err := rm.riskStore.GetPositions(ctx, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user positions: %w", err)
	}

	// Calculate total portfolio value
	var totalPortfolioValue float64
	var symbolValue float64

	for _, pos := range positions {
		totalPortfolioValue += pos.MarketValue
		if pos.Symbol == order.Symbol {
			symbolValue = pos.MarketValue
		}
	}

	// Add new order value
	orderValue := order.Price * order.Quantity
	if order.Side == interfaces.OrderSideBuy {
		symbolValue += orderValue
		totalPortfolioValue += orderValue
	}

	if totalPortfolioValue > 0 {
		concentration := symbolValue / totalPortfolioValue
		concentrationLimit := profile.ConcentrationLimit
		if concentrationLimit == 0 {
			concentrationLimit = rm.config.ConcentrationLimit
		}

		if concentration > concentrationLimit {
			violation := RiskViolation{
				Rule:        "CONCENTRATION_LIMIT",
				Description: "Portfolio concentration exceeds maximum allowed",
				Severity:    "MEDIUM",
				Value:       concentration * 100, // Convert to percentage
				Limit:       concentrationLimit * 100,
			}
			result.Violations = append(result.Violations, violation)
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Diversify portfolio to stay within %.1f%% concentration limit", concentrationLimit*100))
		}
	}

	return nil
}

// checkVolatility validates volatility-based risk limits
func (rm *RiskManager) checkVolatility(ctx context.Context, order *interfaces.Order, result *RiskCheckResult) error {
	// Get volatility data for the symbol
	volatility, err := rm.calculator.GetVolatility(order.Symbol)
	if err != nil {
		// If we can't get volatility data, log warning but don't fail the check
		result.Recommendations = append(result.Recommendations,
			"Unable to assess volatility risk - consider manual review")
		return nil
	}

	if volatility.Volatility > rm.config.VolatilityThreshold {
		violation := RiskViolation{
			Rule:        "VOLATILITY_THRESHOLD",
			Description: "Asset volatility exceeds threshold",
			Severity:    "MEDIUM",
			Value:       volatility.Volatility * 100, // Convert to percentage
			Limit:       rm.config.VolatilityThreshold * 100,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("High volatility asset (%.1f%%) - consider reducing position size", volatility.Volatility*100))
	}

	return nil
}

// checkPortfolioViolations checks for portfolio-level violations
func (rm *RiskManager) checkPortfolioViolations(portfolioRisk *PortfolioRisk, result *RiskCheckResult) {
	// Check VaR limits
	if portfolioRisk.VaR95 > portfolioRisk.TotalValue*0.05 { // 5% VaR limit
		violation := RiskViolation{
			Rule:        "VAR_95_LIMIT",
			Description: "Portfolio VaR 95% exceeds acceptable limit",
			Severity:    "HIGH",
			Value:       portfolioRisk.VaR95,
			Limit:       portfolioRisk.TotalValue * 0.05,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations,
			"Consider reducing portfolio risk through diversification")
	}

	// Check volatility limits
	if portfolioRisk.Volatility > 0.25 { // 25% volatility limit
		violation := RiskViolation{
			Rule:        "PORTFOLIO_VOLATILITY",
			Description: "Portfolio volatility exceeds acceptable limit",
			Severity:    "MEDIUM",
			Value:       portfolioRisk.Volatility * 100,
			Limit:       25.0,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations,
			"High portfolio volatility - consider adding defensive positions")
	}

	// Check maximum drawdown
	if portfolioRisk.MaxDrawdown < -0.20 { // -20% max drawdown limit
		violation := RiskViolation{
			Rule:        "MAX_DRAWDOWN",
			Description: "Portfolio maximum drawdown exceeds acceptable limit",
			Severity:    "HIGH",
			Value:       portfolioRisk.MaxDrawdown * 100,
			Limit:       -20.0,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations,
			"Excessive drawdown detected - review risk management strategy")
	}
}

// checkRealTimeViolations checks for real-time risk violations
func (rm *RiskManager) checkRealTimeViolations(positions []*Position, profile *UserRiskProfile) []*RiskAlert {
	var alerts []*RiskAlert

	// Calculate total exposure
	var totalExposure float64
	var maxSinglePosition float64
	positionMap := make(map[string]float64)

	for _, pos := range positions {
		exposure := pos.MarketValue
		totalExposure += exposure
		positionMap[pos.Symbol] = exposure

		if exposure > maxSinglePosition {
			maxSinglePosition = exposure
		}
	}

	// Check concentration
	if totalExposure > 0 {
		concentration := maxSinglePosition / totalExposure
		if concentration > profile.ConcentrationLimit {
			alert := &RiskAlert{
				ID:        generateRiskAlertID(),
				UserID:    profile.UserID,
				AlertType: string(AlertTypeConcentration),
				Severity:  string(SeverityHigh),
				Message:   fmt.Sprintf("Portfolio concentration %.1f%% exceeds limit %.1f%%", concentration*100, profile.ConcentrationLimit*100),
				Triggered: time.Now(),
				Metadata: map[string]interface{}{
					"concentration": concentration,
					"limit":         profile.ConcentrationLimit,
				},
			}
			alerts = append(alerts, alert)
		}
	}

	// Check individual position limits
	for symbol, exposure := range positionMap {
		if exposure > profile.MaxPositionSize {
			alert := &RiskAlert{
				ID:        generateRiskAlertID(),
				UserID:    profile.UserID,
				AlertType: string(AlertTypePositionLimit),
				Severity:  string(SeverityHigh),
				Message:   fmt.Sprintf("Position in %s ($%.2f) exceeds limit ($%.2f)", symbol, exposure, profile.MaxPositionSize),
				Triggered: time.Now(),
				Metadata: map[string]interface{}{
					"symbol":   symbol,
					"exposure": exposure,
					"limit":    profile.MaxPositionSize,
				},
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// calculateRiskScore calculates overall risk score based on violations
func (rm *RiskManager) calculateRiskScore(violations []RiskViolation) float64 {
	if len(violations) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, violation := range violations {
		switch violation.Severity {
		case "LOW":
			totalScore += 1.0
		case "MEDIUM":
			totalScore += 3.0
		case "HIGH":
			totalScore += 7.0
		case "CRITICAL":
			totalScore += 15.0
		}
	}

	// Normalize score to 0-100 scale
	maxPossibleScore := float64(len(violations)) * 15.0
	if maxPossibleScore > 0 {
		return (totalScore / maxPossibleScore) * 100.0
	}

	return 0.0
}

// RiskCalculator methods

// GetVolatility retrieves volatility data for a symbol
func (rc *RiskCalculator) GetVolatility(symbol string) (*VolatilityData, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Check cache first
	if data, exists := rc.volatilityCache[symbol]; exists {
		if time.Since(data.LastUpdated) < rc.cacheTTL {
			return data, nil
		}
	}

	// In a real implementation, this would fetch from a data provider
	// For now, return mock data
	volatility := &VolatilityData{
		Symbol:      symbol,
		Volatility:  0.20, // 20% default volatility
		LastUpdated: time.Now(),
	}

	// Update cache
	rc.volatilityCache[symbol] = volatility

	return volatility, nil
}

// CalculatePortfolioVolatility calculates portfolio volatility
func (rc *RiskCalculator) CalculatePortfolioVolatility(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	// Simplified calculation - in reality would use correlation matrix
	var totalValue float64
	var weightedVolatility float64

	for _, pos := range positions {
		volatility, err := rc.GetVolatility(pos.Symbol)
		if err != nil {
			continue
		}

		weight := pos.MarketValue
		totalValue += weight
		weightedVolatility += weight * volatility.Volatility
	}

	if totalValue > 0 {
		return weightedVolatility / totalValue
	}

	return 0.0
}

// CalculateVaR calculates Value at Risk
func (rc *RiskCalculator) CalculateVaR(positions []*Position, confidence float64) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	// Simplified VaR calculation using normal distribution
	portfolioValue := 0.0
	for _, pos := range positions {
		portfolioValue += pos.MarketValue
	}

	portfolioVolatility := rc.CalculatePortfolioVolatility(positions)

	// Z-score for confidence level
	var zScore float64
	if confidence >= 0.99 {
		zScore = 2.33
	} else if confidence >= 0.95 {
		zScore = 1.65
	} else {
		zScore = 1.28
	}

	return portfolioValue * portfolioVolatility * zScore
}

// CalculatePortfolioBeta calculates portfolio beta
func (rc *RiskCalculator) CalculatePortfolioBeta(positions []*Position) float64 {
	// Simplified beta calculation - assume average beta of 1.0
	return 1.0
}

// CalculateSharpeRatio calculates Sharpe ratio
func (rc *RiskCalculator) CalculateSharpeRatio(positions []*Position) float64 {
	// Simplified Sharpe ratio calculation
	return 0.8 // Mock value
}

// CalculateMaxDrawdown calculates maximum drawdown
func (rc *RiskCalculator) CalculateMaxDrawdown(positions []*Position) float64 {
	// Simplified max drawdown calculation
	var totalUnrealizedPL float64
	var totalValue float64

	for _, pos := range positions {
		totalUnrealizedPL += pos.UnrealizedPL
		totalValue += pos.MarketValue
	}

	if totalValue > 0 {
		return totalUnrealizedPL / totalValue
	}

	return 0.0
}

// Utility functions

// generateRiskCheckID generates a unique risk check ID
func generateRiskCheckID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "RC-" + hex.EncodeToString(bytes)
}

// generateRiskAlertID generates a unique risk alert ID
func generateRiskAlertID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "RA-" + hex.EncodeToString(bytes)
}
