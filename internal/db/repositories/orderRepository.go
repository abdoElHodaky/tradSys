package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OrderRepository represents a repository for orders
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

// Create creates a new order
func (r *OrderRepository) Create(ctx context.Context, order *db.Order) error {
	result := r.db.WithContext(ctx).Create(order)
	if result.Error != nil {
		r.logger.Error("Failed to create order", zap.Error(result.Error), zap.String("order_id", order.ID))
		return result.Error
	}
	return nil
}

// GetByID gets an order by ID
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*db.Order, error) {
	var order db.Order
	result := r.db.WithContext(ctx).First(&order, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get order by ID", zap.Error(result.Error), zap.String("order_id", id))
		return nil, result.Error
	}
	return &order, nil
}

// GetByClientOrderID gets an order by client order ID
func (r *OrderRepository) GetByClientOrderID(ctx context.Context, userID, clientOrderID string) (*db.Order, error) {
	var order db.Order
	result := r.db.WithContext(ctx).First(&order, "user_id = ? AND client_order_id = ?", userID, clientOrderID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get order by client order ID",
			zap.Error(result.Error),
			zap.String("user_id", userID),
			zap.String("client_order_id", clientOrderID))
		return nil, result.Error
	}
	return &order, nil
}

// Update updates an order
func (r *OrderRepository) Update(ctx context.Context, order *db.Order) error {
	result := r.db.WithContext(ctx).Save(order)
	if result.Error != nil {
		r.logger.Error("Failed to update order", zap.Error(result.Error), zap.String("order_id", order.ID))
		return result.Error
	}
	return nil
}

// UpdateStatus updates an order's status
func (r *OrderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	result := r.db.WithContext(ctx).Model(&db.Order{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		r.logger.Error("Failed to update order status",
			zap.Error(result.Error),
			zap.String("order_id", id),
			zap.String("status", status))
		return result.Error
	}
	return nil
}

// GetOrdersByUserID gets orders by user ID
func (r *OrderRepository) GetOrdersByUserID(ctx context.Context, userID string, limit, offset int) ([]*db.Order, error) {
	var orders []*db.Order
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders)
	if result.Error != nil {
		r.logger.Error("Failed to get orders by user ID",
			zap.Error(result.Error),
			zap.String("user_id", userID))
		return nil, result.Error
	}
	return orders, nil
}

// GetOrdersBySymbol gets orders by symbol
func (r *OrderRepository) GetOrdersBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*db.Order, error) {
	var orders []*db.Order
	result := r.db.WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders)
	if result.Error != nil {
		r.logger.Error("Failed to get orders by symbol",
			zap.Error(result.Error),
			zap.String("symbol", symbol))
		return nil, result.Error
	}
	return orders, nil
}

// GetActiveOrdersByUserID gets active orders by user ID
func (r *OrderRepository) GetActiveOrdersByUserID(ctx context.Context, userID string) ([]*db.Order, error) {
	var orders []*db.Order
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []string{"new", "partially_filled"}).
		Order("created_at DESC").
		Find(&orders)
	if result.Error != nil {
		r.logger.Error("Failed to get active orders by user ID",
			zap.Error(result.Error),
			zap.String("user_id", userID))
		return nil, result.Error
	}
	return orders, nil
}

// GetExpiredOrders gets expired orders
func (r *OrderRepository) GetExpiredOrders(ctx context.Context, now time.Time) ([]*db.Order, error) {
	var orders []*db.Order
	result := r.db.WithContext(ctx).
		Where("status IN ? AND expires_at <= ? AND expires_at IS NOT NULL",
			[]string{"new", "partially_filled"}, now).
		Find(&orders)
	if result.Error != nil {
		r.logger.Error("Failed to get expired orders", zap.Error(result.Error))
		return nil, result.Error
	}
	return orders, nil
}

// BatchUpdate updates multiple orders
func (r *OrderRepository) BatchUpdate(ctx context.Context, orders []*db.Order) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, order := range orders {
			if err := tx.Save(order).Error; err != nil {
				r.logger.Error("Failed to update order in batch",
					zap.Error(err),
					zap.String("order_id", order.ID))
				return err
			}
		}
		return nil
	})
}

// CountOrdersByUserID counts orders by user ID
func (r *OrderRepository) CountOrdersByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&db.Order{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		r.logger.Error("Failed to count orders by user ID",
			zap.Error(result.Error),
			zap.String("user_id", userID))
		return 0, result.Error
	}
	return count, nil
}

// GetOrdersWithTrades gets orders with their trades
func (r *OrderRepository) GetOrdersWithTrades(ctx context.Context, userID string, limit, offset int) ([]*db.Order, error) {
	var orders []*db.Order
	result := r.db.WithContext(ctx).
		Preload("Trades").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders)
	if result.Error != nil {
		r.logger.Error("Failed to get orders with trades",
			zap.Error(result.Error),
			zap.String("user_id", userID))
		return nil, result.Error
	}
	return orders, nil
}
