package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RiskRepository represents a repository for risk management
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

// GetPositionByUserAndSymbol gets a position by user ID and symbol
func (r *RiskRepository) GetPositionByUserAndSymbol(ctx context.Context, userID, symbol string) (*db.Position, error) {
	var position db.Position
	result := r.db.WithContext(ctx).First(&position, "user_id = ? AND symbol = ?", userID, symbol)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get position by user ID and symbol", 
			zap.Error(result.Error), 
			zap.String("user_id", userID),
			zap.String("symbol", symbol))
		return nil, result.Error
	}
	return &position, nil
}

// GetPositionsByUserID gets all positions for a user
func (r *RiskRepository) GetPositionsByUserID(ctx context.Context, userID string) ([]*db.Position, error) {
	var positions []*db.Position
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&positions)
	if result.Error != nil {
		r.logger.Error("Failed to get positions by user ID", 
			zap.Error(result.Error), 
			zap.String("user_id", userID))
		return nil, result.Error
	}
	return positions, nil
}

// UpsertPosition creates or updates a position
func (r *RiskRepository) UpsertPosition(ctx context.Context, position *db.Position) error {
	// Check if position exists
	var existingPosition db.Position
	result := r.db.WithContext(ctx).First(&existingPosition, "user_id = ? AND symbol = ?", position.UserID, position.Symbol)
	
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new position
			position.LastUpdated = time.Now()
			if err := r.db.WithContext(ctx).Create(position).Error; err != nil {
				r.logger.Error("Failed to create position", 
					zap.Error(err), 
					zap.String("user_id", position.UserID),
					zap.String("symbol", position.Symbol))
				return err
			}
			return nil
		}
		r.logger.Error("Failed to check if position exists", 
			zap.Error(result.Error), 
			zap.String("user_id", position.UserID),
			zap.String("symbol", position.Symbol))
		return result.Error
	}
	
	// Update existing position
	position.ID = existingPosition.ID
	position.LastUpdated = time.Now()
	if err := r.db.WithContext(ctx).Save(position).Error; err != nil {
		r.logger.Error("Failed to update position", 
			zap.Error(err), 
			zap.String("user_id", position.UserID),
			zap.String("symbol", position.Symbol))
		return err
	}
	
	return nil
}

// GetRiskLimitsByUserID gets all risk limits for a user
func (r *RiskRepository) GetRiskLimitsByUserID(ctx context.Context, userID string) ([]*db.RiskLimit, error) {
	var limits []*db.RiskLimit
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&limits)
	if result.Error != nil {
		r.logger.Error("Failed to get risk limits by user ID", 
			zap.Error(result.Error), 
			zap.String("user_id", userID))
		return nil, result.Error
	}
	return limits, nil
}

// GetRiskLimitByUserAndType gets a risk limit by user ID and type
func (r *RiskRepository) GetRiskLimitByUserAndType(ctx context.Context, userID, limitType, symbol string) (*db.RiskLimit, error) {
	var limit db.RiskLimit
	query := r.db.WithContext(ctx).Where("user_id = ? AND type = ?", userID, limitType)
	
	// Add symbol filter if provided
	if symbol != "" {
		query = query.Where("symbol = ? OR symbol = ''", symbol)
	}
	
	result := query.First(&limit)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get risk limit by user ID and type", 
			zap.Error(result.Error), 
			zap.String("user_id", userID),
			zap.String("type", limitType),
			zap.String("symbol", symbol))
		return nil, result.Error
	}
	return &limit, nil
}

// CreateRiskLimit creates a new risk limit
func (r *RiskRepository) CreateRiskLimit(ctx context.Context, limit *db.RiskLimit) error {
	// Generate ID if not provided
	if limit.ID == "" {
		limit.ID = uuid.New().String()
	}
	
	result := r.db.WithContext(ctx).Create(limit)
	if result.Error != nil {
		r.logger.Error("Failed to create risk limit", 
			zap.Error(result.Error), 
			zap.String("user_id", limit.UserID),
			zap.String("type", limit.Type))
		return result.Error
	}
	return nil
}

// UpdateRiskLimit updates a risk limit
func (r *RiskRepository) UpdateRiskLimit(ctx context.Context, limit *db.RiskLimit) error {
	result := r.db.WithContext(ctx).Save(limit)
	if result.Error != nil {
		r.logger.Error("Failed to update risk limit", 
			zap.Error(result.Error), 
			zap.String("limit_id", limit.ID))
		return result.Error
	}
	return nil
}

// DeleteRiskLimit deletes a risk limit
func (r *RiskRepository) DeleteRiskLimit(ctx context.Context, limitID string) error {
	result := r.db.WithContext(ctx).Delete(&db.RiskLimit{}, "id = ?", limitID)
	if result.Error != nil {
		r.logger.Error("Failed to delete risk limit", 
			zap.Error(result.Error), 
			zap.String("limit_id", limitID))
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

// UpsertCircuitBreaker creates or updates a circuit breaker
func (r *RiskRepository) UpsertCircuitBreaker(ctx context.Context, circuitBreaker *db.CircuitBreaker) error {
	// Check if circuit breaker exists
	var existingCircuitBreaker db.CircuitBreaker
	result := r.db.WithContext(ctx).First(&existingCircuitBreaker, "symbol = ?", circuitBreaker.Symbol)
	
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new circuit breaker
			if err := r.db.WithContext(ctx).Create(circuitBreaker).Error; err != nil {
				r.logger.Error("Failed to create circuit breaker", 
					zap.Error(err), 
					zap.String("symbol", circuitBreaker.Symbol))
				return err
			}
			return nil
		}
		r.logger.Error("Failed to check if circuit breaker exists", 
			zap.Error(result.Error), 
			zap.String("symbol", circuitBreaker.Symbol))
		return result.Error
	}
	
	// Update existing circuit breaker
	circuitBreaker.ID = existingCircuitBreaker.ID
	if err := r.db.WithContext(ctx).Save(circuitBreaker).Error; err != nil {
		r.logger.Error("Failed to update circuit breaker", 
			zap.Error(err), 
			zap.String("symbol", circuitBreaker.Symbol))
		return err
	}
	
	return nil
}

