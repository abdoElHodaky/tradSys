package peerjs

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a PeerJS message
type Message struct {
	Type    string      `json:"type"`
	Src     string      `json:"src,omitempty"`
	Dst     string      `json:"dst,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// Peer represents a connected peer
type Peer struct {
	ID        string
	Token     string
	Conn      *websocket.Conn
	LastSeen  time.Time
	Connected bool
	mu        sync.RWMutex
}

// PeerServerOptions represents configuration options for the PeerJS server
type PeerServerOptions struct {
	Port            int    `json:"port"`
	Path            string `json:"path"`
	Key             string `json:"key"`
	AllowDiscovery  bool   `json:"allow_discovery"`
	ProxiedRequests bool   `json:"proxied_requests"`
	CleanupOutMsgs  int    `json:"cleanup_out_msgs"`
}
