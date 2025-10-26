package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/errors"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// OrderRepository implements the OrderRepository interface using SQL database
type OrderRepository struct {
	db      *sql.DB
	logger  interfaces.Logger
	metrics interfaces.MetricsCollector
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *sql.DB, logger interfaces.Logger, metrics interfaces.MetricsCollector) *OrderRepository {
	return &OrderRepository{
		db:      db,
		logger:  logger,
		metrics: metrics,
	}
}

// Create creates a new order in the database
func (r *OrderRepository) Create(ctx context.Context, order *types.Order) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.create.duration", time.Since(start), map[string]string{
			"symbol": order.Symbol,
		})
	}()

	query := `
		INSERT INTO orders (
			id, client_order_id, user_id, symbol, side, type, price, quantity,
			filled_quantity, remaining_quantity, status, time_in_force, stop_price,
			created_at, updated_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err := r.db.ExecContext(ctx, query,
		order.ID,
		order.ClientOrderID,
		order.UserID,
		order.Symbol,
		string(order.Side),
		string(order.Type),
		order.Price,
		order.Quantity,
		order.FilledQuantity,
		order.RemainingQuantity,
		string(order.Status),
		string(order.TimeInForce),
		order.StopPrice,
		order.CreatedAt,
		order.UpdatedAt,
		order.ExpiresAt,
	)

	if err != nil {
		r.metrics.IncrementCounter("order_repository.create.errors", map[string]string{
			"symbol": order.Symbol,
		})
		r.logger.Error("Failed to create order", "error", err, "order_id", order.ID)
		return errors.Wrap(err, errors.ErrDatabaseConnection, "failed to insert order")
	}

	r.metrics.IncrementCounter("order_repository.create.success", map[string]string{
		"symbol": order.Symbol,
	})

	r.logger.Debug("Order created successfully", "order_id", order.ID, "symbol", order.Symbol)
	return nil
}

// GetByID retrieves an order by its ID
func (r *OrderRepository) GetByID(ctx context.Context, orderID string) (*types.Order, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.get_by_id.duration", time.Since(start), nil)
	}()

	query := `
		SELECT id, client_order_id, user_id, symbol, side, type, price, quantity,
			   filled_quantity, remaining_quantity, status, time_in_force, stop_price,
			   created_at, updated_at, expires_at
		FROM orders
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, orderID)

	order := &types.Order{}
	var side, orderType, status, timeInForce string

	err := row.Scan(
		&order.ID,
		&order.ClientOrderID,
		&order.UserID,
		&order.Symbol,
		&side,
		&orderType,
		&order.Price,
		&order.Quantity,
		&order.FilledQuantity,
		&order.RemainingQuantity,
		&status,
		&timeInForce,
		&order.StopPrice,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.ExpiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.metrics.IncrementCounter("order_repository.get_by_id.not_found", nil)
			return nil, errors.New(errors.ErrOrderNotFound, "order not found")
		}
		r.metrics.IncrementCounter("order_repository.get_by_id.errors", nil)
		r.logger.Error("Failed to get order by ID", "error", err, "order_id", orderID)
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to query order")
	}

	// Convert string enums back to types
	order.Side = types.OrderSide(side)
	order.Type = types.OrderType(orderType)
	order.Status = types.OrderStatus(status)
	order.TimeInForce = types.TimeInForce(timeInForce)

	r.metrics.IncrementCounter("order_repository.get_by_id.success", map[string]string{
		"symbol": order.Symbol,
	})

	return order, nil
}

// GetByClientOrderID retrieves an order by client order ID and user ID
func (r *OrderRepository) GetByClientOrderID(ctx context.Context, userID, clientOrderID string) (*types.Order, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.get_by_client_id.duration", time.Since(start), nil)
	}()

	query := `
		SELECT id, client_order_id, user_id, symbol, side, type, price, quantity,
			   filled_quantity, remaining_quantity, status, time_in_force, stop_price,
			   created_at, updated_at, expires_at
		FROM orders
		WHERE user_id = $1 AND client_order_id = $2`

	row := r.db.QueryRowContext(ctx, query, userID, clientOrderID)

	order := &types.Order{}
	var side, orderType, status, timeInForce string

	err := row.Scan(
		&order.ID,
		&order.ClientOrderID,
		&order.UserID,
		&order.Symbol,
		&side,
		&orderType,
		&order.Price,
		&order.Quantity,
		&order.FilledQuantity,
		&order.RemainingQuantity,
		&status,
		&timeInForce,
		&order.StopPrice,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.ExpiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.metrics.IncrementCounter("order_repository.get_by_client_id.not_found", nil)
			return nil, errors.New(errors.ErrOrderNotFound, "order not found")
		}
		r.metrics.IncrementCounter("order_repository.get_by_client_id.errors", nil)
		r.logger.Error("Failed to get order by client ID", "error", err, "user_id", userID, "client_order_id", clientOrderID)
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to query order")
	}

	// Convert string enums back to types
	order.Side = types.OrderSide(side)
	order.Type = types.OrderType(orderType)
	order.Status = types.OrderStatus(status)
	order.TimeInForce = types.TimeInForce(timeInForce)

	r.metrics.IncrementCounter("order_repository.get_by_client_id.success", map[string]string{
		"symbol": order.Symbol,
	})

	return order, nil
}

// Update updates an existing order
func (r *OrderRepository) Update(ctx context.Context, order *types.Order) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.update.duration", time.Since(start), map[string]string{
			"symbol": order.Symbol,
		})
	}()

	query := `
		UPDATE orders SET
			filled_quantity = $2,
			remaining_quantity = $3,
			status = $4,
			updated_at = $5
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		order.ID,
		order.FilledQuantity,
		order.RemainingQuantity,
		string(order.Status),
		order.UpdatedAt,
	)

	if err != nil {
		r.metrics.IncrementCounter("order_repository.update.errors", map[string]string{
			"symbol": order.Symbol,
		})
		r.logger.Error("Failed to update order", "error", err, "order_id", order.ID)
		return errors.Wrap(err, errors.ErrDatabaseConnection, "failed to update order")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Warn("Could not get rows affected", "error", err, "order_id", order.ID)
	} else if rowsAffected == 0 {
		r.metrics.IncrementCounter("order_repository.update.not_found", map[string]string{
			"symbol": order.Symbol,
		})
		return errors.New(errors.ErrOrderNotFound, "order not found for update")
	}

	r.metrics.IncrementCounter("order_repository.update.success", map[string]string{
		"symbol": order.Symbol,
	})

	r.logger.Debug("Order updated successfully", "order_id", order.ID, "status", order.Status)
	return nil
}

// Delete deletes an order
func (r *OrderRepository) Delete(ctx context.Context, orderID string) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.delete.duration", time.Since(start), nil)
	}()

	query := `DELETE FROM orders WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, orderID)
	if err != nil {
		r.metrics.IncrementCounter("order_repository.delete.errors", nil)
		r.logger.Error("Failed to delete order", "error", err, "order_id", orderID)
		return errors.Wrap(err, errors.ErrDatabaseConnection, "failed to delete order")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Warn("Could not get rows affected", "error", err, "order_id", orderID)
	} else if rowsAffected == 0 {
		r.metrics.IncrementCounter("order_repository.delete.not_found", nil)
		return errors.New(errors.ErrOrderNotFound, "order not found for deletion")
	}

	r.metrics.IncrementCounter("order_repository.delete.success", nil)
	r.logger.Debug("Order deleted successfully", "order_id", orderID)
	return nil
}

// ListByUser lists orders for a specific user
func (r *OrderRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*types.Order, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.list_by_user.duration", time.Since(start), nil)
	}()

	query := `
		SELECT id, client_order_id, user_id, symbol, side, type, price, quantity,
			   filled_quantity, remaining_quantity, status, time_in_force, stop_price,
			   created_at, updated_at, expires_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		r.metrics.IncrementCounter("order_repository.list_by_user.errors", nil)
		r.logger.Error("Failed to list orders by user", "error", err, "user_id", userID)
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to query orders")
	}
	defer rows.Close()

	orders, err := r.scanOrders(rows)
	if err != nil {
		return nil, err
	}

	r.metrics.IncrementCounter("order_repository.list_by_user.success", nil)
	r.metrics.RecordGauge("order_repository.orders_returned", float64(len(orders)), map[string]string{
		"user_id": userID,
	})

	return orders, nil
}

// ListBySymbol lists orders for a specific symbol
func (r *OrderRepository) ListBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*types.Order, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.list_by_symbol.duration", time.Since(start), map[string]string{
			"symbol": symbol,
		})
	}()

	query := `
		SELECT id, client_order_id, user_id, symbol, side, type, price, quantity,
			   filled_quantity, remaining_quantity, status, time_in_force, stop_price,
			   created_at, updated_at, expires_at
		FROM orders
		WHERE symbol = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, symbol, limit, offset)
	if err != nil {
		r.metrics.IncrementCounter("order_repository.list_by_symbol.errors", map[string]string{
			"symbol": symbol,
		})
		r.logger.Error("Failed to list orders by symbol", "error", err, "symbol", symbol)
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to query orders")
	}
	defer rows.Close()

	orders, err := r.scanOrders(rows)
	if err != nil {
		return nil, err
	}

	r.metrics.IncrementCounter("order_repository.list_by_symbol.success", map[string]string{
		"symbol": symbol,
	})
	r.metrics.RecordGauge("order_repository.orders_returned", float64(len(orders)), map[string]string{
		"symbol": symbol,
	})

	return orders, nil
}

// ListActiveOrders lists all active orders
func (r *OrderRepository) ListActiveOrders(ctx context.Context) ([]*types.Order, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.list_active.duration", time.Since(start), nil)
	}()

	query := `
		SELECT id, client_order_id, user_id, symbol, side, type, price, quantity,
			   filled_quantity, remaining_quantity, status, time_in_force, stop_price,
			   created_at, updated_at, expires_at
		FROM orders
		WHERE status IN ($1, $2)
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, string(types.OrderStatusPending), string(types.OrderStatusPartiallyFilled))
	if err != nil {
		r.metrics.IncrementCounter("order_repository.list_active.errors", nil)
		r.logger.Error("Failed to list active orders", "error", err)
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to query active orders")
	}
	defer rows.Close()

	orders, err := r.scanOrders(rows)
	if err != nil {
		return nil, err
	}

	r.metrics.IncrementCounter("order_repository.list_active.success", nil)
	r.metrics.RecordGauge("order_repository.active_orders", float64(len(orders)), nil)

	return orders, nil
}

// scanOrders scans multiple orders from database rows
func (r *OrderRepository) scanOrders(rows *sql.Rows) ([]*types.Order, error) {
	var orders []*types.Order

	for rows.Next() {
		order := &types.Order{}
		var side, orderType, status, timeInForce string

		err := rows.Scan(
			&order.ID,
			&order.ClientOrderID,
			&order.UserID,
			&order.Symbol,
			&side,
			&orderType,
			&order.Price,
			&order.Quantity,
			&order.FilledQuantity,
			&order.RemainingQuantity,
			&status,
			&timeInForce,
			&order.StopPrice,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ExpiresAt,
		)

		if err != nil {
			r.metrics.IncrementCounter("order_repository.scan.errors", nil)
			r.logger.Error("Failed to scan order row", "error", err)
			return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to scan order")
		}

		// Convert string enums back to types
		order.Side = types.OrderSide(side)
		order.Type = types.OrderType(orderType)
		order.Status = types.OrderStatus(status)
		order.TimeInForce = types.TimeInForce(timeInForce)

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		r.metrics.IncrementCounter("order_repository.scan.errors", nil)
		r.logger.Error("Error iterating over order rows", "error", err)
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "error iterating orders")
	}

	return orders, nil
}

// GetOrderStatistics returns statistics about orders
func (r *OrderRepository) GetOrderStatistics(ctx context.Context, userID string) (*OrderStatistics, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordTimer("order_repository.get_statistics.duration", time.Since(start), nil)
	}()

	query := `
		SELECT 
			COUNT(*) as total_orders,
			COUNT(CASE WHEN status IN ($2, $3) THEN 1 END) as active_orders,
			COUNT(CASE WHEN status = $4 THEN 1 END) as filled_orders,
			COALESCE(SUM(price * quantity), 0) as total_value,
			COALESCE(AVG(price * quantity), 0) as average_order_value
		FROM orders
		WHERE user_id = $1`

	row := r.db.QueryRowContext(ctx, query, userID, 
		string(types.OrderStatusPending), 
		string(types.OrderStatusPartiallyFilled),
		string(types.OrderStatusFilled))

	stats := &OrderStatistics{
		UserID: userID,
	}

	err := row.Scan(
		&stats.TotalOrders,
		&stats.ActiveOrders,
		&stats.FilledOrders,
		&stats.TotalValue,
		&stats.AverageOrderValue,
	)

	if err != nil {
		r.metrics.IncrementCounter("order_repository.get_statistics.errors", nil)
		r.logger.Error("Failed to get order statistics", "error", err, "user_id", userID)
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to get order statistics")
	}

	r.metrics.IncrementCounter("order_repository.get_statistics.success", nil)
	return stats, nil
}

// OrderStatistics contains order statistics
type OrderStatistics struct {
	UserID            string  `json:"user_id"`
	TotalOrders       int     `json:"total_orders"`
	ActiveOrders      int     `json:"active_orders"`
	FilledOrders      int     `json:"filled_orders"`
	TotalValue        float64 `json:"total_value"`
	AverageOrderValue float64 `json:"average_order_value"`
}

// CreateOrdersTable creates the orders table if it doesn't exist
func (r *OrderRepository) CreateOrdersTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS orders (
			id VARCHAR(255) PRIMARY KEY,
			client_order_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			symbol VARCHAR(50) NOT NULL,
			side VARCHAR(10) NOT NULL,
			type VARCHAR(20) NOT NULL,
			price DECIMAL(20,8) NOT NULL DEFAULT 0,
			quantity DECIMAL(20,8) NOT NULL,
			filled_quantity DECIMAL(20,8) NOT NULL DEFAULT 0,
			remaining_quantity DECIMAL(20,8) NOT NULL,
			status VARCHAR(20) NOT NULL,
			time_in_force VARCHAR(10) NOT NULL,
			stop_price DECIMAL(20,8),
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP,
			INDEX idx_user_id (user_id),
			INDEX idx_symbol (symbol),
			INDEX idx_status (status),
			INDEX idx_created_at (created_at),
			UNIQUE KEY unique_client_order (user_id, client_order_id)
		)`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to create orders table", "error", err)
		return errors.Wrap(err, errors.ErrDatabaseConnection, "failed to create orders table")
	}

	r.logger.Info("Orders table created successfully")
	return nil
}
