package repositories

import (
	"context"
	"errors"

	"github.com/abdoElHodaky/tradSys/internal/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RiskRepository represents a repository for risk limits
type RiskRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewRiskRepository creates a new risk repository
func NewRiskRepository(db *gorm.DB, logger *zap.Logger) *RiskRepository {
	return &RiskRepository{
		db:     db,
		logger: logger,
	}
}

// GetRiskLimitByID gets a risk limit by ID
func (r *RiskRepository) GetRiskLimitByID(ctx context.Context, id string) (*db.RiskLimit, error) {
	var riskLimit db.RiskLimit
	result := r.db.WithContext(ctx).First(&riskLimit, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get risk limit by ID",
			zap.Error(result.Error),
			zap.String("risk_limit_id", id))
		return nil, result.Error
	}
	return &riskLimit, nil
}

// GetRiskLimitsByUserID gets risk limits by user ID
func (r *RiskRepository) GetRiskLimitsByUserID(ctx context.Context, userID string) ([]*db.RiskLimit, error) {
	var riskLimits []*db.RiskLimit
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&riskLimits)
	if result.Error != nil {
		r.logger.Error("Failed to get risk limits by user ID",
			zap.Error(result.Error),
			zap.String("user_id", userID))
		return nil, result.Error
	}
	return riskLimits, nil
}

// GetRiskLimitsByType gets risk limits by type
func (r *RiskRepository) GetRiskLimitsByType(ctx context.Context, userID, limitType string) ([]*db.RiskLimit, error) {
	var riskLimits []*db.RiskLimit
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ?", userID, limitType).
		Find(&riskLimits)
	if result.Error != nil {
		r.logger.Error("Failed to get risk limits by type",
			zap.Error(result.Error),
			zap.String("user_id", userID),
			zap.String("type", limitType))
		return nil, result.Error
	}
	return riskLimits, nil
}

// CreateRiskLimit creates a risk limit
func (r *RiskRepository) CreateRiskLimit(ctx context.Context, riskLimit *db.RiskLimit) error {
	result := r.db.WithContext(ctx).Create(riskLimit)
	if result.Error != nil {
		r.logger.Error("Failed to create risk limit",
			zap.Error(result.Error),
			zap.String("risk_limit_id", riskLimit.ID))
		return result.Error
	}
	return nil
}

// UpdateRiskLimit updates a risk limit
func (r *RiskRepository) UpdateRiskLimit(ctx context.Context, riskLimit *db.RiskLimit) error {
	result := r.db.WithContext(ctx).Save(riskLimit)
	if result.Error != nil {
		r.logger.Error("Failed to update risk limit",
			zap.Error(result.Error),
			zap.String("risk_limit_id", riskLimit.ID))
		return result.Error
	}
	return nil
}

// DeleteRiskLimit deletes a risk limit
func (r *RiskRepository) DeleteRiskLimit(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&db.RiskLimit{}, "id = ?", id)
	if result.Error != nil {
		r.logger.Error("Failed to delete risk limit",
			zap.Error(result.Error),
			zap.String("risk_limit_id", id))
		return result.Error
	}
	return nil
}

// GetCircuitBreakerBySymbol gets a circuit breaker by symbol
func (r *RiskRepository) GetCircuitBreakerBySymbol(ctx context.Context, symbol string) (*db.CircuitBreaker, error) {
	var circuitBreaker db.CircuitBreaker
	result := r.db.WithContext(ctx).First(&circuitBreaker, "symbol = ?", symbol)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get circuit breaker by symbol",
			zap.Error(result.Error),
			zap.String("symbol", symbol))
		return nil, result.Error
	}
	return &circuitBreaker, nil
}

// GetAllCircuitBreakers gets all circuit breakers
func (r *RiskRepository) GetAllCircuitBreakers(ctx context.Context) ([]*db.CircuitBreaker, error) {
	var circuitBreakers []*db.CircuitBreaker
	result := r.db.WithContext(ctx).Find(&circuitBreakers)
	if result.Error != nil {
		r.logger.Error("Failed to get all circuit breakers", zap.Error(result.Error))
		return nil, result.Error
	}
	return circuitBreakers, nil
}

// CreateOrUpdateCircuitBreaker creates or updates a circuit breaker
func (r *RiskRepository) CreateOrUpdateCircuitBreaker(ctx context.Context, circuitBreaker *db.CircuitBreaker) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingCircuitBreaker db.CircuitBreaker
		result := tx.First(&existingCircuitBreaker, "symbol = ?", circuitBreaker.Symbol)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Create new circuit breaker
				if err := tx.Create(circuitBreaker).Error; err != nil {
					r.logger.Error("Failed to create circuit breaker",
						zap.Error(err),
						zap.String("symbol", circuitBreaker.Symbol))
					return err
				}
				return nil
			}
			r.logger.Error("Failed to check existing circuit breaker",
				zap.Error(result.Error),
				zap.String("symbol", circuitBreaker.Symbol))
			return result.Error
		}

		// Update existing circuit breaker
		circuitBreaker.ID = existingCircuitBreaker.ID
		circuitBreaker.CreatedAt = existingCircuitBreaker.CreatedAt
		if err := tx.Save(circuitBreaker).Error; err != nil {
			r.logger.Error("Failed to update circuit breaker",
				zap.Error(err),
				zap.String("symbol", circuitBreaker.Symbol))
			return err
		}

		return nil
	})
}

// GetTriggeredCircuitBreakers gets triggered circuit breakers
func (r *RiskRepository) GetTriggeredCircuitBreakers(ctx context.Context) ([]*db.CircuitBreaker, error) {
	var circuitBreakers []*db.CircuitBreaker
	result := r.db.WithContext(ctx).
		Where("triggered = ?", true).
		Find(&circuitBreakers)
	if result.Error != nil {
		r.logger.Error("Failed to get triggered circuit breakers", zap.Error(result.Error))
		return nil, result.Error
	}
	return circuitBreakers, nil
}
