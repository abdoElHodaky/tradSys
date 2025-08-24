package security

import (
	"net/http"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/peerjs"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// SecurePeerServer extends PeerServer with security features
type SecurePeerServer struct {
	*peerjs.PeerServer
	authenticator *PeerAuthenticator
	middleware    *AuthMiddleware
	logger        *zap.Logger
}

// NewSecurePeerServer creates a new secure peer server
func NewSecurePeerServer(logger *zap.Logger, config PeerAuthConfig) *SecurePeerServer {
	// Create the base peer server
	baseServer := peerjs.NewPeerServer(logger)
	
	// Create the authenticator
	authenticator := NewPeerAuthenticator(config, logger)
	
	// Create the middleware
	middleware := NewAuthMiddleware(authenticator, logger)
	
	// Create the secure server
	secureServer := &SecurePeerServer{
		PeerServer:    baseServer,
		authenticator: authenticator,
		middleware:    middleware,
		logger:        logger,
	}
	
	// Start the rate limit cleanup
	authenticator.StartRateLimitCleanup(1 * time.Minute)
	
	return secureServer
}

// HandleConnection handles a new WebSocket connection with security
func (s *SecurePeerServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Extract peer ID from URL
	peerID := r.URL.Query().Get("id")
	if peerID == "" {
		s.logger.Error("Missing peer ID")
		http.Error(w, "Missing peer ID", http.StatusBadRequest)
		return
	}
	
	// Check origin
	if !s.authenticator.CheckOrigin(r) {
		s.logger.Warn("Origin not allowed",
			zap.String("origin", r.Header.Get("Origin")),
			zap.String("remote_addr", r.RemoteAddr))
		
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}
	
	// Check rate limit
	if err := s.authenticator.CheckRateLimit(r); err != nil {
		s.logger.Warn("Rate limit exceeded",
			zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr))
		
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}
	
	// Authenticate the request
	claims, err := s.authenticator.AuthenticateRequest(r)
	if err != nil {
		s.logger.Warn("Authentication failed",
			zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr))
		
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	
	// Verify that the peer ID in the token matches the requested peer ID
	if claims.PeerID != peerID {
		s.logger.Warn("Peer ID mismatch",
			zap.String("token_peer_id", claims.PeerID),
			zap.String("requested_peer_id", peerID),
			zap.String("remote_addr", r.RemoteAddr))
		
		http.Error(w, "Peer ID mismatch", http.StatusForbidden)
		return
	}
	
	// Check connection count
	if err := s.authenticator.IncrementConnectionCount(r); err != nil {
		s.logger.Warn("Connection limit exceeded",
			zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr))
		
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}
	
	// Create a secure upgrader
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return s.authenticator.CheckOrigin(r)
		},
	}
	
	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		s.authenticator.DecrementConnectionCount(r)
		return
	}
	
	// Create peer
	peer := &peerjs.Peer{
		ID:        peerID,
		Conn:      conn,
		LastSeen:  time.Now(),
		Connected: true,
	}
	
	// Register peer
	s.mu.Lock()
	if existingPeer, ok := s.peers[peerID]; ok {
		// Close existing connection
		existingPeer.mu.Lock()
		existingPeer.Connected = false
		existingPeer.mu.Unlock()
		existingPeer.Conn.Close()
	}
	s.peers[peerID] = peer
	s.mu.Unlock()
	
	s.logger.Info("Peer connected", zap.String("peer_id", peerID))
	
	// Send open message
	openMsg := peerjs.Message{
		Type: "OPEN",
	}
	
	if err := conn.WriteJSON(openMsg); err != nil {
		s.logger.Error("Failed to send open message", zap.Error(err), zap.String("peer_id", peerID))
		conn.Close()
		s.authenticator.DecrementConnectionCount(r)
		return
	}
	
	// Handle messages
	go s.handleSecureMessages(peer, r)
}

// handleSecureMessages handles messages from a peer with security
func (s *SecurePeerServer) handleSecureMessages(peer *peerjs.Peer, r *http.Request) {
	defer func() {
		// Unregister peer
		s.mu.Lock()
		if p, ok := s.peers[peer.ID]; ok && p == peer {
			delete(s.peers, peer.ID)
		}
		s.mu.Unlock()
		
		// Close connection
		peer.mu.Lock()
		peer.Connected = false
		peer.mu.Unlock()
		peer.Conn.Close()
		
		// Decrement the connection count
		s.authenticator.DecrementConnectionCount(r)
		
		s.logger.Info("Peer disconnected", zap.String("peer_id", peer.ID))
	}()
	
	for {
		// Read message
		var msg peerjs.Message
		if err := peer.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("Unexpected close error", zap.Error(err), zap.String("peer_id", peer.ID))
			}
			break
		}
		
		// Update last seen
		peer.mu.Lock()
		peer.LastSeen = time.Now()
		peer.mu.Unlock()
		
		// Verify signature if present
		if msg.Signature != "" {
			if !s.authenticator.VerifySignature(string(msg.Data), msg.Signature) {
				s.logger.Warn("Invalid signature", zap.String("peer_id", peer.ID))
				
				// Send error message
				errorMsg := peerjs.Message{
					Type:    "ERROR",
					Payload: "Invalid signature",
				}
				
				if err := peer.Conn.WriteJSON(errorMsg); err != nil {
					s.logger.Error("Failed to send error message", zap.Error(err), zap.String("peer_id", peer.ID))
					break
				}
				
				continue
			}
		}
		
		// Handle message based on type
		switch msg.Type {
		case "HEARTBEAT":
			// Send heartbeat response
			heartbeatMsg := peerjs.Message{
				Type: "HEARTBEAT",
			}
			
			if err := peer.Conn.WriteJSON(heartbeatMsg); err != nil {
				s.logger.Error("Failed to send heartbeat", zap.Error(err), zap.String("peer_id", peer.ID))
				return
			}
			
		case "OFFER", "ANSWER", "CANDIDATE":
			// Forward message to destination peer
			if msg.Dst == "" {
				s.logger.Error("Missing destination", zap.String("peer_id", peer.ID), zap.String("type", msg.Type))
				continue
			}
			
			s.mu.RLock()
			dstPeer, ok := s.peers[msg.Dst]
			s.mu.RUnlock()
			
			if !ok || !dstPeer.Connected {
				// Destination peer not found or not connected
				errorMsg := peerjs.Message{
					Type:    "ERROR",
					Payload: "Peer not found",
				}
				
				if err := peer.Conn.WriteJSON(errorMsg); err != nil {
					s.logger.Error("Failed to send error", zap.Error(err), zap.String("peer_id", peer.ID))
					return
				}
				continue
			}
			
			// Set source
			msg.Src = peer.ID
			
			// Add signature
			msg.Signature = s.authenticator.GenerateSignature(string(msg.Data))
			
			// Forward message
			if err := dstPeer.Conn.WriteJSON(msg); err != nil {
				s.logger.Error("Failed to forward message", zap.Error(err), zap.String("peer_id", peer.ID), zap.String("dst_peer_id", msg.Dst))
				continue
			}
			
		case "LEAVE":
			// Handle leave message
			if msg.Dst == "" {
				s.logger.Error("Missing destination", zap.String("peer_id", peer.ID), zap.String("type", msg.Type))
				continue
			}
			
			s.mu.RLock()
			dstPeer, ok := s.peers[msg.Dst]
			s.mu.RUnlock()
			
			if ok && dstPeer.Connected {
				// Forward leave message
				leaveMsg := peerjs.Message{
					Type: "LEAVE",
					Src:  peer.ID,
				}
				
				if err := dstPeer.Conn.WriteJSON(leaveMsg); err != nil {
					s.logger.Error("Failed to forward leave message", zap.Error(err), zap.String("peer_id", peer.ID), zap.String("dst_peer_id", msg.Dst))
					continue
				}
			}
			
		default:
			s.logger.Warn("Unknown message type", zap.String("type", msg.Type), zap.String("peer_id", peer.ID))
		}
	}
}

