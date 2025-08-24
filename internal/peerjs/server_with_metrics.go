package peerjs

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// PeerServerWithMetrics extends PeerServer with metrics collection
type PeerServerWithMetrics struct {
	*PeerServer
	metrics *metrics.PeerJSMetrics
}

// NewPeerServerWithMetrics creates a new PeerJS server with metrics
func NewPeerServerWithMetrics(logger *zap.Logger, metrics *metrics.PeerJSMetrics) *PeerServerWithMetrics {
	return &PeerServerWithMetrics{
		PeerServer: NewPeerServer(logger),
		metrics:    metrics,
	}
}

// HandleConnection handles a new WebSocket connection with metrics
func (s *PeerServerWithMetrics) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Extract peer ID from URL
	peerID := r.URL.Query().Get("id")
	if peerID == "" {
		s.logger.Error("Missing peer ID")
		s.metrics.RecordPeerError()
		http.Error(w, "Missing peer ID", http.StatusBadRequest)
		return
	}
	
	// Record connection attempt
	s.metrics.RecordConnectionAttempt()
	
	// Upgrade connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		s.metrics.RecordConnectionFailure()
		return
	}
	
	// Record successful connection
	s.metrics.RecordConnectionSuccess()
	
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
		
		// Record peer disconnection
		s.metrics.RecordPeerDisconnected(peerID)
	}
	s.peers[peerID] = peer
	s.mu.Unlock()
	
	// Record peer connection
	s.metrics.RecordPeerConnected(peerID)
	
	s.logger.Info("Peer connected", zap.String("peer_id", peerID))
	
	// Send open message
	openMsg := Message{
		Type: "OPEN",
	}
	
	// Record signal sent
	s.metrics.RecordSignalSent()
	
	startTime := time.Now()
	if err := conn.WriteJSON(openMsg); err != nil {
		s.logger.Error("Failed to send open message", zap.Error(err), zap.String("peer_id", peerID))
		s.metrics.RecordSignalError()
		conn.Close()
		return
	}
	
	// Record signal latency
	s.metrics.RecordSignalLatency(time.Since(startTime))
	
	// Handle messages
	go s.handleMessagesWithMetrics(peer)
}

// handleMessagesWithMetrics handles messages from a peer with metrics
func (s *PeerServerWithMetrics) handleMessagesWithMetrics(peer *Peer) {
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
		
		// Record peer disconnection
		s.metrics.RecordPeerDisconnected(peer.ID)
		
		s.logger.Info("Peer disconnected", zap.String("peer_id", peer.ID))
	}()
	
	for {
		// Read message
		_, data, err := peer.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("Unexpected close error", zap.Error(err), zap.String("peer_id", peer.ID))
				s.metrics.RecordPeerError()
			}
			break
		}
		
		// Record message received
		s.metrics.RecordMessageReceived(len(data))
		
		// Parse message
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			s.logger.Error("Failed to parse message", zap.Error(err), zap.String("peer_id", peer.ID))
			s.metrics.RecordMessageError()
			continue
		}
		
		// Update last seen
		peer.mu.Lock()
		peer.LastSeen = time.Now()
		peer.mu.Unlock()
		
		// Handle message based on type
		switch msg.Type {
		case "HEARTBEAT":
			// Record signal received
			s.metrics.RecordSignalReceived()
			
			// Send heartbeat response
			heartbeatMsg := Message{
				Type: "HEARTBEAT",
			}
			
			// Record signal sent
			s.metrics.RecordSignalSent()
			
			startTime := time.Now()
			if err := peer.Conn.WriteJSON(heartbeatMsg); err != nil {
				s.logger.Error("Failed to send heartbeat", zap.Error(err), zap.String("peer_id", peer.ID))
				s.metrics.RecordSignalError()
				return
			}
			
			// Record signal latency
			s.metrics.RecordSignalLatency(time.Since(startTime))
			
		case "OFFER", "ANSWER", "CANDIDATE":
			// Record signal received
			s.metrics.RecordSignalReceived()
			
			// Forward message to destination peer
			if msg.Dst == "" {
				s.logger.Error("Missing destination", zap.String("peer_id", peer.ID), zap.String("type", msg.Type))
				s.metrics.RecordSignalError()
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
				
				// Record signal sent
				s.metrics.RecordSignalSent()
				
				startTime := time.Now()
				if err := peer.Conn.WriteJSON(errorMsg); err != nil {
					s.logger.Error("Failed to send error", zap.Error(err), zap.String("peer_id", peer.ID))
					s.metrics.RecordSignalError()
					return
				}
				
				// Record signal latency
				s.metrics.RecordSignalLatency(time.Since(startTime))
				continue
			}
			
			// Set source
			msg.Src = peer.ID
			
			// Record signal sent
			s.metrics.RecordSignalSent()
			
			// Forward message
			startTime := time.Now()
			if err := dstPeer.Conn.WriteJSON(msg); err != nil {
				s.logger.Error("Failed to forward message", zap.Error(err), zap.String("peer_id", peer.ID), zap.String("dst_peer_id", msg.Dst))
				s.metrics.RecordSignalError()
				continue
			}
			
			// Record signal latency
			s.metrics.RecordSignalLatency(time.Since(startTime))
			
		case "LEAVE":
			// Record signal received
			s.metrics.RecordSignalReceived()
			
			// Handle leave message
			if msg.Dst == "" {
				s.logger.Error("Missing destination", zap.String("peer_id", peer.ID), zap.String("type", msg.Type))
				s.metrics.RecordSignalError()
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
				
				// Record signal sent
				s.metrics.RecordSignalSent()
				
				startTime := time.Now()
				if err := dstPeer.Conn.WriteJSON(leaveMsg); err != nil {
					s.logger.Error("Failed to forward leave message", zap.Error(err), zap.String("peer_id", peer.ID), zap.String("dst_peer_id", msg.Dst))
					s.metrics.RecordSignalError()
					continue
				}
				
				// Record signal latency
				s.metrics.RecordSignalLatency(time.Since(startTime))
			}
			
		default:
			s.logger.Warn("Unknown message type", zap.String("type", msg.Type), zap.String("peer_id", peer.ID))
		}
	}
}

// CleanupInactivePeersWithMetrics removes inactive peers with metrics
func (s *PeerServerWithMetrics) CleanupInactivePeersWithMetrics(timeout time.Duration) {
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
			
			// Record peer disconnection
			s.metrics.RecordPeerDisconnected(id)
			
			// Remove from peers map
			delete(s.peers, id)
			
			s.logger.Info("Removed inactive peer", zap.String("peer_id", id))
		}
	}
}

// StartCleanupTaskWithMetrics starts a periodic task to clean up inactive peers with metrics
func (s *PeerServerWithMetrics) StartCleanupTaskWithMetrics(interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.CleanupInactivePeersWithMetrics(timeout)
		}
	}()
}

// GetPeerCount returns the number of connected peers
func (s *PeerServerWithMetrics) GetPeerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.peers)
}

// GetPeer gets a peer by ID
func (s *PeerServerWithMetrics) GetPeer(peerID string) (*Peer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	peer, ok := s.peers[peerID]
	return peer, ok
}

