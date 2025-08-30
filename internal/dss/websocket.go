package dss

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketManager manages WebSocket connections for the DSS API
type WebSocketManager struct {
	logger        *zap.Logger
	service       Service
	authService   AuthService
	clients       map[string]*WebSocketClient
	clientsMutex  sync.RWMutex
	upgrader      websocket.Upgrader
	subscriptions map[string]map[string]bool // Map of client ID to subscribed channels
	subsMutex     sync.RWMutex
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID           string
	User         User
	Conn         *websocket.Conn
	Send         chan []byte
	Manager      *WebSocketManager
	Subscriptions map[string]bool
	SubsMutex    sync.RWMutex
	Logger       *zap.Logger
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(logger *zap.Logger, service Service, authService AuthService) *WebSocketManager {
	return &WebSocketManager{
		logger:        logger,
		service:       service,
		authService:   authService,
		clients:       make(map[string]*WebSocketClient),
		subscriptions: make(map[string]map[string]bool),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in this example
				// In production, this should be restricted
				return true
			},
		},
	}
}

// HandleWebSocket upgrades an HTTP connection to WebSocket
func (m *WebSocketManager) HandleWebSocket(c *gin.Context) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "authentication_required",
				"message": "Authentication token is required",
			},
		})
		return
	}

	// Validate token
	user, err := m.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		m.logger.Error("Invalid WebSocket token", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "authentication_required",
				"message": "Invalid or expired token",
			},
		})
		return
	}

	// Upgrade connection to WebSocket
	conn, err := m.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		m.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}

	// Create client
	clientID := generateClientID()
	client := &WebSocketClient{
		ID:            clientID,
		User:          user,
		Conn:          conn,
		Send:          make(chan []byte, 256),
		Manager:       m,
		Subscriptions: make(map[string]bool),
		Logger:        m.logger.With(zap.String("client_id", clientID)),
	}

	// Register client
	m.clientsMutex.Lock()
	m.clients[clientID] = client
	m.clientsMutex.Unlock()

	m.subsMutex.Lock()
	m.subscriptions[clientID] = make(map[string]bool)
	m.subsMutex.Unlock()

	// Start client goroutines
	go client.readPump()
	go client.writePump()

	m.logger.Info("WebSocket client connected", zap.String("client_id", clientID))
}

// BroadcastToChannel sends a message to all clients subscribed to a channel
func (m *WebSocketManager) BroadcastToChannel(channel string, message []byte) {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()

	for clientID, client := range m.clients {
		m.subsMutex.RLock()
		subscribed := m.subscriptions[clientID][channel]
		m.subsMutex.RUnlock()

		if subscribed {
			select {
			case client.Send <- message:
				// Message sent to client
			default:
				// Client send buffer is full, close connection
				m.removeClient(clientID)
			}
		}
	}
}

// BroadcastToSymbol sends a message to all clients subscribed to a symbol
func (m *WebSocketManager) BroadcastToSymbol(symbol string, messageType string, data map[string]interface{}) {
	// Create WebSocket message
	wsMessage := WebSocketMessage{
		Type:      messageType,
		Timestamp: time.Now(),
		Symbol:    symbol,
		Data:      data,
	}

	// Marshal message to JSON
	messageBytes, err := json.Marshal(wsMessage)
	if err != nil {
		m.logger.Error("Failed to marshal WebSocket message", zap.Error(err))
		return
	}

	// Broadcast to symbol channel
	m.BroadcastToChannel("symbol:"+symbol, messageBytes)
}

// removeClient removes a client from the manager
func (m *WebSocketManager) removeClient(clientID string) {
	m.clientsMutex.Lock()
	if client, ok := m.clients[clientID]; ok {
		close(client.Send)
		client.Conn.Close()
		delete(m.clients, clientID)
	}
	m.clientsMutex.Unlock()

	m.subsMutex.Lock()
	delete(m.subscriptions, clientID)
	m.subsMutex.Unlock()

	m.logger.Info("WebSocket client disconnected", zap.String("client_id", clientID))
}

// readPump pumps messages from the WebSocket connection to the manager
func (c *WebSocketClient) readPump() {
	defer func() {
		c.Manager.removeClient(c.ID)
	}()

	c.Conn.SetReadLimit(4096) // Max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		// Process message
		c.processMessage(message)
	}
}

// writePump pumps messages from the client's send channel to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// processMessage processes a message received from the client
func (c *WebSocketClient) processMessage(message []byte) {
	var msg struct {
		Action   string   `json:"action"`
		Channels []string `json:"channels,omitempty"`
		Symbols  []string `json:"symbols,omitempty"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		c.Logger.Error("Failed to parse WebSocket message", zap.Error(err))
		return
	}

	switch msg.Action {
	case "subscribe":
		c.handleSubscribe(msg.Channels, msg.Symbols)
	case "unsubscribe":
		c.handleUnsubscribe(msg.Channels, msg.Symbols)
	case "ping":
		c.handlePing()
	default:
		c.Logger.Warn("Unknown WebSocket action", zap.String("action", msg.Action))
	}
}

// handleSubscribe handles a subscription request
func (c *WebSocketClient) handleSubscribe(channels []string, symbols []string) {
	// Subscribe to channels
	for _, channel := range channels {
		c.SubsMutex.Lock()
		c.Subscriptions[channel] = true
		c.SubsMutex.Unlock()

		c.Manager.subsMutex.Lock()
		c.Manager.subscriptions[c.ID][channel] = true
		c.Manager.subsMutex.Unlock()

		c.Logger.Info("Client subscribed to channel", zap.String("channel", channel))
	}

	// Subscribe to symbols
	for _, symbol := range symbols {
		symbolChannel := "symbol:" + symbol

		c.SubsMutex.Lock()
		c.Subscriptions[symbolChannel] = true
		c.SubsMutex.Unlock()

		c.Manager.subsMutex.Lock()
		c.Manager.subscriptions[c.ID][symbolChannel] = true
		c.Manager.subsMutex.Unlock()

		c.Logger.Info("Client subscribed to symbol", zap.String("symbol", symbol))
	}

	// Send confirmation
	response := map[string]interface{}{
		"type":     "subscription_success",
		"channels": channels,
		"symbols":  symbols,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		c.Logger.Error("Failed to marshal subscription response", zap.Error(err))
		return
	}

	c.Send <- responseBytes
}

// handleUnsubscribe handles an unsubscription request
func (c *WebSocketClient) handleUnsubscribe(channels []string, symbols []string) {
	// Unsubscribe from channels
	for _, channel := range channels {
		c.SubsMutex.Lock()
		delete(c.Subscriptions, channel)
		c.SubsMutex.Unlock()

		c.Manager.subsMutex.Lock()
		delete(c.Manager.subscriptions[c.ID], channel)
		c.Manager.subsMutex.Unlock()

		c.Logger.Info("Client unsubscribed from channel", zap.String("channel", channel))
	}

	// Unsubscribe from symbols
	for _, symbol := range symbols {
		symbolChannel := "symbol:" + symbol

		c.SubsMutex.Lock()
		delete(c.Subscriptions, symbolChannel)
		c.SubsMutex.Unlock()

		c.Manager.subsMutex.Lock()
		delete(c.Manager.subscriptions[c.ID], symbolChannel)
		c.Manager.subsMutex.Unlock()

		c.Logger.Info("Client unsubscribed from symbol", zap.String("symbol", symbol))
	}

	// Send confirmation
	response := map[string]interface{}{
		"type":     "unsubscription_success",
		"channels": channels,
		"symbols":  symbols,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		c.Logger.Error("Failed to marshal unsubscription response", zap.Error(err))
		return
	}

	c.Send <- responseBytes
}

// handlePing handles a ping request
func (c *WebSocketClient) handlePing() {
	response := map[string]interface{}{
		"type": "pong",
		"time": time.Now().UnixNano() / int64(time.Millisecond),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		c.Logger.Error("Failed to marshal ping response", zap.Error(err))
		return
	}

	c.Send <- responseBytes
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return "ws_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of the specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(result)
}

