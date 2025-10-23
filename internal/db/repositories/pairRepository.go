package repositories

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PairRepository handles database operations for trading pairs
type PairRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPairRepository creates a new pair repository
func NewPairRepository(db *gorm.DB, logger *zap.Logger) *PairRepository {
	return &PairRepository{
		db:     db,
		logger: logger,
	}
}

// GetPair retrieves a pair by ID
func (r *PairRepository) GetPair(ctx context.Context, pairID string) (*models.Pair, error) {
	var pair models.Pair
	result := r.db.WithContext(ctx).Where("pair_id = ?", pairID).First(&pair)
	if result.Error != nil {
		r.logger.Error("Failed to get pair",
			zap.Error(result.Error),
			zap.String("pair_id", pairID))
		return nil, result.Error
	}
	return &pair, nil
}

// GetAllPairs retrieves all pairs
func (r *PairRepository) GetAllPairs(ctx context.Context) ([]*models.Pair, error) {
	var pairs []*models.Pair
	result := r.db.WithContext(ctx).Find(&pairs)
	if result.Error != nil {
		r.logger.Error("Failed to get all pairs", zap.Error(result.Error))
		return nil, result.Error
	}
	return pairs, nil
}

// GetActivePairs retrieves all active pairs
func (r *PairRepository) GetActivePairs(ctx context.Context) ([]*models.Pair, error) {
	var pairs []*models.Pair
	result := r.db.WithContext(ctx).Where("status = ?", models.PairStatusActive).Find(&pairs)
	if result.Error != nil {
		r.logger.Error("Failed to get active pairs", zap.Error(result.Error))
		return nil, result.Error
	}
	return pairs, nil
}

// CreatePair creates a new pair
func (r *PairRepository) CreatePair(ctx context.Context, pair *models.Pair) error {
	result := r.db.WithContext(ctx).Create(pair)
	if result.Error != nil {
		r.logger.Error("Failed to create pair",
			zap.Error(result.Error),
			zap.String("pair_id", pair.PairID))
		return result.Error
	}
	return nil
}

// UpdatePair updates an existing pair
func (r *PairRepository) UpdatePair(ctx context.Context, pair *models.Pair) error {
	result := r.db.WithContext(ctx).Save(pair)
	if result.Error != nil {
		r.logger.Error("Failed to update pair",
			zap.Error(result.Error),
			zap.String("pair_id", pair.PairID))
		return result.Error
	}
	return nil
}

// DeletePair deletes a pair
func (r *PairRepository) DeletePair(ctx context.Context, pairID string) error {
	result := r.db.WithContext(ctx).Where("pair_id = ?", pairID).Delete(&models.Pair{})
	if result.Error != nil {
		r.logger.Error("Failed to delete pair",
			zap.Error(result.Error),
			zap.String("pair_id", pairID))
		return result.Error
	}
	return nil
}

// PairStatisticsRepository handles database operations for pair statistics
type PairStatisticsRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPairStatisticsRepository creates a new pair statistics repository
func NewPairStatisticsRepository(db *gorm.DB, logger *zap.Logger) *PairStatisticsRepository {
	return &PairStatisticsRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new pair statistics record
func (r *PairStatisticsRepository) Create(ctx context.Context, stats *models.PairStatistics) error {
	result := r.db.WithContext(ctx).Create(stats)
	if result.Error != nil {
		r.logger.Error("Failed to create pair statistics",
			zap.Error(result.Error),
			zap.String("pair_id", stats.PairID))
		return result.Error
	}
	return nil
}

// GetLatestStatistics retrieves the latest statistics for a pair
func (r *PairStatisticsRepository) GetLatestStatistics(ctx context.Context, pairID string) (*models.PairStatistics, error) {
	var stats models.PairStatistics
	result := r.db.WithContext(ctx).
		Where("pair_id = ?", pairID).
		Order("timestamp DESC").
		First(&stats)

	if result.Error != nil {
		r.logger.Error("Failed to get latest pair statistics",
			zap.Error(result.Error),
			zap.String("pair_id", pairID))
		return nil, result.Error
	}
	return &stats, nil
}

// GetStatisticsHistory retrieves historical statistics for a pair
func (r *PairStatisticsRepository) GetStatisticsHistory(ctx context.Context, pairID string, startTime, endTime time.Time) ([]*models.PairStatistics, error) {
	var stats []*models.PairStatistics
	result := r.db.WithContext(ctx).
		Where("pair_id = ? AND timestamp BETWEEN ? AND ?", pairID, startTime, endTime).
		Order("timestamp ASC").
		Find(&stats)

	if result.Error != nil {
		r.logger.Error("Failed to get pair statistics history",
			zap.Error(result.Error),
			zap.String("pair_id", pairID))
		return nil, result.Error
	}
	return stats, nil
}

// PairPositionRepository handles database operations for pair positions
type PairPositionRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPairPositionRepository creates a new pair position repository
func NewPairPositionRepository(db *gorm.DB, logger *zap.Logger) *PairPositionRepository {
	return &PairPositionRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new pair position
func (r *PairPositionRepository) Create(ctx context.Context, position *models.PairPosition) error {
	result := r.db.WithContext(ctx).Create(position)
	if result.Error != nil {
		r.logger.Error("Failed to create pair position",
			zap.Error(result.Error),
			zap.String("pair_id", position.PairID))
		return result.Error
	}
	return nil
}

// Update updates an existing pair position
func (r *PairPositionRepository) Update(ctx context.Context, position *models.PairPosition) error {
	result := r.db.WithContext(ctx).Save(position)
	if result.Error != nil {
		r.logger.Error("Failed to update pair position",
			zap.Error(result.Error),
			zap.String("pair_id", position.PairID),
			zap.Uint("id", position.ID))
		return result.Error
	}
	return nil
}

// GetOpenPositions retrieves all open positions for a pair
func (r *PairPositionRepository) GetOpenPositions(ctx context.Context, pairID string) ([]*models.PairPosition, error) {
	var positions []*models.PairPosition
	result := r.db.WithContext(ctx).
		Where("pair_id = ? AND status = ?", pairID, "open").
		Find(&positions)

	if result.Error != nil {
		r.logger.Error("Failed to get open pair positions",
			zap.Error(result.Error),
			zap.String("pair_id", pairID))
		return nil, result.Error
	}
	return positions, nil
}

// GetAllOpenPositions retrieves all open positions across all pairs
func (r *PairPositionRepository) GetAllOpenPositions(ctx context.Context) ([]*models.PairPosition, error) {
	var positions []*models.PairPosition
	result := r.db.WithContext(ctx).
		Where("status = ?", "open").
		Find(&positions)

	if result.Error != nil {
		r.logger.Error("Failed to get all open pair positions", zap.Error(result.Error))
		return nil, result.Error
	}
	return positions, nil
}

// GetPositionHistory retrieves historical positions for a pair
func (r *PairPositionRepository) GetPositionHistory(ctx context.Context, pairID string) ([]*models.PairPosition, error) {
	var positions []*models.PairPosition
	result := r.db.WithContext(ctx).
		Where("pair_id = ?", pairID).
		Order("entry_timestamp DESC").
		Find(&positions)

	if result.Error != nil {
		r.logger.Error("Failed to get pair position history",
			zap.Error(result.Error),
			zap.String("pair_id", pairID))
		return nil, result.Error
	}
	return positions, nil
}
