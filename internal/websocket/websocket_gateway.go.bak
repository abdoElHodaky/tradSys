// Package websocket implements Plan 6: Real-Time WebSocket System for TradSys v3
// Provides intelligent WebSocket routing with multi-dimensional connection management
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketGateway manages intelligent WebSocket connections and routing
type WebSocketGateway struct {
	connectionManager   *ConnectionManager
	subscriptionManager *SubscriptionManager
	licenseValidator    *LicenseValidator
	islamicFilter       *IslamicFilter
	complianceEngine    *ComplianceEngine
	analyticsEngine     *AnalyticsEngine
	upgrader            websocket.Upgrader
	connections         map[string]*WebSocketConnection
	channels            map[string]*ExchangeChannel
	mu                  sync.RWMutex
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	ID               string
	UserID           string
	SessionID        string
	Exchange         ExchangeType
	Connection       *websocket.Conn
	Context          *WebSocketConnectionContext
	Subscriptions    map[string]*Subscription
	LastActivity     time.Time
	CreatedAt        time.Time
	MessageCount     int64
	BytesTransferred int64
	IsActive         bool
	mu               sync.RWMutex
}

// WebSocketConnectionContext contains connection context information
type WebSocketConnectionContext struct {
	ConnectionID     string
	UserID           string
	SessionID        string
	Exchange         ExchangeType
	LicenseTier      LicenseTier
	IslamicCompliant bool
	Region           string
	ClientIP         string
	UserAgent        string
	RegionalEndpoint string
	Metadata         map[string]interface{}
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      MessageType            `json:"type"`
	Channel   string                 `json:"channel"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	MessageID string                 `json:"message_id"`
	UserID    string                 `json:"user_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MessageType defines WebSocket message types
type MessageType int

const (
	MessageTypeSubscribe MessageType = iota
	MessageTypeUnsubscribe
	MessageTypeMarketData
	MessageTypeOrderUpdate
	MessageTypePortfolioUpdate
	MessageTypeAlert
	MessageTypeHeartbeat
	MessageTypeError
	MessageTypeAuth
	MessageTypeCompliance
)

// ExchangeType defines supported exchanges for WebSocket
type ExchangeType int

const (
	ExchangeTypeEGX ExchangeType = iota
	ExchangeTypeADX
	ExchangeTypeUnified
)

// LicenseTier defines license tiers for WebSocket access
type LicenseTier int

const (
	LicenseTierBasic LicenseTier = iota
	LicenseTierProfessional
	LicenseTierEnterprise
	LicenseTierIslamic
)

// Subscription represents a WebSocket subscription
type Subscription struct {
	ID               string
	ConnectionID     string
	Channel          string
	Filters          map[string]interface{}
	LicenseValidated bool
	IslamicCompliant bool
	CreatedAt        time.Time
	LastUpdate       time.Time
	MessageCount     int64
}

// ExchangeChannel represents an exchange-specific WebSocket channel
type ExchangeChannel struct {
	Name                 string
	Exchange             ExchangeType
	Subscribers          map[string]*WebSocketConnection
	MessageQueue         chan *WebSocketMessage
	IslamicFiltering     bool
	LicenseRequired      LicenseTier
	RegionalOptimization bool
	mu                   sync.RWMutex
}

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

// handleAuthentication handles WebSocket authentication
func (wsg *WebSocketGateway) handleAuthentication(conn *WebSocketConnection, message *WebSocketMessage) {
	authData, ok := message.Data.(map[string]interface{})
	if !ok {
		wsg.sendError(conn, "Invalid authentication data", nil)
		return
	}

	// Extract authentication information
	userID, _ := authData["user_id"].(string)
	sessionID, _ := authData["session_id"].(string)
	licenseTier, _ := authData["license_tier"].(float64)
	islamicCompliant, _ := authData["islamic_compliant"].(bool)
	exchange, _ := authData["exchange"].(string)

	// Validate license
	licenseValid, err := wsg.licenseValidator.ValidateLicense(userID, LicenseTier(licenseTier))
	if err != nil || !licenseValid {
		wsg.sendError(conn, "License validation failed", err)
		return
	}

	// Update connection context
	conn.mu.Lock()
	conn.UserID = userID
	conn.Context.UserID = userID
	conn.Context.SessionID = sessionID
	conn.Context.LicenseTier = LicenseTier(licenseTier)
	conn.Context.IslamicCompliant = islamicCompliant
	conn.Context.Exchange = parseExchangeType(exchange)
	conn.mu.Unlock()

	// Send authentication success
	response := WebSocketMessage{
		Type:      MessageTypeAuth,
		Channel:   "auth",
		Data:      map[string]interface{}{"status": "authenticated", "user_id": userID},
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	wsg.sendMessage(conn, &response)
	log.Printf("WebSocket authentication successful for user %s on connection %s", userID, conn.ID)
}

// handleSubscription handles WebSocket subscriptions
func (wsg *WebSocketGateway) handleSubscription(conn *WebSocketConnection, message *WebSocketMessage) {
	subData, ok := message.Data.(map[string]interface{})
	if !ok {
		wsg.sendError(conn, "Invalid subscription data", nil)
		return
	}

	channel, _ := subData["channel"].(string)
	filters, _ := subData["filters"].(map[string]interface{})

	// Validate subscription permissions
	if !wsg.validateSubscriptionPermissions(conn, channel) {
		wsg.sendError(conn, "Insufficient permissions for channel", nil)
		return
	}

	// Create subscription
	subscription := &Subscription{
		ID:               generateSubscriptionID(),
		ConnectionID:     conn.ID,
		Channel:          channel,
		Filters:          filters,
		LicenseValidated: true,
		IslamicCompliant: conn.Context.IslamicCompliant,
		CreatedAt:        time.Now(),
		LastUpdate:       time.Now(),
	}

	// Add to connection subscriptions
	conn.mu.Lock()
	conn.Subscriptions[subscription.ID] = subscription
	conn.mu.Unlock()

	// Add to channel subscribers
	wsg.addChannelSubscriber(channel, conn)

	// Send subscription confirmation
	response := WebSocketMessage{
		Type:    MessageTypeSubscribe,
		Channel: channel,
		Data: map[string]interface{}{
			"status":          "subscribed",
			"subscription_id": subscription.ID,
			"channel":         channel,
		},
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	wsg.sendMessage(conn, &response)
	log.Printf("WebSocket subscription created: %s for channel %s", subscription.ID, channel)
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

// initializeChannels initializes exchange-specific channels
func (wsg *WebSocketGateway) initializeChannels() {
	// EGX channels
	wsg.channels["egx.market_data"] = &ExchangeChannel{
		Name:                 "egx.market_data",
		Exchange:             ExchangeTypeEGX,
		Subscribers:          make(map[string]*WebSocketConnection),
		MessageQueue:         make(chan *WebSocketMessage, 10000),
		IslamicFiltering:     true,
		LicenseRequired:      LicenseTierBasic,
		RegionalOptimization: true,
	}

	wsg.channels["egx.order_updates"] = &ExchangeChannel{
		Name:                 "egx.order_updates",
		Exchange:             ExchangeTypeEGX,
		Subscribers:          make(map[string]*WebSocketConnection),
		MessageQueue:         make(chan *WebSocketMessage, 5000),
		IslamicFiltering:     true,
		LicenseRequired:      LicenseTierProfessional,
		RegionalOptimization: true,
	}

	// ADX channels
	wsg.channels["adx.islamic_data"] = &ExchangeChannel{
		Name:                 "adx.islamic_data",
		Exchange:             ExchangeTypeADX,
		Subscribers:          make(map[string]*WebSocketConnection),
		MessageQueue:         make(chan *WebSocketMessage, 10000),
		IslamicFiltering:     true,
		LicenseRequired:      LicenseTierIslamic,
		RegionalOptimization: true,
	}

	wsg.channels["adx.sukuk_prices"] = &ExchangeChannel{
		Name:                 "adx.sukuk_prices",
		Exchange:             ExchangeTypeADX,
		Subscribers:          make(map[string]*WebSocketConnection),
		MessageQueue:         make(chan *WebSocketMessage, 5000),
		IslamicFiltering:     true,
		LicenseRequired:      LicenseTierIslamic,
		RegionalOptimization: true,
	}

	// Unified channels
	wsg.channels["unified.portfolio"] = &ExchangeChannel{
		Name:                 "unified.portfolio",
		Exchange:             ExchangeTypeUnified,
		Subscribers:          make(map[string]*WebSocketConnection),
		MessageQueue:         make(chan *WebSocketMessage, 5000),
		IslamicFiltering:     false,
		LicenseRequired:      LicenseTierProfessional,
		RegionalOptimization: false,
	}

	log.Printf("Initialized %d WebSocket channels", len(wsg.channels))
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

// Additional helper methods for WebSocket gateway
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

func (wsg *WebSocketGateway) handleHeartbeat(conn *WebSocketConnection, message *WebSocketMessage) {
	response := WebSocketMessage{
		Type:      MessageTypeHeartbeat,
		Channel:   "heartbeat",
		Data:      map[string]interface{}{"status": "alive", "timestamp": time.Now()},
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	wsg.sendMessage(conn, &response)
}

func (wsg *WebSocketGateway) handleUnsubscription(conn *WebSocketConnection, message *WebSocketMessage) {
	subData, ok := message.Data.(map[string]interface{})
	if !ok {
		wsg.sendError(conn, "Invalid unsubscription data", nil)
		return
	}

	subscriptionID, _ := subData["subscription_id"].(string)

	// Remove subscription
	conn.mu.Lock()
	delete(conn.Subscriptions, subscriptionID)
	conn.mu.Unlock()

	// Send confirmation
	response := WebSocketMessage{
		Type:    MessageTypeUnsubscribe,
		Channel: "unsubscribe",
		Data: map[string]interface{}{
			"status":          "unsubscribed",
			"subscription_id": subscriptionID,
		},
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	wsg.sendMessage(conn, &response)
}

func (wsg *WebSocketGateway) validateSubscriptionPermissions(conn *WebSocketConnection, channel string) bool {
	// Check license tier requirements
	if channelInfo, exists := wsg.channels[channel]; exists {
		return conn.Context.LicenseTier >= channelInfo.LicenseRequired
	}

	return false
}

func (wsg *WebSocketGateway) addChannelSubscriber(channelName string, conn *WebSocketConnection) {
	if channel, exists := wsg.channels[channelName]; exists {
		channel.mu.Lock()
		channel.Subscribers[conn.ID] = conn
		channel.mu.Unlock()
	}
}

func (wsg *WebSocketGateway) routeMessage(conn *WebSocketConnection, message *WebSocketMessage) {
	// Apply Islamic finance filtering if required
	if conn.Context.IslamicCompliant {
		filteredMessage, err := wsg.islamicFilter.FilterMessage(message, conn.Context)
		if err != nil {
			wsg.sendError(conn, "Islamic compliance filtering failed", err)
			return
		}
		message = filteredMessage
	}

	// Record analytics
	wsg.analyticsEngine.RecordMessage(conn, message, nil)
}

func (wsg *WebSocketGateway) processBinaryMessage(conn *WebSocketConnection, data []byte) {
	// Implementation for processing binary WebSocket messages
	log.Printf("Received binary message from connection %s: %d bytes", conn.ID, len(data))
}

func (wsg *WebSocketGateway) startHeartbeatMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		wsg.performHeartbeatCheck()
	}
}

func (wsg *WebSocketGateway) performHeartbeatCheck() {
	wsg.mu.RLock()
	connections := make([]*WebSocketConnection, 0, len(wsg.connections))
	for _, conn := range wsg.connections {
		connections = append(connections, conn)
	}
	wsg.mu.RUnlock()

	for _, conn := range connections {
		if time.Since(conn.LastActivity) > 60*time.Second {
			// Send ping
			conn.Connection.WriteMessage(websocket.PingMessage, []byte{})
		}
	}
}

func (wsg *WebSocketGateway) startAnalyticsProcessor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		wsg.processAnalytics()
	}
}

func (wsg *WebSocketGateway) processAnalytics() {
	// Process and aggregate analytics data
	stats := wsg.GetConnectionStats()
	log.Printf("WebSocket Analytics: %+v", stats)
}
