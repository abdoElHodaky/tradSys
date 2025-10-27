package repositories

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TradeRepository handles trade data operations
type TradeRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewTradeRepository creates a new trade repository
func NewTradeRepository(db *gorm.DB, logger *zap.Logger) *TradeRepository {
	return &TradeRepository{
		db:     db,
		logger: logger,
	}
}
