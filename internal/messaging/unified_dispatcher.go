// Package messaging provides unified message dispatch implementation
package messaging

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"go.uber.org/zap"
)

// UnifiedMessageDispatcher implements all dispatcher interfaces
type UnifiedMessageDispatcher struct {
	logger *zap.Logger
	
	// Core dispatch
	handlers     map[string][]interfaces.MessageHandler
	handlersMux  sync.RWMutex
	filters      []interfaces.MessageFilter
	filtersMux   sync.RWMutex
	transformers []interfaces.MessageTransformer
	transformersMux sync.RWMutex
	
	// Broadcasting
	channels     map[string]map[string]bool // channel -> clientID -> subscribed
	channelsMux  sync.RWMutex
	clients      map[string]chan interfaces.Message // clientID -> message channel
	clientsMux   sync.RWMutex
	
	// Streaming
	streams     map[string]chan interfaces.Message // clientID -> stream channel
	streamsMux  sync.RWMutex
	
	// Queue
	messageQueue chan queuedMessage
	queueWorkers int
	queueStats   queueStatsData
	
	// Statistics
	stats statsData
	
	// Lifecycle
	started   int32
	startTime time.Time
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// queuedMessage represents a message in the dispatch queue
type queuedMessage struct {
	message  interfaces.Message
	options  *interfaces.DispatchOptions
	priority int
	ctx      context.Context
}

// statsData holds dispatch statistics
type statsData struct {
	totalMessages     int64
	successfulMessages int64
	failedMessages    int64
	totalLatency      int64 // nanoseconds
	handlerCount      int32
	filterCount       int32
	transformerCount  int32
}

// queueStatsData holds queue statistics
type queueStatsData struct {
	queueSize       int32
	processedCount  int64
	totalWaitTime   int64 // nanoseconds
	peakQueueSize   int32
	workerCount     int32
}

// DispatcherConfig contains configuration for the unified dispatcher
type DispatcherConfig struct {
	QueueSize    int
	WorkerCount  int
	BufferSize   int
	Logger       *zap.Logger
}

// NewUnifiedDispatcher creates a new unified message dispatcher
func NewUnifiedDispatcher(config DispatcherConfig) interfaces.UnifiedDispatcher {
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = 4
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	return &UnifiedMessageDispatcher{
		logger:       config.Logger,
		handlers:     make(map[string][]interfaces.MessageHandler),
		filters:      make([]interfaces.MessageFilter, 0),
		transformers: make([]interfaces.MessageTransformer, 0),
		channels:     make(map[string]map[string]bool),
		clients:      make(map[string]chan interfaces.Message),
		streams:      make(map[string]chan interfaces.Message),
		messageQueue: make(chan queuedMessage, config.QueueSize),
		queueWorkers: config.WorkerCount,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start initializes the dispatcher
func (d *UnifiedMessageDispatcher) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&d.started, 0, 1) {
		return fmt.Errorf("dispatcher already started")
	}
	
	d.startTime = time.Now()
	atomic.StoreInt32(&d.queueStats.workerCount, int32(d.queueWorkers))
	
	// Start queue workers
	for i := 0; i < d.queueWorkers; i++ {
		d.wg.Add(1)
		go d.queueWorker(i)
	}
	
	d.logger.Info("Unified message dispatcher started",
		zap.Int("workers", d.queueWorkers),
		zap.Int("queue_size", cap(d.messageQueue)))
	
	return nil
}

// Stop gracefully shuts down the dispatcher
func (d *UnifiedMessageDispatcher) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&d.started, 1, 0) {
		return fmt.Errorf("dispatcher not started")
	}
	
	d.cancel()
	
	// Close message queue
	close(d.messageQueue)
	
	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		d.logger.Info("Unified message dispatcher stopped gracefully")
		return nil
	case <-ctx.Done():
		d.logger.Warn("Dispatcher stop timeout exceeded")
		return ctx.Err()
	}
}

// Health returns the health status of the dispatcher
func (d *UnifiedMessageDispatcher) Health() interfaces.DispatcherHealth {
	isStarted := atomic.LoadInt32(&d.started) == 1
	uptime := time.Since(d.startTime)
	
	return interfaces.DispatcherHealth{
		Status:     d.getHealthStatus(isStarted),
		Uptime:     uptime,
		LastError:  nil, // TODO: Track last error
		ErrorCount: atomic.LoadInt64(&d.stats.failedMessages),
		IsHealthy:  isStarted && d.getQueueSize() < cap(d.messageQueue)*9/10, // Healthy if queue < 90% full
	}
}

// Dispatch sends a message to registered handlers
func (d *UnifiedMessageDispatcher) Dispatch(ctx context.Context, message interfaces.Message, options *interfaces.DispatchOptions) error {
	if atomic.LoadInt32(&d.started) != 1 {
		return fmt.Errorf("dispatcher not started")
	}
	
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		atomic.AddInt64(&d.stats.totalLatency, latency.Nanoseconds())
		atomic.AddInt64(&d.stats.totalMessages, 1)
	}()
	
	// Apply global filters
	if !d.shouldProcessMessage(message) {
		return nil
	}
	
	// Apply transformations
	transformedMessage, err := d.transformMessage(ctx, message)
	if err != nil {
		atomic.AddInt64(&d.stats.failedMessages, 1)
		return fmt.Errorf("message transformation failed: %w", err)
	}
	
	// Get dispatch options
	if options == nil {
		options = &interfaces.DispatchOptions{
			Mode:    interfaces.DispatchAsync,
			Timeout: 30 * time.Second,
		}
	}
	
	// Dispatch based on mode
	switch options.Mode {
	case interfaces.DispatchSync:
		return d.dispatchSync(ctx, transformedMessage, options)
	case interfaces.DispatchAsync:
		return d.dispatchAsync(ctx, transformedMessage, options)
	case interfaces.DispatchBroadcast:
		return d.dispatchBroadcast(ctx, transformedMessage, options)
	case interfaces.DispatchRoundRobin:
		return d.dispatchRoundRobin(ctx, transformedMessage, options)
	default:
		return fmt.Errorf("unsupported dispatch mode: %v", options.Mode)
	}
}

// Subscribe registers a handler for specific message types
func (d *UnifiedMessageDispatcher) Subscribe(messageTypes []string, handler interfaces.MessageHandler) error {
	d.handlersMux.Lock()
	defer d.handlersMux.Unlock()
	
	for _, messageType := range messageTypes {
		if d.handlers[messageType] == nil {
			d.handlers[messageType] = make([]interfaces.MessageHandler, 0)
		}
		d.handlers[messageType] = append(d.handlers[messageType], handler)
	}
	
	atomic.AddInt32(&d.stats.handlerCount, 1)
	
	d.logger.Debug("Handler subscribed",
		zap.String("handler_id", handler.GetHandlerID()),
		zap.Strings("message_types", messageTypes))
	
	return nil
}

// Unsubscribe removes a handler from specific message types
func (d *UnifiedMessageDispatcher) Unsubscribe(messageTypes []string, handlerID string) error {
	d.handlersMux.Lock()
	defer d.handlersMux.Unlock()
	
	for _, messageType := range messageTypes {
		handlers := d.handlers[messageType]
		for i, handler := range handlers {
			if handler.GetHandlerID() == handlerID {
				// Remove handler from slice
				d.handlers[messageType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
		
		// Clean up empty handler lists
		if len(d.handlers[messageType]) == 0 {
			delete(d.handlers, messageType)
		}
	}
	
	atomic.AddInt32(&d.stats.handlerCount, -1)
	
	d.logger.Debug("Handler unsubscribed",
		zap.String("handler_id", handlerID),
		zap.Strings("message_types", messageTypes))
	
	return nil
}

// GetSubscribers returns handlers for a message type
func (d *UnifiedMessageDispatcher) GetSubscribers(messageType string) []interfaces.MessageHandler {
	d.handlersMux.RLock()
	defer d.handlersMux.RUnlock()
	
	handlers := d.handlers[messageType]
	if handlers == nil {
		return []interfaces.MessageHandler{}
	}
	
	// Return a copy to avoid race conditions
	result := make([]interfaces.MessageHandler, len(handlers))
	copy(result, handlers)
	return result
}

// AddFilter adds a global message filter
func (d *UnifiedMessageDispatcher) AddFilter(filter interfaces.MessageFilter) {
	d.filtersMux.Lock()
	defer d.filtersMux.Unlock()
	
	d.filters = append(d.filters, filter)
	atomic.AddInt32(&d.stats.filterCount, 1)
	
	d.logger.Debug("Filter added", zap.String("filter_id", filter.GetFilterID()))
}

// RemoveFilter removes a global message filter
func (d *UnifiedMessageDispatcher) RemoveFilter(filterID string) {
	d.filtersMux.Lock()
	defer d.filtersMux.Unlock()
	
	for i, filter := range d.filters {
		if filter.GetFilterID() == filterID {
			d.filters = append(d.filters[:i], d.filters[i+1:]...)
			atomic.AddInt32(&d.stats.filterCount, -1)
			d.logger.Debug("Filter removed", zap.String("filter_id", filterID))
			break
		}
	}
}

// AddTransformer adds a global message transformer
func (d *UnifiedMessageDispatcher) AddTransformer(transformer interfaces.MessageTransformer) {
	d.transformersMux.Lock()
	defer d.transformersMux.Unlock()
	
	d.transformers = append(d.transformers, transformer)
	atomic.AddInt32(&d.stats.transformerCount, 1)
	
	d.logger.Debug("Transformer added", zap.String("transformer_id", transformer.GetTransformerID()))
}

// RemoveTransformer removes a global message transformer
func (d *UnifiedMessageDispatcher) RemoveTransformer(transformerID string) {
	d.transformersMux.Lock()
	defer d.transformersMux.Unlock()
	
	for i, transformer := range d.transformers {
		if transformer.GetTransformerID() == transformerID {
			d.transformers = append(d.transformers[:i], d.transformers[i+1:]...)
			atomic.AddInt32(&d.stats.transformerCount, -1)
			d.logger.Debug("Transformer removed", zap.String("transformer_id", transformerID))
			break
		}
	}
}

// GetStats returns dispatch statistics
func (d *UnifiedMessageDispatcher) GetStats() interfaces.DispatchStats {
	totalMessages := atomic.LoadInt64(&d.stats.totalMessages)
	totalLatency := atomic.LoadInt64(&d.stats.totalLatency)
	
	var avgLatency time.Duration
	if totalMessages > 0 {
		avgLatency = time.Duration(totalLatency / totalMessages)
	}
	
	return interfaces.DispatchStats{
		TotalMessages:      totalMessages,
		SuccessfulMessages: atomic.LoadInt64(&d.stats.successfulMessages),
		FailedMessages:     atomic.LoadInt64(&d.stats.failedMessages),
		AverageLatency:     avgLatency,
		HandlerCount:       int(atomic.LoadInt32(&d.stats.handlerCount)),
		FilterCount:        int(atomic.LoadInt32(&d.stats.filterCount)),
		TransformerCount:   int(atomic.LoadInt32(&d.stats.transformerCount)),
	}
}

// Helper methods

func (d *UnifiedMessageDispatcher) getHealthStatus(isStarted bool) string {
	if !isStarted {
		return "stopped"
	}
	
	queueSize := d.getQueueSize()
	queueCapacity := cap(d.messageQueue)
	
	if queueSize > queueCapacity*9/10 {
		return "degraded"
	}
	
	return "healthy"
}

func (d *UnifiedMessageDispatcher) getQueueSize() int {
	return int(atomic.LoadInt32(&d.queueStats.queueSize))
}

func (d *UnifiedMessageDispatcher) shouldProcessMessage(message interfaces.Message) bool {
	d.filtersMux.RLock()
	defer d.filtersMux.RUnlock()
	
	for _, filter := range d.filters {
		if !filter.ShouldProcess(message) {
			return false
		}
	}
	return true
}

func (d *UnifiedMessageDispatcher) transformMessage(ctx context.Context, message interfaces.Message) (interfaces.Message, error) {
	d.transformersMux.RLock()
	defer d.transformersMux.RUnlock()
	
	result := message
	for _, transformer := range d.transformers {
		var err error
		result, err = transformer.Transform(ctx, result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (d *UnifiedMessageDispatcher) dispatchSync(ctx context.Context, message interfaces.Message, options *interfaces.DispatchOptions) error {
	handlers := d.GetSubscribers(message.GetType())
	if len(handlers) == 0 {
		return nil
	}
	
	for _, handler := range handlers {
		if handler.CanHandle(message.GetType()) {
			if err := handler.Handle(ctx, message); err != nil {
				atomic.AddInt64(&d.stats.failedMessages, 1)
				d.logger.Error("Handler failed",
					zap.String("handler_id", handler.GetHandlerID()),
					zap.String("message_type", message.GetType()),
					zap.Error(err))
				return err
			}
		}
	}
	
	atomic.AddInt64(&d.stats.successfulMessages, 1)
	return nil
}

func (d *UnifiedMessageDispatcher) dispatchAsync(ctx context.Context, message interfaces.Message, options *interfaces.DispatchOptions) error {
	queuedMsg := queuedMessage{
		message: message,
		options: options,
		priority: 0,
		ctx:     ctx,
	}
	
	select {
	case d.messageQueue <- queuedMsg:
		atomic.AddInt32(&d.queueStats.queueSize, 1)
		// Update peak queue size
		currentSize := atomic.LoadInt32(&d.queueStats.queueSize)
		for {
			peak := atomic.LoadInt32(&d.queueStats.peakQueueSize)
			if currentSize <= peak || atomic.CompareAndSwapInt32(&d.queueStats.peakQueueSize, peak, currentSize) {
				break
			}
		}
		return nil
	case <-ctx.Done():
		atomic.AddInt64(&d.stats.failedMessages, 1)
		return ctx.Err()
	default:
		atomic.AddInt64(&d.stats.failedMessages, 1)
		return fmt.Errorf("message queue full")
	}
}

func (d *UnifiedMessageDispatcher) dispatchBroadcast(ctx context.Context, message interfaces.Message, options *interfaces.DispatchOptions) error {
	// Broadcast to all handlers
	handlers := d.GetSubscribers(message.GetType())
	if len(handlers) == 0 {
		return nil
	}
	
	var wg sync.WaitGroup
	errors := make(chan error, len(handlers))
	
	for _, handler := range handlers {
		if handler.CanHandle(message.GetType()) {
			wg.Add(1)
			go func(h interfaces.MessageHandler) {
				defer wg.Done()
				if err := h.Handle(ctx, message); err != nil {
					errors <- err
				}
			}(handler)
		}
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	var firstError error
	errorCount := 0
	for err := range errors {
		if firstError == nil {
			firstError = err
		}
		errorCount++
	}
	
	if errorCount > 0 {
		atomic.AddInt64(&d.stats.failedMessages, 1)
		d.logger.Error("Broadcast dispatch had errors",
			zap.Int("error_count", errorCount),
			zap.String("message_type", message.GetType()))
		return firstError
	}
	
	atomic.AddInt64(&d.stats.successfulMessages, 1)
	return nil
}

func (d *UnifiedMessageDispatcher) dispatchRoundRobin(ctx context.Context, message interfaces.Message, options *interfaces.DispatchOptions) error {
	handlers := d.GetSubscribers(message.GetType())
	if len(handlers) == 0 {
		return nil
	}
	
	// Simple round-robin: use timestamp to select handler
	handlerIndex := int(message.GetTimestamp().UnixNano()) % len(handlers)
	handler := handlers[handlerIndex]
	
	if handler.CanHandle(message.GetType()) {
		if err := handler.Handle(ctx, message); err != nil {
			atomic.AddInt64(&d.stats.failedMessages, 1)
			return err
		}
	}
	
	atomic.AddInt64(&d.stats.successfulMessages, 1)
	return nil
}

func (d *UnifiedMessageDispatcher) queueWorker(workerID int) {
	defer d.wg.Done()
	
	d.logger.Debug("Queue worker started", zap.Int("worker_id", workerID))
	
	for {
		select {
		case queuedMsg, ok := <-d.messageQueue:
			if !ok {
				d.logger.Debug("Queue worker stopping", zap.Int("worker_id", workerID))
				return
			}
			
			atomic.AddInt32(&d.queueStats.queueSize, -1)
			atomic.AddInt64(&d.queueStats.processedCount, 1)
			
			// Process the message
			if err := d.dispatchSync(queuedMsg.ctx, queuedMsg.message, queuedMsg.options); err != nil {
				d.logger.Error("Queue worker failed to process message",
					zap.Int("worker_id", workerID),
					zap.String("message_type", queuedMsg.message.GetType()),
					zap.Error(err))
			}
			
		case <-d.ctx.Done():
			d.logger.Debug("Queue worker stopping due to context cancellation", zap.Int("worker_id", workerID))
			return
		}
	}
}
