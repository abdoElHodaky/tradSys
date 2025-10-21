package queries

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/trading/metrics"
)

// HFTQueries contains all prepared statements for high-frequency operations
type HFTQueries struct {
	// Order queries
	getOrderByIDStmt     *sql.Stmt
	insertOrderStmt      *sql.Stmt
	updateOrderStmt      *sql.Stmt
	updateOrderStatusStmt *sql.Stmt
	getOrdersByUserStmt  *sql.Stmt
	getActiveOrdersStmt  *sql.Stmt
	
	// Market data queries
	getLatestPriceStmt   *sql.Stmt
	insertPriceStmt      *sql.Stmt
	getPriceHistoryStmt  *sql.Stmt
	
	// User queries
	getUserByIDStmt      *sql.Stmt
	updateUserBalanceStmt *sql.Stmt
	
	// Portfolio queries
	getPortfolioStmt     *sql.Stmt
	updatePositionStmt   *sql.Stmt
	
	db *sql.DB
	mu sync.RWMutex
}

// NewHFTQueries creates and initializes all prepared statements
func NewHFTQueries(db *sql.DB) (*HFTQueries, error) {
	hq := &HFTQueries{db: db}
	
	if err := hq.initPreparedStatements(); err != nil {
		return nil, fmt.Errorf("failed to initialize prepared statements: %w", err)
	}
	
	return hq, nil
}

// initPreparedStatements initializes all prepared statements
func (hq *HFTQueries) initPreparedStatements() error {
	var err error
	
	// Order queries
	hq.getOrderByIDStmt, err = hq.db.Prepare(`
		SELECT id, user_id, symbol, side, type, quantity, price, stop_price, 
		       status, filled_quantity, average_price, commission, 
		       created_at, updated_at, executed_at
		FROM orders WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getOrderByID statement: %w", err)
	}
	
	hq.insertOrderStmt, err = hq.db.Prepare(`
		INSERT INTO orders (id, user_id, symbol, side, type, quantity, price, 
		                   stop_price, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insertOrder statement: %w", err)
	}
	
	hq.updateOrderStmt, err = hq.db.Prepare(`
		UPDATE orders 
		SET filled_quantity = ?, average_price = ?, commission = ?, 
		    status = ?, updated_at = ?, executed_at = ?
		WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare updateOrder statement: %w", err)
	}
	
	hq.updateOrderStatusStmt, err = hq.db.Prepare(`
		UPDATE orders SET status = ?, updated_at = ? WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare updateOrderStatus statement: %w", err)
	}
	
	hq.getOrdersByUserStmt, err = hq.db.Prepare(`
		SELECT id, user_id, symbol, side, type, quantity, price, stop_price, 
		       status, filled_quantity, average_price, commission, 
		       created_at, updated_at, executed_at
		FROM orders WHERE user_id = ? ORDER BY created_at DESC LIMIT ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getOrdersByUser statement: %w", err)
	}
	
	hq.getActiveOrdersStmt, err = hq.db.Prepare(`
		SELECT id, user_id, symbol, side, type, quantity, price, stop_price, 
		       status, filled_quantity, average_price, commission, 
		       created_at, updated_at, executed_at
		FROM orders WHERE status IN ('pending', 'partial') ORDER BY created_at ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getActiveOrders statement: %w", err)
	}
	
	// Market data queries
	hq.getLatestPriceStmt, err = hq.db.Prepare(`
		SELECT symbol, price, volume, timestamp 
		FROM market_data 
		WHERE symbol = ? 
		ORDER BY timestamp DESC 
		LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getLatestPrice statement: %w", err)
	}
	
	hq.insertPriceStmt, err = hq.db.Prepare(`
		INSERT INTO market_data (symbol, price, volume, timestamp)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insertPrice statement: %w", err)
	}
	
	hq.getPriceHistoryStmt, err = hq.db.Prepare(`
		SELECT symbol, price, volume, timestamp 
		FROM market_data 
		WHERE symbol = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getPriceHistory statement: %w", err)
	}
	
	// User queries
	hq.getUserByIDStmt, err = hq.db.Prepare(`
		SELECT id, username, email, balance, created_at, updated_at
		FROM users WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getUserByID statement: %w", err)
	}
	
	hq.updateUserBalanceStmt, err = hq.db.Prepare(`
		UPDATE users SET balance = ?, updated_at = ? WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare updateUserBalance statement: %w", err)
	}
	
	return nil
}

// GetOrderByID retrieves an order by ID using prepared statement
func (hq *HFTQueries) GetOrderByID(orderID string) (*models.Order, error) {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	order := &models.Order{}
	var executedAt sql.NullTime
	
	err := hq.getOrderByIDStmt.QueryRow(orderID).Scan(
		&order.ID, &order.UserID, &order.Symbol, &order.Side, &order.Type,
		&order.Quantity, &order.Price, &order.StopPrice, &order.Status,
		&order.FilledQty, &order.AvgPrice, &order.Notes,
		&order.CreatedAt, &order.UpdatedAt, &executedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Order not found
		}
		return nil, fmt.Errorf("failed to get order by ID: %w", err)
	}
	
	if executedAt.Valid {
		order.ExpiresAt = &executedAt.Time
	}
	
	return order, nil
}

// InsertOrder inserts a new order using prepared statement
func (hq *HFTQueries) InsertOrder(order *models.Order) error {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	_, err := hq.insertOrderStmt.Exec(
		order.ID, order.UserID, order.Symbol, order.Side, order.Type,
		order.Quantity, order.Price, order.StopPrice, order.Status,
		order.CreatedAt, order.UpdatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}
	
	return nil
}

// UpdateOrder updates an existing order using prepared statement
func (hq *HFTQueries) UpdateOrder(order *models.Order) error {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	_, err := hq.updateOrderStmt.Exec(
		order.FilledQty, order.AvgPrice, order.Notes,
		order.Status, order.UpdatedAt, order.ExpiresAt, order.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	
	return nil
}

// UpdateOrderStatus updates only the order status using prepared statement
func (hq *HFTQueries) UpdateOrderStatus(orderID, status string) error {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	_, err := hq.updateOrderStatusStmt.Exec(status, time.Now(), orderID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	
	return nil
}

// GetOrdersByUser retrieves orders for a specific user using prepared statement
func (hq *HFTQueries) GetOrdersByUser(userID string, limit int) ([]*models.Order, error) {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	rows, err := hq.getOrdersByUserStmt.Query(userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by user: %w", err)
	}
	defer rows.Close()
	
	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		var executedAt sql.NullTime
		
		err := rows.Scan(
			&order.ID, &order.UserID, &order.Symbol, &order.Side, &order.Type,
			&order.Quantity, &order.Price, &order.StopPrice, &order.Status,
			&order.FilledQty, &order.AvgPrice, &order.Notes,
			&order.CreatedAt, &order.UpdatedAt, &executedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		
		if executedAt.Valid {
			order.ExpiresAt = &executedAt.Time
		}
		
		orders = append(orders, order)
	}
	
	return orders, nil
}

// GetActiveOrders retrieves all active orders using prepared statement
func (hq *HFTQueries) GetActiveOrders() ([]*models.Order, error) {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	rows, err := hq.getActiveOrdersStmt.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get active orders: %w", err)
	}
	defer rows.Close()
	
	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		var executedAt sql.NullTime
		
		err := rows.Scan(
			&order.ID, &order.UserID, &order.Symbol, &order.Side, &order.Type,
			&order.Quantity, &order.Price, &order.StopPrice, &order.Status,
			&order.FilledQty, &order.AvgPrice, &order.Notes,
			&order.CreatedAt, &order.UpdatedAt, &executedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		
		if executedAt.Valid {
			order.ExpiresAt = &executedAt.Time
		}
		
		orders = append(orders, order)
	}
	
	return orders, nil
}

// MarketData represents market data structure
type MarketData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// GetLatestPrice retrieves the latest price for a symbol using prepared statement
func (hq *HFTQueries) GetLatestPrice(symbol string) (*MarketData, error) {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	data := &MarketData{}
	err := hq.getLatestPriceStmt.QueryRow(symbol).Scan(
		&data.Symbol, &data.Price, &data.Volume, &data.Timestamp,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No data found
		}
		return nil, fmt.Errorf("failed to get latest price: %w", err)
	}
	
	return data, nil
}

// InsertPrice inserts market data using prepared statement
func (hq *HFTQueries) InsertPrice(symbol string, price, volume float64, timestamp time.Time) error {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	_, err := hq.insertPriceStmt.Exec(symbol, price, volume, timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert price: %w", err)
	}
	
	return nil
}

// GetPriceHistory retrieves price history for a symbol using prepared statement
func (hq *HFTQueries) GetPriceHistory(symbol string, start, end time.Time) ([]*MarketData, error) {
	tracker := metrics.TrackDBLatency()
	defer tracker.Finish()
	
	hq.mu.RLock()
	defer hq.mu.RUnlock()
	
	rows, err := hq.getPriceHistoryStmt.Query(symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get price history: %w", err)
	}
	defer rows.Close()
	
	var history []*MarketData
	for rows.Next() {
		data := &MarketData{}
		err := rows.Scan(&data.Symbol, &data.Price, &data.Volume, &data.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market data: %w", err)
		}
		history = append(history, data)
	}
	
	return history, nil
}

// Close closes all prepared statements
func (hq *HFTQueries) Close() error {
	hq.mu.Lock()
	defer hq.mu.Unlock()
	
	statements := []*sql.Stmt{
		hq.getOrderByIDStmt, hq.insertOrderStmt, hq.updateOrderStmt,
		hq.updateOrderStatusStmt, hq.getOrdersByUserStmt, hq.getActiveOrdersStmt,
		hq.getLatestPriceStmt, hq.insertPriceStmt, hq.getPriceHistoryStmt,
		hq.getUserByIDStmt, hq.updateUserBalanceStmt,
	}
	
	for _, stmt := range statements {
		if stmt != nil {
			if err := stmt.Close(); err != nil {
				return fmt.Errorf("failed to close prepared statement: %w", err)
			}
		}
	}
	
	return nil
}

// Global HFT queries instance
var GlobalHFTQueries *HFTQueries

// InitHFTQueries initializes the global HFT queries instance
func InitHFTQueries(db *sql.DB) error {
	var err error
	GlobalHFTQueries, err = NewHFTQueries(db)
	if err != nil {
		return fmt.Errorf("failed to initialize HFT queries: %w", err)
	}
	return nil
}
