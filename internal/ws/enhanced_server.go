package ws

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// EnhancedServer is an enhanced WebSocket server with binary message support
type EnhancedServer struct {
	logger           *zap.Logger
	upgrader         websocket.Upgrader
	pool             *ConnectionPool
	binaryHandler    *BinaryMessageHandler
	supportBinary    bool
	compressionLevel int
	mu               sync.RWMutex
}

// EnhancedServerOptions contains options for the enhanced server
type EnhancedServerOptions struct {
	SupportBinary    bool
	CompressionLevel int
	ReadBufferSize   int
	WriteBufferSize  int
	CheckOrigin      func(r *http.Request) bool
}

// DefaultEnhancedServerOptions returns default options for the enhanced server
func DefaultEnhancedServerOptions() *EnhancedServerOptions {
	return &EnhancedServerOptions{
		SupportBinary:    true,
		CompressionLevel: 2, // Default compression level
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		CheckOrigin: func(r *http.Request) bool {
			return true // In production, implement proper origin checks
		},
	}
}

// NewEnhancedServer creates a new enhanced WebSocket server
func NewEnhancedServer(logger *zap.Logger, options *EnhancedServerOptions) *EnhancedServer {
	if options == nil {
		options = DefaultEnhancedServerOptions()
	}
	
	server := &EnhancedServer{
		logger: logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:    options.ReadBufferSize,
			WriteBufferSize:   options.WriteBufferSize,
			CheckOrigin:       options.CheckOrigin,
			EnableCompression: options.CompressionLevel > 0,
		},
		supportBinary:    options.SupportBinary,
		compressionLevel: options.CompressionLevel,
	}
	
	// Create connection pool
	server.pool = NewConnectionPool(logger)
	
	// Create binary message handler
	server.binaryHandler = NewBinaryMessageHandler(logger, server)
	
	return server
}

// ServeWs handles WebSocket requests from clients
func (s *EnhancedServer) ServeWs(w http.ResponseWriter, r *http.Request) {
	// Set up response headers for WebSocket
	header := http.Header{}
	if s.supportBinary {
		header.Add("Sec-WebSocket-Protocol", "binary")
	}
	
	// Upgrade the connection
	conn, err := s.upgrader.Upgrade(w, r, header)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}
	
	// Enable compression if configured
	if s.compressionLevel > 0 {
		conn.EnableWriteCompression(true)
		conn.SetCompressionLevel(s.compressionLevel)
	}
	
	// Create connection object
	client := &Connection{
		conn:     conn,
		send:     make(chan []byte, 256),
		server:   nil, // Will be set by the legacy server if needed
		symbol:   r.URL.Query().Get("symbol"),
		channels: make(map[string]bool),
	}
	
	// Subscribe to default channels
	client.channels["heartbeat"] = true
	
	// Add to connection pool
	s.pool.AddConnection(client)
	s.pool.SubscribeToChannel(client, "heartbeat")
	
	// Start goroutines for reading and writing
	go s.writePump(client)
	go s.readPump(client)
	
	s.logger.Info("Client connected",
		zap.String("remote_addr", conn.RemoteAddr().String()),
		zap.String("symbol", client.symbol))
}

// readPump pumps messages from the websocket connection to the hub
func (s *EnhancedServer) readPump(c *Connection) {
	defer func() {
		s.pool.RemoveConnection(c)
		c.closeOnce.Do(func() {
			c.conn.Close()
		})
		s.logger.Info("Client disconnected",
			zap.String("remote_addr", c.conn.RemoteAddr().String()))
	}()
	
	c.conn.SetReadLimit(1024 * 1024) // 1MB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("Unexpected close error", zap.Error(err))
			}
			break
		}
		
		// Handle the message based on its type
		if messageType == websocket.BinaryMessage && s.supportBinary {
			// Handle binary message with Protocol Buffers
			if err := s.binaryHandler.HandleIncomingMessage(c, message); err != nil {
				s.logger.Error("Failed to handle binary message",
					zap.Error(err),
					zap.String("remote_addr", c.conn.RemoteAddr().String()))
			}
		} else {
			// Handle text message with JSON
			s.handleTextMessage(c, message)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (s *EnhancedServer) writePump(c *Connection) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.closeOnce.Do(func() {
			c.conn.Close()
		})
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The channel was closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				s.logger.Error("Failed to write message", zap.Error(err))
				return
			}
			
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.Error("Failed to write ping", zap.Error(err))
				return
			}
			
			// Send heartbeat
			if s.supportBinary {
				// Send binary heartbeat
				heartbeat := &ws.WebSocketMessage{
					Type:      "heartbeat",
					Channel:   "heartbeat",
					Timestamp: time.Now().UnixMilli(),
					Payload: &ws.WebSocketMessage_Heartbeat{
						Heartbeat: &ws.HeartbeatPayload{
							Timestamp: time.Now().UnixMilli(),
						},
					},
				}
				
				data, err := proto.Marshal(heartbeat)
				if err != nil {
					s.logger.Error("Failed to marshal heartbeat", zap.Error(err))
					continue
				}
				
				if err := c.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
					s.logger.Error("Failed to write heartbeat", zap.Error(err))
					return
				}
			} else {
				// Send JSON heartbeat
				heartbeat := Message{
					Type:    "heartbeat",
					Channel: "heartbeat",
					Data:    time.Now().Unix(),
				}
				
				if err := c.conn.WriteJSON(heartbeat); err != nil {
					s.logger.Error("Failed to write heartbeat", zap.Error(err))
					return
				}
			}
		}
	}
}

// handleTextMessage handles a text message
func (s *EnhancedServer) handleTextMessage(c *Connection, message []byte) {
	// For now, just log the message
	s.logger.Debug("Received text message",
		zap.String("remote_addr", c.conn.RemoteAddr().String()),
		zap.ByteString("message", message))
	
	// In a real implementation, this would parse the JSON and handle the message
}

// BroadcastBinary broadcasts a binary message to all subscribed clients
func (s *EnhancedServer) BroadcastBinary(message *ws.WebSocketMessage) error {
	return s.binaryHandler.BroadcastBinaryMessage(message)
}

// BroadcastText broadcasts a text message to all subscribed clients
func (s *EnhancedServer) BroadcastText(message *Message) error {
	// Get connections for the channel and symbol
	connections := s.pool.GetConnectionsByChannelAndSymbol(message.Channel, message.Symbol)
	
	// Send to all connections
	for _, conn := range connections {
		conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.conn.WriteJSON(message); err != nil {
			s.logger.Error("Failed to broadcast text message",
				zap.Error(err),
				zap.String("remote_addr", conn.conn.RemoteAddr().String()))
			// Continue sending to other connections
			continue
		}
	}
	
	// Update stats
	s.pool.IncrementMessagesSent(int64(len(connections)))
	
	return nil
}

// GetStats gets the connection pool statistics
func (s *EnhancedServer) GetStats() ConnectionPoolStats {
	return s.pool.GetStats()
}

// ResetStats resets the connection pool statistics
func (s *EnhancedServer) ResetStats() {
	s.pool.ResetStats()
}
