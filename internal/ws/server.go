package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Channel string      `json:"channel"`
	Symbol  string      `json:"symbol,omitempty"`
	Data    interface{} `json:"data"`
}

// Connection represents a WebSocket connection
type Connection struct {
	conn      *websocket.Conn
	send      chan []byte
	server    *Server
	symbol    string
	channels  map[string]bool
	closeOnce sync.Once
	mu        sync.RWMutex
}

// Server manages WebSocket connections
type Server struct {
	logger     *zap.Logger
	upgrader   websocket.Upgrader
	clients    map[*Connection]bool
	register   chan *Connection
	unregister chan *Connection
	broadcast  chan *Message
	mu         sync.RWMutex
}

// NewServer creates a new WebSocket server
func NewServer(logger *zap.Logger) *Server {
	return &Server{
		logger: logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, implement proper origin checks
			},
		},
		clients:    make(map[*Connection]bool),
		register:   make(chan *Connection),
		unregister: make(chan *Connection),
		broadcast:  make(chan *Message),
	}
}

// Run starts the WebSocket server
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			s.logger.Info("Client registered",
				zap.String("remote_addr", client.conn.RemoteAddr().String()))

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
			}
			s.mu.Unlock()
			s.logger.Info("Client unregistered",
				zap.String("remote_addr", client.conn.RemoteAddr().String()))

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				// Check if client is subscribed to this channel and symbol
				client.mu.RLock()
				subscribed := client.channels[message.Channel]
				if message.Symbol != "" {
					subscribed = subscribed && (client.symbol == "" || client.symbol == message.Symbol)
				}
				client.mu.RUnlock()

				if subscribed {
					// Serialize the message
					data, err := json.Marshal(message)
					if err != nil {
						s.logger.Error("Failed to marshal message", zap.Error(err))
						continue
					}

					select {
					case client.send <- data:
					default:
						s.mu.RUnlock()
						s.unregister <- client
						s.mu.RLock()
					}
				}
			}
			s.mu.RUnlock()
		}
	}
}

// ServeWs handles WebSocket requests from clients
func (s *Server) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	client := &Connection{
		conn:     conn,
		send:     make(chan []byte, 256),
		server:   s,
		symbol:   r.URL.Query().Get("symbol"),
		channels: make(map[string]bool),
	}

	// Subscribe to default channels
	client.channels["heartbeat"] = true

	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// writePump pumps messages from the hub to the websocket connection
func (c *Connection) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.closeOnce.Do(func() {
			c.conn.Close()
		})
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.server.logger.Error("Failed to write message", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.server.logger.Error("Failed to write ping", zap.Error(err))
				return
			}

			// Send heartbeat
			heartbeat := Message{
				Type:    "heartbeat",
				Channel: "heartbeat",
				Data:    time.Now().Unix(),
			}
			data, err := json.Marshal(heartbeat)
			if err != nil {
				c.server.logger.Error("Failed to marshal heartbeat", zap.Error(err))
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c.server.logger.Error("Failed to write heartbeat", zap.Error(err))
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Connection) readPump() {
	defer func() {
		c.server.unregister <- c
		c.closeOnce.Do(func() {
			c.conn.Close()
		})
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.server.logger.Error("Unexpected close error", zap.Error(err))
			}
			break
		}

		// Process the message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			c.server.logger.Error("Failed to unmarshal message", zap.Error(err))
			continue
		}

		// Handle subscription messages
		if msg.Type == "subscribe" {
			channel, ok := msg.Data.(string)
			if !ok {
				c.server.logger.Error("Invalid subscription data", zap.Any("data", msg.Data))
				continue
			}

			c.mu.Lock()
			c.channels[channel] = true
			c.mu.Unlock()

			// Send confirmation
			confirmation := Message{
				Type:    "subscribed",
				Channel: channel,
				Symbol:  c.symbol,
				Data:    fmt.Sprintf("Subscribed to %s", channel),
			}
			data, err := json.Marshal(confirmation)
			if err != nil {
				c.server.logger.Error("Failed to marshal confirmation", zap.Error(err))
				continue
			}

			c.send <- data
		} else if msg.Type == "unsubscribe" {
			channel, ok := msg.Data.(string)
			if !ok {
				c.server.logger.Error("Invalid unsubscription data", zap.Any("data", msg.Data))
				continue
			}

			c.mu.Lock()
			delete(c.channels, channel)
			c.mu.Unlock()

			// Send confirmation
			confirmation := Message{
				Type:    "unsubscribed",
				Channel: channel,
				Symbol:  c.symbol,
				Data:    fmt.Sprintf("Unsubscribed from %s", channel),
			}
			data, err := json.Marshal(confirmation)
			if err != nil {
				c.server.logger.Error("Failed to marshal confirmation", zap.Error(err))
				continue
			}

			c.send <- data
		}
	}
}

// Broadcast sends a message to all subscribed clients
func (s *Server) Broadcast(message *Message) {
	s.broadcast <- message
}

