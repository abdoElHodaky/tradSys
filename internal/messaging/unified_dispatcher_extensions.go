// Package messaging provides additional functionality for unified dispatcher
package messaging

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"go.uber.org/zap"
)

// Broadcast sends a message to all connected clients/subscribers
func (d *UnifiedMessageDispatcher) Broadcast(ctx context.Context, message interfaces.Message, channel string) error {
	d.channelsMux.RLock()
	subscribers, exists := d.channels[channel]
	d.channelsMux.RUnlock()
	
	if !exists || len(subscribers) == 0 {
		d.logger.Debug("No subscribers for channel", zap.String("channel", channel))
		return nil
	}
	
	d.clientsMux.RLock()
	defer d.clientsMux.RUnlock()
	
	successCount := 0
	for clientID := range subscribers {
		if clientChan, exists := d.clients[clientID]; exists {
			select {
			case clientChan <- message:
				successCount++
			case <-ctx.Done():
				return ctx.Err()
			default:
				d.logger.Warn("Client channel full, dropping message",
					zap.String("client_id", clientID),
					zap.String("channel", channel))
			}
		}
	}
	
	d.logger.Debug("Broadcast completed",
		zap.String("channel", channel),
		zap.Int("subscribers", len(subscribers)),
		zap.Int("successful", successCount))
	
	return nil
}

// BroadcastToUsers sends a message to specific users
func (d *UnifiedMessageDispatcher) BroadcastToUsers(ctx context.Context, message interfaces.Message, userIDs []string) error {
	d.clientsMux.RLock()
	defer d.clientsMux.RUnlock()
	
	successCount := 0
	for _, userID := range userIDs {
		if clientChan, exists := d.clients[userID]; exists {
			select {
			case clientChan <- message:
				successCount++
			case <-ctx.Done():
				return ctx.Err()
			default:
				d.logger.Warn("User channel full, dropping message",
					zap.String("user_id", userID))
			}
		}
	}
	
	d.logger.Debug("User broadcast completed",
		zap.Int("target_users", len(userIDs)),
		zap.Int("successful", successCount))
	
	return nil
}

// SubscribeToChannel subscribes a client to a channel
func (d *UnifiedMessageDispatcher) SubscribeToChannel(clientID, channel string) error {
	d.channelsMux.Lock()
	defer d.channelsMux.Unlock()
	
	if d.channels[channel] == nil {
		d.channels[channel] = make(map[string]bool)
	}
	
	d.channels[channel][clientID] = true
	
	d.logger.Debug("Client subscribed to channel",
		zap.String("client_id", clientID),
		zap.String("channel", channel))
	
	return nil
}

// UnsubscribeFromChannel unsubscribes a client from a channel
func (d *UnifiedMessageDispatcher) UnsubscribeFromChannel(clientID, channel string) error {
	d.channelsMux.Lock()
	defer d.channelsMux.Unlock()
	
	if subscribers, exists := d.channels[channel]; exists {
		delete(subscribers, clientID)
		
		// Clean up empty channels
		if len(subscribers) == 0 {
			delete(d.channels, channel)
		}
	}
	
	d.logger.Debug("Client unsubscribed from channel",
		zap.String("client_id", clientID),
		zap.String("channel", channel))
	
	return nil
}

// GetChannelSubscribers returns subscribers for a channel
func (d *UnifiedMessageDispatcher) GetChannelSubscribers(channel string) []string {
	d.channelsMux.RLock()
	defer d.channelsMux.RUnlock()
	
	subscribers, exists := d.channels[channel]
	if !exists {
		return []string{}
	}
	
	result := make([]string, 0, len(subscribers))
	for clientID := range subscribers {
		result = append(result, clientID)
	}
	
	return result
}

// StartStream begins streaming messages to a client
func (d *UnifiedMessageDispatcher) StartStream(ctx context.Context, clientID string, messageTypes []string) (<-chan interfaces.Message, error) {
	d.streamsMux.Lock()
	defer d.streamsMux.Unlock()
	
	// Check if stream already exists
	if _, exists := d.streams[clientID]; exists {
		return nil, fmt.Errorf("stream already exists for client: %s", clientID)
	}
	
	// Create stream channel
	streamChan := make(chan interfaces.Message, 100) // Buffered channel
	d.streams[clientID] = streamChan
	
	// Create a stream handler that forwards messages to the stream
	streamHandler := &StreamHandler{
		clientID:     clientID,
		messageTypes: messageTypes,
		streamChan:   streamChan,
		logger:       d.logger,
	}
	
	// Subscribe the stream handler to the specified message types
	if err := d.Subscribe(messageTypes, streamHandler); err != nil {
		delete(d.streams, clientID)
		close(streamChan)
		return nil, fmt.Errorf("failed to subscribe stream handler: %w", err)
	}
	
	d.logger.Debug("Stream started",
		zap.String("client_id", clientID),
		zap.Strings("message_types", messageTypes))
	
	return streamChan, nil
}

// StopStream stops streaming to a client
func (d *UnifiedMessageDispatcher) StopStream(clientID string) error {
	d.streamsMux.Lock()
	defer d.streamsMux.Unlock()
	
	streamChan, exists := d.streams[clientID]
	if !exists {
		return fmt.Errorf("no stream found for client: %s", clientID)
	}
	
	// Close and remove the stream
	close(streamChan)
	delete(d.streams, clientID)
	
	// Unsubscribe the stream handler
	// Note: In a full implementation, we'd need to track the handler ID
	// and unsubscribe it properly
	
	d.logger.Debug("Stream stopped", zap.String("client_id", clientID))
	
	return nil
}

// GetActiveStreams returns active streaming clients
func (d *UnifiedMessageDispatcher) GetActiveStreams() []string {
	d.streamsMux.RLock()
	defer d.streamsMux.RUnlock()
	
	result := make([]string, 0, len(d.streams))
	for clientID := range d.streams {
		result = append(result, clientID)
	}
	
	return result
}

// Enqueue adds a message to the dispatch queue
func (d *UnifiedMessageDispatcher) Enqueue(ctx context.Context, message interfaces.Message, priority int) error {
	queuedMsg := queuedMessage{
		message:  message,
		options:  &interfaces.DispatchOptions{Mode: interfaces.DispatchSync},
		priority: priority,
		ctx:      ctx,
	}
	
	select {
	case d.messageQueue <- queuedMsg:
		atomic.AddInt32(&d.queueStats.queueSize, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("message queue full")
	}
}

// GetQueueSize returns the current queue size
func (d *UnifiedMessageDispatcher) GetQueueSize() int {
	return int(atomic.LoadInt32(&d.queueStats.queueSize))
}

// GetQueueStats returns queue statistics
func (d *UnifiedMessageDispatcher) GetQueueStats() interfaces.QueueStats {
	processedCount := atomic.LoadInt64(&d.queueStats.processedCount)
	totalWaitTime := atomic.LoadInt64(&d.queueStats.totalWaitTime)
	
	var avgWaitTime time.Duration
	if processedCount > 0 {
		avgWaitTime = time.Duration(totalWaitTime / processedCount)
	}
	
	return interfaces.QueueStats{
		QueueSize:       int(atomic.LoadInt32(&d.queueStats.queueSize)),
		ProcessedCount:  processedCount,
		AverageWaitTime: avgWaitTime,
		PeakQueueSize:   int(atomic.LoadInt32(&d.queueStats.peakQueueSize)),
		WorkerCount:     int(atomic.LoadInt32(&d.queueStats.workerCount)),
	}
}

// StreamHandler implements MessageHandler for streaming functionality
type StreamHandler struct {
	clientID     string
	messageTypes []string
	streamChan   chan interfaces.Message
	logger       *zap.Logger
}

// Handle processes a message for streaming
func (h *StreamHandler) Handle(ctx context.Context, message interfaces.Message) error {
	select {
	case h.streamChan <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		h.logger.Warn("Stream channel full, dropping message",
			zap.String("client_id", h.clientID),
			zap.String("message_type", message.GetType()))
		return fmt.Errorf("stream channel full for client: %s", h.clientID)
	}
}

// GetHandlerID returns the handler identifier
func (h *StreamHandler) GetHandlerID() string {
	return fmt.Sprintf("stream-%s", h.clientID)
}

// CanHandle checks if this handler can process the given message type
func (h *StreamHandler) CanHandle(messageType string) bool {
	for _, mt := range h.messageTypes {
		if mt == messageType {
			return true
		}
	}
	return false
}
