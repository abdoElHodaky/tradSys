package risk

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Risk limit errors
var (
	ErrRiskLimitNotFound = errors.New("risk limit not found")
)

// LimitManager handles risk limit management and checking
type LimitManager struct {
	// RiskLimits is a map of user ID to risk limits
	RiskLimits map[string][]*RiskLimit
	// RiskLimitCache is a cache for frequently accessed risk limits
	RiskLimitCache *cache.Cache
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
}

// NewLimitManager creates a new limit manager
func NewLimitManager(logger *zap.Logger) *LimitManager {
	return &LimitManager{
		RiskLimits:     make(map[string][]*RiskLimit),
		RiskLimitCache: cache.New(5*time.Minute, 10*time.Minute),
		logger:         logger,
	}
}

// AddRiskLimit adds a new risk limit
func (lm *LimitManager) AddRiskLimit(ctx context.Context, limit *RiskLimit) (*RiskLimit, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Set timestamps
	now := time.Now()
	limit.CreatedAt = now
	limit.UpdatedAt = now
	limit.Enabled = true

	// Add to user's limits
	if lm.RiskLimits[limit.UserID] == nil {
		lm.RiskLimits[limit.UserID] = make([]*RiskLimit, 0)
	}
	lm.RiskLimits[limit.UserID] = append(lm.RiskLimits[limit.UserID], limit)

	// Update cache
	cacheKey := limit.UserID + ":limits"
	lm.RiskLimitCache.Set(cacheKey, lm.RiskLimits[limit.UserID], cache.DefaultExpiration)

	lm.logger.Info("Risk limit added",
		zap.String("userID", limit.UserID),
		zap.String("symbol", limit.Symbol),
		zap.String("type", string(limit.Type)),
		zap.Float64("value", limit.Limit),
	)

	return limit, nil
}

// CheckRiskLimits checks if an order violates any risk limits
func (lm *LimitManager) CheckRiskLimits(ctx context.Context, userID, symbol string, orderSize, currentPrice float64, currentPosition float64) (*RiskCheckResult, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := &RiskCheckResult{
		Approved:     true,
		Violations:   make([]string, 0),
		RiskScore:    0.0,
		MaxOrderSize: orderSize,
	}

	// Get user's risk limits
	userLimits, exists := lm.RiskLimits[userID]
	if !exists {
		// No limits defined, approve by default
		return result, nil
	}

	orderValue := orderSize * currentPrice
	newPosition := currentPosition + orderSize
	newPositionValue := newPosition * currentPrice

	for _, limit := range userLimits {
		if !limit.Enabled {
			continue
		}

		// Check if limit applies to this symbol (empty symbol means global)
		if limit.Symbol != "" && limit.Symbol != symbol {
			continue
		}

		violation := false
		violationMsg := ""

		switch limit.Type {
		case RiskLimitTypeMaxOrderSize:
			if orderSize > limit.Limit {
				violation = true
				violationMsg = "Order size exceeds maximum allowed"
				result.MaxOrderSize = limit.Limit
			}

		case RiskLimitTypeMaxOrderValue:
			if orderValue > limit.Limit {
				violation = true
				violationMsg = "Order value exceeds maximum allowed"
			}

		case RiskLimitTypeMaxPositionSize:
			if abs(newPosition) > limit.Limit {
				violation = true
				violationMsg = "Position size would exceed maximum allowed"
			}

		case RiskLimitTypeMaxPositionValue:
			if abs(newPositionValue) > limit.Limit {
				violation = true
				violationMsg = "Position value would exceed maximum allowed"
			}

		case RiskLimitTypeMaxDailyLoss:
			// This would require daily P&L tracking - simplified for now
			// In a real implementation, you'd track daily P&L and check against this limit

		case RiskLimitTypeMaxDrawdown:
			// This would require drawdown calculation - simplified for now
			// In a real implementation, you'd calculate current drawdown and check against this limit
		}

		if violation {
			result.Approved = false
			result.Violations = append(result.Violations, violationMsg)
			result.RiskScore += 1.0 // Simple risk scoring

			lm.logger.Warn("Risk limit violation",
				zap.String("userID", userID),
				zap.String("symbol", symbol),
				zap.String("limitType", string(limit.Type)),
				zap.Float64("limitValue", limit.Limit),
				zap.String("violation", violationMsg),
			)
		}
	}

	return result, nil
}

// GetUserLimits retrieves all risk limits for a user
func (lm *LimitManager) GetUserLimits(userID string) []*RiskLimit {
	// Try cache first
	cacheKey := userID + ":limits"
	if cached, found := lm.RiskLimitCache.Get(cacheKey); found {
		if limits, ok := cached.([]*RiskLimit); ok {
			return limits
		}
	}

	lm.mu.RLock()
	defer lm.mu.RUnlock()

	limits, exists := lm.RiskLimits[userID]
	if !exists {
		return []*RiskLimit{}
	}

	// Update cache
	lm.RiskLimitCache.Set(cacheKey, limits, cache.DefaultExpiration)

	return limits
}

// UpdateRiskLimit updates an existing risk limit
func (lm *LimitManager) UpdateRiskLimit(ctx context.Context, limitID string, newValue float64) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Find and update the limit
	for userID, userLimits := range lm.RiskLimits {
		for _, limit := range userLimits {
			if limit.ID == limitID {
				limit.Limit = newValue
				limit.UpdatedAt = time.Now()

				// Update cache
				cacheKey := userID + ":limits"
				lm.RiskLimitCache.Set(cacheKey, userLimits, cache.DefaultExpiration)

				lm.logger.Info("Risk limit updated",
					zap.String("limitID", limitID),
					zap.String("userID", userID),
					zap.Float64("newValue", newValue),
				)

				return nil
			}
		}
	}

	return ErrRiskLimitNotFound
}

// EnableRiskLimit enables or disables a risk limit
func (lm *LimitManager) EnableRiskLimit(ctx context.Context, limitID string, enabled bool) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Find and update the limit
	for userID, userLimits := range lm.RiskLimits {
		for _, limit := range userLimits {
			if limit.ID == limitID {
				limit.Enabled = enabled
				limit.UpdatedAt = time.Now()

				// Update cache
				cacheKey := userID + ":limits"
				lm.RiskLimitCache.Set(cacheKey, userLimits, cache.DefaultExpiration)

				lm.logger.Info("Risk limit enabled/disabled",
					zap.String("limitID", limitID),
					zap.String("userID", userID),
					zap.Bool("enabled", enabled),
				)

				return nil
			}
		}
	}

	return ErrRiskLimitNotFound
}

// abs returns the absolute value of x
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
