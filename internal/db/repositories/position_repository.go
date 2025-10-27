package repositories

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PositionRepository handles position data operations
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
