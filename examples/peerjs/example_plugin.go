package main

import (
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/peerjs"
	"github.com/abdoElHodaky/tradSys/internal/peerjs/plugin"
	"go.uber.org/zap"
)

// PluginInfo is the exported plugin information
var PluginInfo = &plugin.PluginInfo{
	Name:        "ExamplePlugin",
	Version:     "1.0.0",
	Description: "An example PeerJS plugin",
}

// CreatePlugin is the exported function to create a plugin
func CreatePlugin() plugin.PeerJSPlugin {
	return &ExamplePlugin{}
}

// ExamplePlugin implements the PeerJSPlugin interface
type ExamplePlugin struct {
	server *peerjs.PeerServer
	logger *zap.Logger
	peers  map[string]time.Time
}

// Initialize initializes the plugin
func (p *ExamplePlugin) Initialize(server *peerjs.PeerServer, logger *zap.Logger) error {
	p.server = server
	p.logger = logger
	p.peers = make(map[string]time.Time)
	
	p.logger.Info("Example PeerJS plugin initialized")
	return nil
}

// GetName returns the name of the plugin
func (p *ExamplePlugin) GetName() string {
	return PluginInfo.Name
}

// GetVersion returns the version of the plugin
func (p *ExamplePlugin) GetVersion() string {
	return PluginInfo.Version
}

// GetDescription returns the description of the plugin
func (p *ExamplePlugin) GetDescription() string {
	return PluginInfo.Description
}

// OnPeerConnected is called when a peer connects
func (p *ExamplePlugin) OnPeerConnected(peerID string) {
	p.logger.Info("Peer connected",
		zap.String("plugin", p.GetName()),
		zap.String("peer_id", peerID))
	
	p.peers[peerID] = time.Now()
	
	// Send a welcome message to the peer
	welcomeMsg := &peerjs.Message{
		Type: "plugin.welcome",
		Dst:  peerID,
		Payload: map[string]interface{}{
			"message": fmt.Sprintf("Welcome from %s plugin!", p.GetName()),
			"time":    time.Now().Format(time.RFC3339),
		},
	}
	
	// Use the server to send the message
	// Note: This is a simplified example, actual implementation would depend on the PeerServer API
	if peer, ok := p.server.GetPeer(peerID); ok {
		peer.SendMessage(welcomeMsg)
	}
}

// OnPeerDisconnected is called when a peer disconnects
func (p *ExamplePlugin) OnPeerDisconnected(peerID string) {
	p.logger.Info("Peer disconnected",
		zap.String("plugin", p.GetName()),
		zap.String("peer_id", peerID))
	
	if connectTime, ok := p.peers[peerID]; ok {
		sessionDuration := time.Since(connectTime)
		p.logger.Info("Peer session ended",
			zap.String("plugin", p.GetName()),
			zap.String("peer_id", peerID),
			zap.Duration("session_duration", sessionDuration))
		
		delete(p.peers, peerID)
	}
}

// OnMessage is called when a message is received
func (p *ExamplePlugin) OnMessage(msg *peerjs.Message) bool {
	// Check if this is a message for our plugin
	if msg.Type == "plugin.example.ping" {
		p.logger.Info("Received ping message",
			zap.String("plugin", p.GetName()),
			zap.String("from", msg.Src))
		
		// Send a pong response
		pongMsg := &peerjs.Message{
			Type: "plugin.example.pong",
			Dst:  msg.Src,
			Src:  "server",
			Payload: map[string]interface{}{
				"time": time.Now().Format(time.RFC3339),
			},
		}
		
		// Use the server to send the message
		if peer, ok := p.server.GetPeer(msg.Src); ok {
			peer.SendMessage(pongMsg)
		}
		
		return true // Message handled
	}
	
	return false // Message not handled
}

