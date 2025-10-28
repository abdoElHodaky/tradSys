package websocket

import (
	"encoding/json"
	"sync"

	"go.uber.org/zap"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Message handlers
	messageHandlers map[string]MessageHandler

	// Symbol subscriptions
	symbolSubscriptions map[string]map[*Client]bool

	// Logger
	logger *zap.Logger

	// Mutex for thread safety
	mu sync.RWMutex
}

// MessageHandler defines the signature for message handlers
type MessageHandler func(client *Client, message *Message)

// Message represents a WebSocket message
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// NewHub creates a new WebSocket hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:             make(map[*Client]bool),
		broadcast:           make(chan []byte),
		register:            make(chan *Client),
		unregister:          make(chan *Client),
		messageHandlers:     make(map[string]MessageHandler),
		symbolSubscriptions: make(map[string]map[*Client]bool),
		logger:              logger,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("Client registered", zap.String("client_id", client.ID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				
				// Remove from all symbol subscriptions
				for symbol, subscribers := range h.symbolSubscriptions {
					if _, exists := subscribers[client]; exists {
						delete(subscribers, client)
						if len(subscribers) == 0 {
							delete(h.symbolSubscriptions, symbol)
						}
					}
				}
			}
			h.mu.Unlock()
			h.logger.Info("Client unregistered", zap.String("client_id", client.ID))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// RegisterClient registers a new client
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// RegisterMessageHandler registers a message handler for a specific message type
func (h *Hub) RegisterMessageHandler(messageType string, handler MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messageHandlers[messageType] = handler
}

// HandleMessage handles an incoming message from a client
func (h *Hub) HandleMessage(client *Client, messageData []byte) {
	var msg Message
	if err := json.Unmarshal(messageData, &msg); err != nil {
		h.logger.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	h.mu.RLock()
	handler, exists := h.messageHandlers[msg.Type]
	h.mu.RUnlock()

	if exists {
		handler(client, &msg)
	} else {
		h.logger.Warn("No handler for message type", zap.String("type", msg.Type))
	}
}

// SubscribeToSymbol subscribes a client to a symbol
func (h *Hub) SubscribeToSymbol(client *Client, symbol string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.symbolSubscriptions[symbol] == nil {
		h.symbolSubscriptions[symbol] = make(map[*Client]bool)
	}
	h.symbolSubscriptions[symbol][client] = true
}

// UnsubscribeFromSymbol unsubscribes a client from a symbol
func (h *Hub) UnsubscribeFromSymbol(client *Client, symbol string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, exists := h.symbolSubscriptions[symbol]; exists {
		delete(subscribers, client)
		if len(subscribers) == 0 {
			delete(h.symbolSubscriptions, symbol)
		}
	}
}

// BroadcastToSymbol broadcasts a message to all clients subscribed to a symbol
func (h *Hub) BroadcastToSymbol(symbol string, message []byte) {
	h.mu.RLock()
	subscribers, exists := h.symbolSubscriptions[symbol]
	h.mu.RUnlock()

	if !exists {
		return
	}

	h.mu.RLock()
	for client := range subscribers {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
	h.mu.RUnlock()
}

