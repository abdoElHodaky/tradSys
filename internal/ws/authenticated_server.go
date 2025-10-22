package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/auth"
	pbws "github.com/abdoElHodaky/tradSys/proto/ws"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// AuthenticatedServer represents a WebSocket server with authentication
type AuthenticatedServer struct {
	logger           *zap.Logger
	upgrader         *AuthenticatedUpgrader
	connections      map[string]*AuthenticatedConnection
	connectionsMutex sync.RWMutex
	handlers         map[string]MessageHandler
	handlersMutex    sync.RWMutex
	closeCh          chan struct{}
}

// NewAuthenticatedServer creates a new authenticated WebSocket server
func NewAuthenticatedServer(logger *zap.Logger, jwtService *auth.JWTService) *AuthenticatedServer {
	return &AuthenticatedServer{
		logger:      logger,
		upgrader:    NewAuthenticatedUpgrader(logger, jwtService),
		connections: make(map[string]*AuthenticatedConnection),
		handlers:    make(map[string]MessageHandler),
		closeCh:     make(chan struct{}),
	}
}

// HandleWebSocket handles WebSocket connections
func (s *AuthenticatedServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection with authentication
	conn, err := s.upgrader.Upgrade(w, r)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Add connection to map
	s.connectionsMutex.Lock()
	s.connections[conn.UserID] = conn
	s.connectionsMutex.Unlock()

	// Start message handler
	go s.handleMessages(conn)

	// Start ping handler
	go s.pingConnection(conn)

	s.logger.Info("WebSocket connection established", 
		zap.String("user_id", conn.UserID), 
		zap.String("username", conn.Username),
		zap.String("role", conn.Role))
}

// handleMessages handles messages from a connection
func (s *AuthenticatedServer) handleMessages(conn *AuthenticatedConnection) {
	defer func() {
		// Remove connection from map
		s.connectionsMutex.Lock()
		delete(s.connections, conn.UserID)
		s.connectionsMutex.Unlock()

		// Close connection
		conn.Close()

		s.logger.Info("WebSocket connection closed", 
			zap.String("user_id", conn.UserID), 
			zap.String("username", conn.Username))
	}()

	for {
		// Check if server is closed
		select {
		case <-s.closeCh:
			return
		default:
		}

		// Read message
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("Unexpected close error", zap.Error(err))
			}
			break
		}

		// Handle message based on type
		switch messageType {
		case websocket.TextMessage:
			// Parse JSON message
			var message Message
			if err := json.Unmarshal(data, &message); err != nil {
				s.logger.Error("Failed to parse message", zap.Error(err))
				continue
			}

			// Handle message
			s.handleMessage(conn, message)

		case websocket.BinaryMessage:
			// Parse Protocol Buffers message
			var pbMessage pbws.WebSocketMessage
			if err := proto.Unmarshal(data, &pbMessage); err != nil {
				s.logger.Error("Failed to parse binary message", zap.Error(err))
				continue
			}

			// Convert to internal message format
			wsMessage := &WebSocketMessage{
				Type:    pbMessage.Type,
				Channel: pbMessage.Channel,
				Symbol:  pbMessage.Symbol,
				Data:    json.RawMessage("{}"), // Convert payload to JSON if needed
			}

			// Handle binary message
			s.handleBinaryMessage(conn, wsMessage)

		default:
			s.logger.Warn("Unsupported message type", zap.Int("type", messageType))
		}
	}
}

// handleMessage handles a text message
func (s *AuthenticatedServer) handleMessage(conn *AuthenticatedConnection, message Message) {
	// Get handler for message type
	s.handlersMutex.RLock()
	handler, ok := s.handlers[message.Type]
	s.handlersMutex.RUnlock()

	if !ok {
		s.logger.Warn("No handler for message type", zap.String("type", message.Type))
		return
	}

	// Handle message
	if err := handler(context.Background(), conn, message); err != nil {
		s.logger.Error("Failed to handle message", zap.Error(err), zap.String("type", message.Type))
	}
}

// handleBinaryMessage handles a binary message
func (s *AuthenticatedServer) handleBinaryMessage(conn *AuthenticatedConnection, message *WebSocketMessage) {
	// Get handler for message type
	s.handlersMutex.RLock()
	handler, ok := s.handlers[message.Type]
	s.handlersMutex.RUnlock()

	if !ok {
		s.logger.Warn("No handler for message type", zap.String("type", message.Type))
		return
	}

	// Convert to text message for handler
	textMessage := Message{
		Type:    message.Type,
		Channel: message.Channel,
		Symbol:  message.Symbol,
		Data:    message.Data,
	}

	// Handle message
	if err := handler(context.Background(), conn, textMessage); err != nil {
		s.logger.Error("Failed to handle binary message", zap.Error(err), zap.String("type", message.Type))
	}
}

// pingConnection sends periodic pings to a connection
func (s *AuthenticatedServer) pingConnection(conn *AuthenticatedConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeCh:
			return
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				s.logger.Error("Failed to send ping", zap.Error(err))
				return
			}
		}
	}
}

// RegisterHandler registers a handler for a message type
func (s *AuthenticatedServer) RegisterHandler(messageType string, handler MessageHandler) {
	s.handlersMutex.Lock()
	defer s.handlersMutex.Unlock()

	s.handlers[messageType] = handler
}

// BroadcastMessage broadcasts a message to all connections with the specified roles
func (s *AuthenticatedServer) BroadcastMessage(message Message, roles ...string) {
	// Marshal message
	data, err := json.Marshal(message)
	if err != nil {
		s.logger.Error("Failed to marshal message", zap.Error(err))
		return
	}

	// Get connections
	s.connectionsMutex.RLock()
	defer s.connectionsMutex.RUnlock()

	// Broadcast to connections with the specified roles
	for _, conn := range s.connections {
		if len(roles) == 0 || AuthorizeConnection(conn, roles...) {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				s.logger.Error("Failed to send message", zap.Error(err), zap.String("user_id", conn.UserID))
			}
		}
	}
}

// BroadcastBinaryMessage broadcasts a binary message to all connections with the specified roles
func (s *AuthenticatedServer) BroadcastBinaryMessage(message *WebSocketMessage, roles ...string) {
	// Convert to proto message
	pbMessage := &pbws.WebSocketMessage{
		Type:    message.Type,
		Channel: message.Channel,
		Symbol:  message.Symbol,
	}
	
	// Marshal message
	data, err := proto.Marshal(pbMessage)
	if err != nil {
		s.logger.Error("Failed to marshal binary message", zap.Error(err))
		return
	}

	// Get connections
	s.connectionsMutex.RLock()
	defer s.connectionsMutex.RUnlock()

	// Broadcast to connections with the specified roles
	for _, conn := range s.connections {
		if len(roles) == 0 || AuthorizeConnection(conn, roles...) {
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				s.logger.Error("Failed to send binary message", zap.Error(err), zap.String("user_id", conn.UserID))
			}
		}
	}
}

// SendMessage sends a message to a specific user
func (s *AuthenticatedServer) SendMessage(userID string, message Message) error {
	// Marshal message
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Get connection
	s.connectionsMutex.RLock()
	conn, ok := s.connections[userID]
	s.connectionsMutex.RUnlock()

	if !ok {
		return ErrConnectionNotFound
	}

	// Send message
	return conn.WriteMessage(websocket.TextMessage, data)
}

// SendBinaryMessage sends a binary message to a specific user
func (s *AuthenticatedServer) SendBinaryMessage(userID string, message *WebSocketMessage) error {
	// Convert to proto message
	pbMessage := &pbws.WebSocketMessage{
		Type:    message.Type,
		Channel: message.Channel,
		Symbol:  message.Symbol,
	}
	
	// Marshal message
	data, err := proto.Marshal(pbMessage)
	if err != nil {
		return err
	}

	// Get connection
	s.connectionsMutex.RLock()
	conn, ok := s.connections[userID]
	s.connectionsMutex.RUnlock()

	if !ok {
		return ErrConnectionNotFound
	}

	// Send message
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

// Close closes the server
func (s *AuthenticatedServer) Close() {
	close(s.closeCh)

	// Close all connections
	s.connectionsMutex.Lock()
	defer s.connectionsMutex.Unlock()

	for _, conn := range s.connections {
		conn.Close()
	}

	s.connections = make(map[string]*AuthenticatedConnection)
}

// GetConnectionCount returns the number of active connections
func (s *AuthenticatedServer) GetConnectionCount() int {
	s.connectionsMutex.RLock()
	defer s.connectionsMutex.RUnlock()

	return len(s.connections)
}

// GetConnectionCountByRole returns the number of active connections by role
func (s *AuthenticatedServer) GetConnectionCountByRole() map[string]int {
	s.connectionsMutex.RLock()
	defer s.connectionsMutex.RUnlock()

	counts := make(map[string]int)
	for _, conn := range s.connections {
		counts[conn.Role]++
	}

	return counts
}

// MessageHandler is a function that handles a message
type MessageHandler func(ctx context.Context, conn *AuthenticatedConnection, msg Message) error
