package websocket

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client represents a WebSocket client
type Client struct {
	// ID is the unique identifier for the client
	ID string
	
	// Hub is the hub that the client belongs to
	Hub *Hub
	
	// Conn is the WebSocket connection
	Conn *websocket.Conn
	
	// Send is a channel of messages to send to the client
	Send chan []byte
	
	// Logger is the logger for the client
	Logger *zap.Logger
}

// ClientConfig contains configuration for the client
type ClientConfig struct {
	// SendBufferSize is the size of the send buffer
	SendBufferSize int
	
	// PingInterval is the interval at which ping messages are sent
	PingInterval time.Duration
	
	// PongWait is the time to wait for a pong response
	PongWait time.Duration
	
	// WriteWait is the time to wait for a write to complete
	WriteWait time.Duration
	
	// MaxMessageSize is the maximum message size
	MaxMessageSize int64
}

// DefaultClientConfig returns the default configuration
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		SendBufferSize: 256,
		PingInterval:   30 * time.Second,
		PongWait:       60 * time.Second,
		WriteWait:      10 * time.Second,
		MaxMessageSize: 1024 * 1024, // 1MB
	}
}

// NewClient creates a new client
func NewClient(id string, conn *websocket.Conn, hub *Hub, logger *zap.Logger) *Client {
	config := DefaultClientConfig()
	
	return &Client{
		ID:     id,
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, config.SendBufferSize),
		Logger: logger,
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	config := DefaultClientConfig()
	
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	
	// Set read limit
	c.Conn.SetReadLimit(config.MaxMessageSize)
	
	// Set read deadline
	c.Conn.SetReadDeadline(time.Now().Add(config.PongWait))
	
	// Set pong handler
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(config.PongWait))
		return nil
	})
	
	// Read messages
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Logger.Error("Unexpected close error", zap.Error(err))
			}
			break
		}
		
		// Process the message
		message = bytes.TrimSpace(bytes.Replace(message, []byte{'\n'}, []byte{' '}, -1))
		
		// Parse the message
		var msg Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			c.Logger.Error("Failed to parse message", zap.Error(err))
			continue
		}
		
		// Handle the message
		c.Hub.HandleMessage(c, &msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	config := DefaultClientConfig()
	
	ticker := time.NewTicker(config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.Send:
			// Set write deadline
			c.Conn.SetWriteDeadline(time.Now().Add(config.WriteWait))
			
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			// Get the next writer
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			
			// Write the message
			w.Write(message)
			
			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}
			
			// Close the writer
			if err := w.Close(); err != nil {
				return
			}
			
		case <-ticker.C:
			// Set write deadline
			c.Conn.SetWriteDeadline(time.Now().Add(config.WriteWait))
			
			// Send ping message
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send sends a message to the client
func (c *Client) SendMessage(msg *Message) {
	// Marshal the message
	payload, err := json.Marshal(msg)
	if err != nil {
		c.Logger.Error("Failed to marshal message", zap.Error(err))
		return
	}
	
	// Send the message
	c.Send <- payload
}

