package performance

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketOptimizer provides WebSocket optimization functionality
type WebSocketOptimizer struct {
	logger           *zap.Logger
	compressionLevel int
	writeBufferSize  int
	readBufferSize   int
	pingInterval     time.Duration
	pongWait         time.Duration
	writeWait        time.Duration
	mu               sync.Mutex
	connections      map[*websocket.Conn]struct{}
}

// WebSocketOptimizerOptions contains options for the WebSocket optimizer
type WebSocketOptimizerOptions struct {
	CompressionLevel int
	WriteBufferSize  int
	ReadBufferSize   int
	PingInterval     time.Duration
	PongWait         time.Duration
	WriteWait        time.Duration
}

// DefaultWebSocketOptimizerOptions returns default WebSocket optimizer options
func DefaultWebSocketOptimizerOptions() WebSocketOptimizerOptions {
	return WebSocketOptimizerOptions{
		CompressionLevel: 2, // Default compression level
		WriteBufferSize:  4096,
		ReadBufferSize:   4096,
		PingInterval:     30 * time.Second,
		PongWait:         60 * time.Second,
		WriteWait:        10 * time.Second,
	}
}

// NewWebSocketOptimizer creates a new WebSocket optimizer
func NewWebSocketOptimizer(logger *zap.Logger, options WebSocketOptimizerOptions) *WebSocketOptimizer {
	return &WebSocketOptimizer{
		logger:           logger,
		compressionLevel: options.CompressionLevel,
		writeBufferSize:  options.WriteBufferSize,
		readBufferSize:   options.ReadBufferSize,
		pingInterval:     options.PingInterval,
		pongWait:         options.PongWait,
		writeWait:        options.WriteWait,
		connections:      make(map[*websocket.Conn]struct{}),
	}
}

// GetUpgrader returns a WebSocket upgrader with optimized settings
func (o *WebSocketOptimizer) GetUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  o.readBufferSize,
		WriteBufferSize: o.writeBufferSize,
		CheckOrigin:     func(r *websocket.Request) bool { return true },
		EnableCompression: true,
	}
}

// OptimizeConnection optimizes a WebSocket connection
func (o *WebSocketOptimizer) OptimizeConnection(conn *websocket.Conn) {
	// Enable compression
	conn.EnableWriteCompression(true)
	
	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(o.pongWait))
	
	// Set pong handler
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(o.pongWait))
		return nil
	})
	
	// Add connection to the map
	o.mu.Lock()
	o.connections[conn] = struct{}{}
	o.mu.Unlock()
	
	// Start ping routine
	go o.pingConnection(conn)
	
	o.logger.Debug("Optimized WebSocket connection")
}

// pingConnection sends periodic pings to a WebSocket connection
func (o *WebSocketOptimizer) pingConnection(conn *websocket.Conn) {
	ticker := time.NewTicker(o.pingInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		o.mu.Lock()
		_, exists := o.connections[conn]
		o.mu.Unlock()
		
		if !exists {
			return
		}
		
		conn.SetWriteDeadline(time.Now().Add(o.writeWait))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			o.removeConnection(conn)
			return
		}
	}
}

// removeConnection removes a connection from the map
func (o *WebSocketOptimizer) removeConnection(conn *websocket.Conn) {
	o.mu.Lock()
	delete(o.connections, conn)
	o.mu.Unlock()
}

// GetConnectionCount returns the number of active connections
func (o *WebSocketOptimizer) GetConnectionCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.connections)
}

