package trading

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// NewRiskManager creates a new risk manager
func NewRiskManager(config *RiskManagerConfig, riskStore RiskStore) *RiskManager {
	if config == nil {
		config = GetDefaultRiskManagerConfig()
	}

	return &RiskManager{
		config:     config,
		riskStore:  riskStore,
		calculator: NewRiskCalculator(),
	}
}

// NewRiskCalculator creates a new risk calculator
func NewRiskCalculator() *RiskCalculator {
	return &RiskCalculator{
		volatilityCache: make(map[string]*VolatilityData),
		cacheTTL:        5 * time.Minute,
	}
}

// GetDefaultRiskManagerConfig returns default risk manager configuration
func GetDefaultRiskManagerConfig() *RiskManagerConfig {
	return &RiskManagerConfig{
		MaxPositionSize:     1000000.0, // $1M
		MaxDailyVolume:      5000000.0, // $5M
		MaxOrderValue:       500000.0,  // $500K
		VolatilityThreshold: 0.30,      // 30%
		ConcentrationLimit:  0.20,      // 20%
		EnableRealTimeCheck: true,
		RiskCheckTimeout:    5 * time.Second,
	}
}

// ValidateOrder performs comprehensive risk validation for an order
func (rm *RiskManager) ValidateOrder(ctx context.Context, order *interfaces.Order) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Approved:        true,
		RiskScore:       0.0,
		Violations:      []RiskViolation{},
		Recommendations: []string{},
		CheckedAt:       time.Now(),
		Metadata:        make(map[string]interface{}),
	}

	// Get user risk profile
	profile, err := rm.riskStore.GetUserRisk(ctx, order.UserID)
	if err != nil {
		return result, fmt.Errorf("failed to get user risk profile: %w", err)
	}

	if !profile.IsActive {
		result.Approved = false
		result.Reason = "user risk profile is inactive"
		return result, nil
	}

	// 1. Order value check
	orderValue := order.Price * order.Quantity
	if err := rm.checkOrderValue(orderValue, result); err != nil {
		return result, err
	}

	// 2. Position size check
	if err := rm.checkPositionSize(ctx, order, profile, result); err != nil {
		return result, err
	}

	// 3. Daily volume check
	if err := rm.checkDailyVolume(ctx, order, profile, result); err != nil {
		return result, err
	}

	// 4. Concentration check
	if err := rm.checkConcentration(ctx, order, profile, result); err != nil {
		return result, err
	}

	// 5. Volatility check
	if err := rm.checkVolatility(ctx, order, result); err != nil {
		return result, err
	}

	// Calculate overall risk score
	result.RiskScore = rm.calculateRiskScore(result.Violations)

	// Determine approval based on violations
	if len(result.Violations) > 0 {
		for _, violation := range result.Violations {
			if violation.Severity == "CRITICAL" {
				result.Approved = false
				result.Reason = "critical risk violation detected"
				break
			}
		}
	}

	// Save risk check record
	record := &RiskCheckRecord{
		ID:        generateRiskCheckID(),
		UserID:    order.UserID,
		OrderID:   order.ID,
		Result:    result,
		Order:     order,
		CheckedAt: time.Now(),
	}

	if err := rm.riskStore.SaveRiskCheck(ctx, record); err != nil {
		// Log error but don't fail the check
		fmt.Printf("Failed to save risk check record: %v\n", err)
	}

	return result, nil
}

// GetUserRiskProfile retrieves a user's risk profile
func (rm *RiskManager) GetUserRiskProfile(ctx context.Context, userID string) (*UserRiskProfile, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.riskStore.GetUserRisk(ctx, userID)
}

// UpdateUserRiskProfile updates a user's risk profile
func (rm *RiskManager) UpdateUserRiskProfile(ctx context.Context, profile *UserRiskProfile) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	profile.LastUpdated = time.Now()
	return rm.riskStore.UpdateUserRisk(ctx, profile)
}

// GetUserPositions retrieves all positions for a user
func (rm *RiskManager) GetUserPositions(ctx context.Context, userID string) ([]*Position, error) {
	return rm.riskStore.GetPositions(ctx, userID)
}

// CalculatePortfolioRisk calculates portfolio-level risk metrics
func (rm *RiskManager) CalculatePortfolioRisk(ctx context.Context, userID string) (*PortfolioRisk, error) {
	positions, err := rm.riskStore.GetPositions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	if len(positions) == 0 {
		return &PortfolioRisk{
			UserID:         userID,
			LastCalculated: time.Now(),
		}, nil
	}

	// Calculate total portfolio value
	var totalValue float64
	for _, pos := range positions {
		totalValue += pos.MarketValue
	}

	// Calculate basic risk metrics
	portfolioRisk := &PortfolioRisk{
		UserID:         userID,
		TotalValue:     totalValue,
		LastCalculated: time.Now(),
	}

	// Calculate volatility and other metrics
	portfolioRisk.Volatility = rm.calculator.CalculatePortfolioVolatility(positions)
	portfolioRisk.VaR95 = rm.calculator.CalculateVaR(positions, 0.95)
	portfolioRisk.VaR99 = rm.calculator.CalculateVaR(positions, 0.99)
	portfolioRisk.Beta = rm.calculator.CalculatePortfolioBeta(positions)
	portfolioRisk.Sharpe = rm.calculator.CalculateSharpeRatio(positions)
	portfolioRisk.MaxDrawdown = rm.calculator.CalculateMaxDrawdown(positions)

	return portfolioRisk, nil
}

// GetRiskMetrics calculates and returns risk metrics for a user
func (rm *RiskManager) GetRiskMetrics(ctx context.Context, userID string) (*RiskMetrics, error) {
	positions, err := rm.riskStore.GetPositions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	dailyVolume, err := rm.riskStore.GetDailyVolume(ctx, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get daily volume: %w", err)
	}

	// Calculate total exposure
	var totalExposure float64
	var maxSinglePosition float64

	for _, pos := range positions {
		exposure := pos.MarketValue
		totalExposure += exposure

		if exposure > maxSinglePosition {
			maxSinglePosition = exposure
		}
	}

	// Calculate concentration ratio
	var concentrationRatio float64
	if totalExposure > 0 {
		concentrationRatio = maxSinglePosition / totalExposure
	}

	return &RiskMetrics{
		UserID:             userID,
		TotalExposure:      totalExposure,
		ConcentrationRatio: concentrationRatio,
		DailyVolumeUsed:    dailyVolume,
		PositionCount:      len(positions),
		LastCalculated:     time.Now(),
	}, nil
}

// CreateRiskAlert creates a new risk alert
func (rm *RiskManager) CreateRiskAlert(userID string, alertType AlertType, severity ViolationSeverity, message string, metadata map[string]interface{}) *RiskAlert {
	return &RiskAlert{
		ID:           generateRiskAlertID(),
		UserID:       userID,
		AlertType:    string(alertType),
		Severity:     string(severity),
		Message:      message,
		Triggered:    time.Now(),
		Acknowledged: false,
		Metadata:     metadata,
	}
}

// MonitorRealTimeRisk performs real-time risk monitoring
func (rm *RiskManager) MonitorRealTimeRisk(ctx context.Context, userID string) error {
	if !rm.config.EnableRealTimeCheck {
		return nil
	}

	// Get current positions
	positions, err := rm.riskStore.GetPositions(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// Get user risk profile
	profile, err := rm.riskStore.GetUserRisk(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user risk profile: %w", err)
	}

	// Check for risk violations
	alerts := rm.checkRealTimeViolations(positions, profile)

	// Process alerts (in a real implementation, this would send notifications)
	for _, alert := range alerts {
		fmt.Printf("Risk Alert: %s - %s\n", alert.AlertType, alert.Message)
	}

	return nil
}

// SetRiskLimits updates risk limits for a user
func (rm *RiskManager) SetRiskLimits(ctx context.Context, userID string, limits *RiskLimits) error {
	profile, err := rm.riskStore.GetUserRisk(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user risk profile: %w", err)
	}

	// Update profile with new limits
	profile.MaxPositionSize = limits.MaxPositionSize
	profile.MaxDailyVolume = limits.MaxDailyVolume
	profile.ConcentrationLimit = limits.ConcentrationLimit
	profile.LastUpdated = time.Now()

	return rm.riskStore.UpdateUserRisk(ctx, profile)
}

// GetRiskStatus returns the current risk status for a user
func (rm *RiskManager) GetRiskStatus(ctx context.Context, userID string) (RiskStatus, error) {
	profile, err := rm.riskStore.GetUserRisk(ctx, userID)
	if err != nil {
		return StatusInactive, fmt.Errorf("failed to get user risk profile: %w", err)
	}

	if !profile.IsActive {
		return StatusInactive, nil
	}

	// Check for any critical violations
	metrics, err := rm.GetRiskMetrics(ctx, userID)
	if err != nil {
		return StatusInactive, fmt.Errorf("failed to get risk metrics: %w", err)
	}

	// Determine status based on metrics
	if metrics.ConcentrationRatio > profile.ConcentrationLimit {
		return StatusSuspended, nil
	}

	if metrics.DailyVolumeUsed > profile.MaxDailyVolume {
		return StatusBlocked, nil
	}

	return StatusActive, nil
}

// ValidatePortfolio performs comprehensive portfolio validation
func (rm *RiskManager) ValidatePortfolio(ctx context.Context, userID string) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Approved:        true,
		RiskScore:       0.0,
		Violations:      []RiskViolation{},
		Recommendations: []string{},
		CheckedAt:       time.Now(),
		Metadata:        make(map[string]interface{}),
	}

	// Get portfolio risk metrics
	portfolioRisk, err := rm.CalculatePortfolioRisk(ctx, userID)
	if err != nil {
		return result, fmt.Errorf("failed to calculate portfolio risk: %w", err)
	}

	// Check portfolio-level violations
	rm.checkPortfolioViolations(portfolioRisk, result)

	// Calculate overall risk score
	result.RiskScore = rm.calculateRiskScore(result.Violations)

	return result, nil
}
