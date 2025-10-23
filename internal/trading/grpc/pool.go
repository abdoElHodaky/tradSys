package grpc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// HFTGRPCPoolConfig contains gRPC connection pool configuration
type HFTGRPCPoolConfig struct {
	// Pool settings
	InitialSize int           `yaml:"initial_size" default:"5"`
	MaxSize     int           `yaml:"max_size" default:"20"`
	MaxIdleTime time.Duration `yaml:"max_idle_time" default:"5m"`
	MaxLifetime time.Duration `yaml:"max_lifetime" default:"30m"`

	// Connection settings
	ConnectTimeout time.Duration `yaml:"connect_timeout" default:"5s"`
	RequestTimeout time.Duration `yaml:"request_timeout" default:"10s"`

	// Keep-alive settings
	KeepAliveTime       time.Duration `yaml:"keep_alive_time" default:"30s"`
	KeepAliveTimeout    time.Duration `yaml:"keep_alive_timeout" default:"5s"`
	PermitWithoutStream bool          `yaml:"permit_without_stream" default:"true"`

	// Retry settings
	MaxRetries    int           `yaml:"max_retries" default:"3"`
	RetryDelay    time.Duration `yaml:"retry_delay" default:"100ms"`
	BackoffFactor float64       `yaml:"backoff_factor" default:"2.0"`
}

// PooledConnection represents a pooled gRPC connection
type PooledConnection struct {
	conn       *grpc.ClientConn
	createdAt  time.Time
	lastUsedAt time.Time
	usageCount int64
	inUse      bool
	mu         sync.RWMutex
}

// IsHealthy checks if the connection is healthy
func (pc *PooledConnection) IsHealthy() bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if pc.conn == nil {
		return false
	}

	state := pc.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// IsExpired checks if the connection has expired
func (pc *PooledConnection) IsExpired(maxLifetime, maxIdleTime time.Duration) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	now := time.Now()

	// Check lifetime
	if maxLifetime > 0 && now.Sub(pc.createdAt) > maxLifetime {
		return true
	}

	// Check idle time
	if maxIdleTime > 0 && now.Sub(pc.lastUsedAt) > maxIdleTime {
		return true
	}

	return false
}

// MarkUsed marks the connection as used
func (pc *PooledConnection) MarkUsed() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.lastUsedAt = time.Now()
	atomic.AddInt64(&pc.usageCount, 1)
	pc.inUse = true
}

// MarkUnused marks the connection as unused
func (pc *PooledConnection) MarkUnused() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.inUse = false
}

// Close closes the connection
func (pc *PooledConnection) Close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.conn != nil {
		return pc.conn.Close()
	}
	return nil
}

// HFTGRPCPool manages a pool of gRPC connections
type HFTGRPCPool struct {
	config      *HFTGRPCPoolConfig
	target      string
	connections []*PooledConnection
	mu          sync.RWMutex
	closed      bool

	// Statistics
	totalConnections  int64
	activeConnections int64
	totalRequests     int64
	failedRequests    int64
}

// NewHFTGRPCPool creates a new gRPC connection pool
func NewHFTGRPCPool(target string, config *HFTGRPCPoolConfig) (*HFTGRPCPool, error) {
	if config == nil {
		config = &HFTGRPCPoolConfig{
			InitialSize:         5,
			MaxSize:             20,
			MaxIdleTime:         5 * time.Minute,
			MaxLifetime:         30 * time.Minute,
			ConnectTimeout:      5 * time.Second,
			RequestTimeout:      10 * time.Second,
			KeepAliveTime:       30 * time.Second,
			KeepAliveTimeout:    5 * time.Second,
			PermitWithoutStream: true,
			MaxRetries:          3,
			RetryDelay:          100 * time.Millisecond,
			BackoffFactor:       2.0,
		}
	}

	pool := &HFTGRPCPool{
		config:      config,
		target:      target,
		connections: make([]*PooledConnection, 0, config.MaxSize),
	}

	// Create initial connections
	for i := 0; i < config.InitialSize; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			// Close any connections created so far
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection %d: %w", i, err)
		}
		pool.connections = append(pool.connections, conn)
		atomic.AddInt64(&pool.totalConnections, 1)
	}

	// Start maintenance goroutine
	go pool.maintenanceLoop()

	return pool, nil
}

// createConnection creates a new gRPC connection
func (p *HFTGRPCPool) createConnection() (*PooledConnection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectTimeout)
	defer cancel()

	// Configure keep-alive parameters
	kacp := keepalive.ClientParameters{
		Time:                p.config.KeepAliveTime,
		Timeout:             p.config.KeepAliveTimeout,
		PermitWithoutStream: p.config.PermitWithoutStream,
	}

	// Create connection with optimized settings
	conn, err := grpc.DialContext(ctx, p.target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kacp),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
		grpc.WithInitialWindowSize(1024*1024),     // 1MB
		grpc.WithInitialConnWindowSize(1024*1024), // 1MB
		grpc.WithWriteBufferSize(32*1024),         // 32KB
		grpc.WithReadBufferSize(32*1024),          // 32KB
	)

	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &PooledConnection{
		conn:       conn,
		createdAt:  now,
		lastUsedAt: now,
	}, nil
}

// GetConnection retrieves a connection from the pool
func (p *HFTGRPCPool) GetConnection() (*PooledConnection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, fmt.Errorf("connection pool is closed")
	}

	atomic.AddInt64(&p.totalRequests, 1)

	// Find an available healthy connection
	for _, conn := range p.connections {
		if !conn.inUse && conn.IsHealthy() && !conn.IsExpired(p.config.MaxLifetime, p.config.MaxIdleTime) {
			conn.MarkUsed()
			atomic.AddInt64(&p.activeConnections, 1)
			return conn, nil
		}
	}

	// Create a new connection if pool is not at max capacity
	if len(p.connections) < p.config.MaxSize {
		conn, err := p.createConnection()
		if err != nil {
			atomic.AddInt64(&p.failedRequests, 1)
			return nil, err
		}

		conn.MarkUsed()
		p.connections = append(p.connections, conn)
		atomic.AddInt64(&p.totalConnections, 1)
		atomic.AddInt64(&p.activeConnections, 1)
		return conn, nil
	}

	// Pool is full, wait for a connection to become available
	// For HFT, we don't want to wait too long
	atomic.AddInt64(&p.failedRequests, 1)
	return nil, fmt.Errorf("connection pool exhausted")
}

// ReturnConnection returns a connection to the pool
func (p *HFTGRPCPool) ReturnConnection(conn *PooledConnection) {
	if conn == nil {
		return
	}

	conn.MarkUnused()
	atomic.AddInt64(&p.activeConnections, -1)
}

// maintenanceLoop performs periodic maintenance on the pool
func (p *HFTGRPCPool) maintenanceLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return
		}

		// Remove expired or unhealthy connections
		var activeConnections []*PooledConnection
		for _, conn := range p.connections {
			if conn.inUse {
				// Keep connections that are in use
				activeConnections = append(activeConnections, conn)
			} else if conn.IsHealthy() && !conn.IsExpired(p.config.MaxLifetime, p.config.MaxIdleTime) {
				// Keep healthy, non-expired connections
				activeConnections = append(activeConnections, conn)
			} else {
				// Close expired or unhealthy connections
				conn.Close()
				atomic.AddInt64(&p.totalConnections, -1)
			}
		}

		p.connections = activeConnections
		p.mu.Unlock()
	}
}

// ExecuteWithRetry executes a function with retry logic
func (p *HFTGRPCPool) ExecuteWithRetry(fn func(*grpc.ClientConn) error) error {
	var lastErr error
	delay := p.config.RetryDelay

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		conn, err := p.GetConnection()
		if err != nil {
			lastErr = err
			if attempt < p.config.MaxRetries {
				time.Sleep(delay)
				delay = time.Duration(float64(delay) * p.config.BackoffFactor)
			}
			continue
		}

		err = fn(conn.conn)
		p.ReturnConnection(conn)

		if err == nil {
			return nil
		}

		lastErr = err
		if attempt < p.config.MaxRetries {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * p.config.BackoffFactor)
		}
	}

	return lastErr
}

// GetStats returns pool statistics
func (p *HFTGRPCPool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"total_connections":  atomic.LoadInt64(&p.totalConnections),
		"active_connections": atomic.LoadInt64(&p.activeConnections),
		"idle_connections":   atomic.LoadInt64(&p.totalConnections) - atomic.LoadInt64(&p.activeConnections),
		"total_requests":     atomic.LoadInt64(&p.totalRequests),
		"failed_requests":    atomic.LoadInt64(&p.failedRequests),
		"pool_size":          len(p.connections),
		"max_pool_size":      p.config.MaxSize,
		"target":             p.target,
	}
}

// Close closes all connections in the pool
func (p *HFTGRPCPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	var errs []error
	for _, conn := range p.connections {
		if err := conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	p.connections = nil

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// HFTGRPCPoolManager manages multiple gRPC connection pools
type HFTGRPCPoolManager struct {
	pools map[string]*HFTGRPCPool
	mu    sync.RWMutex
}

// NewHFTGRPCPoolManager creates a new gRPC pool manager
func NewHFTGRPCPoolManager() *HFTGRPCPoolManager {
	return &HFTGRPCPoolManager{
		pools: make(map[string]*HFTGRPCPool),
	}
}

// GetPool gets or creates a connection pool for a target
func (m *HFTGRPCPoolManager) GetPool(target string, config *HFTGRPCPoolConfig) (*HFTGRPCPool, error) {
	m.mu.RLock()
	if pool, exists := m.pools[target]; exists {
		m.mu.RUnlock()
		return pool, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := m.pools[target]; exists {
		return pool, nil
	}

	// Create new pool
	pool, err := NewHFTGRPCPool(target, config)
	if err != nil {
		return nil, err
	}

	m.pools[target] = pool
	return pool, nil
}

// CloseAll closes all connection pools
func (m *HFTGRPCPoolManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for target, pool := range m.pools {
		if err := pool.Close(); err != nil {
			errs = append(errs, fmt.Errorf("error closing pool for %s: %w", target, err))
		}
	}

	m.pools = make(map[string]*HFTGRPCPool)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing pools: %v", errs)
	}

	return nil
}

// GetAllStats returns statistics for all pools
func (m *HFTGRPCPoolManager) GetAllStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	for target, pool := range m.pools {
		stats[target] = pool.GetStats()
	}

	return stats
}

// Global pool manager instance
var GlobalPoolManager = NewHFTGRPCPoolManager()
