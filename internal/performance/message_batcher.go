package performance

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"go.uber.org/zap"
)

// MessagePriority defines the priority of a message
type MessagePriority int

const (
	// PriorityLow is for non-critical messages
	PriorityLow MessagePriority = iota
	// PriorityMedium is for normal messages
	PriorityMedium
	// PriorityHigh is for important messages
	PriorityHigh
	// PriorityCritical is for critical messages that should be sent immediately
	PriorityCritical
)

// BatchableMessage represents a message that can be batched
type BatchableMessage struct {
	// Type is the type of the message
	Type string `json:"type"`
	
	// Data is the data of the message
	Data json.RawMessage `json:"data"`
	
	// Priority is the priority of the message
	Priority MessagePriority `json:"-"`
	
	// Timestamp is when the message was created
	Timestamp time.Time `json:"-"`
	
	// Size is the size of the message in bytes
	Size int `json:"-"`
}

// MessageBatcherConfig contains configuration for the message batcher
type MessageBatcherConfig struct {
	// MaxBatchSize is the maximum number of messages in a batch
	MaxBatchSize int
	
	// MaxBatchBytes is the maximum size of a batch in bytes
	MaxBatchBytes int
	
	// BatchInterval is the interval at which batches are flushed
	BatchInterval time.Duration
	
	// PriorityThresholds defines the batch size thresholds for each priority
	// Messages with higher priority will be sent in smaller batches
	PriorityThresholds map[MessagePriority]int
	
	// EnableAdaptiveBatching enables adaptive batching based on system load
	EnableAdaptiveBatching bool
	
	// AdaptiveFactors defines the factors for adaptive batching
	AdaptiveFactors AdaptiveFactors
}

// AdaptiveFactors defines factors for adaptive batching
type AdaptiveFactors struct {
	// LoadThreshold is the load threshold for adaptive batching
	LoadThreshold float64
	
	// BatchSizeReductionFactor is the factor by which batch size is reduced under high load
	BatchSizeReductionFactor float64
	
	// BatchIntervalReductionFactor is the factor by which batch interval is reduced under high load
	BatchIntervalReductionFactor float64
}

// DefaultMessageBatcherConfig returns the default configuration
func DefaultMessageBatcherConfig() MessageBatcherConfig {
	return MessageBatcherConfig{
		MaxBatchSize:  100,
		MaxBatchBytes: 1024 * 1024, // 1MB
		BatchInterval: 100 * time.Millisecond,
		PriorityThresholds: map[MessagePriority]int{
			PriorityLow:      100,
			PriorityMedium:   50,
			PriorityHigh:     10,
			PriorityCritical: 1,
		},
		EnableAdaptiveBatching: true,
		AdaptiveFactors: AdaptiveFactors{
			LoadThreshold:              0.7,
			BatchSizeReductionFactor:   0.5,
			BatchIntervalReductionFactor: 0.5,
		},
	}
}

// MessageBatcher batches messages for efficient transmission
type MessageBatcher struct {
	// Configuration
	config MessageBatcherConfig
	
	// Batches for each priority
	batches map[MessagePriority][]BatchableMessage
	
	// Batch sizes in bytes
	batchSizes map[MessagePriority]int
	
	// Mutex for protecting the batches
	mu sync.Mutex
	
	// Timer for flushing batches
	timer *time.Timer
	
	// Channel for receiving messages
	messageCh chan BatchableMessage
	
	// Channel for flushing batches
	flushCh chan MessagePriority
	
	// Channel for stopping the batcher
	stopCh chan struct{}
	
	// Function for sending batches
	sendBatchFunc func([]BatchableMessage) error
	
	// Logger
	logger *zap.Logger
	
	// Metrics
	metrics *metrics.WebSocketMetrics
	
	// Current system load (0.0 - 1.0)
	currentLoad float64
	
	// Mutex for protecting the current load
	loadMu sync.RWMutex
}

// NewMessageBatcher creates a new message batcher
func NewMessageBatcher(
	config MessageBatcherConfig,
	sendBatchFunc func([]BatchableMessage) error,
	logger *zap.Logger,
	metrics *metrics.WebSocketMetrics,
) *MessageBatcher {
	batcher := &MessageBatcher{
		config:        config,
		batches:       make(map[MessagePriority][]BatchableMessage),
		batchSizes:    make(map[MessagePriority]int),
		messageCh:     make(chan BatchableMessage, 1000),
		flushCh:       make(chan MessagePriority, 10),
		stopCh:        make(chan struct{}),
		sendBatchFunc: sendBatchFunc,
		logger:        logger,
		metrics:       metrics,
		currentLoad:   0.0,
	}
	
	// Initialize batches for each priority
	for priority := PriorityLow; priority <= PriorityCritical; priority++ {
		batcher.batches[priority] = make([]BatchableMessage, 0, config.MaxBatchSize)
		batcher.batchSizes[priority] = 0
	}
	
	// Start the timer
	batcher.timer = time.NewTimer(config.BatchInterval)
	
	// Start the processing goroutine
	go batcher.processMessages()
	
	return batcher
}

// AddMessage adds a message to the batcher
func (b *MessageBatcher) AddMessage(message BatchableMessage) {
	// Set the timestamp if not set
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	
	// Send critical messages immediately
	if message.Priority == PriorityCritical {
		b.sendImmediately(message)
		return
	}
	
	// Send the message to the channel
	b.messageCh <- message
}

// sendImmediately sends a message immediately
func (b *MessageBatcher) sendImmediately(message BatchableMessage) {
	err := b.sendBatchFunc([]BatchableMessage{message})
	if err != nil {
		b.logger.Error("Failed to send message",
			zap.Error(err),
			zap.String("type", message.Type))
	}
}

// processMessages processes messages from the channel
func (b *MessageBatcher) processMessages() {
	for {
		select {
		case <-b.stopCh:
			// Stop the batcher
			return
			
		case <-b.timer.C:
			// Flush all batches
			b.flushAllBatches()
			
			// Reset the timer
			b.timer.Reset(b.getAdaptiveBatchInterval())
			
		case priority := <-b.flushCh:
			// Flush a specific batch
			b.flushBatch(priority)
			
		case message := <-b.messageCh:
			// Add the message to the batch
			b.addMessageToBatch(message)
		}
	}
}

// addMessageToBatch adds a message to the appropriate batch
func (b *MessageBatcher) addMessageToBatch(message BatchableMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	priority := message.Priority
	
	// Add the message to the batch
	b.batches[priority] = append(b.batches[priority], message)
	b.batchSizes[priority] += message.Size
	
	// Check if the batch should be flushed
	threshold := b.getAdaptiveBatchSize(priority)
	if len(b.batches[priority]) >= threshold || b.batchSizes[priority] >= b.config.MaxBatchBytes {
		// Flush the batch asynchronously
		go func() {
			b.flushCh <- priority
		}()
	}
}

// flushBatch flushes a specific batch
func (b *MessageBatcher) flushBatch(priority MessagePriority) {
	b.mu.Lock()
	
	// Get the batch
	batch := b.batches[priority]
	if len(batch) == 0 {
		b.mu.Unlock()
		return
	}
	
	// Clear the batch
	b.batches[priority] = make([]BatchableMessage, 0, b.config.MaxBatchSize)
	b.batchSizes[priority] = 0
	
	b.mu.Unlock()
	
	// Record batch metrics
	startTime := time.Now()
	
	// Send the batch
	err := b.sendBatchFunc(batch)
	
	// Record batch metrics
	if b.metrics != nil {
		b.metrics.RecordBatch(len(batch), time.Since(startTime))
	}
	
	if err != nil {
		b.logger.Error("Failed to send batch",
			zap.Error(err),
			zap.Int("size", len(batch)),
			zap.Stringer("priority", priority))
	}
}

// flushAllBatches flushes all batches
func (b *MessageBatcher) flushAllBatches() {
	for priority := PriorityLow; priority <= PriorityHigh; priority++ {
		b.flushBatch(priority)
	}
}

// Stop stops the batcher
func (b *MessageBatcher) Stop() {
	// Flush all batches
	b.flushAllBatches()
	
	// Stop the timer
	b.timer.Stop()
	
	// Stop the processing goroutine
	close(b.stopCh)
}

// SetCurrentLoad sets the current system load
func (b *MessageBatcher) SetCurrentLoad(load float64) {
	b.loadMu.Lock()
	defer b.loadMu.Unlock()
	
	// Clamp the load to [0.0, 1.0]
	if load < 0.0 {
		load = 0.0
	} else if load > 1.0 {
		load = 1.0
	}
	
	b.currentLoad = load
}

// getCurrentLoad gets the current system load
func (b *MessageBatcher) getCurrentLoad() float64 {
	b.loadMu.RLock()
	defer b.loadMu.RUnlock()
	return b.currentLoad
}

// getAdaptiveBatchSize gets the adaptive batch size for a priority
func (b *MessageBatcher) getAdaptiveBatchSize(priority MessagePriority) int {
	// Get the base threshold
	threshold := b.config.PriorityThresholds[priority]
	
	// If adaptive batching is disabled, return the base threshold
	if !b.config.EnableAdaptiveBatching {
		return threshold
	}
	
	// Get the current load
	load := b.getCurrentLoad()
	
	// If the load is below the threshold, return the base threshold
	if load < b.config.AdaptiveFactors.LoadThreshold {
		return threshold
	}
	
	// Calculate the adaptive threshold
	adaptiveFactor := 1.0 - ((load - b.config.AdaptiveFactors.LoadThreshold) / 
		(1.0 - b.config.AdaptiveFactors.LoadThreshold) * 
		b.config.AdaptiveFactors.BatchSizeReductionFactor)
	
	adaptiveThreshold := int(float64(threshold) * adaptiveFactor)
	
	// Ensure the threshold is at least 1
	if adaptiveThreshold < 1 {
		adaptiveThreshold = 1
	}
	
	return adaptiveThreshold
}

// getAdaptiveBatchInterval gets the adaptive batch interval
func (b *MessageBatcher) getAdaptiveBatchInterval() time.Duration {
	// If adaptive batching is disabled, return the base interval
	if !b.config.EnableAdaptiveBatching {
		return b.config.BatchInterval
	}
	
	// Get the current load
	load := b.getCurrentLoad()
	
	// If the load is below the threshold, return the base interval
	if load < b.config.AdaptiveFactors.LoadThreshold {
		return b.config.BatchInterval
	}
	
	// Calculate the adaptive interval
	adaptiveFactor := 1.0 - ((load - b.config.AdaptiveFactors.LoadThreshold) / 
		(1.0 - b.config.AdaptiveFactors.LoadThreshold) * 
		b.config.AdaptiveFactors.BatchIntervalReductionFactor)
	
	adaptiveInterval := time.Duration(float64(b.config.BatchInterval) * adaptiveFactor)
	
	// Ensure the interval is at least 1ms
	if adaptiveInterval < time.Millisecond {
		adaptiveInterval = time.Millisecond
	}
	
	return adaptiveInterval
}

// String returns a string representation of the priority
func (p MessagePriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

