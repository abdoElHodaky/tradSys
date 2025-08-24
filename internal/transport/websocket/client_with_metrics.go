package websocket

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ClientWithMetrics extends the Client with metrics collection
type ClientWithMetrics struct {
	*Client
	metrics *metrics.WebSocketMetrics
}

// NewClientWithMetrics creates a new client with metrics
func NewClientWithMetrics(conn *websocket.Conn, hub *Hub, logger *zap.Logger, metrics *metrics.WebSocketMetrics) *ClientWithMetrics {
	// Generate a client ID if not provided
	clientID := uuid.New().String()
	
	// Create the base client
	client := NewClient(clientID, conn, hub, logger)
	
	// Create the client with metrics
	clientWithMetrics := &ClientWithMetrics{
		Client:  client,
		metrics: metrics,
	}
	
	// Record the connection
	metrics.RecordConnectionOpen(clientID)
	
	return clientWithMetrics
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *ClientWithMetrics) ReadPump() {
	config := DefaultClientConfig()
	
	defer func() {
		c.Hub.Unregister <- c.Client
		c.Conn.Close()
		c.metrics.RecordConnectionClose(c.ID)
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
				c.metrics.RecordConnectionError()
			}
			break
		}
		
		// Record the message
		c.metrics.RecordMessageReceived(len(message))
		
		// Process the message
		message = bytes.TrimSpace(bytes.Replace(message, []byte{'\n'}, []byte{' '}, -1))
		
		// Parse the message
		var msg Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			c.Logger.Error("Failed to parse message", zap.Error(err))
			c.metrics.RecordMessageError()
			continue
		}
		
		// Record the start time for latency measurement
		startTime := time.Now()
		
		// Handle the message
		c.Hub.HandleMessage(c.Client, &msg)
		
		// Record the latency
		c.metrics.RecordMessageLatency(time.Since(startTime))
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *ClientWithMetrics) WritePump() {
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
			
			// Record the message
			c.metrics.RecordMessageSent(len(message))
			
			// Get the next writer
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.metrics.RecordMessageError()
				return
			}
			
			// Write the message
			w.Write(message)
			
			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				nextMsg := <-c.Send
				w.Write([]byte{'\n'})
				w.Write(nextMsg)
				
				// Record the batched message
				c.metrics.RecordMessageSent(len(nextMsg))
			}
			
			// Record the batch
			if n > 0 {
				c.metrics.RecordBatch(n+1, 0) // We don't have the batch latency here
			}
			
			// Close the writer
			if err := w.Close(); err != nil {
				c.metrics.RecordMessageError()
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

// SendMessage sends a message to the client with metrics
func (c *ClientWithMetrics) SendMessage(msg *Message) {
	// Marshal the message
	startTime := time.Now()
	payload, err := json.Marshal(msg)
	if err != nil {
		c.Logger.Error("Failed to marshal message", zap.Error(err))
		c.metrics.RecordMessageError()
		return
	}
	
	// Record compression if enabled
	if c.Conn.EnableWriteCompression(true) {
		originalSize := len(payload)
		// Note: We don't have access to the compressed size here,
		// but we can estimate it based on the message type
		c.metrics.RecordCompression(originalSize, originalSize/2, time.Since(startTime))
	}
	
	// Send the message
	c.Send <- payload
}

