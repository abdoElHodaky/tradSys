package websocket

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Type aliases for backward compatibility
type Message = WebSocketMessage

// Connection represents a WebSocket connection with additional metadata
type Connection struct {
	conn      *websocket.Conn
	mu        sync.RWMutex
	channels  map[string]bool
	userID    string
	clientID  string
	symbol    string
	send      chan []byte
	server    interface{} // Can be *Server or *EnhancedServer
	closeOnce sync.Once
}

// NewConnection creates a new Connection
func NewConnection(conn *websocket.Conn, userID, clientID string) *Connection {
	return &Connection{
		conn:     conn,
		channels: make(map[string]bool),
		userID:   userID,
		clientID: clientID,
		send:     make(chan []byte, 256),
	}
}

// GetConn returns the underlying websocket connection
func (c *Connection) GetConn() *websocket.Conn {
	return c.conn
}

// GetUserID returns the user ID
func (c *Connection) GetUserID() string {
	return c.userID
}

// GetClientID returns the client ID
func (c *Connection) GetClientID() string {
	return c.clientID
}

// AddChannel adds a channel subscription
func (c *Connection) AddChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channels[channel] = true
}

// RemoveChannel removes a channel subscription
func (c *Connection) RemoveChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.channels, channel)
}

// GetChannels returns a copy of subscribed channels
func (c *Connection) GetChannels() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	channels := make(map[string]bool)
	for k, v := range c.channels {
		channels[k] = v
	}
	return channels
}

// GetSymbol returns the symbol
func (c *Connection) GetSymbol() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.symbol
}

// SetSymbol sets the symbol
func (c *Connection) SetSymbol(symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.symbol = symbol
}

// Use WebSocketMessage from websocket_gateway.go to avoid duplication

// NewWebSocketMessage creates a new WebSocket message
func NewWebSocketMessage(messageType MessageType, data interface{}) (*WebSocketMessage, error) {
	// Create the message
	return &WebSocketMessage{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
	}, nil
}

// UnmarshalData unmarshals the message data into the provided struct
func (m *WebSocketMessage) UnmarshalData(v interface{}) error {
	return json.Unmarshal(m.Data, v)
}

// NewErrorMessage creates a new error message
func NewErrorMessage(errorMessage string) *WebSocketMessage {
	return &WebSocketMessage{
		Type:      "error",
		Error:     errorMessage,
		Timestamp: time.Now(),
	}
}

// SuccessMessage creates a new success message
func SuccessMessage(data interface{}) (*WebSocketMessage, error) {
	// Marshal the data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Create the message
	return &WebSocketMessage{
		Type:      "success",
		Data:      dataBytes,
		Timestamp: time.Now(),
	}, nil
}

// PingMessage creates a new ping message
func PingMessage() *WebSocketMessage {
	return &WebSocketMessage{
		Type:      "ping",
		Timestamp: time.Now(),
	}
}

// PongMessage creates a new pong message
func PongMessage() *WebSocketMessage {
	return &WebSocketMessage{
		Type:      "pong",
		Timestamp: time.Now(),
	}
}

// NewAuthMessage creates a new authentication message
func NewAuthMessage(token string) *WebSocketMessage {
	data := map[string]string{
		"token": token,
	}

	// Marshal the data
	dataBytes, _ := json.Marshal(data)

	// Create the message
	return &WebSocketMessage{
		Type:      "auth",
		Data:      dataBytes,
		Timestamp: time.Now(),
	}
}

// SubscribeMessage creates a new subscribe message
func SubscribeMessage(channel string, params map[string]interface{}) (*WebSocketMessage, error) {
	// Create the data
	data := map[string]interface{}{
		"channel": channel,
	}

	// Add params if provided
	if params != nil {
		data["params"] = params
	}

	// Marshal the data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Create the message
	return &WebSocketMessage{
		Type:      "subscribe",
		Data:      dataBytes,
		Timestamp: time.Now(),
	}, nil
}

// UnsubscribeMessage creates a new unsubscribe message
func UnsubscribeMessage(channel string) (*WebSocketMessage, error) {
	// Create the data
	data := map[string]string{
		"channel": channel,
	}

	// Marshal the data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Create the message
	return &WebSocketMessage{
		Type:      "unsubscribe",
		Data:      dataBytes,
		Timestamp: time.Now(),
	}, nil
}

// Common error messages
var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrInvalidMessage     = errors.New("invalid message")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidChannel     = errors.New("invalid channel")
	ErrInvalidParams      = errors.New("invalid parameters")
	ErrInternalError      = errors.New("internal error")
)
