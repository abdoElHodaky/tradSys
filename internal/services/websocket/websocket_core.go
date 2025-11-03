package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// NewWebSocketGateway creates a new WebSocket gateway instance
func NewWebSocketGateway() *WebSocketGateway {
	gateway := &WebSocketGateway{
		connectionManager:   NewConnectionManager(),
		subscriptionManager: NewSubscriptionManager(),
		licenseValidator:    NewLicenseValidator(),
		islamicFilter:       NewIslamicFilter(),
		complianceEngine:    NewComplianceEngine(),
		analyticsEngine:     NewAnalyticsEngine(),
		connections:         make(map[string]*WebSocketConnection),
		channels:            make(map[string]*ExchangeChannel),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
		},
	}

	// Initialize exchange-specific channels
	gateway.initializeChannels()

	// Start background processes
	go gateway.startHeartbeatMonitor()
	go gateway.startAnalyticsProcessor()

	return gateway
}

// HandleConnection handles new WebSocket connections
func (wsg *WebSocketGateway) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := wsg.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create connection context
	ctx := &WebSocketConnectionContext{
		ConnectionID: generateConnectionID(),
		ClientIP:     r.RemoteAddr,
		UserAgent:    r.UserAgent(),
		Metadata:     make(map[string]interface{}),
	}

	// Create WebSocket connection
	wsConn := &WebSocketConnection{
		ID:            ctx.ConnectionID,
		Connection:    conn,
		Context:       ctx,
		Subscriptions: make(map[string]*Subscription),
		CreatedAt:     time.Now(),
		LastActivity:  time.Now(),
		IsActive:      true,
	}

	// Register connection
	wsg.mu.Lock()
	wsg.connections[wsConn.ID] = wsConn
	wsg.mu.Unlock()

	// Start connection handler
	go wsg.handleConnection(wsConn)

	log.Printf("WebSocket connection established: %s from %s", wsConn.ID, ctx.ClientIP)
}

// handleConnection handles messages for a specific connection
func (wsg *WebSocketGateway) handleConnection(conn *WebSocketConnection) {
	defer wsg.closeConnection(conn)

	// Set read deadline
	conn.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Set pong handler for heartbeat
	conn.Connection.SetPongHandler(func(string) error {
		conn.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.LastActivity = time.Now()
		return nil
	})

	for {
		// Read message
		messageType, data, err := conn.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for connection %s: %v", conn.ID, err)
			}
			break
		}

		// Update activity
		conn.mu.Lock()
		conn.LastActivity = time.Now()
		conn.MessageCount++
		conn.BytesTransferred += int64(len(data))
		conn.mu.Unlock()

		// Process message based on type
		if messageType == websocket.TextMessage {
			wsg.processTextMessage(conn, data)
		} else if messageType == websocket.BinaryMessage {
			wsg.processBinaryMessage(conn, data)
		}
	}
}

// processTextMessage processes text WebSocket messages
func (wsg *WebSocketGateway) processTextMessage(conn *WebSocketConnection, data []byte) {
	var message WebSocketMessage
	if err := json.Unmarshal(data, &message); err != nil {
		wsg.sendError(conn, "Invalid message format", err)
		return
	}

	// Set message metadata
	message.Timestamp = time.Now()
	message.MessageID = generateMessageID()
	message.UserID = conn.UserID

	// Route message based on type
	switch message.Type {
	case MessageTypeAuth:
		wsg.handleAuthentication(conn, &message)
	case MessageTypeSubscribe:
		wsg.handleSubscription(conn, &message)
	case MessageTypeUnsubscribe:
		wsg.handleUnsubscription(conn, &message)
	case MessageTypeHeartbeat:
		wsg.handleHeartbeat(conn, &message)
	default:
		wsg.routeMessage(conn, &message)
	}
}

// BroadcastToChannel broadcasts a message to all subscribers of a channel
func (wsg *WebSocketGateway) BroadcastToChannel(channelName string, message *WebSocketMessage) {
	wsg.mu.RLock()
	channel, exists := wsg.channels[channelName]
	wsg.mu.RUnlock()

	if !exists {
		log.Printf("Channel not found: %s", channelName)
		return
	}

	channel.mu.RLock()
	subscribers := make([]*WebSocketConnection, 0, len(channel.Subscribers))
	for _, conn := range channel.Subscribers {
		subscribers = append(subscribers, conn)
	}
	channel.mu.RUnlock()

	// Broadcast to all subscribers
	for _, conn := range subscribers {
		// Apply filtering if required
		filteredMessage := message
		if conn.Context.IslamicCompliant && channel.IslamicFiltering {
			var err error
			filteredMessage, err = wsg.islamicFilter.FilterMessage(message, conn.Context)
			if err != nil {
				log.Printf("Islamic filtering failed for connection %s: %v", conn.ID, err)
				continue
			}
		}

		// Send message
		wsg.sendMessage(conn, filteredMessage)
	}

	log.Printf("Broadcasted message to %d subscribers on channel %s", len(subscribers), channelName)
}

// sendMessage sends a message to a WebSocket connection
func (wsg *WebSocketGateway) sendMessage(conn *WebSocketConnection, message *WebSocketMessage) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if !conn.IsActive {
		return
	}

	// Set write deadline
	conn.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Send message
	if err := conn.Connection.WriteJSON(message); err != nil {
		log.Printf("Failed to send message to connection %s: %v", conn.ID, err)
		conn.IsActive = false
		return
	}

	// Update metrics
	conn.MessageCount++
	conn.LastActivity = time.Now()
}

// GetConnectionStats returns WebSocket connection statistics
func (wsg *WebSocketGateway) GetConnectionStats() map[string]interface{} {
	wsg.mu.RLock()
	defer wsg.mu.RUnlock()

	stats := make(map[string]interface{})

	// Connection counts
	totalConnections := len(wsg.connections)
	activeConnections := 0

	// Exchange distribution
	exchangeCounts := make(map[ExchangeType]int)
	licenseCounts := make(map[LicenseTier]int)

	for _, conn := range wsg.connections {
		if conn.IsActive {
			activeConnections++
		}
		exchangeCounts[conn.Context.Exchange]++
		licenseCounts[conn.Context.LicenseTier]++
	}

	stats["total_connections"] = totalConnections
	stats["active_connections"] = activeConnections
	stats["exchange_distribution"] = exchangeCounts
	stats["license_distribution"] = licenseCounts
	stats["total_channels"] = len(wsg.channels)
	stats["timestamp"] = time.Now()

	return stats
}

// closeConnection closes a WebSocket connection and cleans up resources
func (wsg *WebSocketGateway) closeConnection(conn *WebSocketConnection) {
	conn.mu.Lock()
	conn.IsActive = false
	conn.mu.Unlock()

	// Remove from connections map
	wsg.mu.Lock()
	delete(wsg.connections, conn.ID)
	wsg.mu.Unlock()

	// Remove from all channels
	for _, channel := range wsg.channels {
		channel.mu.Lock()
		delete(channel.Subscribers, conn.ID)
		channel.mu.Unlock()
	}

	// Close WebSocket connection
	conn.Connection.Close()
}

// sendError sends an error message to a WebSocket connection
func (wsg *WebSocketGateway) sendError(conn *WebSocketConnection, message string, err error) {
	errorDetails := "Unknown error"
	if err != nil {
		errorDetails = err.Error()
	}

	errorMsg := WebSocketMessage{
		Type:    MessageTypeError,
		Channel: "error",
		Data: map[string]interface{}{
			"error":   message,
			"details": errorDetails,
		},
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	wsg.sendMessage(conn, &errorMsg)
}

// Helper functions
func generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

func parseExchangeType(exchange string) ExchangeType {
	switch exchange {
	case "EGX":
		return ExchangeTypeEGX
	case "ADX":
		return ExchangeTypeADX
	case "UNIFIED":
		return ExchangeTypeUnified
	default:
		return ExchangeTypeUnified
	}
}

// Constructor functions for dependencies
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*WebSocketConnection),
	}
}

func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string]*Subscription),
	}
}

func NewLicenseValidator() *LicenseValidator {
	return &LicenseValidator{}
}

func NewIslamicFilter() *IslamicFilter {
	return &IslamicFilter{}
}

func NewComplianceEngine() *ComplianceEngine {
	return &ComplianceEngine{}
}

func NewAnalyticsEngine() *AnalyticsEngine {
	return &AnalyticsEngine{}
}

// ValidateLicense validates a user's license for WebSocket access
func (lv *LicenseValidator) ValidateLicense(userID string, tier LicenseTier) (bool, error) {
	// Implementation for license validation
	return true, nil
}

// FilterMessage filters a WebSocket message for Islamic compliance
func (f *IslamicFilter) FilterMessage(message *WebSocketMessage, ctx *WebSocketConnectionContext) (*WebSocketMessage, error) {
	// Implementation for Islamic filtering
	return message, nil
}

// RecordMessage records a WebSocket message for analytics
func (ae *AnalyticsEngine) RecordMessage(conn *WebSocketConnection, message *WebSocketMessage, err error) {
	// Implementation for analytics recording
}
