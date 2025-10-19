package ws

import (
	"context"
	"net/http"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/unified-config"
	"github.com/gorilla/websocket"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServerParams contains the parameters for creating a WebSocket server
type ServerParams struct {
	fx.In

	Logger    *zap.Logger
	Config    *config.Config
	Lifecycle fx.Lifecycle
}

// Server represents a WebSocket server
type Server struct {
	logger     *zap.Logger
	config     *config.Config
	upgrader   websocket.Upgrader
	clients    map[string]*Client
	clientsMux sync.RWMutex
}

// Client represents a WebSocket client connection
type Client struct {
	ID           string
	UserID       string
	Conn         *websocket.Conn
	Subscriptions map[string]bool
	Send         chan []byte
}

// NewServer creates a new WebSocket server with fx dependency injection
func NewServer(p ServerParams) *Server {
	server := &Server{
		logger: p.Logger,
		config: p.Config,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, this should be more restrictive
			},
		},
		clients: make(map[string]*Client),
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.Logger.Info("WebSocket server initialized")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("Stopping WebSocket server")
			server.closeAllConnections()
			return nil
		},
	})

	return server
}

// closeAllConnections closes all WebSocket connections
func (s *Server) closeAllConnections() {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	for id, client := range s.clients {
		if client.Conn != nil {
			client.Conn.Close()
		}
		delete(s.clients, id)
	}
}

// HandleWebSocket handles a WebSocket connection
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Create a new client
	client := &Client{
		ID:            r.URL.Query().Get("client_id"),
		UserID:        r.URL.Query().Get("user_id"),
		Conn:          conn,
		Subscriptions: make(map[string]bool),
		Send:          make(chan []byte, 256),
	}

	// Register the client
	s.clientsMux.Lock()
	s.clients[client.ID] = client
	s.clientsMux.Unlock()

	s.logger.Info("New WebSocket connection",
		zap.String("client_id", client.ID),
		zap.String("user_id", client.UserID),
		zap.String("remote_addr", r.RemoteAddr))

	// Start goroutines for reading and writing
	go s.readPump(client)
	go s.writePump(client)
}

// readPump pumps messages from the WebSocket connection to the hub
func (s *Server) readPump(client *Client) {
	defer func() {
		s.clientsMux.Lock()
		delete(s.clients, client.ID)
		s.clientsMux.Unlock()
		client.Conn.Close()
		close(client.Send)
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		// Process the message
		s.logger.Debug("Received WebSocket message",
			zap.String("client_id", client.ID),
			zap.Int("message_size", len(message)))

		// In a real implementation, you would parse and handle the message here
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (s *Server) writePump(client *Client) {
	defer client.Conn.Close()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// The hub closed the channel
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				s.logger.Error("WebSocket write error", zap.Error(err))
				return
			}
		}
	}
}

// Broadcast sends a message to all clients subscribed to a topic
func (s *Server) Broadcast(topic string, message []byte) int {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	count := 0
	for _, client := range s.clients {
		if client.Subscriptions[topic] {
			select {
			case client.Send <- message:
				count++
			default:
				// Client's send buffer is full, skip this client
			}
		}
	}

	return count
}

// ServerModule provides the WebSocket server module for fx
var ServerModule = fx.Options(
	fx.Provide(NewServer),
)

