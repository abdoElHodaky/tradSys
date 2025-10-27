package repositories

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RiskRepository handles risk data operations
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
