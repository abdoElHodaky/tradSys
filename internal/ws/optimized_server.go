package ws

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool/performance"
	"github.com/abdoElHodaky/tradSys/internal/performance/latency"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// Default buffer sizes
	defaultReadBufferSize  = 1024 * 4  // 4KB
	defaultWriteBufferSize = 1024 * 16 // 16KB

	// Message size limits
	maxMessageSize = 1024 * 1024 // 1MB

	// Connection pool settings
	defaultMaxConnections = 10000

	// Worker pool settings
	defaultWorkerCount = 0 // Will use NumCPU() if 0
)

// OptimizedMessageHandler defines the signature for optimized message handlers
type OptimizedMessageHandler func(ctx context.Context, data []byte) ([]byte, error)

// OptimizedWebSocketServer is an enhanced WebSocket server optimized for HFT
type OptimizedWebSocketServer struct {
	upgrader      websocket.Upgrader
	connections   map[*websocket.Conn]bool
	connectionsMu sync.RWMutex
	logger        *zap.Logger

	// Message handling
	handlers   map[string]OptimizedMessageHandler
	handlersMu sync.RWMutex

	// Performance optimizations
	bufferPool     *pools.BufferPool
	workerPool     chan struct{}
	latencyTracker *latency.LatencyTracker

	// Connection management
	maxConnections int
	connCount      int32 // atomic

	// Statistics
	messagesReceived uint64 // atomic
	messagesSent     uint64 // atomic
	bytesReceived    uint64 // atomic
	bytesSent        uint64 // atomic
	errors           uint64 // atomic
}

// OptimizedWebSocketServerConfig contains configuration for the optimized WebSocket server
type OptimizedWebSocketServerConfig struct {
	ReadBufferSize  int
	WriteBufferSize int
	MaxMessageSize  int
	MaxConnections  int
	WorkerCount     int
	Logger          *zap.Logger
}

// NewOptimizedWebSocketServer creates a new optimized WebSocket server
func NewOptimizedWebSocketServer(config OptimizedWebSocketServerConfig) *OptimizedWebSocketServer {
	// Set defaults for unspecified values
	if config.ReadBufferSize <= 0 {
		config.ReadBufferSize = defaultReadBufferSize
	}
	if config.WriteBufferSize <= 0 {
		config.WriteBufferSize = defaultWriteBufferSize
	}
	if config.MaxMessageSize <= 0 {
		config.MaxMessageSize = maxMessageSize
	}
	if config.MaxConnections <= 0 {
		config.MaxConnections = defaultMaxConnections
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = runtime.NumCPU()
	}
	if config.Logger == nil {
		var err error
		config.Logger, err = zap.NewProduction()
		if err != nil {
			panic("failed to create logger: " + err.Error())
		}
	}

	server := &OptimizedWebSocketServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all connections by default
				// In production, implement proper origin checking
				return true
			},
		},
		connections:    make(map[*websocket.Conn]bool),
		handlers:       make(map[string]OptimizedMessageHandler),
		bufferPool:     pools.NewBufferPool(config.MaxMessageSize),
		workerPool:     make(chan struct{}, config.WorkerCount),
		latencyTracker: latency.NewLatencyTracker(config.Logger),
		maxConnections: config.MaxConnections,
		logger:         config.Logger,
	}

	return server
}

// HandleFunc registers a message handler for a specific message type
func (s *OptimizedWebSocketServer) HandleFunc(messageType string, handler OptimizedMessageHandler) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()

	s.handlers[messageType] = handler
	s.logger.Info("Registered message handler", zap.String("type", messageType))
}

// ServeHTTP handles WebSocket connections
func (s *OptimizedWebSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if we've reached the connection limit
	if int(atomic.LoadInt32(&s.connCount)) >= s.maxConnections {
		s.logger.Warn("Connection limit reached, rejecting connection")
		http.Error(w, "Too many connections", http.StatusServiceUnavailable)
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		atomic.AddUint64(&s.errors, 1)
		return
	}

	// Configure the connection
	conn.SetReadLimit(int64(maxMessageSize))

	// Register the connection
	s.connectionsMu.Lock()
	s.connections[conn] = true
	s.connectionsMu.Unlock()

	// Increment connection count
	atomic.AddInt32(&s.connCount, 1)

	s.logger.Info("Client connected",
		zap.String("remote_addr", conn.RemoteAddr().String()),
		zap.Int32("active_connections", atomic.LoadInt32(&s.connCount)))

	// Handle the connection in a goroutine
	go s.handleConnection(conn)
}

// handleConnection handles a WebSocket connection
func (s *OptimizedWebSocketServer) handleConnection(conn *websocket.Conn) {
	defer func() {
		// Unregister the connection
		s.connectionsMu.Lock()
		delete(s.connections, conn)
		s.connectionsMu.Unlock()

		// Decrement connection count
		atomic.AddInt32(&s.connCount, -1)

		// Close the connection
		conn.Close()

		s.logger.Info("Client disconnected",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.Int32("active_connections", atomic.LoadInt32(&s.connCount)))
	}()

	for {
		// Get a buffer from the pool
		buffer := s.bufferPool.Get()

		// Read the next message
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				s.logger.Error("Unexpected close error",
					zap.Error(err),
					zap.String("remote_addr", conn.RemoteAddr().String()))
				atomic.AddUint64(&s.errors, 1)
			}
			s.bufferPool.Put(buffer)
			break
		}

		// Update statistics
		atomic.AddUint64(&s.messagesReceived, 1)
		atomic.AddUint64(&s.bytesReceived, uint64(len(message)))

		// Handle the message based on its type
		if messageType == websocket.TextMessage {
			s.handleTextMessage(conn, message, buffer)
		} else if messageType == websocket.BinaryMessage {
			s.handleBinaryMessage(conn, message, buffer)
		}

		// Return the buffer to the pool
		s.bufferPool.Put(buffer)
	}
}

// handleTextMessage handles a text message
func (s *OptimizedWebSocketServer) handleTextMessage(conn *websocket.Conn, message []byte, buffer []byte) {
	// Start latency tracking
	startTime := time.Now()

	// Try to get a worker from the pool
	select {
	case s.workerPool <- struct{}{}:
		// Got a worker, process the message in a separate goroutine
		go func() {
			defer func() {
				<-s.workerPool
				s.latencyTracker.TrackMarketDataProcessing("text_message", startTime)
			}()

			// Parse the message to determine its type
			msgType, err := parseMessageType(message)
			if err != nil {
				s.logger.Error("Failed to parse message type",
					zap.Error(err),
					zap.String("remote_addr", conn.RemoteAddr().String()))
				atomic.AddUint64(&s.errors, 1)
				return
			}

			// Find the appropriate handler
			s.handlersMu.RLock()
			handler, exists := s.handlers[msgType]
			s.handlersMu.RUnlock()

			if !exists {
				s.logger.Warn("No handler for message type",
					zap.String("type", msgType),
					zap.String("remote_addr", conn.RemoteAddr().String()))
				return
			}

			// Handle the message
			response, err := handler(context.Background(), message)
			if err != nil {
				s.logger.Error("Failed to handle message",
					zap.Error(err),
					zap.String("type", msgType),
					zap.String("remote_addr", conn.RemoteAddr().String()))
				atomic.AddUint64(&s.errors, 1)
				return
			}

			// Send the response if there is one
			if response != nil {
				if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
					s.logger.Error("Failed to send response",
						zap.Error(err),
						zap.String("type", msgType),
						zap.String("remote_addr", conn.RemoteAddr().String()))
					atomic.AddUint64(&s.errors, 1)
					return
				}

				// Update statistics
				atomic.AddUint64(&s.messagesSent, 1)
				atomic.AddUint64(&s.bytesSent, uint64(len(response)))
			}
		}()
	default:
		// Worker pool is full, process the message in the current goroutine
		s.logger.Warn("Worker pool full, processing message in current goroutine",
			zap.String("remote_addr", conn.RemoteAddr().String()))

		// Parse the message to determine its type
		msgType, err := parseMessageType(message)
		if err != nil {
			s.logger.Error("Failed to parse message type",
				zap.Error(err),
				zap.String("remote_addr", conn.RemoteAddr().String()))
			atomic.AddUint64(&s.errors, 1)
			return
		}

		// Find the appropriate handler
		s.handlersMu.RLock()
		handler, exists := s.handlers[msgType]
		s.handlersMu.RUnlock()

		if !exists {
			s.logger.Warn("No handler for message type",
				zap.String("type", msgType),
				zap.String("remote_addr", conn.RemoteAddr().String()))
			return
		}

		// Handle the message
		response, err := handler(context.Background(), message)
		if err != nil {
			s.logger.Error("Failed to handle message",
				zap.Error(err),
				zap.String("type", msgType),
				zap.String("remote_addr", conn.RemoteAddr().String()))
			atomic.AddUint64(&s.errors, 1)
			return
		}

		// Send the response if there is one
		if response != nil {
			if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
				s.logger.Error("Failed to send response",
					zap.Error(err),
					zap.String("type", msgType),
					zap.String("remote_addr", conn.RemoteAddr().String()))
				atomic.AddUint64(&s.errors, 1)
				return
			}

			// Update statistics
			atomic.AddUint64(&s.messagesSent, 1)
			atomic.AddUint64(&s.bytesSent, uint64(len(response)))
		}

		// Track latency
		s.latencyTracker.TrackMarketDataProcessing("text_message", startTime)
	}
}

// handleBinaryMessage handles a binary message
func (s *OptimizedWebSocketServer) handleBinaryMessage(conn *websocket.Conn, message []byte, buffer []byte) {
	// Implementation similar to handleTextMessage but for binary messages
	// For brevity, not duplicating the entire implementation
	// In a real implementation, this would handle binary protocol messages

	// Update statistics
	atomic.AddUint64(&s.messagesSent, 1)
	atomic.AddUint64(&s.bytesSent, uint64(len(message)))
}

// Broadcast sends a message to all connected clients
func (s *OptimizedWebSocketServer) Broadcast(messageType int, message []byte) {
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	for conn := range s.connections {
		// Send in a non-blocking way to avoid one slow client affecting others
		go func(c *websocket.Conn) {
			if err := c.WriteMessage(messageType, message); err != nil {
				s.logger.Error("Failed to broadcast message",
					zap.Error(err),
					zap.String("remote_addr", c.RemoteAddr().String()))
				atomic.AddUint64(&s.errors, 1)
				return
			}

			// Update statistics
			atomic.AddUint64(&s.messagesSent, 1)
			atomic.AddUint64(&s.bytesSent, uint64(len(message)))
		}(conn)
	}
}

// GetStats returns server statistics
func (s *OptimizedWebSocketServer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"active_connections": atomic.LoadInt32(&s.connCount),
		"messages_received":  atomic.LoadUint64(&s.messagesReceived),
		"messages_sent":      atomic.LoadUint64(&s.messagesSent),
		"bytes_received":     atomic.LoadUint64(&s.bytesReceived),
		"bytes_sent":         atomic.LoadUint64(&s.bytesSent),
		"errors":             atomic.LoadUint64(&s.errors),
	}
}

// GetLatencyTracker returns the latency tracker
func (s *OptimizedWebSocketServer) GetLatencyTracker() *latency.LatencyTracker {
	return s.latencyTracker
}

// parseMessageType extracts the message type from a message
// This is a placeholder - implement according to your message format
func parseMessageType(message []byte) (string, error) {
	// In a real implementation, this would parse the message to extract its type
	// For example, if using JSON:
	// var msg map[string]interface{}
	// if err := json.Unmarshal(message, &msg); err != nil {
	//     return "", err
	// }
	// return msg["type"].(string), nil

	// Placeholder implementation
	return "default", nil
}
