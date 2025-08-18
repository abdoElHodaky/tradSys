package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/query"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OrderRepository handles database operations for orders
type OrderRepository struct {
	db        *gorm.DB
	logger    *zap.Logger
	optimizer *query.Optimizer
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *gorm.DB, logger *zap.Logger) *OrderRepository {
	repo := &OrderRepository{
		db:        db,
		logger:    logger,
		optimizer: query.NewOptimizer(db, logger),
	}
	
	return repo
}

// Create inserts a new order into the database
func (r *OrderRepository) Create(ctx context.Context, order *models.Order) error {
	result := r.db.WithContext(ctx).Create(order)
	if result.Error != nil {
		r.logger.Error("Failed to create order", 
			zap.Error(result.Error),
			zap.String("order_id", order.OrderID))
		return result.Error
	}
	return nil
}

// FindByID retrieves an order by its ID using the query builder
func (r *OrderRepository) FindByID(ctx context.Context, orderID string) (*models.Order, error) {
	var order models.Order
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("orders").
		Where("order_id = ?", orderID)
	
	err := builder.First(&order)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to find order",
			zap.Error(err),
			zap.String("order_id", orderID))
		return nil, err
	}
	
	return &order, nil
}

// Update updates an existing order
func (r *OrderRepository) Update(ctx context.Context, order *models.Order) error {
	result := r.db.WithContext(ctx).Save(order)
	if result.Error != nil {
		r.logger.Error("Failed to update order", 
			zap.Error(result.Error),
			zap.String("order_id", order.OrderID))
		return result.Error
	}
	return nil
}

// FindActiveOrdersBySymbol retrieves all active orders for a symbol
func (r *OrderRepository) FindActiveOrdersBySymbol(ctx context.Context, symbol string) ([]*models.Order, error) {
	var orders []*models.Order
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("orders").
		UseIndex("idx_orders_symbol_status").
		Where("symbol = ?", symbol).
		Where("status NOT IN (?, ?, ?)", 
			string(models.OrderStatusFilled), 
			string(models.OrderStatusCancelled), 
			string(models.OrderStatusRejected)).
		OrderBy("created_at DESC")
	
	err := builder.Execute(&orders)
	if err != nil {
		r.logger.Error("Failed to find active orders",
			zap.Error(err),
			zap.String("symbol", symbol))
		return nil, err
	}
	
	return orders, nil
}

// FindOrdersByTimeRange finds orders within a time range
func (r *OrderRepository) FindOrdersByTimeRange(ctx context.Context, symbol string, start, end time.Time) ([]*models.Order, error) {
	var orders []*models.Order
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("orders").
		Where("symbol = ?", symbol).
		Where("created_at BETWEEN ? AND ?", start, end).
		OrderBy("created_at ASC")
	
	// Analyze the query plan before execution
	query, args := builder.Build()
	plan, err := r.optimizer.AnalyzeQuery(query, args...)
	if err == nil {
		r.logger.Debug("Query execution plan", zap.String("plan", plan))
	}
	
	err = builder.Execute(&orders)
	if err != nil {
		r.logger.Error("Failed to find orders by time range",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.Time("start", start),
			zap.Time("end", end))
		return nil, err
	}
	
	return orders, nil
}

// CreateTrade inserts a new trade into the database
func (r *OrderRepository) CreateTrade(ctx context.Context, trade *models.Trade) error {
	result := r.db.WithContext(ctx).Create(trade)
	if result.Error != nil {
		r.logger.Error("Failed to create trade", 
			zap.Error(result.Error),
			zap.String("trade_id", trade.TradeID))
		return result.Error
	}
	return nil
}

// FindTradesByOrderID retrieves all trades for an order
func (r *OrderRepository) FindTradesByOrderID(ctx context.Context, orderID string) ([]*models.Trade, error) {
	var trades []*models.Trade
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("trades").
		UseIndex("idx_trades_order_id").
		Where("order_id = ?", orderID).
		OrderBy("timestamp ASC")
	
	err := builder.Execute(&trades)
	if err != nil {
		r.logger.Error("Failed to find trades",
			zap.Error(err),
			zap.String("order_id", orderID))
		return nil, err
	}
	
	return trades, nil
}

// GetOrderStatistics gets statistics about orders
func (r *OrderRepository) GetOrderStatistics(ctx context.Context, symbol string) (map[string]interface{}, error) {
	var result map[string]interface{}
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("orders").
		Select(
			"COUNT(*) as total_orders",
			"SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as filled_orders",
			"SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as cancelled_orders",
			"AVG(CASE WHEN status = ? THEN price ELSE 0 END) as avg_fill_price",
		).
		Where("symbol = ?", symbol).
		Where("created_at > ?", time.Now().Add(-24*time.Hour))
	
	// Add parameters for the CASE statements
	// Create a new builder with the CASE statement parameters
	newBuilder := query.NewBuilder(r.db, r.logger).
		Table(builder.GetTable()).
		Select(builder.GetFields()...)
	
	// Add the CASE statement parameters first
	newBuilder.Where("symbol = ?", symbol)
	newBuilder.Where("created_at > ?", time.Now().Add(-24*time.Hour))
	
	// Replace the builder with the new one
	builder = newBuilder
	
	err := builder.First(&result)
	if err != nil {
		r.logger.Error("Failed to get order statistics",
			zap.Error(err),
			zap.String("symbol", symbol))
		return nil, err
	}
	
	return result, nil
}

// GetPosition gets the current position for a symbol
func (r *OrderRepository) GetPosition(ctx context.Context, symbol, accountID string) (*models.Position, error) {
	var position models.Position
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("positions").
		Where("symbol = ?", symbol).
		Where("account_id = ?", accountID)
	
	err := builder.First(&position)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return an empty position if not found
			return &models.Position{
				Symbol:       symbol,
				AccountID:    accountID,
				Quantity:     0,
				AveragePrice: 0,
				LastUpdated:  time.Now(),
			}, nil
		}
		
		r.logger.Error("Failed to get position",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("account_id", accountID))
		return nil, err
	}
	
	return &position, nil
}

// UpdatePosition updates a position
func (r *OrderRepository) UpdatePosition(ctx context.Context, position *models.Position) error {
	// Use a transaction to ensure atomicity
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// Try to update existing position
	result := tx.Model(&models.Position{}).
		Where("symbol = ? AND account_id = ?", position.Symbol, position.AccountID).
		Updates(map[string]interface{}{
			"quantity":       position.Quantity,
			"average_price":  position.AveragePrice,
			"unrealized_pnl": position.UnrealizedPnL,
			"realized_pnl":   position.RealizedPnL,
			"last_updated":   position.LastUpdated,
			"updated_at":     time.Now(),
		})
	
	// If no record was updated, create a new one
	if result.RowsAffected == 0 {
		if err := tx.Create(position).Error; err != nil {
			tx.Rollback()
			r.logger.Error("Failed to create position", 
				zap.Error(err),
				zap.String("symbol", position.Symbol))
			return err
		}
	} else if result.Error != nil {
		tx.Rollback()
		r.logger.Error("Failed to update position", 
			zap.Error(result.Error),
			zap.String("symbol", position.Symbol))
		return result.Error
	}
	
	return tx.Commit().Error
}
