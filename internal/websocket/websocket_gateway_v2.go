package websocket

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Use MessageType from websocket_gateway.go to avoid duplication

// Gateway manages WebSocket connections and routing for high-performance trading
type Gateway struct {
	// Core components
	connectionManager *ConnectionManager
	messageHandler    *MessageHandler
	performanceOpt    *PerformanceOptimizer
	
	// Configuration
	config *GatewayConfig
	logger *zap.Logger
	
	// Connection tracking
	connections map[string]*V2Connection
	mu          sync.RWMutex
	
	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	
	// Performance metrics
	metrics *GatewayMetrics
}

// GatewayConfig contains gateway configuration
type GatewayConfig struct {
	MaxConnections     int           `json:"max_connections"`
	MaxMessageSize     int64         `json:"max_message_size"`
	WriteTimeout       time.Duration `json:"write_timeout"`
	ReadTimeout        time.Duration `json:"read_timeout"`
	PingInterval       time.Duration `json:"ping_interval"`
	PongTimeout        time.Duration `json:"pong_timeout"`
	EnableCompression  bool          `json:"enable_compression"`
	BufferSize         int           `json:"buffer_size"`
	MaxSubscriptions   int           `json:"max_subscriptions"`
	RateLimitPerSecond int           `json:"rate_limit_per_second"`
}

// V2Connection represents a WebSocket connection optimized for trading
type V2Connection struct {
	ID       string
	UserID   string
	Exchange string
	
	// WebSocket connection
	conn *websocket.Conn
	
	// Message channels
	send    chan []byte
	receive chan []byte
	
	// Subscriptions
	subscriptions map[string]*V2Subscription
	subMu         sync.RWMutex
	
	// Performance tracking
	lastActivity     time.Time
	messageCount     int64
	bytesTransferred int64
	latencySum       int64
	
	// Connection state
	isActive bool
	mu       sync.RWMutex
	
	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// V2Subscription represents a channel subscription
type V2Subscription struct {
	ID       string
	Channel  string
	Symbol   string
	Type     SubscriptionType
	Filters  map[string]interface{}
	Created  time.Time
	LastData time.Time
}

// SubscriptionType defines subscription types
type SubscriptionType string

const (
	SubTypeMarketData    SubscriptionType = "market_data"
	SubTypeOrderBook     SubscriptionType = "order_book"
	SubTypeTrades        SubscriptionType = "trades"
	SubTypeOrderUpdates  SubscriptionType = "order_updates"
	SubTypePortfolio     SubscriptionType = "portfolio"
	SubTypeAlerts        SubscriptionType = "alerts"
)

// GatewayMetrics tracks gateway performance
type GatewayMetrics struct {
	TotalConnections    int64         `json:"total_connections"`
	ActiveConnections   int64         `json:"active_connections"`
	MessagesPerSecond   float64       `json:"messages_per_second"`
	AverageLatency      time.Duration `json:"average_latency"`
	BytesPerSecond      float64       `json:"bytes_per_second"`
	ErrorRate           float64       `json:"error_rate"`
	SubscriptionCount   int64         `json:"subscription_count"`
	LastUpdated         time.Time     `json:"last_updated"`
}

// NewGateway creates a new WebSocket gateway
func NewGateway(config *GatewayConfig, logger *zap.Logger) *Gateway {
	ctx, cancel := context.WithCancel(context.Background())
	
	gateway := &Gateway{
		config:      config,
		logger:      logger,
		connections: make(map[string]*V2Connection),
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
func (g *Gateway) HandleConnection(conn *websocket.Conn, userID, exchange string) (*V2Connection, error) {
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

// createConnection creates a new connection instance
func (g *Gateway) createConnection(conn *websocket.Conn, userID, exchange string) *V2Connection {
	ctx, cancel := context.WithCancel(g.ctx)
	
	connection := &V2Connection{
		ID:            generateV2ConnectionID(),
		UserID:        userID,
		Exchange:      exchange,
		conn:          conn,
		send:          make(chan []byte, g.config.BufferSize),
		receive:       make(chan []byte, g.config.BufferSize),
		subscriptions: make(map[string]*V2Subscription),
		lastActivity:  time.Now(),
		isActive:      true,
		ctx:           ctx,
		cancel:        cancel,
	}
	
	return connection
}

// checkConnectionLimits checks if new connections are allowed
func (g *Gateway) checkConnectionLimits() error {
	g.mu.RLock()
	currentConnections := len(g.connections)
	g.mu.RUnlock()
	
	if currentConnections >= g.config.MaxConnections {
		return ErrMaxConnectionsReached
	}
	
	return nil
}

// handleConnectionRead handles reading messages from a connection
func (g *Gateway) handleConnectionRead(conn *V2Connection) {
	defer func() {
		conn.Close()
		g.removeConnection(conn.ID)
	}()
	
	conn.conn.SetReadLimit(g.config.MaxMessageSize)
	conn.conn.SetReadDeadline(time.Now().Add(g.config.ReadTimeout))
	conn.conn.SetPongHandler(func(string) error {
		conn.conn.SetReadDeadline(time.Now().Add(g.config.ReadTimeout))
		return nil
	})
	
	for {
		select {
		case <-conn.ctx.Done():
			return
		default:
			_, message, err := conn.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					g.logger.Error("WebSocket read error",
						zap.String("connection_id", conn.ID),
						zap.Error(err))
				}
				return
			}
			
			// Update connection metrics
			conn.mu.Lock()
			conn.lastActivity = time.Now()
			conn.messageCount++
			conn.bytesTransferred += int64(len(message))
			conn.mu.Unlock()
			
			// Process message
			if err := g.messageHandler.ProcessMessage(conn, message); err != nil {
				g.logger.Error("Failed to process message",
					zap.String("connection_id", conn.ID),
					zap.Error(err))
			}
		}
	}
}

// handleConnectionWrite handles writing messages to a connection
func (g *Gateway) handleConnectionWrite(conn *V2Connection) {
	ticker := time.NewTicker(g.config.PingInterval)
	defer func() {
		ticker.Stop()
		conn.conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-conn.send:
			conn.conn.SetWriteDeadline(time.Now().Add(g.config.WriteTimeout))
			if !ok {
				conn.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			if err := conn.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				g.logger.Error("WebSocket write error",
					zap.String("connection_id", conn.ID),
					zap.Error(err))
				return
			}
			
			// Update metrics
			conn.mu.Lock()
			conn.bytesTransferred += int64(len(message))
			conn.mu.Unlock()
			
		case <-ticker.C:
			conn.conn.SetWriteDeadline(time.Now().Add(g.config.WriteTimeout))
			if err := conn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			
		case <-conn.ctx.Done():
			return
		}
	}
}

// handleConnectionPing handles ping/pong for connection health
func (g *Gateway) handleConnectionPing(conn *V2Connection) {
	ticker := time.NewTicker(g.config.PingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if time.Since(conn.lastActivity) > g.config.PongTimeout {
				g.logger.Warn("Connection ping timeout",
					zap.String("connection_id", conn.ID))
				conn.Close()
				return
			}
			
		case <-conn.ctx.Done():
			return
		}
	}
}

// removeConnection removes a connection from the gateway
func (g *Gateway) removeConnection(connectionID string) {
	g.mu.Lock()
	delete(g.connections, connectionID)
	g.mu.Unlock()
	
	g.logger.Debug("Connection removed",
		zap.String("connection_id", connectionID))
}

// metricsCollector collects and updates gateway metrics
func (g *Gateway) metricsCollector() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			g.updateMetrics()
		case <-g.ctx.Done():
			return
		}
	}
}

// connectionCleaner cleans up inactive connections
func (g *Gateway) connectionCleaner() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			g.cleanInactiveConnections()
		case <-g.ctx.Done():
			return
		}
	}
}

// performanceMonitor monitors gateway performance
func (g *Gateway) performanceMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			g.performanceOpt.OptimizePerformance()
		case <-g.ctx.Done():
			return
		}
	}
}

// updateMetrics updates gateway metrics
func (g *Gateway) updateMetrics() {
	g.mu.RLock()
	activeConnections := int64(len(g.connections))
	g.mu.RUnlock()
	
	g.metrics.ActiveConnections = activeConnections
	g.metrics.LastUpdated = time.Now()
}

// cleanInactiveConnections removes inactive connections
func (g *Gateway) cleanInactiveConnections() {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	cutoff := time.Now().Add(-5 * time.Minute)
	
	for id, conn := range g.connections {
		conn.mu.RLock()
		lastActivity := conn.lastActivity
		conn.mu.RUnlock()
		
		if lastActivity.Before(cutoff) {
			conn.Close()
			delete(g.connections, id)
			g.logger.Info("Cleaned inactive connection",
				zap.String("connection_id", id))
		}
	}
}

// Connection methods

// Subscribe adds a subscription to the connection
func (c *V2Connection) Subscribe(channel, symbol string, subType SubscriptionType, filters map[string]interface{}) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	
	if len(c.subscriptions) >= 100 { // Max subscriptions per connection
		return ErrMaxSubscriptionsReached
	}
	
	subscriptionID := generateV2SubscriptionID()
	subscription := &V2Subscription{
		ID:      subscriptionID,
		Channel: channel,
		Symbol:  symbol,
		Type:    subType,
		Filters: filters,
		Created: time.Now(),
	}
	
	c.subscriptions[subscriptionID] = subscription
	
	return nil
}

// Unsubscribe removes a subscription from the connection
func (c *V2Connection) Unsubscribe(subscriptionID string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	
	delete(c.subscriptions, subscriptionID)
	return nil
}

// IsSubscribedToChannel checks if connection is subscribed to a channel
func (c *V2Connection) IsSubscribedToChannel(channel string) bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	
	for _, sub := range c.subscriptions {
		if sub.Channel == channel {
			return true
		}
	}
	
	return false
}

// Close closes the connection
func (c *V2Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.isActive {
		return
	}
	
	c.isActive = false
	c.cancel()
	close(c.send)
	c.conn.Close()
}

// Helper functions
func generateV2ConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

func generateV2SubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

// Error definitions
var (
	ErrMaxConnectionsReached     = errors.New("maximum connections reached")
	ErrMaxSubscriptionsReached   = errors.New("maximum subscriptions reached")
	ErrConnectionSendBufferFull  = errors.New("connection send buffer full")
)
