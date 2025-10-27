package repositories

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OrderRepository handles order data operations
type OrderRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *gorm.DB, logger *zap.Logger) *OrderRepository {
	return &OrderRepository{
		db:     db,
		logger: logger,
	}
}
