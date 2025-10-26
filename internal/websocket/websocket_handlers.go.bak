package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// MessageHandler handles WebSocket message processing
type MessageHandler struct {
	gateway *Gateway
	logger  *zap.Logger
}

// Message represents a WebSocket message
type Message struct {
	Type      MessageType            `json:"type"`
	Channel   string                 `json:"channel,omitempty"`
	Symbol    string                 `json:"symbol,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	MessageID string                 `json:"message_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SubscribeRequest represents a subscription request
type SubscribeRequest struct {
	Channel string                 `json:"channel"`
	Symbol  string                 `json:"symbol,omitempty"`
	Type    SubscriptionType       `json:"type"`
	Filters map[string]interface{} `json:"filters,omitempty"`
}

// UnsubscribeRequest represents an unsubscription request
type UnsubscribeRequest struct {
	SubscriptionID string `json:"subscription_id"`
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(gateway *Gateway, logger *zap.Logger) *MessageHandler {
	return &MessageHandler{
		gateway: gateway,
		logger:  logger,
	}
}

// ProcessMessage processes an incoming WebSocket message
func (h *MessageHandler) ProcessMessage(conn *Connection, messageBytes []byte) error {
	var message Message
	if err := json.Unmarshal(messageBytes, &message); err != nil {
		h.logger.Error("Failed to unmarshal message",
			zap.String("connection_id", conn.ID),
			zap.Error(err))
		return h.sendError(conn, "invalid_message", "Failed to parse message")
	}

	// Update connection activity
	conn.mu.Lock()
	conn.lastActivity = time.Now()
	conn.mu.Unlock()

	// Process message based on type
	switch message.Type {
	case MessageTypeSubscribe:
		return h.handleSubscribe(conn, &message)
	case MessageTypeUnsubscribe:
		return h.handleUnsubscribe(conn, &message)
	case MessageTypeHeartbeat:
		return h.handleHeartbeat(conn, &message)
	default:
		h.logger.Warn("Unknown message type",
			zap.String("connection_id", conn.ID),
			zap.String("message_type", string(message.Type)))
		return h.sendError(conn, "unknown_message_type", "Unknown message type")
	}
}

// CreateMessage creates a new message
func (h *MessageHandler) CreateMessage(messageType MessageType, channel string, data interface{}) (*Message, error) {
	return &Message{
		Type:      messageType,
		Channel:   channel,
		Data:      data,
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}, nil
}

// SerializeMessage serializes a message to JSON bytes
func (h *MessageHandler) SerializeMessage(message *Message) ([]byte, error) {
	return json.Marshal(message)
}

// handleSubscribe handles subscription requests
func (h *MessageHandler) handleSubscribe(conn *Connection, message *Message) error {
	var subReq SubscribeRequest
	
	// Parse subscription request from message data
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return h.sendError(conn, "invalid_subscription", "Invalid subscription data")
	}
	
	if err := json.Unmarshal(dataBytes, &subReq); err != nil {
		return h.sendError(conn, "invalid_subscription", "Failed to parse subscription request")
	}

	// Validate subscription request
	if err := h.validateSubscriptionRequest(&subReq); err != nil {
		return h.sendError(conn, "invalid_subscription", err.Error())
	}

	// Add subscription
	if err := conn.Subscribe(subReq.Channel, subReq.Symbol, subReq.Type, subReq.Filters); err != nil {
		return h.sendError(conn, "subscription_failed", err.Error())
	}

	// Send confirmation
	response := map[string]interface{}{
		"status":  "subscribed",
		"channel": subReq.Channel,
		"symbol":  subReq.Symbol,
		"type":    subReq.Type,
	}

	return h.sendMessage(conn, MessageTypeSubscribe, "", response)
}

// handleUnsubscribe handles unsubscription requests
func (h *MessageHandler) handleUnsubscribe(conn *Connection, message *Message) error {
	var unsubReq UnsubscribeRequest
	
	// Parse unsubscription request from message data
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return h.sendError(conn, "invalid_unsubscription", "Invalid unsubscription data")
	}
	
	if err := json.Unmarshal(dataBytes, &unsubReq); err != nil {
		return h.sendError(conn, "invalid_unsubscription", "Failed to parse unsubscription request")
	}

	// Remove subscription
	if err := conn.Unsubscribe(unsubReq.SubscriptionID); err != nil {
		return h.sendError(conn, "unsubscription_failed", err.Error())
	}

	// Send confirmation
	response := map[string]interface{}{
		"status":          "unsubscribed",
		"subscription_id": unsubReq.SubscriptionID,
	}

	return h.sendMessage(conn, MessageTypeUnsubscribe, "", response)
}

// handleHeartbeat handles heartbeat messages
func (h *MessageHandler) handleHeartbeat(conn *Connection, message *Message) error {
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
	}

	return h.sendMessage(conn, MessageTypeHeartbeat, "", response)
}

// validateSubscriptionRequest validates a subscription request
func (h *MessageHandler) validateSubscriptionRequest(req *SubscribeRequest) error {
	if req.Channel == "" {
		return errors.New("channel is required")
	}

	if req.Type == "" {
		return errors.New("subscription type is required")
	}

	// Validate subscription type
	validTypes := []SubscriptionType{
		SubTypeMarketData,
		SubTypeOrderBook,
		SubTypeTrades,
		SubTypeOrderUpdates,
		SubTypePortfolio,
		SubTypeAlerts,
	}

	isValidType := false
	for _, validType := range validTypes {
		if req.Type == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		return errors.New("invalid subscription type")
	}

	// For market data subscriptions, symbol is required
	if req.Type == SubTypeMarketData || req.Type == SubTypeOrderBook || req.Type == SubTypeTrades {
		if req.Symbol == "" {
			return errors.New("symbol is required for market data subscriptions")
		}
	}

	return nil
}

// sendMessage sends a message to a connection
func (h *MessageHandler) sendMessage(conn *Connection, messageType MessageType, channel string, data interface{}) error {
	message := &Message{
		Type:      messageType,
		Channel:   channel,
		Data:      data,
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal message",
			zap.String("connection_id", conn.ID),
			zap.Error(err))
		return err
	}

	select {
	case conn.send <- messageBytes:
		return nil
	default:
		h.logger.Warn("Connection send buffer full",
			zap.String("connection_id", conn.ID))
		return ErrConnectionSendBufferFull
	}
}

// sendError sends an error message to a connection
func (h *MessageHandler) sendError(conn *Connection, errorCode, errorMessage string) error {
	errorData := map[string]interface{}{
		"code":    errorCode,
		"message": errorMessage,
	}

	return h.sendMessage(conn, MessageTypeError, "", errorData)
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// ConnectionManager manages WebSocket connections
type ConnectionManager struct {
	gateway *Gateway
	logger  *zap.Logger
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(gateway *Gateway, logger *zap.Logger) *ConnectionManager {
	return &ConnectionManager{
		gateway: gateway,
		logger:  logger,
	}
}

// PerformanceOptimizer optimizes WebSocket performance
type PerformanceOptimizer struct {
	gateway *Gateway
	logger  *zap.Logger
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(gateway *Gateway, logger *zap.Logger) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		gateway: gateway,
		logger:  logger,
	}
}

// OptimizePerformance performs performance optimizations
func (p *PerformanceOptimizer) OptimizePerformance() {
	// Get current metrics
	metrics := p.gateway.GetMetrics()
	
	// Log performance metrics
	p.logger.Debug("WebSocket performance metrics",
		zap.Int64("active_connections", metrics.ActiveConnections),
		zap.Float64("messages_per_second", metrics.MessagesPerSecond),
		zap.Duration("average_latency", metrics.AverageLatency),
		zap.Int64("subscription_count", metrics.SubscriptionCount))
	
	// Perform optimizations based on metrics
	if metrics.AverageLatency > 100*time.Millisecond {
		p.logger.Warn("High WebSocket latency detected",
			zap.Duration("average_latency", metrics.AverageLatency))
		// Could implement latency optimization strategies here
	}
	
	if metrics.ActiveConnections > int64(p.gateway.config.MaxConnections*0.8) {
		p.logger.Warn("High connection count",
			zap.Int64("active_connections", metrics.ActiveConnections),
			zap.Int("max_connections", p.gateway.config.MaxConnections))
		// Could implement connection throttling here
	}
}
