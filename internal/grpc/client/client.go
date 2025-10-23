package client

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// ConnectionPool provides a pool of gRPC connections
type ConnectionPool struct {
	target      string
	maxSize     int
	connections []*grpc.ClientConn
	index       int
	mu          sync.Mutex
	logger      *zap.Logger
	dialOptions []grpc.DialOption
}

// ConnectionPoolOptions contains options for the connection pool
type ConnectionPoolOptions struct {
	MaxSize           int
	DialTimeout       time.Duration
	KeepAliveTime     time.Duration
	KeepAliveTimeout  time.Duration
	MaxBackoffDelay   time.Duration
	BackoffMultiplier float64
	MinConnectTimeout time.Duration
}

// DefaultConnectionPoolOptions returns default connection pool options
func DefaultConnectionPoolOptions() ConnectionPoolOptions {
	return ConnectionPoolOptions{
		MaxSize:           10,
		DialTimeout:       5 * time.Second,
		KeepAliveTime:     30 * time.Second,
		KeepAliveTimeout:  10 * time.Second,
		MaxBackoffDelay:   10 * time.Second,
		BackoffMultiplier: 1.5,
		MinConnectTimeout: 1 * time.Second,
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(target string, logger *zap.Logger, options ConnectionPoolOptions) (*ConnectionPool, error) {
	// Create dial options
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                options.KeepAliveTime,
			Timeout:             options.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				Multiplier: options.BackoffMultiplier,
				Jitter:     0.2,
				MaxDelay:   options.MaxBackoffDelay,
			},
			MinConnectTimeout: options.MinConnectTimeout,
		}),
		// Enable wait for ready semantics
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		// Increase max receive and send message sizes
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(50*1024*1024),
			grpc.MaxCallSendMsgSize(50*1024*1024),
		),
	}

	pool := &ConnectionPool{
		target:      target,
		maxSize:     options.MaxSize,
		connections: make([]*grpc.ClientConn, 0, options.MaxSize),
		logger:      logger,
		dialOptions: dialOptions,
	}

	// Initialize the pool with connections
	for i := 0; i < options.MaxSize; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), options.DialTimeout)
		defer cancel()

		conn, err := grpc.DialContext(ctx, target, dialOptions...)
		if err != nil {
			pool.Close()
			return nil, err
		}

		pool.connections = append(pool.connections, conn)
	}

	// Start a goroutine to monitor connections
	go pool.monitorConnections()

	return pool, nil
}

// Get gets a connection from the pool
func (p *ConnectionPool) Get() (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.connections) == 0 {
		return nil, ErrNoConnections
	}

	// Get the next connection
	conn := p.connections[p.index%len(p.connections)]
	p.index++

	// Check if the connection is ready
	if conn.GetState() != connectivity.Ready {
		// Try to find a ready connection
		for i := 0; i < len(p.connections); i++ {
			if p.connections[i].GetState() == connectivity.Ready {
				conn = p.connections[i]
				break
			}
		}
	}

	return conn, nil
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.Close()
	}

	p.connections = nil
}

// monitorConnections monitors connections and reconnects if necessary
func (p *ConnectionPool) monitorConnections() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		for i, conn := range p.connections {
			state := conn.GetState()
			if state == connectivity.TransientFailure || state == connectivity.Shutdown {
				p.logger.Warn("Connection in bad state, reconnecting",
					zap.String("target", p.target),
					zap.Int("index", i),
					zap.String("state", state.String()))

				// Close the old connection
				conn.Close()

				// Create a new connection
				newConn, err := grpc.Dial(p.target, p.dialOptions...)
				if err != nil {
					p.logger.Error("Failed to reconnect",
						zap.String("target", p.target),
						zap.Int("index", i),
						zap.Error(err))
					continue
				}

				// Replace the old connection
				p.connections[i] = newConn
			}
		}
		p.mu.Unlock()
	}
}

// ErrNoConnections is returned when there are no connections in the pool
var ErrNoConnections = &PoolError{Message: "no connections available in the pool"}

// PoolError represents an error from the connection pool
type PoolError struct {
	Message string
}

// Error returns the error message
func (e *PoolError) Error() string {
	return e.Message
}
