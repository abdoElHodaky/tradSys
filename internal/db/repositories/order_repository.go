package repositories

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// OrderRepositoryParams contains the parameters for creating an order repository
type OrderRepositoryParams struct {
	fx.In

	Logger *zap.Logger
	DB     *sqlx.DB `optional:"true"`
}

// OrderRepository provides database operations for orders
type OrderRepository struct {
	logger *zap.Logger
	db     *sqlx.DB
}

// Order represents an order in the database
type Order struct {
	ID             int64     `db:"id"`
	OrderID        string    `db:"order_id"`
	Symbol         string    `db:"symbol"`
	Type           int       `db:"type"`
	Side           int       `db:"side"`
	Status         int       `db:"status"`
	Quantity       float64   `db:"quantity"`
	FilledQuantity float64   `db:"filled_quantity"`
	Price          float64   `db:"price"`
	StopPrice      float64   `db:"stop_price"`
	ClientOrderID  string    `db:"client_order_id"`
	AccountID      string    `db:"account_id"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// NewOrderRepository creates a new order repository with fx dependency injection
func NewOrderRepository(p OrderRepositoryParams) *OrderRepository {
	return &OrderRepository{
		logger: p.Logger,
		db:     p.DB,
	}
}

// CreateOrder creates a new order in the database
func (r *OrderRepository) CreateOrder(ctx context.Context, order *Order) error {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil
	}

	query := `
		INSERT INTO orders (
			order_id, symbol, type, side, status, quantity, filled_quantity,
			price, stop_price, client_order_id, account_id, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		RETURNING id
	`

	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	row := r.db.QueryRowContext(ctx, query,
		order.OrderID,
		order.Symbol,
		order.Type,
		order.Side,
		order.Status,
		order.Quantity,
		order.FilledQuantity,
		order.Price,
		order.StopPrice,
		order.ClientOrderID,
		order.AccountID,
		order.CreatedAt,
		order.UpdatedAt,
	)

	err := row.Scan(&order.ID)
	if err != nil {
		r.logger.Error("Failed to create order",
			zap.String("order_id", order.OrderID),
			zap.Error(err))
		return err
	}

	return nil
}

// GetOrder retrieves an order by ID
func (r *OrderRepository) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil, nil
	}

	query := `
		SELECT id, order_id, symbol, type, side, status, quantity, filled_quantity,
			price, stop_price, client_order_id, account_id, created_at, updated_at
		FROM orders
		WHERE order_id = $1
	`

	var order Order
	err := r.db.GetContext(ctx, &order, query, orderID)
	if err != nil {
		r.logger.Error("Failed to get order",
			zap.String("order_id", orderID),
			zap.Error(err))
		return nil, err
	}

	return &order, nil
}

// UpdateOrder updates an existing order
func (r *OrderRepository) UpdateOrder(ctx context.Context, order *Order) error {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil
	}

	query := `
		UPDATE orders
		SET status = $1, filled_quantity = $2, updated_at = $3
		WHERE order_id = $4
	`

	order.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		order.Status,
		order.FilledQuantity,
		order.UpdatedAt,
		order.OrderID,
	)

	if err != nil {
		r.logger.Error("Failed to update order",
			zap.String("order_id", order.OrderID),
			zap.Error(err))
		return err
	}

	return nil
}

// GetOrders retrieves a list of orders
func (r *OrderRepository) GetOrders(ctx context.Context, symbol string, status int, startTime, endTime time.Time, limit int) ([]*Order, error) {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil, nil
	}

	query := `
		SELECT id, order_id, symbol, type, side, status, quantity, filled_quantity,
			price, stop_price, client_order_id, account_id, created_at, updated_at
		FROM orders
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if symbol != "" {
		query += " AND symbol = $" + string(argIndex)
		args = append(args, symbol)
		argIndex++
	}

	if status != int(orders.OrderStatus_PENDING) {
		query += " AND status = $" + string(argIndex)
		args = append(args, status)
		argIndex++
	}

	if !startTime.IsZero() {
		query += " AND created_at >= $" + string(argIndex)
		args = append(args, startTime)
		argIndex++
	}

	if !endTime.IsZero() {
		query += " AND created_at <= $" + string(argIndex)
		args = append(args, endTime)
		argIndex++
	}

	query += " ORDER BY created_at DESC LIMIT $" + string(argIndex)
	args = append(args, limit)

	var orders []*Order
	err := r.db.SelectContext(ctx, &orders, query, args...)
	if err != nil {
		r.logger.Error("Failed to get orders",
			zap.String("symbol", symbol),
			zap.Int("status", status),
			zap.Error(err))
		return nil, err
	}

	return orders, nil
}

// OrderRepositoryModule provides the order repository module for fx
var OrderRepositoryModule = fx.Options(
	fx.Provide(NewOrderRepository),
)

