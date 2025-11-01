package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// createConnection creates a new connection instance
func (g *Gateway) createConnection(conn *websocket.Conn, userID, exchange string) *Connection {
	ctx, cancel := context.WithCancel(g.ctx)

	connection := &Connection{
		ID:            generateConnectionID(),
		UserID:        userID,
		Exchange:      exchange,
		conn:          conn,
		send:          make(chan []byte, g.config.BufferSize),
		receive:       make(chan []byte, g.config.BufferSize),
		subscriptions: make(map[string]*Subscription),
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
	defer g.mu.RUnlock()

	if len(g.connections) >= g.config.MaxConnections {
		return ErrMaxConnectionsReached
	}

	return nil
}

// handleConnectionRead handles reading messages from a connection
func (g *Gateway) handleConnectionRead(conn *Connection) {
	defer func() {
		conn.Close()
		g.removeConnection(conn.ID)
	}()

	conn.conn.SetReadLimit(g.config.MaxMessageSize)
	conn.conn.SetReadDeadline(time.Now().Add(g.config.ReadTimeout))
	conn.conn.SetPongHandler(func(string) error {
		conn.conn.SetReadDeadline(time.Now().Add(g.config.ReadTimeout))
		conn.mu.Lock()
		conn.lastActivity = time.Now()
		conn.mu.Unlock()
		return nil
	})

	for {
		_, message, err := conn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				g.logger.Error("WebSocket read error",
					zap.String("connection_id", conn.ID),
					zap.Error(err))
			}
			break
		}

		// Update activity
		conn.mu.Lock()
		conn.lastActivity = time.Now()
		conn.messageCount++
		conn.mu.Unlock()

		// Process message
		if err := g.processMessage(conn, message); err != nil {
			g.logger.Error("Message processing error",
				zap.String("connection_id", conn.ID),
				zap.Error(err))
		}
	}
}

// handleConnectionWrite handles writing messages to a connection
func (g *Gateway) handleConnectionWrite(conn *Connection) {
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
func (g *Gateway) handleConnectionPing(conn *Connection) {
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

// processMessage processes incoming WebSocket messages
func (g *Gateway) processMessage(conn *Connection, messageBytes []byte) error {
	var message Message
	if err := json.Unmarshal(messageBytes, &message); err != nil {
		return fmt.Errorf("invalid message format: %w", err)
	}

	switch message.Type {
	case MessageTypeSubscribe:
		return g.handleSubscribeMessage(conn, &message)
	case MessageTypeUnsubscribe:
		return g.handleUnsubscribeMessage(conn, &message)
	case MessageTypeHeartbeat:
		return g.handleHeartbeatMessage(conn, &message)
	default:
		return fmt.Errorf("unknown message type: %s", message.Type)
	}
}

// handleSubscribeMessage handles subscription requests
func (g *Gateway) handleSubscribeMessage(conn *Connection, message *Message) error {
	if message.Channel == "" {
		return ErrInvalidMessage
	}

	subType := SubTypeMarketData
	if message.Metadata != nil {
		if t, ok := message.Metadata["type"].(string); ok {
			subType = SubscriptionType(t)
		}
	}

	filters := make(map[string]interface{})
	if message.Metadata != nil {
		if f, ok := message.Metadata["filters"].(map[string]interface{}); ok {
			filters = f
		}
	}

	return conn.Subscribe(message.Channel, message.Symbol, subType, filters)
}

// handleUnsubscribeMessage handles unsubscription requests
func (g *Gateway) handleUnsubscribeMessage(conn *Connection, message *Message) error {
	if message.ID == "" {
		return ErrInvalidMessage
	}

	return conn.Unsubscribe(message.ID)
}

// handleHeartbeatMessage handles heartbeat messages
func (g *Gateway) handleHeartbeatMessage(conn *Connection, message *Message) error {
	// Send heartbeat response
	response := &Message{
		Type:      MessageTypeHeartbeat,
		Timestamp: time.Now(),
		Data:      "pong",
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return err
	}

	select {
	case conn.send <- responseBytes:
		return nil
	default:
		return ErrConnectionSendBufferFull
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
			g.cleanupInactiveConnections()
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
			g.monitorPerformance()
		case <-g.ctx.Done():
			return
		}
	}
}

// updateMetrics updates gateway metrics
func (g *Gateway) updateMetrics() {
	g.mu.RLock()
	defer g.mu.RUnlock()

	g.metrics.ActiveConnections = int64(len(g.connections))
	g.metrics.LastUpdated = time.Now()

	// Calculate messages per second and other metrics
	var totalMessages, totalBytes int64
	for _, conn := range g.connections {
		conn.mu.RLock()
		totalMessages += conn.messageCount
		totalBytes += conn.bytesTransferred
		conn.mu.RUnlock()
	}

	// Simple rate calculation (would be more sophisticated in production)
	if g.metrics.LastUpdated.Sub(time.Time{}).Seconds() > 0 {
		g.metrics.MessagesPerSecond = float64(totalMessages) / g.metrics.LastUpdated.Sub(time.Time{}).Seconds()
		g.metrics.BytesPerSecond = float64(totalBytes) / g.metrics.LastUpdated.Sub(time.Time{}).Seconds()
	}
}

// cleanupInactiveConnections removes inactive connections
func (g *Gateway) cleanupInactiveConnections() {
	g.mu.Lock()
	defer g.mu.Unlock()

	inactiveThreshold := time.Now().Add(-5 * time.Minute)
	var toRemove []string

	for id, conn := range g.connections {
		conn.mu.RLock()
		if conn.lastActivity.Before(inactiveThreshold) || !conn.isActive {
			toRemove = append(toRemove, id)
		}
		conn.mu.RUnlock()
	}

	for _, id := range toRemove {
		if conn, exists := g.connections[id]; exists {
			conn.Close()
			delete(g.connections, id)
			g.logger.Info("Cleaned up inactive connection",
				zap.String("connection_id", id))
		}
	}
}

// monitorPerformance monitors and logs performance metrics
func (g *Gateway) monitorPerformance() {
	metrics := g.GetMetrics()

	g.logger.Debug("Gateway performance metrics",
		zap.Int64("active_connections", metrics.ActiveConnections),
		zap.Float64("messages_per_second", metrics.MessagesPerSecond),
		zap.Duration("average_latency", metrics.AverageLatency),
		zap.Int64("subscription_count", metrics.SubscriptionCount))

	// Alert on high connection count
	if metrics.ActiveConnections > int64(g.config.MaxConnections)*8/10 {
		g.logger.Warn("High connection count",
			zap.Int64("active_connections", metrics.ActiveConnections),
			zap.Int("max_connections", g.config.MaxConnections))
	}
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Connection methods

// Subscribe adds a subscription to the connection
func (c *Connection) Subscribe(channel, symbol string, subType SubscriptionType, filters map[string]interface{}) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	// Check subscription limits
	if len(c.subscriptions) >= 100 { // Max 100 subscriptions per connection
		return fmt.Errorf("maximum subscriptions reached")
	}

	subscriptionID := generateSubscriptionID(channel, symbol)

	subscription := &Subscription{
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
func (c *Connection) Unsubscribe(subscriptionID string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	if _, exists := c.subscriptions[subscriptionID]; !exists {
		return ErrSubscriptionNotFound
	}

	delete(c.subscriptions, subscriptionID)
	return nil
}

// IsSubscribedToChannel checks if connection is subscribed to a channel
func (c *Connection) IsSubscribedToChannel(channel string) bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	for _, sub := range c.subscriptions {
		if sub.Channel == channel {
			return true
		}
	}

	return false
}

// IsSubscribedToSymbol checks if connection is subscribed to a symbol
func (c *Connection) IsSubscribedToSymbol(symbol string) bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	for _, sub := range c.subscriptions {
		if sub.Symbol == symbol {
			return true
		}
	}

	return false
}

// Close closes the connection
func (c *Connection) Close() {
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

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID(channel, symbol string) string {
	return fmt.Sprintf("%s:%s:%d", channel, symbol, time.Now().UnixNano())
}

// Component constructors

// NewConnectionManager creates a new connection manager
func NewConnectionManager(gateway *Gateway, logger *zap.Logger) *ConnectionManager {
	return &ConnectionManager{
		gateway: gateway,
		logger:  logger,
	}
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(gateway *Gateway, logger *zap.Logger) *MessageHandler {
	return &MessageHandler{
		gateway: gateway,
		logger:  logger,
	}
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(gateway *Gateway, logger *zap.Logger) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		gateway: gateway,
		logger:  logger,
	}
}

// MessageHandler methods

// CreateMessage creates a new message
func (mh *MessageHandler) CreateMessage(msgType MessageType, channel string, data interface{}) (*Message, error) {
	return &Message{
		Type:      msgType,
		Channel:   channel,
		Data:      data,
		Timestamp: time.Now(),
		ID:        generateConnectionID(), // Reuse the ID generator
	}, nil
}

// SerializeMessage serializes a message to bytes
func (mh *MessageHandler) SerializeMessage(message *Message) ([]byte, error) {
	return json.Marshal(message)
}
