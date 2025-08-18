package peerjs

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// AuthenticatedPeerConnection represents a PeerJS connection with authentication
type AuthenticatedPeerConnection struct {
	*websocket.Conn
	UserID   string
	Username string
	Role     string
	PeerID   string
}

// AuthenticatedPeerServer extends the PeerServer with authentication
type AuthenticatedPeerServer struct {
	*PeerServer
	logger *zap.Logger
}

// NewAuthenticatedPeerServer creates a new authenticated PeerJS server
func NewAuthenticatedPeerServer(logger *zap.Logger, options *PeerServerOptions) *AuthenticatedPeerServer {
	return &AuthenticatedPeerServer{
		PeerServer: NewPeerServer(options),
		logger:     logger,
	}
}

// HandleConnection handles a new WebSocket connection with authentication
func (s *AuthenticatedPeerServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter or Authorization header
	token := r.URL.Query().Get("token")
	if token == "" {
		// Try Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.logger.Error("Missing authentication token")
			http.Error(w, "Missing authentication token", http.StatusUnauthorized)
			return
		}

		// Check if the header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			s.logger.Error("Invalid authorization header format")
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token = parts[1]
	}

	// Validate token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		s.logger.Error("Invalid authentication token", zap.Error(err))
		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
		return
	}

	// Get peer ID from URL
	peerID := r.URL.Query().Get("id")
	if peerID == "" {
		s.logger.Error("Missing peer ID")
		http.Error(w, "Missing peer ID", http.StatusBadRequest)
		return
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Create authenticated connection
	authConn := &AuthenticatedPeerConnection{
		Conn:     conn,
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		PeerID:   peerID,
	}

	// Add connection to peers
	s.peers.Store(peerID, authConn)

	// Start message handler
	go s.handleAuthenticatedMessages(authConn)

	s.logger.Info("PeerJS connection authenticated",
		zap.String("user_id", claims.UserID),
		zap.String("username", claims.Username),
		zap.String("role", claims.Role),
		zap.String("peer_id", peerID))
}

// handleAuthenticatedMessages handles messages from an authenticated connection
func (s *AuthenticatedPeerServer) handleAuthenticatedMessages(conn *AuthenticatedPeerConnection) {
	defer func() {
		// Remove connection from peers
		s.peers.Delete(conn.PeerID)

		// Close connection
		conn.Close()

		s.logger.Info("PeerJS connection closed",
			zap.String("user_id", conn.UserID),
			zap.String("username", conn.Username),
			zap.String("peer_id", conn.PeerID))
	}()

	for {
		// Read message
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("Unexpected close error", zap.Error(err))
			}
			break
		}

		// Parse message
		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			s.logger.Error("Failed to parse message", zap.Error(err))
			continue
		}

		// Validate message
		if err := s.validateMessage(message); err != nil {
			s.logger.Error("Invalid message", zap.Error(err))
			// Send error message
			errorMsg := Message{
				Type: "ERROR",
				Payload: map[string]interface{}{
					"error": err.Error(),
				},
			}
			if err := conn.WriteJSON(errorMsg); err != nil {
				s.logger.Error("Failed to send error message", zap.Error(err))
			}
			continue
		}

		// Handle message based on type
		switch message.Type {
		case "HEARTBEAT":
			// Send heartbeat response
			heartbeatMsg := Message{
				Type: "HEARTBEAT",
			}
			if err := conn.WriteJSON(heartbeatMsg); err != nil {
				s.logger.Error("Failed to send heartbeat", zap.Error(err))
			}

		case "OFFER", "ANSWER", "CANDIDATE":
			// Check if destination peer exists
			dst := message.Dst
			if dst == "" {
				s.logger.Error("Missing destination peer ID")
				continue
			}

			// Get destination peer
			dstPeer, ok := s.peers.Load(dst)
			if !ok {
				s.logger.Error("Destination peer not found", zap.String("dst", dst))
				// Send error message
				errorMsg := Message{
					Type: "ERROR",
					Payload: map[string]interface{}{
						"error": "Destination peer not found",
					},
				}
				if err := conn.WriteJSON(errorMsg); err != nil {
					s.logger.Error("Failed to send error message", zap.Error(err))
				}
				continue
			}

			// Set source peer ID
			message.Src = conn.PeerID

			// Forward message to destination peer
			if err := dstPeer.(*AuthenticatedPeerConnection).WriteJSON(message); err != nil {
				s.logger.Error("Failed to forward message", zap.Error(err))
			}

		case "LEAVE":
			// Peer is leaving, nothing to do
			s.logger.Info("Peer leaving", zap.String("peer_id", conn.PeerID))

		default:
			s.logger.Warn("Unknown message type", zap.String("type", message.Type))
		}
	}
}

// validateMessage validates a PeerJS message
func (s *AuthenticatedPeerServer) validateMessage(message Message) error {
	// Validate message type
	if message.Type == "" {
		return errors.New("message type is required")
	}

	// Validate message based on type
	switch message.Type {
	case "OFFER", "ANSWER", "CANDIDATE":
		if message.Dst == "" {
			return errors.New("destination peer ID is required")
		}
		if message.Payload == nil {
			return errors.New("payload is required")
		}
	case "LEAVE":
		// No additional validation needed
	case "HEARTBEAT":
		// No additional validation needed
	default:
		return errors.New("invalid message type")
	}

	return nil
}

// WriteJSON writes a JSON message to the connection
func (c *AuthenticatedPeerConnection) WriteJSON(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, data)
}
