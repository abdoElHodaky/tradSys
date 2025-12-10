package websocket

import (
	"context"
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrConnectionNotFound       = errors.New("connection not found")
	ErrConnectionSendBufferFull = errors.New("connection send buffer full")
	ErrMaxConnectionsReached    = errors.New("maximum connections reached")
	ErrInvalidMessage           = errors.New("invalid message")
	ErrSubscriptionNotFound     = errors.New("subscription not found")
	ErrRateLimitExceeded        = errors.New("rate limit exceeded")
)

// NewGateway creates a new WebSocket gateway
func NewGateway(config *GatewayConfig, logger *zap.Logger) *Gateway {
	ctx, cancel := context.WithCancel(context.Background())

	gateway := &Gateway{
		config:      config,
		logger:      logger,
		connections: make(map[string]*Connection),
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &GatewayMetrics{LastUpdated: time.Now()},
	}

	// Initialize components
	gateway.connectionManager = NewConnectionManager(gateway, logger)
	gateway.messageHandler = NewMessageHandler(gateway, logger)
	gateway.performanceOpt = NewPerformanceOptimizer(gateway, logger)

	return gateway
}

// Start starts the WebSocket gateway
func (g *Gateway) Start() error {
	g.logger.Info("Starting WebSocket gateway",
		zap.Int("max_connections", g.config.MaxConnections),
		zap.Duration("ping_interval", g.config.PingInterval))

	// Start background processes
	go g.metricsCollector()
	go g.connectionCleaner()
	go g.performanceMonitor()

	return nil
}

// Stop stops the WebSocket gateway
func (g *Gateway) Stop() error {
	g.logger.Info("Stopping WebSocket gateway")

	g.cancel()

	// Close all connections
	g.mu.Lock()
	for _, conn := range g.connections {
		conn.Close()
	}
	g.mu.Unlock()

	return nil
}

// HandleConnection handles a new WebSocket connection
func (g *Gateway) HandleConnection(conn *websocket.Conn, userID, exchange string) (*Connection, error) {
	// Check connection limits
	if err := g.checkConnectionLimits(); err != nil {
		return nil, err
	}

	// Create connection
	connection := g.createConnection(conn, userID, exchange)

	// Register connection
	g.mu.Lock()
	g.connections[connection.ID] = connection
	g.mu.Unlock()

	// Start connection handlers
	go g.handleConnectionRead(connection)
	go g.handleConnectionWrite(connection)
	go g.handleConnectionPing(connection)

	g.logger.Info("WebSocket connection established",
		zap.String("connection_id", connection.ID),
		zap.String("user_id", userID),
		zap.String("exchange", exchange))

	return connection, nil
}

// Subscribe adds a subscription to a connection
func (g *Gateway) Subscribe(connectionID, channel, symbol string, subType SubscriptionType, filters map[string]interface{}) error {
	g.mu.RLock()
	conn, exists := g.connections[connectionID]
	g.mu.RUnlock()

	if !exists {
		return ErrConnectionNotFound
	}

	return conn.Subscribe(channel, symbol, subType, filters)
}

// Unsubscribe removes a subscription from a connection
func (g *Gateway) Unsubscribe(connectionID, subscriptionID string) error {
	g.mu.RLock()
	conn, exists := g.connections[connectionID]
	g.mu.RUnlock()

	if !exists {
		return ErrConnectionNotFound
	}

	return conn.Unsubscribe(subscriptionID)
}

// Broadcast sends a message to all connections subscribed to a channel
func (g *Gateway) Broadcast(channel string, data interface{}) error {
	message, err := g.messageHandler.CreateMessage(MessageTypeMarketData, channel, data)
	if err != nil {
		return err
	}

	messageBytes, err := g.messageHandler.SerializeMessage(message)
	if err != nil {
		return err
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, conn := range g.connections {
		if conn.IsSubscribedToChannel(channel) {
			select {
			case conn.send <- messageBytes:
			default:
				g.logger.Warn("Connection send buffer full",
					zap.String("connection_id", conn.ID))
			}
		}
	}

	return nil
}

// BroadcastWithOptions sends a message with advanced options
func (g *Gateway) BroadcastWithOptions(options *BroadcastOptions, data interface{}) error {
	message, err := g.messageHandler.CreateMessage(MessageTypeMarketData, options.Channel, data)
	if err != nil {
		return err
	}

	// Add metadata if provided
	if options.Metadata != nil {
		message.Metadata = options.Metadata
	}

	messageBytes, err := g.messageHandler.SerializeMessage(message)
	if err != nil {
		return err
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, conn := range g.connections {
		// Apply user filter if provided
		if options.UserFilter != nil && !options.UserFilter(conn.UserID) {
			continue
		}

		// Check channel subscription
		if conn.IsSubscribedToChannel(options.Channel) {
			// Check symbol filter if provided
			if options.Symbol != "" && !conn.IsSubscribedToSymbol(options.Symbol) {
				continue
			}

			select {
			case conn.send <- messageBytes:
			default:
				g.logger.Warn("Connection send buffer full",
					zap.String("connection_id", conn.ID))
			}
		}
	}

	return nil
}

// SendToConnection sends a message to a specific connection
func (g *Gateway) SendToConnection(connectionID string, messageType MessageType, data interface{}) error {
	g.mu.RLock()
	conn, exists := g.connections[connectionID]
	g.mu.RUnlock()

	if !exists {
		return ErrConnectionNotFound
	}

	message, err := g.messageHandler.CreateMessage(messageType, "", data)
	if err != nil {
		return err
	}

	messageBytes, err := g.messageHandler.SerializeMessage(message)
	if err != nil {
		return err
	}

	select {
	case conn.send <- messageBytes:
		return nil
	default:
		return ErrConnectionSendBufferFull
	}
}

// SendToUser sends a message to all connections for a specific user
func (g *Gateway) SendToUser(userID string, messageType MessageType, data interface{}) error {
	message, err := g.messageHandler.CreateMessage(messageType, "", data)
	if err != nil {
		return err
	}

	messageBytes, err := g.messageHandler.SerializeMessage(message)
	if err != nil {
		return err
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	sent := 0
	for _, conn := range g.connections {
		if conn.UserID == userID {
			select {
			case conn.send <- messageBytes:
				sent++
			default:
				g.logger.Warn("Connection send buffer full",
					zap.String("connection_id", conn.ID),
					zap.String("user_id", userID))
			}
		}
	}

	if sent == 0 {
		return ErrConnectionNotFound
	}

	return nil
}

// GetMetrics returns gateway metrics
func (g *Gateway) GetMetrics() *GatewayMetrics {
	g.mu.RLock()
	defer g.mu.RUnlock()

	metrics := &GatewayMetrics{
		TotalConnections:  g.metrics.TotalConnections,
		ActiveConnections: int64(len(g.connections)),
		LastUpdated:       time.Now(),
	}

	// Calculate aggregated metrics
	var totalMessages, totalBytes, totalLatency int64
	var subscriptionCount int64

	for _, conn := range g.connections {
		conn.mu.RLock()
		totalMessages += conn.messageCount
		totalBytes += conn.bytesTransferred
		totalLatency += conn.latencySum
		subscriptionCount += int64(len(conn.subscriptions))
		conn.mu.RUnlock()
	}

	if totalMessages > 0 {
		metrics.AverageLatency = time.Duration(totalLatency / totalMessages)
	}

	metrics.SubscriptionCount = subscriptionCount

	return metrics
}

// GetConnectionStats returns statistics for all connections
func (g *Gateway) GetConnectionStats() []ConnectionStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := make([]ConnectionStats, 0, len(g.connections))

	for _, conn := range g.connections {
		conn.mu.RLock()
		stat := ConnectionStats{
			ID:                conn.ID,
			UserID:            conn.UserID,
			Exchange:          conn.Exchange,
			LastActivity:      conn.lastActivity,
			MessageCount:      conn.messageCount,
			BytesTransferred:  conn.bytesTransferred,
			SubscriptionCount: len(conn.subscriptions),
			IsActive:          conn.isActive,
		}

		if conn.messageCount > 0 {
			stat.AverageLatency = time.Duration(conn.latencySum / conn.messageCount)
		}

		conn.mu.RUnlock()
		stats = append(stats, stat)
	}

	return stats
}

// GetConnection returns a connection by ID
func (g *Gateway) GetConnection(connectionID string) (*Connection, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	conn, exists := g.connections[connectionID]
	if !exists {
		return nil, ErrConnectionNotFound
	}

	return conn, nil
}

// CloseConnection closes a specific connection
func (g *Gateway) CloseConnection(connectionID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	conn, exists := g.connections[connectionID]
	if !exists {
		return ErrConnectionNotFound
	}

	conn.Close()
	delete(g.connections, connectionID)

	g.logger.Info("Connection closed",
		zap.String("connection_id", connectionID),
		zap.String("user_id", conn.UserID))

	return nil
}

// GetHealthStatus returns the health status of the gateway
func (g *Gateway) GetHealthStatus() *HealthStatus {
	metrics := g.GetMetrics()

	status := "healthy"
	if metrics.ActiveConnections > int64(g.config.MaxConnections)*8/10 {
		status = "warning"
	}
	if metrics.ActiveConnections >= int64(g.config.MaxConnections) {
		status = "critical"
	}

	return &HealthStatus{
		Status:            status,
		ActiveConnections: metrics.ActiveConnections,
		TotalConnections:  metrics.TotalConnections,
		MessagesPerSecond: metrics.MessagesPerSecond,
		AverageLatency:    metrics.AverageLatency,
		ErrorRate:         metrics.ErrorRate,
		LastCheck:         time.Now(),
		Components: map[string]string{
			"connection_manager":    "healthy",
			"message_handler":       "healthy",
			"performance_optimizer": "healthy",
		},
	}
}
