package peerjs

import (
	"encoding/json"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"go.uber.org/zap"
)

// PeerWithMetrics extends Peer with metrics collection
type PeerWithMetrics struct {
	*Peer
	metrics *metrics.PeerJSMetrics
	logger  *zap.Logger
}

// NewPeerWithMetrics creates a new peer with metrics
func NewPeerWithMetrics(peer *Peer, metrics *metrics.PeerJSMetrics, logger *zap.Logger) *PeerWithMetrics {
	return &PeerWithMetrics{
		Peer:    peer,
		metrics: metrics,
		logger:  logger,
	}
}

// SendMessage sends a message to the peer with metrics
func (p *PeerWithMetrics) SendMessage(msg *Message) error {
	// Marshal the message to get its size
	data, err := json.Marshal(msg)
	if err != nil {
		p.logger.Error("Failed to marshal message", zap.Error(err))
		p.metrics.RecordMessageError()
		return err
	}
	
	// Record message sent
	p.metrics.RecordMessageSent(len(data))
	
	// Send the message
	startTime := time.Now()
	err = p.Conn.WriteJSON(msg)
	if err != nil {
		p.logger.Error("Failed to send message", zap.Error(err))
		p.metrics.RecordMessageError()
		return err
	}
	
	// Record message latency
	p.metrics.RecordMessageLatency(time.Since(startTime))
	
	return nil
}

// UpdateLastSeen updates the last seen time with metrics
func (p *PeerWithMetrics) UpdateLastSeen() {
	p.mu.Lock()
	p.LastSeen = time.Now()
	p.mu.Unlock()
}

// IsConnected checks if the peer is connected
func (p *PeerWithMetrics) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Connected
}

// Disconnect disconnects the peer with metrics
func (p *PeerWithMetrics) Disconnect() {
	p.mu.Lock()
	wasConnected := p.Connected
	p.Connected = false
	p.mu.Unlock()
	
	if wasConnected {
		p.Conn.Close()
		p.metrics.RecordPeerDisconnected(p.ID)
	}
}

// MessageHandler is a function that handles a message
type MessageHandler func(peer *Peer, msg *Message) error

// MessageHandlerWithMetrics wraps a message handler with metrics
func MessageHandlerWithMetrics(handler MessageHandler, metrics *metrics.PeerJSMetrics, logger *zap.Logger) MessageHandler {
	return func(peer *Peer, msg *Message) error {
		// Record message received
		data, _ := json.Marshal(msg)
		metrics.RecordMessageReceived(len(data))
		
		// Handle the message
		startTime := time.Now()
		err := handler(peer, msg)
		
		// Record message latency
		metrics.RecordMessageLatency(time.Since(startTime))
		
		// Record error if any
		if err != nil {
			logger.Error("Message handler error", zap.Error(err))
			metrics.RecordMessageError()
		}
		
		return err
	}
}

