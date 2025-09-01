package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
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
	
	// BatchTimeout is the maximum time to wait before sending a batch
	BatchTimeout time.Duration
	
	// PriorityThresholds defines the thresholds for each priority
	PriorityThresholds map[MessagePriority]int
	
	// EnableCompression enables compression of batches
	EnableCompression bool
	
	// CompressionLevel is the compression level (1-9, higher is more compression)
	CompressionLevel int
	
	// EnableMetrics enables metrics collection
	EnableMetrics bool
}

// DefaultMessageBatcherConfig returns the default message batcher configuration
func DefaultMessageBatcherConfig() MessageBatcherConfig {
	return MessageBatcherConfig{
		MaxBatchSize:    100,
		MaxBatchBytes:   1024 * 1024, // 1MB
		BatchTimeout:    100 * time.Millisecond,
		PriorityThresholds: map[MessagePriority]int{
			PriorityLow:      100, // Wait for up to 100 messages
			PriorityMedium:   50,  // Wait for up to 50 messages
			PriorityHigh:     10,  // Wait for up to 10 messages
			PriorityCritical: 1,   // Send immediately
		},
		EnableCompression: true,
		CompressionLevel:  6, // Default compression level
		EnableMetrics:     true,
	}
}

// MessageBatch represents a batch of messages
type MessageBatch struct {
	// Messages is the list of messages in the batch
	Messages []*BatchableMessage `json:"messages"`
	
	// Timestamp is when the batch was created
	Timestamp time.Time `json:"timestamp"`
	
	// Size is the total size of the batch in bytes
	Size int `json:"-"`
	
	// Compressed indicates whether the batch is compressed
	Compressed bool `json:"compressed,omitempty"`
}

// MessageBatcher batches messages for efficient transmission
type MessageBatcher struct {
	// Configuration
	config MessageBatcherConfig
	
	// Batches by priority
	batches map[MessagePriority]*MessageBatch
	
	// Batch locks by priority
	batchLocks map[MessagePriority]*sync.Mutex
	
	// Batch send function
	sendFunc func(batch *MessageBatch) error
	
	// Metrics
	messageCount        uint64
	batchCount          uint64
	bytesSent           uint64
	bytesCompressed     uint64
	compressionRatio    float64
	batchSizeHistogram  *metrics.Histogram
	batchTimeHistogram  *metrics.Histogram
	
	// State
	running      bool
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	
	// Logger
	logger *zap.Logger
}

// NewMessageBatcher creates a new message batcher
func NewMessageBatcher(config MessageBatcherConfig, sendFunc func(batch *MessageBatch) error, logger *zap.Logger) *MessageBatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	// Initialize batches and locks
	batches := make(map[MessagePriority]*MessageBatch)
	batchLocks := make(map[MessagePriority]*sync.Mutex)
	
	for priority := PriorityLow; priority <= PriorityCritical; priority++ {
		batches[priority] = &MessageBatch{
			Messages:  make([]*BatchableMessage, 0, config.MaxBatchSize),
			Timestamp: time.Now(),
			Size:      0,
		}
		batchLocks[priority] = &sync.Mutex{}
	}
	
	// Create histograms for metrics
	batchSizeHistogram := metrics.NewHistogram(
		"message_batcher_batch_size",
		"Size of message batches",
		[]float64{10, 50, 100, 200, 500, 1000},
	)
	
	batchTimeHistogram := metrics.NewHistogram(
		"message_batcher_batch_time",
		"Time to process message batches",
		[]float64{1, 5, 10, 50, 100, 500},
	)
	
	return &MessageBatcher{
		config:             config,
		batches:            batches,
		batchLocks:         batchLocks,
		sendFunc:           sendFunc,
		batchSizeHistogram: batchSizeHistogram,
		batchTimeHistogram: batchTimeHistogram,
		stopCh:             make(chan struct{}),
		logger:             logger,
	}
}

// Start starts the message batcher
func (b *MessageBatcher) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.running {
		return fmt.Errorf("message batcher already running")
	}
	
	b.logger.Info("Starting message batcher",
		zap.Int("maxBatchSize", b.config.MaxBatchSize),
		zap.Int("maxBatchBytes", b.config.MaxBatchBytes),
		zap.Duration("batchTimeout", b.config.BatchTimeout),
	)
	
	b.running = true
	b.stopCh = make(chan struct{})
	
	// Start the batch flusher goroutine
	b.wg.Add(1)
	go b.batchFlusher(ctx)
	
	return nil
}

// Stop stops the message batcher
func (b *MessageBatcher) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.running {
		return nil
	}
	
	b.logger.Info("Stopping message batcher")
	
	// Signal the batch flusher to stop
	close(b.stopCh)
	
	// Wait for the batch flusher to stop
	b.wg.Wait()
	
	// Flush any remaining batches
	for priority := PriorityLow; priority <= PriorityCritical; priority++ {
		b.flushBatch(priority)
	}
	
	b.running = false
	
	return nil
}

// AddMessage adds a message to the batch
func (b *MessageBatcher) AddMessage(message *BatchableMessage) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}
	
	// Set the timestamp if not set
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	
	// Calculate the message size if not set
	if message.Size == 0 {
		data, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}
		message.Size = len(data)
	}
	
	// Check if the message is too large
	if message.Size > b.config.MaxBatchBytes {
		return fmt.Errorf("message size %d exceeds maximum batch size %d", message.Size, b.config.MaxBatchBytes)
	}
	
	// Increment message count
	atomic.AddUint64(&b.messageCount, 1)
	
	// Lock the batch for this priority
	b.batchLocks[message.Priority].Lock()
	defer b.batchLocks[message.Priority].Unlock()
	
	// Get the batch for this priority
	batch := b.batches[message.Priority]
	
	// Check if adding this message would exceed the maximum batch size
	if batch.Size+message.Size > b.config.MaxBatchBytes || len(batch.Messages) >= b.config.MaxBatchSize {
		// Flush the current batch
		if err := b.flushBatchLocked(message.Priority); err != nil {
			return fmt.Errorf("failed to flush batch: %w", err)
		}
	}
	
	// Add the message to the batch
	batch.Messages = append(batch.Messages, message)
	batch.Size += message.Size
	
	// Check if we should flush the batch based on priority thresholds
	threshold, ok := b.config.PriorityThresholds[message.Priority]
	if ok && len(batch.Messages) >= threshold {
		if err := b.flushBatchLocked(message.Priority); err != nil {
			return fmt.Errorf("failed to flush batch: %w", err)
		}
	}
	
	return nil
}

// flushBatch flushes a batch for a specific priority
func (b *MessageBatcher) flushBatch(priority MessagePriority) error {
	b.batchLocks[priority].Lock()
	defer b.batchLocks[priority].Unlock()
	
	return b.flushBatchLocked(priority)
}

// flushBatchLocked flushes a batch for a specific priority (must be called with lock held)
func (b *MessageBatcher) flushBatchLocked(priority MessagePriority) error {
	batch := b.batches[priority]
	
	// Skip empty batches
	if len(batch.Messages) == 0 {
		return nil
	}
	
	// Record metrics
	if b.config.EnableMetrics {
		b.batchSizeHistogram.Observe(float64(len(batch.Messages)))
		startTime := time.Now()
		defer func() {
			b.batchTimeHistogram.Observe(float64(time.Since(startTime).Milliseconds()))
		}()
	}
	
	// Compress the batch if enabled
	if b.config.EnableCompression && batch.Size > 1024 {
		compressedBatch, err := b.compressBatch(batch)
		if err != nil {
			b.logger.Error("Failed to compress batch",
				zap.Error(err),
			)
		} else {
			batch = compressedBatch
		}
	}
	
	// Send the batch
	if err := b.sendFunc(batch); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}
	
	// Update metrics
	atomic.AddUint64(&b.batchCount, 1)
	atomic.AddUint64(&b.bytesSent, uint64(batch.Size))
	
	// Create a new batch
	b.batches[priority] = &MessageBatch{
		Messages:  make([]*BatchableMessage, 0, b.config.MaxBatchSize),
		Timestamp: time.Now(),
		Size:      0,
	}
	
	return nil
}

// compressBatch compresses a batch
func (b *MessageBatcher) compressBatch(batch *MessageBatch) (*MessageBatch, error) {
	// Marshal the batch
	data, err := json.Marshal(batch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch: %w", err)
	}
	
	// Compress the data
	compressor := NewMessageCompressor(b.config.CompressionLevel)
	compressed, err := compressor.Compress(data)
	if err != nil {
		return nil, fmt.Errorf("failed to compress batch: %w", err)
	}
	
	// Update compression metrics
	if b.config.EnableMetrics {
		atomic.AddUint64(&b.bytesCompressed, uint64(len(compressed)))
		if len(data) > 0 {
			b.compressionRatio = float64(len(compressed)) / float64(len(data))
		}
	}
	
	// Create a new batch with compressed data
	return &MessageBatch{
		Messages:   batch.Messages,
		Timestamp:  batch.Timestamp,
		Size:       len(compressed),
		Compressed: true,
	}, nil
}

// batchFlusher periodically flushes batches
func (b *MessageBatcher) batchFlusher(ctx context.Context) {
	defer b.wg.Done()
	
	ticker := time.NewTicker(b.config.BatchTimeout)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Flush batches that have been waiting too long
			for priority := PriorityLow; priority <= PriorityCritical; priority++ {
				b.batchLocks[priority].Lock()
				batch := b.batches[priority]
				if len(batch.Messages) > 0 && time.Since(batch.Timestamp) >= b.config.BatchTimeout {
					if err := b.flushBatchLocked(priority); err != nil {
						b.logger.Error("Failed to flush batch",
							zap.Error(err),
							zap.Int("priority", int(priority)),
						)
					}
				}
				b.batchLocks[priority].Unlock()
			}
		case <-b.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// GetStats returns statistics about the message batcher
func (b *MessageBatcher) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["messageCount"] = atomic.LoadUint64(&b.messageCount)
	stats["batchCount"] = atomic.LoadUint64(&b.batchCount)
	stats["bytesSent"] = atomic.LoadUint64(&b.bytesSent)
	stats["bytesCompressed"] = atomic.LoadUint64(&b.bytesCompressed)
	stats["compressionRatio"] = b.compressionRatio
	
	// Get batch sizes
	batchSizes := make(map[string]int)
	for priority := PriorityLow; priority <= PriorityCritical; priority++ {
		b.batchLocks[priority].Lock()
		batchSizes[fmt.Sprintf("priority%d", priority)] = len(b.batches[priority].Messages)
		b.batchLocks[priority].Unlock()
	}
	stats["batchSizes"] = batchSizes
	
	return stats
}

// IsRunning returns whether the message batcher is running
func (b *MessageBatcher) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// SetSendFunc sets the function used to send batches
func (b *MessageBatcher) SetSendFunc(sendFunc func(batch *MessageBatch) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sendFunc = sendFunc
}

// SetBatchTimeout sets the batch timeout
func (b *MessageBatcher) SetBatchTimeout(timeout time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.config.BatchTimeout = timeout
}

// SetMaxBatchSize sets the maximum batch size
func (b *MessageBatcher) SetMaxBatchSize(size int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.config.MaxBatchSize = size
}

// SetMaxBatchBytes sets the maximum batch bytes
func (b *MessageBatcher) SetMaxBatchBytes(bytes int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.config.MaxBatchBytes = bytes
}

// SetPriorityThreshold sets the threshold for a specific priority
func (b *MessageBatcher) SetPriorityThreshold(priority MessagePriority, threshold int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.config.PriorityThresholds[priority] = threshold
}

// SetEnableCompression sets whether compression is enabled
func (b *MessageBatcher) SetEnableCompression(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.config.EnableCompression = enabled
}

// SetCompressionLevel sets the compression level
func (b *MessageBatcher) SetCompressionLevel(level int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if level < 1 {
		level = 1
	} else if level > 9 {
		level = 9
	}
	b.config.CompressionLevel = level
}

