package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LimitsManager handles risk limits management
type LimitsManager struct {
	logger      *zap.Logger
	limits      map[string]map[string]*RiskLimit // userID -> limitType -> limit
	mu          sync.RWMutex
	calculator  *RiskCalculator
}

// NewLimitsManager creates a new limits manager
func NewLimitsManager(logger *zap.Logger, calculator *RiskCalculator) *LimitsManager {
	return &LimitsManager{
		logger:     logger,
		limits:     make(map[string]map[string]*RiskLimit),
		calculator: calculator,
	}
}

// SetLimit sets a risk limit for a user
func (lm *LimitsManager) SetLimit(userID string, limit *RiskLimit) error {
	if limit == nil {
		return fmt.Errorf("limit cannot be nil")
	}
	
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}
	
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if lm.limits[userID] == nil {
		lm.limits[userID] = make(map[string]*RiskLimit)
	}
	
	lm.limits[userID][string(limit.Type)] = limit
	
	lm.logger.Info("Risk limit set",
		zap.String("user_id", userID),
		zap.String("limit_type", string(limit.Type)),
		zap.Float64("value", limit.Value),
		zap.String("symbol", limit.Symbol))
	
	return nil
}

// GetLimit gets a risk limit for a user
func (lm *LimitsManager) GetLimit(userID string, limitType RiskLimitType) (*RiskLimit, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	userLimits, exists := lm.limits[userID]
	if !exists {
		return nil, fmt.Errorf("no limits found for user %s", userID)
	}
	
	limit, exists := userLimits[string(limitType)]
	if !exists {
		return nil, fmt.Errorf("limit type %s not found for user %s", limitType, userID)
	}
	
	return limit, nil
}

// GetAllLimits gets all risk limits for a user
func (lm *LimitsManager) GetAllLimits(userID string) (map[string]*RiskLimit, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	userLimits, exists := lm.limits[userID]
	if !exists {
		return nil, fmt.Errorf("no limits found for user %s", userID)
	}
	
	// Return a copy to prevent external modification
	result := make(map[string]*RiskLimit)
	for k, v := range userLimits {
		result[k] = v
	}
	
	return result, nil
}

// RemoveLimit removes a risk limit for a user
func (lm *LimitsManager) RemoveLimit(userID string, limitType RiskLimitType) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	userLimits, exists := lm.limits[userID]
	if !exists {
		return fmt.Errorf("no limits found for user %s", userID)
	}
	
	delete(userLimits, string(limitType))
	
	lm.logger.Info("Risk limit removed",
		zap.String("user_id", userID),
		zap.String("limit_type", string(limitType)))
	
	return nil
}

// CheckPositionLimit checks if a position violates risk limits
func (lm *LimitsManager) CheckPositionLimit(userID, symbol string, quantity, price float64) (*RiskCheckResult, error) {
	positionValue := quantity * price
	
	// Check position size limit
	positionLimit, err := lm.GetLimit(userID, RiskLimitTypePositionSize)
	if err == nil {
		if positionLimit.Symbol == "" || positionLimit.Symbol == symbol {
			if positionValue > positionLimit.Value {
				return &RiskCheckResult{
					Passed:     false,
					RiskLevel:  RiskLevelHigh,
					Violations: []string{fmt.Sprintf("Position size %.2f exceeds limit %.2f", positionValue, positionLimit.Value)},
					Warnings:   []string{},
					CheckedAt:  time.Now(),
				}, nil
			}
		}
	}
	
	// Check daily loss limit
	dailyLossLimit, err := lm.GetLimit(userID, RiskLimitTypeDailyLoss)
	if err == nil {
		// In a real implementation, this would calculate actual daily P&L
		// For now, we'll assume the position represents potential loss
		if positionValue > dailyLossLimit.Value {
			return &RiskCheckResult{
				Passed:     false,
				RiskLevel:  RiskLevelHigh,
				Violations: []string{fmt.Sprintf("Potential daily loss %.2f exceeds limit %.2f", positionValue, dailyLossLimit.Value)},
				Warnings:   []string{},
				CheckedAt:  time.Now(),
			}, nil
		}
	}
	
	return &RiskCheckResult{
		Passed:     true,
		RiskLevel:  RiskLevelLow,
		Violations: []string{},
		Warnings:   []string{},
		CheckedAt:  time.Now(),
	}, nil
}

// CheckDrawdownLimit checks if drawdown violates limits
func (lm *LimitsManager) CheckDrawdownLimit(userID string, currentValue, peakValue float64) (*RiskCheckResult, error) {
	drawdown := lm.calculator.CalculateDrawdown(userID, currentValue, peakValue)
	
	// Check max drawdown limit
	drawdownLimit, err := lm.GetLimit(userID, RiskLimitTypeMaxDrawdown)
	if err == nil {
		if drawdown > drawdownLimit.Value {
			return &RiskCheckResult{
				Passed:     false,
				RiskLevel:  RiskLevelHigh,
				Violations: []string{fmt.Sprintf("Drawdown %.2f%% exceeds limit %.2f%%", drawdown*100, drawdownLimit.Value*100)},
				Warnings:   []string{},
				CheckedAt:  time.Now(),
			}, nil
		}
		
		// Warning at 80% of limit
		if drawdown > drawdownLimit.Value*0.8 {
			return &RiskCheckResult{
				Passed:     true,
				RiskLevel:  RiskLevelMedium,
				Violations: []string{},
				Warnings:   []string{fmt.Sprintf("Drawdown %.2f%% approaching limit %.2f%%", drawdown*100, drawdownLimit.Value*100)},
				CheckedAt:  time.Now(),
			}, nil
		}
	}
	
	return &RiskCheckResult{
		Passed:     true,
		RiskLevel:  RiskLevelLow,
		Violations: []string{},
		Warnings:   []string{},
		CheckedAt:  time.Now(),
	}, nil
}

// CheckConcentrationLimit checks if concentration violates limits
func (lm *LimitsManager) CheckConcentrationLimit(userID, symbol string, positionValue, totalPortfolioValue float64) (*RiskCheckResult, error) {
	if totalPortfolioValue <= 0 {
		return &RiskCheckResult{
			Passed:     true,
			RiskLevel:  RiskLevelLow,
			Violations: []string{},
			Warnings:   []string{},
			CheckedAt:  time.Now(),
		}, nil
	}
	
	concentration := positionValue / totalPortfolioValue
	
	// Check concentration limit
	concentrationLimit, err := lm.GetLimit(userID, RiskLimitTypeConcentration)
	if err == nil {
		if concentrationLimit.Symbol == "" || concentrationLimit.Symbol == symbol {
			if concentration > concentrationLimit.Value {
				return &RiskCheckResult{
					Passed:     false,
					RiskLevel:  RiskLevelHigh,
					Violations: []string{fmt.Sprintf("Concentration %.2f%% exceeds limit %.2f%%", concentration*100, concentrationLimit.Value*100)},
					Warnings:   []string{},
					CheckedAt:  time.Now(),
				}, nil
			}
			
			// Warning at 80% of limit
			if concentration > concentrationLimit.Value*0.8 {
				return &RiskCheckResult{
					Passed:     true,
					RiskLevel:  RiskLevelMedium,
					Violations: []string{},
					Warnings:   []string{fmt.Sprintf("Concentration %.2f%% approaching limit %.2f%%", concentration*100, concentrationLimit.Value*100)},
					CheckedAt:  time.Now(),
				}, nil
			}
		}
	}
	
	return &RiskCheckResult{
		Passed:     true,
		RiskLevel:  RiskLevelLow,
		Violations: []string{},
		Warnings:   []string{},
		CheckedAt:  time.Now(),
	}, nil
}

// CheckAllLimits checks all applicable limits for a position
func (lm *LimitsManager) CheckAllLimits(ctx context.Context, userID, symbol string, quantity, price float64) (*RiskCheckResult, error) {
	var allViolations []string
	var allWarnings []string
	var highestRiskLevel RiskLevel = RiskLevelLow
	
	// Check position limit
	positionResult, err := lm.CheckPositionLimit(userID, symbol, quantity, price)
	if err == nil {
		allViolations = append(allViolations, positionResult.Violations...)
		allWarnings = append(allWarnings, positionResult.Warnings...)
		if positionResult.RiskLevel == RiskLevelHigh {
			highestRiskLevel = RiskLevelHigh
		} else if positionResult.RiskLevel == RiskLevelMedium && highestRiskLevel != RiskLevelHigh {
			highestRiskLevel = RiskLevelMedium
		}
	}
	
	// In a real implementation, we would also check:
	// - Portfolio-level limits
	// - Sector concentration limits
	// - Leverage limits
	// - VaR limits
	// - Correlation limits
	
	return &RiskCheckResult{
		Passed:     len(allViolations) == 0,
		RiskLevel:  highestRiskLevel,
		Violations: allViolations,
		Warnings:   allWarnings,
		CheckedAt:  time.Now(),
	}, nil
}

// SetDefaultLimits sets default risk limits for a user
func (lm *LimitsManager) SetDefaultLimits(userID string) error {
	defaultLimits := []*RiskLimit{
		{
			Type:      RiskLimitTypePositionSize,
			Value:     1000000, // $1M position limit
			Symbol:    "",      // Applies to all symbols
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Type:      RiskLimitTypeMaxDrawdown,
			Value:     0.20, // 20% max drawdown
			Symbol:    "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Type:      RiskLimitTypeConcentration,
			Value:     0.25, // 25% max concentration
			Symbol:    "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Type:      RiskLimitTypeDailyLoss,
			Value:     100000, // $100K daily loss limit
			Symbol:    "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	
	for _, limit := range defaultLimits {
		if err := lm.SetLimit(userID, limit); err != nil {
			return fmt.Errorf("failed to set default limit %s: %w", limit.Type, err)
		}
	}
	
	lm.logger.Info("Default risk limits set", zap.String("user_id", userID))
	return nil
}
