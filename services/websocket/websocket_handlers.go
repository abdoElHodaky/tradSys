package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

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

// handleUnsubscription handles WebSocket unsubscriptions
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

// handleHeartbeat handles WebSocket heartbeat messages
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

// validateSubscriptionPermissions validates if a connection can subscribe to a channel
func (wsg *WebSocketGateway) validateSubscriptionPermissions(conn *WebSocketConnection, channel string) bool {
	// Check license tier requirements
	if channelInfo, exists := wsg.channels[channel]; exists {
		return conn.Context.LicenseTier >= channelInfo.LicenseRequired
	}

	return false
}

// addChannelSubscriber adds a connection to a channel's subscriber list
func (wsg *WebSocketGateway) addChannelSubscriber(channelName string, conn *WebSocketConnection) {
	if channel, exists := wsg.channels[channelName]; exists {
		channel.mu.Lock()
		channel.Subscribers[conn.ID] = conn
		channel.mu.Unlock()
	}
}

// routeMessage routes a WebSocket message based on its content and context
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

// processBinaryMessage processes binary WebSocket messages
func (wsg *WebSocketGateway) processBinaryMessage(conn *WebSocketConnection, data []byte) {
	// Implementation for processing binary WebSocket messages
	log.Printf("Received binary message from connection %s: %d bytes", conn.ID, len(data))
}

// startHeartbeatMonitor starts the heartbeat monitoring process
func (wsg *WebSocketGateway) startHeartbeatMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		wsg.performHeartbeatCheck()
	}
}

// performHeartbeatCheck performs heartbeat checks on all connections
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

// startAnalyticsProcessor starts the analytics processing routine
func (wsg *WebSocketGateway) startAnalyticsProcessor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		wsg.processAnalytics()
	}
}

// processAnalytics processes and aggregates analytics data
func (wsg *WebSocketGateway) processAnalytics() {
	// Process and aggregate analytics data
	stats := wsg.GetConnectionStats()
	log.Printf("WebSocket Analytics: %+v", stats)
}
