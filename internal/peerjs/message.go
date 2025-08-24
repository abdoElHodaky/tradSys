package peerjs

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a message between peers
type Message struct {
	// Type is the type of the message
	Type string `json:"type"`
	
	// Src is the source peer ID
	Src string `json:"src,omitempty"`
	
	// Dst is the destination peer ID
	Dst string `json:"dst,omitempty"`
	
	// Payload is the payload of the message
	Payload interface{} `json:"payload,omitempty"`
	
	// Data is the raw data of the message
	Data json.RawMessage `json:"data,omitempty"`
	
	// Signature is the HMAC signature of the message
	Signature string `json:"signature,omitempty"`
}

// Peer represents a peer in the PeerJS network
type Peer struct {
	// ID is the unique identifier for the peer
	ID string
	
	// Conn is the WebSocket connection
	Conn *websocket.Conn
	
	// LastSeen is when the peer was last seen
	LastSeen time.Time
	
	// Connected indicates if the peer is connected
	Connected bool
	
	// Mutex for protecting the peer
	mu sync.RWMutex
}

// PeerServer is a server for PeerJS
type PeerServer struct {
	// Peers is a map of peer ID to peer
	peers map[string]*Peer
	
	// Mutex for protecting the peers map
	mu sync.RWMutex
	
	// Upgrader for upgrading HTTP connections to WebSocket
	upgrader websocket.Upgrader
	
	// Logger
	logger *zap.Logger
}

// NewPeerServer creates a new PeerJS server
func NewPeerServer(logger *zap.Logger) *PeerServer {
	return &PeerServer{
		peers: make(map[string]*Peer),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		logger: logger,
	}
}

// HandleConnection handles a new WebSocket connection
func (s *PeerServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Extract peer ID from URL
	peerID := r.URL.Query().Get("id")
	if peerID == "" {
		s.logger.Error("Missing peer ID")
		http.Error(w, "Missing peer ID", http.StatusBadRequest)
		return
	}
	
	// Upgrade connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}
	
	// Create peer
	peer := &Peer{
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
	openMsg := Message{
		Type: "OPEN",
	}
	
	if err := conn.WriteJSON(openMsg); err != nil {
		s.logger.Error("Failed to send open message", zap.Error(err), zap.String("peer_id", peerID))
		conn.Close()
		return
	}
	
	// Handle messages
	go s.handleMessages(peer)
}

// handleMessages handles messages from a peer
func (s *PeerServer) handleMessages(peer *Peer) {
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
		
		s.logger.Info("Peer disconnected", zap.String("peer_id", peer.ID))
	}()
	
	for {
		// Read message
		var msg Message
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
		
		// Handle message based on type
		switch msg.Type {
		case "HEARTBEAT":
			// Send heartbeat response
			heartbeatMsg := Message{
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
				errorMsg := Message{
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
				leaveMsg := Message{
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

// CleanupInactivePeers removes inactive peers
func (s *PeerServer) CleanupInactivePeers(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	for id, peer := range s.peers {
		peer.mu.RLock()
		lastSeen := peer.LastSeen
		connected := peer.Connected
		peer.mu.RUnlock()
		
		if connected && now.Sub(lastSeen) > timeout {
			// Peer is inactive, close connection
			peer.mu.Lock()
			peer.Connected = false
			peer.mu.Unlock()
			peer.Conn.Close()
			
			// Remove from peers map
			delete(s.peers, id)
			
			s.logger.Info("Removed inactive peer", zap.String("peer_id", id))
		}
	}
}

// StartCleanupTask starts a periodic task to clean up inactive peers
func (s *PeerServer) StartCleanupTask(interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.CleanupInactivePeers(timeout)
		}
	}()
}

