package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// ConnectionPool manages a pool of database connections
type ConnectionPool struct {
	db           *sqlx.DB
	logger       *zap.Logger
	maxOpenConns int
	maxIdleConns int
	connLifetime time.Duration
	metrics      *ConnectionMetrics
	mutex        sync.RWMutex
}

// ConnectionMetrics tracks database connection metrics
type ConnectionMetrics struct {
	OpenConnections    int64
	InUseConnections   int64
	IdleConnections    int64
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleTimeClosed  int64
	MaxLifetimeClosed  int64
	QueryCount         int64
	QueryErrors        int64
	QueryDuration      time.Duration
	SlowQueryThreshold time.Duration
	SlowQueries        int64
	mutex              sync.RWMutex
}

// ConnectionPoolOptions contains options for the connection pool
type ConnectionPoolOptions struct {
	MaxOpenConns      int
	MaxIdleConns      int
	ConnLifetime      time.Duration
	SlowQueryThreshold time.Duration
}

// NewConnectionPool creates a new database connection pool
func NewConnectionPool(db *sqlx.DB, logger *zap.Logger, options ConnectionPoolOptions) *ConnectionPool {
	// Set default values if not provided
	if options.MaxOpenConns == 0 {
		options.MaxOpenConns = 25
	}
	if options.MaxIdleConns == 0 {
		options.MaxIdleConns = 10
	}
	if options.ConnLifetime == 0 {
		options.ConnLifetime = 5 * time.Minute
	}
	if options.SlowQueryThreshold == 0 {
		options.SlowQueryThreshold = 100 * time.Millisecond
	}

	// Configure connection pool
	db.SetMaxOpenConns(options.MaxOpenConns)
	db.SetMaxIdleConns(options.MaxIdleConns)
	db.SetConnMaxLifetime(options.ConnLifetime)

	metrics := &ConnectionMetrics{
		SlowQueryThreshold: options.SlowQueryThreshold,
	}

	pool := &ConnectionPool{
		db:           db,
		logger:       logger,
		maxOpenConns: options.MaxOpenConns,
		maxIdleConns: options.MaxIdleConns,
		connLifetime: options.ConnLifetime,
		metrics:      metrics,
	}

	// Start metrics collection
	go pool.collectMetrics()

	logger.Info("Database connection pool initialized",
		zap.Int("max_open_conns", options.MaxOpenConns),
		zap.Int("max_idle_conns", options.MaxIdleConns),
		zap.Duration("conn_lifetime", options.ConnLifetime),
		zap.Duration("slow_query_threshold", options.SlowQueryThreshold),
	)

	return pool
}

// collectMetrics periodically collects connection pool metrics
func (p *ConnectionPool) collectMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := p.db.Stats()

		p.metrics.mutex.Lock()
		p.metrics.OpenConnections = int64(stats.OpenConnections)
		p.metrics.InUseConnections = int64(stats.InUse)
		p.metrics.IdleConnections = int64(stats.Idle)
		p.metrics.WaitCount = stats.WaitCount
		p.metrics.WaitDuration = stats.WaitDuration
		p.metrics.MaxIdleTimeClosed = stats.MaxIdleClosed
		p.metrics.MaxLifetimeClosed = stats.MaxLifetimeClosed
		p.metrics.mutex.Unlock()

		p.logger.Debug("Database connection pool metrics",
			zap.Int("open_connections", stats.OpenConnections),
			zap.Int("in_use_connections", stats.InUse),
			zap.Int("idle_connections", stats.Idle),
			zap.Int64("wait_count", stats.WaitCount),
			zap.Duration("wait_duration", stats.WaitDuration),
			zap.Int64("max_idle_closed", stats.MaxIdleClosed),
			zap.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
		)
	}
}

// GetDB returns the underlying database connection
func (p *ConnectionPool) GetDB() *sqlx.DB {
	return p.db
}

// Exec executes a query without returning any rows
func (p *ConnectionPool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	startTime := time.Now()
	result, err := p.db.ExecContext(ctx, query, args...)
	duration := time.Since(startTime)

	p.trackQuery(query, duration, err)

	return result, err
}

// Query executes a query that returns rows
func (p *ConnectionPool) Query(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	startTime := time.Now()
	rows, err := p.db.QueryxContext(ctx, query, args...)
	duration := time.Since(startTime)

	p.trackQuery(query, duration, err)

	return rows, err
}

// QueryRow executes a query that returns a single row
func (p *ConnectionPool) QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	startTime := time.Now()
	row := p.db.QueryRowxContext(ctx, query, args...)
	duration := time.Since(startTime)

	p.trackQuery(query, duration, nil)

	return row
}

// NamedExec executes a named query without returning any rows
func (p *ConnectionPool) NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	startTime := time.Now()
	result, err := p.db.NamedExecContext(ctx, query, arg)
	duration := time.Since(startTime)

	p.trackQuery(query, duration, err)

	return result, err
}

// NamedQuery executes a named query that returns rows
func (p *ConnectionPool) NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	startTime := time.Now()
	rows, err := p.db.NamedQueryContext(ctx, query, arg)
	duration := time.Since(startTime)

	p.trackQuery(query, duration, err)

	return rows, err
}

// Select executes a query and scans the results into dest
func (p *ConnectionPool) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	startTime := time.Now()
	err := p.db.SelectContext(ctx, dest, query, args...)
	duration := time.Since(startTime)

	p.trackQuery(query, duration, err)

	return err
}

// Get executes a query and scans the result into dest
func (p *ConnectionPool) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	startTime := time.Now()
	err := p.db.GetContext(ctx, dest, query, args...)
	duration := time.Since(startTime)

	p.trackQuery(query, duration, err)

	return err
}

// Begin starts a transaction
func (p *ConnectionPool) Begin(ctx context.Context) (*sqlx.Tx, error) {
	return p.db.BeginTxx(ctx, nil)
}

// BeginTx starts a transaction with options
func (p *ConnectionPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	return p.db.BeginTxx(ctx, opts)
}

// Ping verifies a connection to the database is still alive
func (p *ConnectionPool) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Close closes the database connection pool
func (p *ConnectionPool) Close() error {
	p.logger.Info("Closing database connection pool")
	return p.db.Close()
}

// trackQuery tracks query metrics
func (p *ConnectionPool) trackQuery(query string, duration time.Duration, err error) {
	p.metrics.mutex.Lock()
	defer p.metrics.mutex.Unlock()

	p.metrics.QueryCount++
	p.metrics.QueryDuration += duration

	if err != nil {
		p.metrics.QueryErrors++
		p.logger.Error("Database query error",
			zap.Error(err),
			zap.String("query", query),
			zap.Duration("duration", duration),
		)
	}

	if duration >= p.metrics.SlowQueryThreshold {
		p.metrics.SlowQueries++
		p.logger.Warn("Slow database query",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Duration("threshold", p.metrics.SlowQueryThreshold),
		)
	}
}

// GetMetrics returns the current connection metrics
func (p *ConnectionPool) GetMetrics() ConnectionMetrics {
	p.metrics.mutex.RLock()
	defer p.metrics.mutex.RUnlock()

	return *p.metrics
}

// ResetMetrics resets the query metrics
func (p *ConnectionPool) ResetMetrics() {
	p.metrics.mutex.Lock()
	defer p.metrics.mutex.Unlock()

	p.metrics.QueryCount = 0
	p.metrics.QueryErrors = 0
	p.metrics.QueryDuration = 0
	p.metrics.SlowQueries = 0

	p.logger.Info("Database query metrics reset")
}

// GetStats returns the current database stats
func (p *ConnectionPool) GetStats() sql.DBStats {
	return p.db.Stats()
}

// LogStats logs the current database stats
func (p *ConnectionPool) LogStats() {
	stats := p.db.Stats()
	p.logger.Info("Database connection pool stats",
		zap.Int("max_open_conns", p.maxOpenConns),
		zap.Int("max_idle_conns", p.maxIdleConns),
		zap.Duration("conn_lifetime", p.connLifetime),
		zap.Int("open_connections", stats.OpenConnections),
		zap.Int("in_use_connections", stats.InUse),
		zap.Int("idle_connections", stats.Idle),
		zap.Int64("wait_count", stats.WaitCount),
		zap.Duration("wait_duration", stats.WaitDuration),
		zap.Int64("max_idle_closed", stats.MaxIdleClosed),
		zap.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
	)
}

// WithTransaction executes a function within a transaction
func (p *ConnectionPool) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// Rollback on panic
			_ = tx.Rollback()
			panic(p) // Re-throw panic after rollback
		} else if err != nil {
			// Rollback on error
			_ = tx.Rollback()
		} else {
			// Commit if no error or panic
			err = tx.Commit()
			if err != nil {
				err = fmt.Errorf("failed to commit transaction: %w", err)
			}
		}
	}()

	err = fn(tx)
	return err
}
