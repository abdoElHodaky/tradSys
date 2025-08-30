package ws

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ConnectionState represents the state of a WebSocket connection
type ConnectionState int

const (
	// ConnectionStateConnecting indicates the connection is being established
	ConnectionStateConnecting ConnectionState = iota
	// ConnectionStateOpen indicates the connection is open and ready for communication
	ConnectionStateOpen
	// ConnectionStateClosing indicates the connection is in the process of closing
	ConnectionStateClosing
	// ConnectionStateClosed indicates the connection is closed
	ConnectionStateClosed
)

// MessageType represents the type of WebSocket message
type MessageType int

const (
	// TextMessage denotes a text data message
	TextMessage MessageType = websocket.TextMessage
	// BinaryMessage denotes a binary data message
	BinaryMessage MessageType = websocket.BinaryMessage
	// CloseMessage denotes a close control message
	CloseMessage MessageType = websocket.CloseMessage
	// PingMessage denotes a ping control message
	PingMessage MessageType = websocket.PingMessage
	// PongMessage denotes a pong control message
	PongMessage MessageType = websocket.PongMessage
)

// Config represents the configuration for the WebSocket server
type Config struct {
	// ReadBufferSize is the size of the read buffer for the WebSocket connection
	ReadBufferSize int
	// WriteBufferSize is the size of the write buffer for the WebSocket connection
	WriteBufferSize int
	// HandshakeTimeout is the timeout for the WebSocket handshake
	HandshakeTimeout time.Duration
	// PingInterval is the interval at which ping messages are sent
	PingInterval time.Duration
	// PongTimeout is the timeout for receiving a pong response
	PongTimeout time.Duration
	// MaxMessageSize is the maximum size of a message in bytes
	MaxMessageSize int64
	// MessageBufferSize is the size of the message buffer
	MessageBufferSize int
	// CheckOrigin is a function that checks the origin of the request
	CheckOrigin func(r *http.Request) bool
}

// DefaultConfig returns the default configuration for the WebSocket server
func DefaultConfig() Config {
	return Config{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		HandshakeTimeout:  10 * time.Second,
		PingInterval:      30 * time.Second,
		PongTimeout:       60 * time.Second,
		MaxMessageSize:    512 * 1024, // 512KB
		MessageBufferSize: 256,
		CheckOrigin:       func(r *http.Request) bool { return true },
	}
}

// OptimizedServer represents an optimized WebSocket server
type OptimizedServer struct {
	config     Config
	upgrader   websocket.Upgrader
	logger     *zap.Logger
	handlers   map[string]MessageHandler
	middleware []Middleware
	
	// Connection tracking
	connections      map[string]*Connection
	connectionsMutex sync.RWMutex
	
	// Broadcast channels
	broadcastChan chan BroadcastMessage
	
	// Lifecycle
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewOptimizedServer creates a new optimized WebSocket server
func NewOptimizedServer(config Config) (*OptimizedServer, error) {
	// Create logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, errors.New("failed to create logger: " + err.Error())
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	server := &OptimizedServer{
		config: config,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin:     config.CheckOrigin,
		},
		logger:        logger,
		handlers:      make(map[string]MessageHandler),
		middleware:    []Middleware{},
		connections:   make(map[string]*Connection),
		broadcastChan: make(chan BroadcastMessage, config.MessageBufferSize),
		ctx:           ctx,
		cancelFunc:    cancel,
	}
	
	// Start broadcast handler
	go server.handleBroadcasts()
	
	return server, nil
}

// WithLogger sets the logger for the server
func (s *OptimizedServer) WithLogger(logger *zap.Logger) *OptimizedServer {
	s.logger = logger
	return s
}

// RegisterHandler registers a message handler for a specific message type
func (s *OptimizedServer) RegisterHandler(messageType string, handler MessageHandler) {
	s.handlers[messageType] = handler
}

// Use adds middleware to the server
func (s *OptimizedServer) Use(middleware ...Middleware) {
	s.middleware = append(s.middleware, middleware...)
}

// HandleConnection handles a WebSocket connection
func (s *OptimizedServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection",
			zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr))
		return
	}
	
	// Create connection object
	connection := NewConnection(conn, s.config.MessageBufferSize)
	connection.SetLogger(s.logger)
	
	// Configure connection
	conn.SetReadLimit(s.config.MaxMessageSize)
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(s.config.PongTimeout))
	})
	
	// Register connection
	s.connectionsMutex.Lock()
	s.connections[connection.ID()] = connection
	s.connectionsMutex.Unlock()
	
	s.logger.Info("New WebSocket connection established",
		zap.String("connection_id", connection.ID()),
		zap.String("remote_addr", conn.RemoteAddr().String()))
	
	// Start ping ticker
	pingTicker := time.NewTicker(s.config.PingInterval)
	
	// Handle connection in separate goroutines
	go func() {
		defer func() {
			pingTicker.Stop()
			s.closeConnection(connection)
		}()
		
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-pingTicker.C:
				// Send ping
				if err := connection.WriteControl(PingMessage, []byte{}, time.Now().Add(s.config.PingInterval/2)); err != nil {
					s.logger.Warn("Failed to send ping",
						zap.String("connection_id", connection.ID()),
						zap.Error(err))
					return
				}
			}
		}
	}()
	
	// Handle incoming messages
	go s.readPump(connection)
}

// readPump handles reading messages from a connection
func (s *OptimizedServer) readPump(connection *Connection) {
	defer s.closeConnection(connection)
	
	for {
		messageType, message, err := connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, 
				websocket.CloseGoingAway, 
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure) {
				s.logger.Error("Unexpected close error",
					zap.String("connection_id", connection.ID()),
					zap.Error(err))
			}
			break
		}
		
		// Process message
		go s.processMessage(connection, messageType, message)
	}
}

// processMessage processes an incoming message
func (s *OptimizedServer) processMessage(connection *Connection, messageType MessageType, message []byte) {
	// Apply middleware
	ctx := context.Background()
	for _, middleware := range s.middleware {
		var err error
		ctx, err = middleware(ctx, connection, messageType, message)
		if err != nil {
			s.logger.Warn("Middleware rejected message",
				zap.String("connection_id", connection.ID()),
				zap.Error(err))
			return
		}
	}
	
	// Parse message to determine type
	// This is a simplified example - in a real application, you would parse the message
	// to determine its type and then route it to the appropriate handler
	
	// For now, just log the message
	s.logger.Debug("Received message",
		zap.String("connection_id", connection.ID()),
		zap.Int("message_type", int(messageType)),
		zap.Int("message_size", len(message)))
	
	// Route message to handler
	// This is a simplified example - in a real application, you would parse the message
	// to determine its type and then route it to the appropriate handler
	for handlerType, handler := range s.handlers {
		if err := handler(ctx, connection, message); err != nil {
			s.logger.Error("Handler failed",
				zap.String("connection_id", connection.ID()),
				zap.String("handler_type", handlerType),
				zap.Error(err))
		}
	}
}

// closeConnection closes a WebSocket connection
func (s *OptimizedServer) closeConnection(connection *Connection) {
	// Remove from connections map
	s.connectionsMutex.Lock()
	delete(s.connections, connection.ID())
	s.connectionsMutex.Unlock()
	
	// Close connection
	connection.Close()
	
	s.logger.Info("WebSocket connection closed",
		zap.String("connection_id", connection.ID()))
}

// Broadcast sends a message to all connections
func (s *OptimizedServer) Broadcast(message []byte, messageType MessageType) {
	s.broadcastChan <- BroadcastMessage{
		Message:     message,
		MessageType: messageType,
	}
}

// BroadcastToGroup sends a message to a specific group of connections
func (s *OptimizedServer) BroadcastToGroup(group string, message []byte, messageType MessageType) {
	s.broadcastChan <- BroadcastMessage{
		Message:     message,
		MessageType: messageType,
		Group:       group,
	}
}

// handleBroadcasts processes broadcast messages
func (s *OptimizedServer) handleBroadcasts() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case broadcast := <-s.broadcastChan:
			s.connectionsMutex.RLock()
			for _, connection := range s.connections {
				// If group is specified, only send to connections in that group
				if broadcast.Group != "" && !connection.InGroup(broadcast.Group) {
					continue
				}
				
				// Send message
				err := connection.WriteMessage(broadcast.MessageType, broadcast.Message)
				if err != nil {
					s.logger.Warn("Failed to broadcast message",
						zap.String("connection_id", connection.ID()),
						zap.Error(err))
				}
			}
			s.connectionsMutex.RUnlock()
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *OptimizedServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down WebSocket server")
	
	// Cancel context to stop all goroutines
	s.cancelFunc()
	
	// Close all connections
	s.connectionsMutex.Lock()
	for _, connection := range s.connections {
		connection.Close()
	}
	s.connections = make(map[string]*Connection)
	s.connectionsMutex.Unlock()
	
	return nil
}

// GetConnectionCount returns the number of active connections
func (s *OptimizedServer) GetConnectionCount() int {
	s.connectionsMutex.RLock()
	defer s.connectionsMutex.RUnlock()
	return len(s.connections)
}

// BroadcastMessage represents a message to be broadcast
type BroadcastMessage struct {
	Message     []byte
	MessageType MessageType
	Group       string
}

// MessageHandler is a function that handles a WebSocket message
type MessageHandler func(ctx context.Context, connection *Connection, message []byte) error

// Middleware is a function that processes a message before it is handled
type Middleware func(ctx context.Context, connection *Connection, messageType MessageType, message []byte) (context.Context, error)

