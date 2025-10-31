// Package interfaces provides message dispatch abstractions for unified messaging
package interfaces

import (
	"context"
	"time"
)

// Message represents a generic message in the system
type Message interface {
	// GetType returns the message type identifier
	GetType() string
	
	// GetPayload returns the message payload
	GetPayload() interface{}
	
	// GetTimestamp returns when the message was created
	GetTimestamp() time.Time
	
	// GetMetadata returns message metadata
	GetMetadata() map[string]interface{}
}

// MessageHandler handles incoming messages
type MessageHandler interface {
	// Handle processes a message and returns an error if processing fails
	Handle(ctx context.Context, message Message) error
	
	// GetHandlerID returns a unique identifier for this handler
	GetHandlerID() string
	
	// CanHandle checks if this handler can process the given message type
	CanHandle(messageType string) bool
}

// MessageFilter filters messages based on criteria
type MessageFilter interface {
	// ShouldProcess determines if a message should be processed
	ShouldProcess(message Message) bool
	
	// GetFilterID returns a unique identifier for this filter
	GetFilterID() string
}

// MessageTransformer transforms messages before dispatch
type MessageTransformer interface {
	// Transform modifies a message and returns the transformed version
	Transform(ctx context.Context, message Message) (Message, error)
	
	// GetTransformerID returns a unique identifier for this transformer
	GetTransformerID() string
}

// DispatchMode defines how messages should be dispatched
type DispatchMode int

const (
	// DispatchSync processes messages synchronously
	DispatchSync DispatchMode = iota
	
	// DispatchAsync processes messages asynchronously
	DispatchAsync
	
	// DispatchBroadcast sends messages to all matching handlers
	DispatchBroadcast
	
	// DispatchRoundRobin distributes messages among handlers
	DispatchRoundRobin
)

// DispatchOptions configures message dispatch behavior
type DispatchOptions struct {
	Mode         DispatchMode
	Timeout      time.Duration
	RetryCount   int
	RetryDelay   time.Duration
	Filters      []MessageFilter
	Transformers []MessageTransformer
}

// MessageDispatcher defines the core message dispatch interface
type MessageDispatcher interface {
	// Dispatch sends a message to registered handlers
	Dispatch(ctx context.Context, message Message, options *DispatchOptions) error
	
	// Subscribe registers a handler for specific message types
	Subscribe(messageTypes []string, handler MessageHandler) error
	
	// Unsubscribe removes a handler from specific message types
	Unsubscribe(messageTypes []string, handlerID string) error
	
	// GetSubscribers returns handlers for a message type
	GetSubscribers(messageType string) []MessageHandler
	
	// AddFilter adds a global message filter
	AddFilter(filter MessageFilter)
	
	// RemoveFilter removes a global message filter
	RemoveFilter(filterID string)
	
	// AddTransformer adds a global message transformer
	AddTransformer(transformer MessageTransformer)
	
	// RemoveTransformer removes a global message transformer
	RemoveTransformer(transformerID string)
	
	// GetStats returns dispatch statistics
	GetStats() DispatchStats
}

// DispatchStats provides statistics about message dispatch
type DispatchStats struct {
	TotalMessages     int64
	SuccessfulMessages int64
	FailedMessages    int64
	AverageLatency    time.Duration
	HandlerCount      int
	FilterCount       int
	TransformerCount  int
}

// BroadcastDispatcher extends MessageDispatcher for broadcasting capabilities
type BroadcastDispatcher interface {
	MessageDispatcher
	
	// Broadcast sends a message to all connected clients/subscribers
	Broadcast(ctx context.Context, message Message, channel string) error
	
	// BroadcastToUsers sends a message to specific users
	BroadcastToUsers(ctx context.Context, message Message, userIDs []string) error
	
	// Subscribe a client to a channel
	SubscribeToChannel(clientID, channel string) error
	
	// Unsubscribe a client from a channel
	UnsubscribeFromChannel(clientID, channel string) error
	
	// GetChannelSubscribers returns subscribers for a channel
	GetChannelSubscribers(channel string) []string
}

// StreamDispatcher extends MessageDispatcher for streaming capabilities
type StreamDispatcher interface {
	MessageDispatcher
	
	// StartStream begins streaming messages to a client
	StartStream(ctx context.Context, clientID string, messageTypes []string) (<-chan Message, error)
	
	// StopStream stops streaming to a client
	StopStream(clientID string) error
	
	// GetActiveStreams returns active streaming clients
	GetActiveStreams() []string
}

// QueueDispatcher extends MessageDispatcher for queue-based dispatch
type QueueDispatcher interface {
	MessageDispatcher
	
	// Enqueue adds a message to the dispatch queue
	Enqueue(ctx context.Context, message Message, priority int) error
	
	// GetQueueSize returns the current queue size
	GetQueueSize() int
	
	// GetQueueStats returns queue statistics
	GetQueueStats() QueueStats
}

// QueueStats provides statistics about message queues
type QueueStats struct {
	QueueSize        int
	ProcessedCount   int64
	AverageWaitTime  time.Duration
	PeakQueueSize    int
	WorkerCount      int
}

// UnifiedDispatcher combines all dispatch capabilities
type UnifiedDispatcher interface {
	MessageDispatcher
	BroadcastDispatcher
	StreamDispatcher
	QueueDispatcher
	
	// Start initializes the dispatcher
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the dispatcher
	Stop(ctx context.Context) error
	
	// Health returns the health status of the dispatcher
	Health() DispatcherHealth
}

// DispatcherHealth represents the health status of a dispatcher
type DispatcherHealth struct {
	Status      string
	Uptime      time.Duration
	LastError   error
	ErrorCount  int64
	IsHealthy   bool
}
