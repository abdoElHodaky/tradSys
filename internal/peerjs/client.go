package peerjs

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/proto/ws"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// PeerClient represents a client for the PeerJS server
type PeerClient struct {
	logger    *zap.Logger
	serverURL string
	peerID    string
	conn      *websocket.Conn
	connected bool
	peers     map[string]*PeerConnection
	handlers  map[string]MessageHandler
	closeCh   chan struct{}
	mu        sync.RWMutex
}

// PeerConnection represents a connection to another peer
type PeerConnection struct {
	ID        string
	Connected bool
	LastSeen  time.Time
	DataCh    chan []byte
	mu        sync.RWMutex
}



// MessageHandler is a function that handles a message
type MessageHandler func(msg Message) error

// NewPeerClient creates a new PeerJS client
func NewPeerClient(logger *zap.Logger, serverURL, peerID string) *PeerClient {
	return &PeerClient{
		logger:    logger,
		serverURL: serverURL,
		peerID:    peerID,
		peers:     make(map[string]*PeerConnection),
		handlers:  make(map[string]MessageHandler),
		closeCh:   make(chan struct{}),
	}
}

// Connect connects to the PeerJS server
func (c *PeerClient) Connect() error {
	// Connect to server
	conn, _, err := websocket.DefaultDialer.Dial(c.serverURL+"?id="+c.peerID, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	// Start message handler
	go c.handleMessages()

	// Start heartbeat
	go c.sendHeartbeats()

	c.logger.Info("Connected to PeerJS server", zap.String("server_url", c.serverURL), zap.String("peer_id", c.peerID))

	return nil
}

// Disconnect disconnects from the PeerJS server
func (c *PeerClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// Close connection
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	// Signal close
	close(c.closeCh)

	c.connected = false

	c.logger.Info("Disconnected from PeerJS server")

	return nil
}

// handleMessages handles messages from the server
func (c *PeerClient) handleMessages() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
	}()

	for {
		// Check if closed
		select {
		case <-c.closeCh:
			return
		default:
		}

		// Read message
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("Unexpected close error", zap.Error(err))
			}
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Error("Failed to parse message", zap.Error(err))
			continue
		}

		// Handle message based on type
		switch msg.Type {
		case "OPEN":
			c.logger.Info("Received OPEN message")

		case "HEARTBEAT":
			// Heartbeat response, nothing to do

		case "OFFER", "ANSWER", "CANDIDATE":
			// Handle connection negotiation
			if msg.Src == "" {
				c.logger.Error("Missing source", zap.String("type", msg.Type))
				continue
			}

			// Get or create peer connection
			c.mu.Lock()
			peer, ok := c.peers[msg.Src]
			if !ok {
				peer = &PeerConnection{
					ID:       msg.Src,
					DataCh:   make(chan []byte, 100),
					LastSeen: time.Now(),
				}
				c.peers[msg.Src] = peer
			}
			c.mu.Unlock()

			// Update last seen
			peer.mu.Lock()
			peer.LastSeen = time.Now()
			peer.mu.Unlock()

			// Handle message with appropriate handler
			c.mu.RLock()
			handler, ok := c.handlers[msg.Type]
			c.mu.RUnlock()

			if ok {
				if err := handler(msg); err != nil {
					c.logger.Error("Failed to handle message", zap.Error(err), zap.String("type", msg.Type))
				}
			} else {
				c.logger.Warn("No handler for message type", zap.String("type", msg.Type))
			}

		case "LEAVE":
			// Handle peer disconnect
			if msg.Src == "" {
				c.logger.Error("Missing source", zap.String("type", msg.Type))
				continue
			}

			c.mu.Lock()
			peer, ok := c.peers[msg.Src]
			if ok {
				peer.mu.Lock()
				peer.Connected = false
				peer.mu.Unlock()
				delete(c.peers, msg.Src)
			}
			c.mu.Unlock()

			c.logger.Info("Peer disconnected", zap.String("peer_id", msg.Src))

		case "ERROR":
			c.logger.Error("Received error message", zap.Any("payload", msg.Payload))

		default:
			c.logger.Warn("Unknown message type", zap.String("type", msg.Type))
		}
	}
}

// sendHeartbeats sends periodic heartbeats to the server
func (c *PeerClient) sendHeartbeats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeCh:
			return
		case <-ticker.C:
			c.mu.RLock()
			connected := c.connected
			c.mu.RUnlock()

			if !connected {
				return
			}

			// Send heartbeat
			heartbeatMsg := Message{
				Type: "HEARTBEAT",
			}

			c.mu.RLock()
			err := c.conn.WriteJSON(heartbeatMsg)
			c.mu.RUnlock()

			if err != nil {
				c.logger.Error("Failed to send heartbeat", zap.Error(err))
				return
			}
		}
	}
}

// Connect to a peer
func (c *PeerClient) ConnectToPeer(peerID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	// Check if already connected
	if peer, ok := c.peers[peerID]; ok && peer.Connected {
		return nil
	}

	// Create peer connection
	peer := &PeerConnection{
		ID:       peerID,
		DataCh:   make(chan []byte, 100),
		LastSeen: time.Now(),
	}
	c.peers[peerID] = peer

	// Send offer
	offerMsg := Message{
		Type: "OFFER",
		Dst:  peerID,
		Payload: map[string]interface{}{
			"sdp": "offer_sdp", // In a real implementation, this would be a WebRTC SDP offer
		},
	}

	if err := c.conn.WriteJSON(offerMsg); err != nil {
		delete(c.peers, peerID)
		return fmt.Errorf("failed to send offer: %w", err)
	}

	c.logger.Info("Sent connection offer", zap.String("peer_id", peerID))

	return nil
}

// Disconnect from a peer
func (c *PeerClient) DisconnectFromPeer(peerID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	// Check if connected
	peer, ok := c.peers[peerID]
	if !ok || !peer.Connected {
		return nil
	}

	// Send leave message
	leaveMsg := Message{
		Type: "LEAVE",
		Dst:  peerID,
	}

	if err := c.conn.WriteJSON(leaveMsg); err != nil {
		return fmt.Errorf("failed to send leave message: %w", err)
	}

	// Update peer state
	peer.mu.Lock()
	peer.Connected = false
	peer.mu.Unlock()

	// Remove peer
	delete(c.peers, peerID)

	c.logger.Info("Disconnected from peer", zap.String("peer_id", peerID))

	return nil
}

// SendToPeer sends a message to a peer
func (c *PeerClient) SendToPeer(peerID string, data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	// Check if connected to peer
	peer, ok := c.peers[peerID]
	if !ok || !peer.Connected {
		return fmt.Errorf("not connected to peer %s", peerID)
	}

	// In a real implementation, this would use WebRTC data channels
	// For now, we'll just simulate it

	c.logger.Debug("Sent data to peer", zap.String("peer_id", peerID), zap.Int("data_size", len(data)))

	return nil
}

// SendMarketDataToPeer sends market data to a peer
func (c *PeerClient) SendMarketDataToPeer(peerID string, marketData *ws.MarketDataPayload) error {
	// Create WebSocket message
	message := &ws.WebSocketMessage{
		Type:      "marketData",
		Channel:   "marketData",
		Symbol:    marketData.Symbol,
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WebSocketMessage_MarketData{
			MarketData: marketData,
		},
	}

	// Serialize message
	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send to peer
	return c.SendToPeer(peerID, data)
}

// RegisterHandler registers a handler for a message type
func (c *PeerClient) RegisterHandler(messageType string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[messageType] = handler
}

// GetConnectedPeers returns a list of connected peers
func (c *PeerClient) GetConnectedPeers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var peers []string
	for id, peer := range c.peers {
		peer.mu.RLock()
		connected := peer.Connected
		peer.mu.RUnlock()

		if connected {
			peers = append(peers, id)
		}
	}

	return peers
}
