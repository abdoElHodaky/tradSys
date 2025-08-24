package performance

import (
	"errors"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ConnectionState represents the state of a connection
type ConnectionState int

const (
	// ConnectionStateIdle means the connection is idle
	ConnectionStateIdle ConnectionState = iota
	// ConnectionStateActive means the connection is active
	ConnectionStateActive
	// ConnectionStateClosed means the connection is closed
	ConnectionStateClosed
)

// PooledConnection represents a connection in the pool
type PooledConnection struct {
	// Conn is the WebSocket connection
	Conn *websocket.Conn
	
	// State is the state of the connection
	State ConnectionState
	
	// LastUsed is when the connection was last used
	LastUsed time.Time
	
	// CreatedAt is when the connection was created
	CreatedAt time.Time
	
	// ID is the unique identifier for the connection
	ID string
	
	// Stats contains connection statistics
	Stats ConnectionStats
	
	// Mutex for protecting the connection
	mu sync.RWMutex
}

// ConnectionStats contains statistics for a connection
type ConnectionStats struct {
	// BytesSent is the number of bytes sent
	BytesSent int64
	
	// BytesReceived is the number of bytes received
	BytesReceived int64
	
	// MessagesSent is the number of messages sent
	MessagesSent int64
	
	// MessagesReceived is the number of messages received
	MessagesReceived int64
	
	// Errors is the number of errors
	Errors int64
	
	// LastError is the last error that occurred
	LastError error
	
	// LastErrorTime is when the last error occurred
	LastErrorTime time.Time
}

// ConnectionPoolConfig contains configuration for the connection pool
type ConnectionPoolConfig struct {
	// MinConnections is the minimum number of connections to maintain
	MinConnections int
	
	// MaxConnections is the maximum number of connections to maintain
	MaxConnections int
	
	// MaxIdleTime is the maximum time a connection can be idle
	MaxIdleTime time.Duration
	
	// MaxLifetime is the maximum lifetime of a connection
	MaxLifetime time.Duration
	
	// HealthCheckInterval is the interval at which health checks are performed
	HealthCheckInterval time.Duration
	
	// ConnectionTimeout is the timeout for establishing a connection
	ConnectionTimeout time.Duration
	
	// EnableConnectionReuse enables connection reuse
	EnableConnectionReuse bool
	
	// EnableHealthChecks enables health checks
	EnableHealthChecks bool
}

// DefaultConnectionPoolConfig returns the default configuration
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MinConnections:      5,
		MaxConnections:      100,
		MaxIdleTime:         5 * time.Minute,
		MaxLifetime:         30 * time.Minute,
		HealthCheckInterval: 1 * time.Minute,
		ConnectionTimeout:   10 * time.Second,
		EnableConnectionReuse: true,
		EnableHealthChecks:  true,
	}
}

// ConnectionPool manages a pool of WebSocket connections
type ConnectionPool struct {
	// Configuration
	config ConnectionPoolConfig
	
	// Connections
	connections map[string]*PooledConnection
	
	// Idle connections
	idleConnections []*PooledConnection
	
	// Mutex for protecting the connections
	mu sync.RWMutex
	
	// Function for creating a new connection
	createConnFunc func() (*websocket.Conn, error)
	
	// Logger
	logger *zap.Logger
	
	// Metrics
	metrics *metrics.WebSocketMetrics
	
	// Stop channel
	stopCh chan struct{}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(
	config ConnectionPoolConfig,
	createConnFunc func() (*websocket.Conn, error),
	logger *zap.Logger,
	metrics *metrics.WebSocketMetrics,
) *ConnectionPool {
	pool := &ConnectionPool{
		config:          config,
		connections:     make(map[string]*PooledConnection),
		idleConnections: make([]*PooledConnection, 0, config.MaxConnections),
		createConnFunc:  createConnFunc,
		logger:          logger,
		metrics:         metrics,
		stopCh:          make(chan struct{}),
	}
	
	// Initialize the pool
	pool.initialize()
	
	// Start the health check goroutine if enabled
	if config.EnableHealthChecks {
		go pool.healthCheckLoop()
	}
	
	return pool
}

// initialize initializes the connection pool
func (p *ConnectionPool) initialize() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create the minimum number of connections
	for i := 0; i < p.config.MinConnections; i++ {
		conn, err := p.createConnection()
		if err != nil {
			p.logger.Error("Failed to create connection during initialization",
				zap.Error(err),
				zap.Int("attempt", i+1))
			continue
		}
		
		// Add the connection to the pool
		p.addConnectionLocked(conn)
	}
}

// GetConnection gets a connection from the pool
func (p *ConnectionPool) GetConnection() (*PooledConnection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check if there are any idle connections
	if len(p.idleConnections) > 0 {
		// Get the last idle connection
		conn := p.idleConnections[len(p.idleConnections)-1]
		p.idleConnections = p.idleConnections[:len(p.idleConnections)-1]
		
		// Update the connection state
		conn.mu.Lock()
		conn.State = ConnectionStateActive
		conn.LastUsed = time.Now()
		conn.mu.Unlock()
		
		return conn, nil
	}
	
	// Check if we can create a new connection
	if len(p.connections) >= p.config.MaxConnections {
		return nil, errors.New("connection pool is full")
	}
	
	// Create a new connection
	conn, err := p.createConnection()
	if err != nil {
		return nil, err
	}
	
	// Add the connection to the pool
	pooledConn := p.addConnectionLocked(conn)
	
	return pooledConn, nil
}

// ReleaseConnection releases a connection back to the pool
func (p *ConnectionPool) ReleaseConnection(conn *PooledConnection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check if the connection is closed
	if conn.State == ConnectionStateClosed {
		// Remove the connection from the pool
		delete(p.connections, conn.ID)
		return
	}
	
	// Check if connection reuse is disabled
	if !p.config.EnableConnectionReuse {
		// Close the connection
		conn.mu.Lock()
		conn.State = ConnectionStateClosed
		conn.mu.Unlock()
		
		conn.Conn.Close()
		
		// Remove the connection from the pool
		delete(p.connections, conn.ID)
		return
	}
	
	// Update the connection state
	conn.mu.Lock()
	conn.State = ConnectionStateIdle
	conn.LastUsed = time.Now()
	conn.mu.Unlock()
	
	// Add the connection to the idle list
	p.idleConnections = append(p.idleConnections, conn)
}

// CloseConnection closes a connection
func (p *ConnectionPool) CloseConnection(conn *PooledConnection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Update the connection state
	conn.mu.Lock()
	conn.State = ConnectionStateClosed
	conn.mu.Unlock()
	
	// Close the connection
	conn.Conn.Close()
	
	// Remove the connection from the pool
	delete(p.connections, conn.ID)
}

// createConnection creates a new WebSocket connection
func (p *ConnectionPool) createConnection() (*websocket.Conn, error) {
	// Set a timeout for connection creation
	timeoutCh := make(chan struct{})
	go func() {
		time.Sleep(p.config.ConnectionTimeout)
		close(timeoutCh)
	}()
	
	// Create a channel for the connection result
	resultCh := make(chan struct {
		conn *websocket.Conn
		err  error
	})
	
	// Create the connection in a goroutine
	go func() {
		conn, err := p.createConnFunc()
		resultCh <- struct {
			conn *websocket.Conn
			err  error
		}{conn, err}
	}()
	
	// Wait for the connection or timeout
	select {
	case result := <-resultCh:
		return result.conn, result.err
	case <-timeoutCh:
		return nil, errors.New("connection timeout")
	}
}

// addConnectionLocked adds a connection to the pool (must be called with lock held)
func (p *ConnectionPool) addConnectionLocked(conn *websocket.Conn) *PooledConnection {
	// Create a unique ID for the connection
	id := generateConnectionID()
	
	// Create the pooled connection
	pooledConn := &PooledConnection{
		Conn:      conn,
		State:     ConnectionStateIdle,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
		ID:        id,
		Stats: ConnectionStats{
			BytesSent:       0,
			BytesReceived:   0,
			MessagesSent:    0,
			MessagesReceived: 0,
			Errors:          0,
			LastError:       nil,
			LastErrorTime:   time.Time{},
		},
	}
	
	// Add the connection to the pool
	p.connections[id] = pooledConn
	p.idleConnections = append(p.idleConnections, pooledConn)
	
	return pooledConn
}

// healthCheckLoop performs periodic health checks
func (p *ConnectionPool) healthCheckLoop() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.performHealthCheck()
		case <-p.stopCh:
			return
		}
	}
}

// performHealthCheck performs a health check on all connections
func (p *ConnectionPool) performHealthCheck() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	now := time.Now()
	
	// Check all connections
	for id, conn := range p.connections {
		conn.mu.RLock()
		state := conn.State
		lastUsed := conn.LastUsed
		createdAt := conn.CreatedAt
		conn.mu.RUnlock()
		
		// Check if the connection is idle and has exceeded the max idle time
		if state == ConnectionStateIdle && now.Sub(lastUsed) > p.config.MaxIdleTime {
			p.logger.Debug("Closing idle connection",
				zap.String("id", id),
				zap.Duration("idle_time", now.Sub(lastUsed)))
			
			// Close the connection
			conn.mu.Lock()
			conn.State = ConnectionStateClosed
			conn.mu.Unlock()
			
			conn.Conn.Close()
			
			// Remove the connection from the pool
			delete(p.connections, id)
			
			// Remove the connection from the idle list
			for i, idleConn := range p.idleConnections {
				if idleConn.ID == id {
					p.idleConnections = append(p.idleConnections[:i], p.idleConnections[i+1:]...)
					break
				}
			}
			
			continue
		}
		
		// Check if the connection has exceeded the max lifetime
		if now.Sub(createdAt) > p.config.MaxLifetime {
			p.logger.Debug("Closing connection due to max lifetime",
				zap.String("id", id),
				zap.Duration("lifetime", now.Sub(createdAt)))
			
			// Close the connection
			conn.mu.Lock()
			conn.State = ConnectionStateClosed
			conn.mu.Unlock()
			
			conn.Conn.Close()
			
			// Remove the connection from the pool
			delete(p.connections, id)
			
			// Remove the connection from the idle list if it's idle
			if state == ConnectionStateIdle {
				for i, idleConn := range p.idleConnections {
					if idleConn.ID == id {
						p.idleConnections = append(p.idleConnections[:i], p.idleConnections[i+1:]...)
						break
					}
				}
			}
			
			continue
		}
		
		// Check if the connection is idle and we need to ping it
		if state == ConnectionStateIdle {
			// Send a ping message
			err := conn.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
			if err != nil {
				p.logger.Debug("Ping failed, closing connection",
					zap.String("id", id),
					zap.Error(err))
				
				// Close the connection
				conn.mu.Lock()
				conn.State = ConnectionStateClosed
				conn.Stats.Errors++
				conn.Stats.LastError = err
				conn.Stats.LastErrorTime = now
				conn.mu.Unlock()
				
				conn.Conn.Close()
				
				// Remove the connection from the pool
				delete(p.connections, id)
				
				// Remove the connection from the idle list
				for i, idleConn := range p.idleConnections {
					if idleConn.ID == id {
						p.idleConnections = append(p.idleConnections[:i], p.idleConnections[i+1:]...)
						break
					}
				}
			}
		}
	}
	
	// Ensure we have the minimum number of connections
	if len(p.connections) < p.config.MinConnections {
		needed := p.config.MinConnections - len(p.connections)
		p.logger.Debug("Creating additional connections to meet minimum",
			zap.Int("current", len(p.connections)),
			zap.Int("minimum", p.config.MinConnections),
			zap.Int("needed", needed))
		
		for i := 0; i < needed; i++ {
			conn, err := p.createConnection()
			if err != nil {
				p.logger.Error("Failed to create connection during health check",
					zap.Error(err),
					zap.Int("attempt", i+1))
				continue
			}
			
			// Add the connection to the pool
			p.addConnectionLocked(conn)
		}
	}
}

// Close closes the connection pool
func (p *ConnectionPool) Close() {
	// Stop the health check goroutine
	close(p.stopCh)
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Close all connections
	for _, conn := range p.connections {
		conn.mu.Lock()
		conn.State = ConnectionStateClosed
		conn.mu.Unlock()
		
		conn.Conn.Close()
	}
	
	// Clear the connections
	p.connections = make(map[string]*PooledConnection)
	p.idleConnections = p.idleConnections[:0]
}

// GetStats gets statistics for the connection pool
func (p *ConnectionPool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return map[string]interface{}{
		"total_connections":  len(p.connections),
		"idle_connections":   len(p.idleConnections),
		"active_connections": len(p.connections) - len(p.idleConnections),
		"min_connections":    p.config.MinConnections,
		"max_connections":    p.config.MaxConnections,
	}
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return "conn-" + time.Now().Format("20060102-150405-999999999")
}

