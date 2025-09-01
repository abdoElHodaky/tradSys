package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConnectionPool manages a pool of database connections
type ConnectionPool struct {
	// Database connection
	db *sql.DB
	
	// Configuration
	maxConnections int
	maxIdleTime    time.Duration
	
	// Connection tracking
	activeConnections int
	mu                sync.Mutex
	
	// Metrics
	metrics ConnectionPoolMetrics
	
	// Logger
	logger *zap.Logger
}

// ConnectionPoolMetrics tracks metrics for the connection pool
type ConnectionPoolMetrics struct {
	TotalConnections     int64
	ActiveConnections    int64
	IdleConnections      int64
	WaitCount            int64
	WaitDuration         time.Duration
	MaxIdleTimeClosed    int64
	MaxLifetimeClosed    int64
	ConnectionErrors     int64
	ConnectionTimeouts   int64
	QueryCount           int64
	QueryErrors          int64
	QueryDuration        time.Duration
	TransactionCount     int64
	TransactionErrors    int64
	TransactionDuration  time.Duration
	TransactionRollbacks int64
}

// ConnectionPoolConfig represents the configuration for the connection pool
type ConnectionPoolConfig struct {
	// MaxConnections is the maximum number of connections in the pool
	MaxConnections int
	// MaxIdleConnections is the maximum number of idle connections in the pool
	MaxIdleConnections int
	// MaxIdleTime is the maximum time a connection can be idle before being closed
	MaxIdleTime time.Duration
	// MaxLifetime is the maximum time a connection can be used before being closed
	MaxLifetime time.Duration
	// ConnectionTimeout is the timeout for establishing a connection
	ConnectionTimeout time.Duration
}

// DefaultConnectionPoolConfig returns the default configuration for the connection pool
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxConnections:     10,
		MaxIdleConnections: 5,
		MaxIdleTime:        5 * time.Minute,
		MaxLifetime:        1 * time.Hour,
		ConnectionTimeout:  10 * time.Second,
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(db *sql.DB, config ConnectionPoolConfig, logger *zap.Logger) (*ConnectionPool, error) {
	if db == nil {
		return nil, errors.New("database connection is nil")
	}
	
	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}
	
	// Configure connection pool
	db.SetMaxOpenConns(config.MaxConnections)
	db.SetMaxIdleConns(config.MaxIdleConnections)
	db.SetConnMaxIdleTime(config.MaxIdleTime)
	db.SetConnMaxLifetime(config.MaxLifetime)
	
	return &ConnectionPool{
		db:             db,
		maxConnections: config.MaxConnections,
		maxIdleTime:    config.MaxIdleTime,
		logger:         logger.With(zap.String("component", "connection_pool")),
	}, nil
}

// GetConnection gets a connection from the pool
func (p *ConnectionPool) GetConnection(ctx context.Context) (*sql.Conn, error) {
	p.mu.Lock()
	if p.activeConnections >= p.maxConnections {
		p.mu.Unlock()
		return nil, errors.New("connection pool exhausted")
	}
	p.activeConnections++
	p.metrics.ActiveConnections++
	p.metrics.TotalConnections++
	p.mu.Unlock()
	
	// Get connection from pool
	conn, err := p.db.Conn(ctx)
	if err != nil {
		p.mu.Lock()
		p.activeConnections--
		p.metrics.ActiveConnections--
		p.metrics.ConnectionErrors++
		p.mu.Unlock()
		
		p.logger.Error("Failed to get connection from pool",
			zap.Error(err))
		
		return nil, fmt.Errorf("failed to get connection from pool: %w", err)
	}
	
	return conn, nil
}

// ReleaseConnection releases a connection back to the pool
func (p *ConnectionPool) ReleaseConnection(conn *sql.Conn) error {
	if conn == nil {
		return errors.New("connection is nil")
	}
	
	p.mu.Lock()
	p.activeConnections--
	p.metrics.ActiveConnections--
	p.mu.Unlock()
	
	return conn.Close()
}

// Close closes the connection pool
func (p *ConnectionPool) Close() error {
	return p.db.Close()
}

// Stats returns statistics about the connection pool
func (p *ConnectionPool) Stats() sql.DBStats {
	return p.db.Stats()
}

// Metrics returns metrics for the connection pool
func (p *ConnectionPool) Metrics() ConnectionPoolMetrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Update metrics from DB stats
	stats := p.db.Stats()
	p.metrics.IdleConnections = int64(stats.Idle)
	p.metrics.WaitCount = stats.WaitCount
	p.metrics.WaitDuration = stats.WaitDuration
	p.metrics.MaxIdleTimeClosed = stats.MaxIdleTimeClosed
	p.metrics.MaxLifetimeClosed = stats.MaxLifetimeClosed
	
	return p.metrics
}

// Exec executes a query without returning any rows
func (p *ConnectionPool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	
	p.mu.Lock()
	p.metrics.QueryCount++
	p.mu.Unlock()
	
	result, err := p.db.ExecContext(ctx, query, args...)
	
	p.mu.Lock()
	p.metrics.QueryDuration += time.Since(start)
	if err != nil {
		p.metrics.QueryErrors++
	}
	p.mu.Unlock()
	
	if err != nil {
		p.logger.Error("Query execution failed",
			zap.String("query", query),
			zap.Error(err))
	}
	
	return result, err
}

// Query executes a query that returns rows
func (p *ConnectionPool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	
	p.mu.Lock()
	p.metrics.QueryCount++
	p.mu.Unlock()
	
	rows, err := p.db.QueryContext(ctx, query, args...)
	
	p.mu.Lock()
	p.metrics.QueryDuration += time.Since(start)
	if err != nil {
		p.metrics.QueryErrors++
	}
	p.mu.Unlock()
	
	if err != nil {
		p.logger.Error("Query execution failed",
			zap.String("query", query),
			zap.Error(err))
	}
	
	return rows, err
}

// QueryRow executes a query that returns a single row
func (p *ConnectionPool) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	
	p.mu.Lock()
	p.metrics.QueryCount++
	p.mu.Unlock()
	
	row := p.db.QueryRowContext(ctx, query, args...)
	
	p.mu.Lock()
	p.metrics.QueryDuration += time.Since(start)
	p.mu.Unlock()
	
	return row
}

// Begin starts a new transaction
func (p *ConnectionPool) Begin(ctx context.Context) (*sql.Tx, error) {
	startTime := time.Now()
	
	p.mu.Lock()
	p.metrics.TransactionCount++
	p.mu.Unlock()
	
	tx, err := p.db.BeginTx(ctx, nil)
	
	p.mu.Lock()
	if err != nil {
		p.metrics.TransactionErrors++
	} else {
		p.metrics.TransactionDuration += time.Since(startTime)
	}
	p.mu.Unlock()
	
	if err != nil {
		p.logger.Error("Failed to start transaction",
			zap.Error(err))
	}
	
	return tx, err
}

// Select executes a query that returns rows and scans them into dest
func (p *ConnectionPool) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	
	p.mu.Lock()
	p.metrics.QueryCount++
	p.mu.Unlock()
	
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		p.mu.Lock()
		p.metrics.QueryErrors++
		p.metrics.QueryDuration += time.Since(start)
		p.mu.Unlock()
		
		p.logger.Error("Query execution failed",
			zap.String("query", query),
			zap.Error(err))
		
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()
	
	// Scan rows into dest
	// This is a simplified implementation - in a real application,
	// you would use a library like sqlx or sqlc to handle scanning
	// For now, we'll just return an error if dest is not a pointer to a slice
	
	p.mu.Lock()
	p.metrics.QueryDuration += time.Since(start)
	p.mu.Unlock()
	
	return fmt.Errorf("not implemented: use a proper SQL mapper library")
}

// WithTransaction executes a function within a transaction
func (p *ConnectionPool) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	start := time.Now()
	
	p.mu.Lock()
	p.metrics.TransactionCount++
	p.mu.Unlock()
	
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		p.mu.Lock()
		p.metrics.TransactionErrors++
		p.mu.Unlock()
		
		p.logger.Error("Failed to start transaction",
			zap.Error(err))
		
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	
	// Execute function within transaction
	err = fn(tx)
	
	// Handle panic

	if rec := recover(); rec != nil {

	if r := recover(); r != nil {

		// Attempt to roll back the transaction
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			p.logger.Error("Failed to roll back transaction after panic",
				zap.Error(rollbackErr))
		}
		
		p.mu.Lock()
		p.metrics.TransactionRollbacks++
		p.mu.Unlock()
		
		// Re-throw the original panic
 
		if err, ok := rec.(error); ok {

		if err, ok := r.(error); ok {

			p.logger.Error("Panic in transaction",
				zap.Error(err))
			return err
		}
		
 
		return fmt.Errorf("panic in transaction: %v", rec)

		return fmt.Errorf("panic in transaction: %v", r)

	}
	
	// Handle error
	if err != nil {
		// Attempt to roll back the transaction
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			p.logger.Error("Failed to roll back transaction after error",
				zap.Error(rollbackErr))
		}
		
		p.mu.Lock()
		p.metrics.TransactionErrors++
		p.metrics.TransactionRollbacks++
		p.mu.Unlock()
		
		p.logger.Error("Transaction failed",
			zap.Error(err))
		
		return fmt.Errorf("transaction failed: %w", err)
	}
	
	// Commit the transaction
	if err := tx.Commit(); err != nil {
		p.mu.Lock()
		p.metrics.TransactionErrors++
		p.mu.Unlock()
		
		p.logger.Error("Failed to commit transaction",
			zap.Error(err))
		
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	p.mu.Lock()
	p.metrics.TransactionDuration += time.Since(start)
	p.mu.Unlock()
	
	return nil
}

