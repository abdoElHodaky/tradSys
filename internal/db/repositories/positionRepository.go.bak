package repositories

import (
	"context"
	"errors"

	"github.com/abdoElHodaky/tradSys/internal/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PositionRepository represents a repository for positions
type PositionRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPositionRepository creates a new position repository
func NewPositionRepository(db *gorm.DB, logger *zap.Logger) *PositionRepository {
	return &PositionRepository{
		db:     db,
		logger: logger,
	}
}

// GetByUserIDAndSymbol gets a position by user ID and symbol
func (r *PositionRepository) GetByUserIDAndSymbol(ctx context.Context, userID, symbol string) (*db.Position, error) {
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

// GetPositionsByUserID gets positions by user ID
func (r *PositionRepository) GetPositionsByUserID(ctx context.Context, userID string) ([]*db.Position, error) {
	var positions []*db.Position
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&positions)
	if result.Error != nil {
		r.logger.Error("Failed to get positions by user ID",
			zap.Error(result.Error),
			zap.String("user_id", userID))
		return nil, result.Error
	}
	return positions, nil
}

// CreateOrUpdate creates or updates a position
func (r *PositionRepository) CreateOrUpdate(ctx context.Context, position *db.Position) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingPosition db.Position
		result := tx.First(&existingPosition, "user_id = ? AND symbol = ?", position.UserID, position.Symbol)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Create new position
				if err := tx.Create(position).Error; err != nil {
					r.logger.Error("Failed to create position",
						zap.Error(err),
						zap.String("user_id", position.UserID),
						zap.String("symbol", position.Symbol))
					return err
				}
				return nil
			}
			r.logger.Error("Failed to check existing position",
				zap.Error(result.Error),
				zap.String("user_id", position.UserID),
				zap.String("symbol", position.Symbol))
			return result.Error
		}

		// Update existing position
		position.ID = existingPosition.ID
		position.CreatedAt = existingPosition.CreatedAt
		if err := tx.Save(position).Error; err != nil {
			r.logger.Error("Failed to update position",
				zap.Error(err),
				zap.String("user_id", position.UserID),
				zap.String("symbol", position.Symbol))
			return err
		}

		return nil
	})
}

// BatchUpdate updates multiple positions
func (r *PositionRepository) BatchUpdate(ctx context.Context, positions []*db.Position) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, position := range positions {
			var existingPosition db.Position
			result := tx.First(&existingPosition, "user_id = ? AND symbol = ?", position.UserID, position.Symbol)

			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					// Create new position
					if err := tx.Create(position).Error; err != nil {
						r.logger.Error("Failed to create position in batch",
							zap.Error(err),
							zap.String("user_id", position.UserID),
							zap.String("symbol", position.Symbol))
						return err
					}
					continue
				}
				r.logger.Error("Failed to check existing position in batch",
					zap.Error(result.Error),
					zap.String("user_id", position.UserID),
					zap.String("symbol", position.Symbol))
				return result.Error
			}

			// Update existing position
			position.ID = existingPosition.ID
			position.CreatedAt = existingPosition.CreatedAt
			if err := tx.Save(position).Error; err != nil {
				r.logger.Error("Failed to update position in batch",
					zap.Error(err),
					zap.String("user_id", position.UserID),
					zap.String("symbol", position.Symbol))
				return err
			}
		}

		return nil
	})
}

// GetNonZeroPositions gets non-zero positions
func (r *PositionRepository) GetNonZeroPositions(ctx context.Context) ([]*db.Position, error) {
	var positions []*db.Position
	result := r.db.WithContext(ctx).
		Where("quantity != 0").
		Find(&positions)
	if result.Error != nil {
		r.logger.Error("Failed to get non-zero positions", zap.Error(result.Error))
		return nil, result.Error
	}
	return positions, nil
}

// GetTotalExposure gets the total exposure for a user
func (r *PositionRepository) GetTotalExposure(ctx context.Context, userID string) (float64, error) {
	var totalExposure float64
	result := r.db.WithContext(ctx).Model(&db.Position{}).
		Select("COALESCE(SUM(ABS(quantity) * average_entry_price), 0)").
		Where("user_id = ?", userID).
		Scan(&totalExposure)
	if result.Error != nil {
		r.logger.Error("Failed to get total exposure",
			zap.Error(result.Error),
			zap.String("user_id", userID))
		return 0, result.Error
	}
	return totalExposure, nil
}

// GetTotalPnL gets the total PnL for a user
func (r *PositionRepository) GetTotalPnL(ctx context.Context, userID string) (float64, float64, error) {
	var positions []*db.Position
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&positions)
	if result.Error != nil {
		r.logger.Error("Failed to get positions for total PnL",
			zap.Error(result.Error),
			zap.String("user_id", userID))
		return 0, 0, result.Error
	}

	var totalUnrealizedPnL, totalRealizedPnL float64
	for _, position := range positions {
		totalUnrealizedPnL += position.UnrealizedPnL
		totalRealizedPnL += position.RealizedPnL
	}

	return totalUnrealizedPnL, totalRealizedPnL, nil
}
