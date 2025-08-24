package websocket

import (
	"encoding/json"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"github.com/abdoElHodaky/tradSys/internal/performance"
	"go.uber.org/zap"
)

// OptimizedHub extends Hub with optimized message handling
type OptimizedHub struct {
	*Hub
	
	// Message batcher for each priority
	batchers map[performance.MessagePriority]*performance.MessageBatcher
	
	// Message compressor
	compressor *performance.MessageCompressor
	
	// Connection pool
	connectionPool *performance.ConnectionPool
	
	// Metrics
	metrics *metrics.WebSocketMetrics
}

// OptimizedHubConfig contains configuration for the optimized hub
type OptimizedHubConfig struct {
	// BatcherConfig is the configuration for the message batcher
	BatcherConfig performance.MessageBatcherConfig
	
	// CompressorConfig is the configuration for the message compressor
	CompressorConfig performance.MessageCompressorConfig
	
	// ConnectionPoolConfig is the configuration for the connection pool
	ConnectionPoolConfig performance.ConnectionPoolConfig
}

// DefaultOptimizedHubConfig returns the default configuration
func DefaultOptimizedHubConfig() OptimizedHubConfig {
	return OptimizedHubConfig{
		BatcherConfig:        performance.DefaultMessageBatcherConfig(),
		CompressorConfig:     performance.DefaultMessageCompressorConfig(),
		ConnectionPoolConfig: performance.DefaultConnectionPoolConfig(),
	}
}

// NewOptimizedHub creates a new optimized hub
func NewOptimizedHub(logger *zap.Logger, metrics *metrics.WebSocketMetrics, config OptimizedHubConfig) *OptimizedHub {
	// Create the base hub
	hub := NewHub(logger)
	
	// Create the optimized hub
	optimizedHub := &OptimizedHub{
		Hub:     hub,
		batchers: make(map[performance.MessagePriority]*performance.MessageBatcher),
		metrics: metrics,
	}
	
	// Create the message compressor
	optimizedHub.compressor = performance.NewMessageCompressor(
		config.CompressorConfig,
		logger,
		metrics,
	)
	
	// Create the message batchers
	for priority := performance.PriorityLow; priority <= performance.PriorityCritical; priority++ {
		optimizedHub.batchers[priority] = performance.NewMessageBatcher(
			config.BatcherConfig,
			func(messages []performance.BatchableMessage) error {
				return optimizedHub.sendBatch(messages)
			},
			logger,
			metrics,
		)
	}
	
	// Create the connection pool
	optimizedHub.connectionPool = performance.NewConnectionPool(
		config.ConnectionPoolConfig,
		func() (*websocket.Conn, error) {
			// This is a placeholder - in a real implementation, this would create a new connection
			return nil, errors.New("connection creation not implemented")
		},
		logger,
		metrics,
	)
	
	return optimizedHub
}

// BroadcastOptimized broadcasts a message to all clients with optimization
func (h *OptimizedHub) BroadcastOptimized(message *Message, priority performance.MessagePriority) {
	// Convert the message to a batchable message
	data, err := json.Marshal(message)
	if err != nil {
		h.Logger.Error("Failed to marshal message", zap.Error(err))
		return
	}
	
	batchableMessage := performance.BatchableMessage{
		Type:      message.Type,
		Data:      message.Data,
		Priority:  priority,
		Timestamp: time.Now(),
		Size:      len(data),
	}
	
	// Add the message to the batcher
	h.batchers[priority].AddMessage(batchableMessage)
}

// SendToClientOptimized sends a message to a specific client with optimization
func (h *OptimizedHub) SendToClientOptimized(clientID string, msg *Message, priority performance.MessagePriority) {
	// Find the client
	h.mu.RLock()
	client, ok := h.Clients[clientID]
	h.mu.RUnlock()
	
	if !ok {
		h.Logger.Warn("Client not found", zap.String("client_id", clientID))
		return
	}
	
	// Convert the message to a batchable message
	data, err := json.Marshal(msg)
	if err != nil {
		h.Logger.Error("Failed to marshal message", zap.Error(err))
		return
	}
	
	batchableMessage := performance.BatchableMessage{
		Type:      msg.Type,
		Data:      msg.Data,
		Priority:  priority,
		Timestamp: time.Now(),
		Size:      len(data),
	}
	
	// For critical messages, send immediately
	if priority == performance.PriorityCritical {
		h.sendMessageToClient(client, batchableMessage)
		return
	}
	
	// Add the message to the batcher
	h.batchers[priority].AddMessage(batchableMessage)
}

// sendBatch sends a batch of messages
func (h *OptimizedHub) sendBatch(messages []performance.BatchableMessage) error {
	// Group messages by client
	clientMessages := make(map[string][]performance.BatchableMessage)
	
	h.mu.RLock()
	for _, message := range messages {
		// Convert the message to a Message
		msg := &Message{
			Type: message.Type,
			Data: message.Data,
		}
		
		// Compress the message if it's large enough
		compressedData, err := h.compressor.CompressMessage(message.Data, message.Type)
		if err != nil {
			h.Logger.Error("Failed to compress message", zap.Error(err))
			continue
		}
		
		// Update the message data
		msg.Data = compressedData
		
		// Broadcast to all clients
		for clientID, client := range h.Clients {
			if _, ok := clientMessages[clientID]; !ok {
				clientMessages[clientID] = make([]performance.BatchableMessage, 0, len(messages))
			}
			
			// Add the message to the client's messages
			clientMessages[clientID] = append(clientMessages[clientID], performance.BatchableMessage{
				Type:      msg.Type,
				Data:      msg.Data,
				Priority:  message.Priority,
				Timestamp: message.Timestamp,
				Size:      len(msg.Data),
			})
		}
	}
	h.mu.RUnlock()
	
	// Send messages to each client
	for clientID, messages := range clientMessages {
		h.mu.RLock()
		client, ok := h.Clients[clientID]
		h.mu.RUnlock()
		
		if !ok {
			continue
		}
		
		// Send the messages to the client
		for _, message := range messages {
			h.sendMessageToClient(client, message)
		}
	}
	
	return nil
}

// sendMessageToClient sends a message to a client
func (h *OptimizedHub) sendMessageToClient(client *Client, message performance.BatchableMessage) {
	// Create the message
	msg := &Message{
		Type: message.Type,
		Data: message.Data,
	}
	
	// Send the message
	client.SendMessage(msg)
}

// Stop stops the optimized hub
func (h *OptimizedHub) Stop() {
	// Stop the batchers
	for _, batcher := range h.batchers {
		batcher.Stop()
	}
	
	// Close the connection pool
	h.connectionPool.Close()
}

