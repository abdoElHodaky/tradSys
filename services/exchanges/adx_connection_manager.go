package exchanges

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ADXConnectionManager handles connection management for ADX exchange
type ADXConnectionManager struct {
	config           *ADXConfig
	logger           *zap.Logger
	connections      map[string]*ADXConnection
	connectionPool   *ConnectionPool
	healthChecker    *ConnectionHealthChecker
	reconnectManager *ReconnectManager
	mu               sync.RWMutex
	
	// Connection metrics
	totalConnections    int64
	activeConnections   int64
	failedConnections   int64
	reconnectAttempts   int64
}

// ADXConnection represents a single connection to ADX
type ADXConnection struct {
	ID              string
	Type            ConnectionType
	Status          ConnectionStatus
	Endpoint        string
	LastHeartbeat   time.Time
	ConnectedAt     time.Time
	ReconnectCount  int
	ErrorCount      int64
	BytesSent       int64
	BytesReceived   int64
	mu              sync.RWMutex
}

// ConnectionType defines the type of ADX connection
type ConnectionType int

const (
	ConnectionTypeMarketData ConnectionType = iota
	ConnectionTypeTrading
	ConnectionTypeReference
	ConnectionTypeCompliance
)

// String returns the string representation of connection type
func (ct ConnectionType) String() string {
	switch ct {
	case ConnectionTypeMarketData:
		return "market_data"
	case ConnectionTypeTrading:
		return "trading"
	case ConnectionTypeReference:
		return "reference"
	case ConnectionTypeCompliance:
		return "compliance"
	default:
		return "unknown"
	}
}

// ConnectionStatus defines the status of a connection
type ConnectionStatus int

const (
	ConnectionStatusDisconnected ConnectionStatus = iota
	ConnectionStatusConnecting
	ConnectionStatusConnected
	ConnectionStatusReconnecting
	ConnectionStatusError
)

// String returns the string representation of connection status
func (cs ConnectionStatus) String() string {
	switch cs {
	case ConnectionStatusDisconnected:
		return "disconnected"
	case ConnectionStatusConnecting:
		return "connecting"
	case ConnectionStatusConnected:
		return "connected"
	case ConnectionStatusReconnecting:
		return "reconnecting"
	case ConnectionStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// ConnectionPool manages a pool of ADX connections
type ConnectionPool struct {
	maxConnections int
	connections    chan *ADXConnection
	factory        func() (*ADXConnection, error)
	mu             sync.Mutex
}

// ConnectionHealthChecker monitors connection health
type ConnectionHealthChecker struct {
	interval        time.Duration
	timeout         time.Duration
	maxFailures     int
	healthChecks    map[string]*HealthCheck
	mu              sync.RWMutex
}

// HealthCheck represents a connection health check
type HealthCheck struct {
	ConnectionID    string
	LastCheck       time.Time
	ConsecutiveFails int
	IsHealthy       bool
}

// ReconnectManager handles automatic reconnection logic
type ReconnectManager struct {
	maxRetries      int
	baseDelay       time.Duration
	maxDelay        time.Duration
	backoffFactor   float64
	reconnectQueue  chan *ReconnectRequest
	mu              sync.RWMutex
}

// ReconnectRequest represents a reconnection request
type ReconnectRequest struct {
	ConnectionID string
	Attempt      int
	LastAttempt  time.Time
	Reason       string
}

// NewADXConnectionManager creates a new connection manager
func NewADXConnectionManager(config *ADXConfig, logger *zap.Logger) *ADXConnectionManager {
	return &ADXConnectionManager{
		config:           config,
		logger:           logger,
		connections:      make(map[string]*ADXConnection),
		connectionPool:   NewConnectionPool(config.MaxConnections),
		healthChecker:    NewConnectionHealthChecker(30*time.Second, 5*time.Second, 3),
		reconnectManager: NewReconnectManager(5, time.Second, 30*time.Second, 2.0),
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxConnections int) *ConnectionPool {
	return &ConnectionPool{
		maxConnections: maxConnections,
		connections:    make(chan *ADXConnection, maxConnections),
	}
}

// NewConnectionHealthChecker creates a new health checker
func NewConnectionHealthChecker(interval, timeout time.Duration, maxFailures int) *ConnectionHealthChecker {
	return &ConnectionHealthChecker{
		interval:     interval,
		timeout:      timeout,
		maxFailures:  maxFailures,
		healthChecks: make(map[string]*HealthCheck),
	}
}

// NewReconnectManager creates a new reconnect manager
func NewReconnectManager(maxRetries int, baseDelay, maxDelay time.Duration, backoffFactor float64) *ReconnectManager {
	return &ReconnectManager{
		maxRetries:     maxRetries,
		baseDelay:      baseDelay,
		maxDelay:       maxDelay,
		backoffFactor:  backoffFactor,
		reconnectQueue: make(chan *ReconnectRequest, 100),
	}
}

// Connect establishes a connection to ADX
func (acm *ADXConnectionManager) Connect(ctx context.Context, connType ConnectionType) (*ADXConnection, error) {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	connectionID := fmt.Sprintf("adx_%s_%d", connType.String(), time.Now().UnixNano())
	
	connection := &ADXConnection{
		ID:            connectionID,
		Type:          connType,
		Status:        ConnectionStatusConnecting,
		Endpoint:      acm.getEndpointForType(connType),
		ConnectedAt:   time.Now(),
	}

	// Simulate connection establishment
	if err := acm.establishConnection(ctx, connection); err != nil {
		connection.Status = ConnectionStatusError
		acm.failedConnections++
		return nil, fmt.Errorf("failed to establish connection: %w", err)
	}

	connection.Status = ConnectionStatusConnected
	connection.LastHeartbeat = time.Now()
	
	acm.connections[connectionID] = connection
	acm.activeConnections++
	acm.totalConnections++

	// Start health monitoring
	go acm.monitorConnection(ctx, connection)

	acm.logger.Info("ADX connection established",
		zap.String("connectionID", connectionID),
		zap.String("type", connType.String()),
		zap.String("endpoint", connection.Endpoint),
	)

	return connection, nil
}

// Disconnect closes a connection
func (acm *ADXConnectionManager) Disconnect(ctx context.Context, connectionID string) error {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	connection, exists := acm.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	connection.mu.Lock()
	connection.Status = ConnectionStatusDisconnected
	connection.mu.Unlock()

	delete(acm.connections, connectionID)
	acm.activeConnections--

	acm.logger.Info("ADX connection disconnected",
		zap.String("connectionID", connectionID),
		zap.String("type", connection.Type.String()),
	)

	return nil
}

// GetConnection retrieves a connection by ID
func (acm *ADXConnectionManager) GetConnection(connectionID string) (*ADXConnection, error) {
	acm.mu.RLock()
	defer acm.mu.RUnlock()

	connection, exists := acm.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", connectionID)
	}

	return connection, nil
}

// GetConnectionsByType retrieves all connections of a specific type
func (acm *ADXConnectionManager) GetConnectionsByType(connType ConnectionType) []*ADXConnection {
	acm.mu.RLock()
	defer acm.mu.RUnlock()

	var connections []*ADXConnection
	for _, conn := range acm.connections {
		if conn.Type == connType {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetHealthyConnections returns all healthy connections
func (acm *ADXConnectionManager) GetHealthyConnections() []*ADXConnection {
	acm.mu.RLock()
	defer acm.mu.RUnlock()

	var healthyConnections []*ADXConnection
	for _, conn := range acm.connections {
		if conn.Status == ConnectionStatusConnected {
			healthyConnections = append(healthyConnections, conn)
		}
	}

	return healthyConnections
}

// establishConnection simulates establishing a connection
func (acm *ADXConnectionManager) establishConnection(ctx context.Context, connection *ADXConnection) error {
	// Simulate connection establishment delay
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// getEndpointForType returns the endpoint URL for a connection type
func (acm *ADXConnectionManager) getEndpointForType(connType ConnectionType) string {
	switch connType {
	case ConnectionTypeMarketData:
		return "wss://adx.ae/marketdata"
	case ConnectionTypeTrading:
		return "wss://adx.ae/trading"
	case ConnectionTypeReference:
		return "https://adx.ae/reference"
	case ConnectionTypeCompliance:
		return "https://adx.ae/compliance"
	default:
		return "https://adx.ae/default"
	}
}

// monitorConnection monitors a connection's health
func (acm *ADXConnectionManager) monitorConnection(ctx context.Context, connection *ADXConnection) {
	ticker := time.NewTicker(acm.healthChecker.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := acm.performHealthCheck(ctx, connection); err != nil {
				acm.logger.Warn("Connection health check failed",
					zap.String("connectionID", connection.ID),
					zap.Error(err),
				)
				
				// Trigger reconnection if needed
				acm.scheduleReconnect(connection, "health_check_failed")
			}
		}
	}
}

// performHealthCheck performs a health check on a connection
func (acm *ADXConnectionManager) performHealthCheck(ctx context.Context, connection *ADXConnection) error {
	connection.mu.Lock()
	defer connection.mu.Unlock()

	// Simulate health check
	connection.LastHeartbeat = time.Now()
	
	// Check if connection is stale
	if time.Since(connection.LastHeartbeat) > 2*acm.healthChecker.interval {
		connection.ErrorCount++
		return fmt.Errorf("connection stale")
	}

	return nil
}

// scheduleReconnect schedules a reconnection attempt
func (acm *ADXConnectionManager) scheduleReconnect(connection *ADXConnection, reason string) {
	request := &ReconnectRequest{
		ConnectionID: connection.ID,
		Attempt:      connection.ReconnectCount + 1,
		LastAttempt:  time.Now(),
		Reason:       reason,
	}

	select {
	case acm.reconnectManager.reconnectQueue <- request:
		connection.ReconnectCount++
	default:
		acm.logger.Warn("Reconnect queue full, dropping request",
			zap.String("connectionID", connection.ID),
		)
	}
}

// StartReconnectWorker starts the reconnection worker
func (acm *ADXConnectionManager) StartReconnectWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case request := <-acm.reconnectManager.reconnectQueue:
				acm.handleReconnectRequest(ctx, request)
			}
		}
	}()
}

// handleReconnectRequest handles a reconnection request
func (acm *ADXConnectionManager) handleReconnectRequest(ctx context.Context, request *ReconnectRequest) {
	if request.Attempt > acm.reconnectManager.maxRetries {
		acm.logger.Error("Max reconnect attempts exceeded",
			zap.String("connectionID", request.ConnectionID),
			zap.Int("attempts", request.Attempt),
		)
		return
	}

	// Calculate backoff delay
	delay := time.Duration(float64(acm.reconnectManager.baseDelay) * 
		float64(request.Attempt) * acm.reconnectManager.backoffFactor)
	if delay > acm.reconnectManager.maxDelay {
		delay = acm.reconnectManager.maxDelay
	}

	// Wait before reconnecting
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return
	}

	// Attempt reconnection
	connection, exists := acm.connections[request.ConnectionID]
	if !exists {
		return
	}

	acm.logger.Info("Attempting reconnection",
		zap.String("connectionID", request.ConnectionID),
		zap.Int("attempt", request.Attempt),
		zap.String("reason", request.Reason),
	)

	connection.Status = ConnectionStatusReconnecting
	if err := acm.establishConnection(ctx, connection); err != nil {
		acm.logger.Error("Reconnection failed",
			zap.String("connectionID", request.ConnectionID),
			zap.Error(err),
		)
		
		// Schedule another attempt
		acm.scheduleReconnect(connection, "reconnect_failed")
		return
	}

	connection.Status = ConnectionStatusConnected
	connection.LastHeartbeat = time.Now()
	acm.reconnectAttempts++

	acm.logger.Info("Reconnection successful",
		zap.String("connectionID", request.ConnectionID),
		zap.Int("attempt", request.Attempt),
	)
}

// GetMetrics returns connection manager metrics
func (acm *ADXConnectionManager) GetMetrics() map[string]interface{} {
	acm.mu.RLock()
	defer acm.mu.RUnlock()

	return map[string]interface{}{
		"total_connections":    acm.totalConnections,
		"active_connections":   acm.activeConnections,
		"failed_connections":   acm.failedConnections,
		"reconnect_attempts":   acm.reconnectAttempts,
		"connection_types": map[string]int{
			"market_data": len(acm.GetConnectionsByType(ConnectionTypeMarketData)),
			"trading":     len(acm.GetConnectionsByType(ConnectionTypeTrading)),
			"reference":   len(acm.GetConnectionsByType(ConnectionTypeReference)),
			"compliance":  len(acm.GetConnectionsByType(ConnectionTypeCompliance)),
		},
	}
}

// Close closes all connections and stops the connection manager
func (acm *ADXConnectionManager) Close(ctx context.Context) error {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	acm.logger.Info("Closing ADX connection manager")

	// Close all connections
	for connectionID := range acm.connections {
		if err := acm.Disconnect(ctx, connectionID); err != nil {
			acm.logger.Error("Failed to close connection",
				zap.String("connectionID", connectionID),
				zap.Error(err),
			)
		}
	}

	acm.logger.Info("ADX connection manager closed")
	return nil
}
