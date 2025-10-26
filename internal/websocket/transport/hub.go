package websocket

import (
	"encoding/json"
	"sync"

	"go.uber.org/zap"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Clients is a map of client ID to client
	Clients map[string]*Client

	// register is a channel for registering clients
	register chan *Client

	// unregister is a channel for unregistering clients
	unregister chan *Client

	// broadcast is a channel for broadcasting messages to all clients
	broadcast chan *Message

	// MessageHandlers is a map of message type to handler
	MessageHandlers map[string]MessageHandler

	// Logger is the logger for the hub
	Logger *zap.Logger

	// Mutex for protecting the clients map
	mu sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	// Type is the type of the message
	Type string `json:"type"`

	// Data is the data of the message
	Data json.RawMessage `json:"data"`

	// ClientID is the ID of the client that sent the message
	ClientID string `json:"client_id,omitempty"`
}

// MessageHandler is a function that handles a message
type MessageHandler func(client *Client, msg *Message)

// NewHub creates a new hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		Clients:         make(map[string]*Client),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		broadcast:       make(chan *Message),
		MessageHandlers: make(map[string]MessageHandler),
		Logger:          logger,
	}
}

// Run runs the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			// Register the client
			h.mu.Lock()
			h.Clients[client.ID] = client
			h.mu.Unlock()

			h.Logger.Info("Client registered", zap.String("client_id", client.ID))

		case client := <-h.unregister:
			// Unregister the client
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()

			h.Logger.Info("Client unregistered", zap.String("client_id", client.ID))

		case message := <-h.broadcast:
			// Broadcast the message to all clients
			h.mu.RLock()
			for _, client := range h.Clients {
				select {
				case client.Send <- h.serializeMessage(message):
				default:
					// Client's send buffer is full, unregister the client
					h.mu.RUnlock()
					h.Unregister(client)
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register registers a client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast broadcasts a message to all clients
func (h *Hub) Broadcast(message *Message) {
	h.broadcast <- message
}

// RegisterMessageHandler registers a message handler
func (h *Hub) RegisterMessageHandler(messageType string, handler MessageHandler) {
	h.MessageHandlers[messageType] = handler
}

// HandleMessage handles a message
func (h *Hub) HandleMessage(client *Client, msg *Message) {
	// Set the client ID
	msg.ClientID = client.ID

	// Find the handler for the message type
	handler, ok := h.MessageHandlers[msg.Type]
	if !ok {
		h.Logger.Warn("No handler for message type", zap.String("type", msg.Type))
		return
	}

	// Handle the message
	handler(client, msg)
}

// serializeMessage serializes a message to JSON
func (h *Hub) serializeMessage(msg *Message) []byte {
	payload, err := json.Marshal(msg)
	if err != nil {
		h.Logger.Error("Failed to marshal message", zap.Error(err))
		return []byte{}
	}

	return payload
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, msg *Message) {
	// Find the client
	h.mu.RLock()
	client, ok := h.Clients[clientID]
	h.mu.RUnlock()

	if !ok {
		h.Logger.Warn("Client not found", zap.String("client_id", clientID))
		return
	}

	// Send the message
	client.SendMessage(msg)
}

// BroadcastToClients broadcasts a message to specific clients
func (h *Hub) BroadcastToClients(clientIDs []string, msg *Message) {
	// Find the clients
	h.mu.RLock()
	for _, clientID := range clientIDs {
		client, ok := h.Clients[clientID]
		if !ok {
			h.Logger.Warn("Client not found", zap.String("client_id", clientID))
			continue
		}

		// Send the message
		client.SendMessage(msg)
	}
	h.mu.RUnlock()
}
