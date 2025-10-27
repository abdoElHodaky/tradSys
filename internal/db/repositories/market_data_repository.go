package repositories

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MarketDataRepository handles market data operations
type MarketDataRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewMarketDataRepository creates a new market data repository
func NewMarketDataRepository(db *gorm.DB, logger *zap.Logger) *MarketDataRepository {
	return &MarketDataRepository{
		db:     db,
		logger: logger,
	}
}
