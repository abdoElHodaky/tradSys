package ws

import (
	"encoding/json"
	"time"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	// Type is the message type
	Type string `json:"type"`

	// Data is the message data
	Data json.RawMessage `json:"data"`

	// ID is the message ID
	ID string `json:"id,omitempty"`

	// Timestamp is the message timestamp
	Timestamp time.Time `json:"timestamp,omitempty"`

	// Source is the message source
	Source string `json:"source,omitempty"`

	// Target is the message target
	Target string `json:"target,omitempty"`

	// Error is the error message if any
	Error string `json:"error,omitempty"`
}

// NewWebSocketMessage creates a new WebSocket message
func NewWebSocketMessage(messageType string, data interface{}) (*WebSocketMessage, error) {
	// Marshal the data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Create the message
	return &WebSocketMessage{
		Type:      messageType,
		Data:      dataBytes,
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
	ErrConnectionNotFound = "connection not found"
	ErrInvalidMessage     = "invalid message"
	ErrUnauthorized       = "unauthorized"
	ErrInvalidToken       = "invalid token"
	ErrInvalidChannel     = "invalid channel"
	ErrInvalidParams      = "invalid parameters"
	ErrInternalError      = "internal error"
)
