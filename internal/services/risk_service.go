package services

import (
	"context"
	"fmt"
	"math"
)

// RiskServiceImpl implements the RiskService interface
type RiskServiceImpl struct {
	riskLimits map[string]*RiskLimits
}

// NewRiskService creates a new risk service instance
func NewRiskService() RiskService {
	return &RiskServiceImpl{
		riskLimits: make(map[string]*RiskLimits),
	}
}

// CheckRisk performs risk assessment on an order
func (s *RiskServiceImpl) CheckRisk(ctx context.Context, order *Order) (*RiskCheckResult, error) {
	if order == nil {
		return nil, fmt.Errorf("order cannot be nil")
	}

	result := &RiskCheckResult{
		Approved: true,
		Reasons:  []string{},
		Warnings: []string{},
	}

	// Get risk limits for the account
	limits, exists := s.riskLimits[order.AccountID]
	if !exists {
		// Use default limits if none are set
		limits = s.getDefaultRiskLimits()
	}

	// Check position size limit
	orderValue := order.Quantity * order.Price
	if orderValue > limits.MaxPositionSize {
		result.Approved = false
		result.Reasons = append(result.Reasons, 
			fmt.Sprintf("Order value %.2f exceeds maximum position size %.2f", 
				orderValue, limits.MaxPositionSize))
	}

	// Check if order value is close to limit (warning)
	if orderValue > limits.MaxPositionSize*0.8 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Order value %.2f is close to maximum position size %.2f",
				orderValue, limits.MaxPositionSize))
	}

	// Additional risk checks could be added here:
	// - Daily loss limits
	// - Leverage checks
	// - Exposure limits
	// - Volatility checks
	// - Correlation checks

	return result, nil
}

// ValidatePosition validates a trading position against risk limits
func (s *RiskServiceImpl) ValidatePosition(ctx context.Context, position *Position) error {
	if position == nil {
		return fmt.Errorf("position cannot be nil")
	}

	// Get risk limits for the account
	limits, exists := s.riskLimits[position.AccountID]
	if !exists {
		limits = s.getDefaultRiskLimits()
	}

	// Check position size
	if math.Abs(position.MarketValue) > limits.MaxPositionSize {
		return fmt.Errorf("position market value %.2f exceeds maximum position size %.2f",
			math.Abs(position.MarketValue), limits.MaxPositionSize)
	}

	// Check unrealized P&L
	if position.UnrealizedPL < -limits.MaxDailyLoss {
		return fmt.Errorf("position unrealized loss %.2f exceeds maximum daily loss %.2f",
			math.Abs(position.UnrealizedPL), limits.MaxDailyLoss)
	}

	return nil
}

// GetRiskMetrics calculates risk metrics for a portfolio
func (s *RiskServiceImpl) GetRiskMetrics(ctx context.Context, portfolio *Portfolio) (*RiskMetrics, error) {
	if portfolio == nil {
		return nil, fmt.Errorf("portfolio cannot be nil")
	}

	metrics := &RiskMetrics{}

	// Calculate total exposure
	totalExposure := 0.0
	totalValue := 0.0
	totalPL := 0.0

	for _, position := range portfolio.Positions {
		totalExposure += math.Abs(position.MarketValue)
		totalValue += position.MarketValue
		totalPL += position.UnrealizedPL
	}

	metrics.Exposure = totalExposure

	// Simple VaR calculation (1% of total value)
	metrics.VaR = totalValue * 0.01

	// Simple volatility calculation (placeholder)
	metrics.Volatility = 0.15 // 15% annualized volatility

	// Simple Sharpe ratio calculation (placeholder)
	if metrics.Volatility > 0 {
		metrics.Sharpe = (totalPL / totalValue) / metrics.Volatility
	}

	// Simple beta calculation (placeholder - would need market data)
	metrics.Beta = 1.0

	// Simple max drawdown calculation (placeholder)
	metrics.MaxDrawdown = math.Min(0, totalPL/totalValue)

	return metrics, nil
}

// MonitorRisk performs ongoing risk monitoring
func (s *RiskServiceImpl) MonitorRisk(ctx context.Context) error {
	// In a real implementation, this would:
	// - Monitor all active positions
	// - Check for limit breaches
	// - Send alerts
	// - Trigger automatic risk controls
	
	// For now, this is a placeholder
	return nil
}

// GetRiskLimits retrieves risk limits for an account
func (s *RiskServiceImpl) GetRiskLimits(ctx context.Context, accountID string) (*RiskLimits, error) {
	limits, exists := s.riskLimits[accountID]
	if !exists {
		return s.getDefaultRiskLimits(), nil
	}

	return limits, nil
}

// UpdateRiskLimits updates risk limits for an account
func (s *RiskServiceImpl) UpdateRiskLimits(ctx context.Context, accountID string, limits *RiskLimits) error {
	if limits == nil {
		return fmt.Errorf("risk limits cannot be nil")
	}

	// Validate limits
	if err := s.validateRiskLimits(limits); err != nil {
		return fmt.Errorf("invalid risk limits: %w", err)
	}

	s.riskLimits[accountID] = limits
	return nil
}

// getDefaultRiskLimits returns default risk limits
func (s *RiskServiceImpl) getDefaultRiskLimits() *RiskLimits {
	return &RiskLimits{
		MaxPositionSize: 100000.0, // $100,000
		MaxDailyLoss:    10000.0,  // $10,000
		MaxLeverage:     10.0,     // 10:1
		MaxExposure:     500000.0, // $500,000
	}
}

// validateRiskLimits validates risk limit values
func (s *RiskServiceImpl) validateRiskLimits(limits *RiskLimits) error {
	if limits.MaxPositionSize <= 0 {
		return fmt.Errorf("max position size must be positive")
	}
	if limits.MaxDailyLoss <= 0 {
		return fmt.Errorf("max daily loss must be positive")
	}
	if limits.MaxLeverage <= 0 {
		return fmt.Errorf("max leverage must be positive")
	}
	if limits.MaxExposure <= 0 {
		return fmt.Errorf("max exposure must be positive")
	}

	return nil
}
