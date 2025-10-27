package peerjs

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// PeerServer represents a PeerJS signaling server
type PeerServer struct {
	logger   *zap.Logger
	upgrader websocket.Upgrader
	peers    sync.Map
	mu       sync.RWMutex
}

// NewPeerServer creates a new PeerJS server
func NewPeerServer(logger *zap.Logger) *PeerServer {
	return &PeerServer{
		logger: logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, implement proper origin checks
			},
		},
		peers: sync.Map{},
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
	if existingPeerValue, ok := s.peers.Load(peerID); ok {
		existingPeer := existingPeerValue.(*Peer)
		// Close existing connection
		existingPeer.mu.Lock()
		existingPeer.Connected = false
		existingPeer.mu.Unlock()
		existingPeer.Conn.Close()
	}
	s.peers.Store(peerID, peer)
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
		if pValue, ok := s.peers.Load(peer.ID); ok && pValue.(*Peer) == peer {
			s.peers.Delete(peer.ID)
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
		_, data, err := peer.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("Unexpected close error", zap.Error(err), zap.String("peer_id", peer.ID))
			}
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			s.logger.Error("Failed to parse message", zap.Error(err), zap.String("peer_id", peer.ID))
			continue
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
			dstPeerValue, ok := s.peers.Load(msg.Dst)
			s.mu.RUnlock()

			if !ok {
				// Destination peer not found
				errorMsg := Message{
					Type:    "ERROR",
					Payload: "Peer not found",
				}
				peer.Conn.WriteJSON(errorMsg)
				continue
			}

			dstPeer := dstPeerValue.(*Peer)
			if !dstPeer.Connected {
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
			dstPeerValue, ok := s.peers.Load(msg.Dst)
			s.mu.RUnlock()

			if ok {
				dstPeer := dstPeerValue.(*Peer)
				if dstPeer.Connected {
					// Forward leave message
					leaveMsg := Message{
						Type: "LEAVE",
						Src:  peer.ID,
					}
					dstPeer.Conn.WriteJSON(leaveMsg)
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
	var inactivePeers []string

	s.peers.Range(func(key, value interface{}) bool {
		id := key.(string)
		peer := value.(*Peer)

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

			inactivePeers = append(inactivePeers, id)
			s.logger.Info("Removed inactive peer", zap.String("peer_id", id))
		}
		return true
	})

	// Remove inactive peers from the map
	for _, id := range inactivePeers {
		s.peers.Delete(id)
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

// GetPeerCount returns the number of connected peers
func (s *PeerServer) GetPeerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	s.peers.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
