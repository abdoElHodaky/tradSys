package unit

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/performance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestMessageBatcher(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	t.Run("Basic Batching", func(t *testing.T) {
		// Create a channel to receive batches
		batchCh := make(chan *performance.MessageBatch, 10)

		// Create a send function
		sendFunc := func(batch *performance.MessageBatch) error {
			batchCh <- batch
			return nil
		}

		// Create a message batcher with a small batch size
		config := performance.DefaultMessageBatcherConfig()
		config.MaxBatchSize = 5
		config.BatchTimeout = 1 * time.Second
		batcher := performance.NewMessageBatcher(config, sendFunc, logger)

		// Create a context for the batcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the batcher
		err := batcher.Start(ctx)
		require.NoError(t, err)

		// Add messages
		for i := 0; i < 5; i++ {
			message := &performance.BatchableMessage{
				Type:     "test",
				Data:     json.RawMessage([]byte(`{"id":` + fmt.Sprintf("%d", i) + `}`)),
				Priority: performance.PriorityMedium,
			}
			err := batcher.AddMessage(message)
			assert.NoError(t, err)
		}

		// Wait for the batch to be sent
		select {
		case batch := <-batchCh:
			assert.Len(t, batch.Messages, 5)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for batch")
		}

		// Stop the batcher
		err = batcher.Stop()
		require.NoError(t, err)
	})

	t.Run("Batch Timeout", func(t *testing.T) {
		// Create a channel to receive batches
		batchCh := make(chan *performance.MessageBatch, 10)

		// Create a send function
		sendFunc := func(batch *performance.MessageBatch) error {
			batchCh <- batch
			return nil
		}

		// Create a message batcher with a short timeout
		config := performance.DefaultMessageBatcherConfig()
		config.MaxBatchSize = 100
		config.BatchTimeout = 100 * time.Millisecond
		batcher := performance.NewMessageBatcher(config, sendFunc, logger)

		// Create a context for the batcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the batcher
		err := batcher.Start(ctx)
		require.NoError(t, err)

		// Add a few messages (not enough to trigger batch size)
		for i := 0; i < 3; i++ {
			message := &performance.BatchableMessage{
				Type:     "test",
				Data:     json.RawMessage([]byte(`{"id":` + fmt.Sprintf("%d", i) + `}`)),
				Priority: performance.PriorityMedium,
			}
			err := batcher.AddMessage(message)
			assert.NoError(t, err)
		}

		// Wait for the batch to be sent due to timeout
		select {
		case batch := <-batchCh:
			assert.Len(t, batch.Messages, 3)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for batch")
		}

		// Stop the batcher
		err = batcher.Stop()
		require.NoError(t, err)
	})

	t.Run("Priority Batching", func(t *testing.T) {
		// Create a channel to receive batches
		batchCh := make(chan *performance.MessageBatch, 10)

		// Create a send function
		sendFunc := func(batch *performance.MessageBatch) error {
			batchCh <- batch
			return nil
		}

		// Create a message batcher with priority thresholds
		config := performance.DefaultMessageBatcherConfig()
		config.MaxBatchSize = 100
		config.BatchTimeout = 1 * time.Second
		config.PriorityThresholds = map[performance.MessagePriority]int{
			performance.PriorityLow:      10,
			performance.PriorityMedium:   5,
			performance.PriorityHigh:     2,
			performance.PriorityCritical: 1,
		}
		batcher := performance.NewMessageBatcher(config, sendFunc, logger)

		// Create a context for the batcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the batcher
		err := batcher.Start(ctx)
		require.NoError(t, err)

		// Add a critical message
		criticalMessage := &performance.BatchableMessage{
			Type:     "critical",
			Data:     json.RawMessage([]byte(`{"id":999}`)),
			Priority: performance.PriorityCritical,
		}
		err = batcher.AddMessage(criticalMessage)
		assert.NoError(t, err)

		// Wait for the critical batch to be sent immediately
		select {
		case batch := <-batchCh:
			assert.Len(t, batch.Messages, 1)
			assert.Equal(t, "critical", batch.Messages[0].Type)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for critical batch")
		}

		// Add high priority messages
		for i := 0; i < 2; i++ {
			message := &performance.BatchableMessage{
				Type:     "high",
				Data:     json.RawMessage([]byte(`{"id":` + fmt.Sprintf("%d", i) + `}`)),
				Priority: performance.PriorityHigh,
			}
			err := batcher.AddMessage(message)
			assert.NoError(t, err)
		}

		// Wait for the high priority batch to be sent
		select {
		case batch := <-batchCh:
			assert.Len(t, batch.Messages, 2)
			assert.Equal(t, "high", batch.Messages[0].Type)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for high priority batch")
		}

		// Stop the batcher
		err = batcher.Stop()
		require.NoError(t, err)
	})

	t.Run("Batch Size Limit", func(t *testing.T) {
		// Create a channel to receive batches
		batchCh := make(chan *performance.MessageBatch, 10)

		// Create a send function
		sendFunc := func(batch *performance.MessageBatch) error {
			batchCh <- batch
			return nil
		}

		// Create a message batcher with a small batch size
		config := performance.DefaultMessageBatcherConfig()
		config.MaxBatchSize = 3
		config.BatchTimeout = 1 * time.Second
		batcher := performance.NewMessageBatcher(config, sendFunc, logger)

		// Create a context for the batcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the batcher
		err := batcher.Start(ctx)
		require.NoError(t, err)

		// Add messages
		for i := 0; i < 7; i++ {
			message := &performance.BatchableMessage{
				Type:     "test",
				Data:     json.RawMessage([]byte(`{"id":` + fmt.Sprintf("%d", i) + `}`)),
				Priority: performance.PriorityMedium,
			}
			err := batcher.AddMessage(message)
			assert.NoError(t, err)
		}

		// Wait for the first batch to be sent
		select {
		case batch := <-batchCh:
			assert.Len(t, batch.Messages, 3)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for first batch")
		}

		// Wait for the second batch to be sent
		select {
		case batch := <-batchCh:
			assert.Len(t, batch.Messages, 3)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for second batch")
		}

		// Wait for the third batch to be sent
		select {
		case batch := <-batchCh:
			assert.Len(t, batch.Messages, 1)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for third batch")
		}

		// Stop the batcher
		err = batcher.Stop()
		require.NoError(t, err)
	})

	t.Run("Batch Bytes Limit", func(t *testing.T) {
		// Create a channel to receive batches
		batchCh := make(chan *performance.MessageBatch, 10)

		// Create a send function
		sendFunc := func(batch *performance.MessageBatch) error {
			batchCh <- batch
			return nil
		}

		// Create a message batcher with a small batch bytes limit
		config := performance.DefaultMessageBatcherConfig()
		config.MaxBatchSize = 100
		config.MaxBatchBytes = 100
		config.BatchTimeout = 1 * time.Second
		batcher := performance.NewMessageBatcher(config, sendFunc, logger)

		// Create a context for the batcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the batcher
		err := batcher.Start(ctx)
		require.NoError(t, err)

		// Add a large message
		largeData := make([]byte, 80)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		largeMessage := &performance.BatchableMessage{
			Type:     "large",
			Data:     json.RawMessage(largeData),
			Priority: performance.PriorityMedium,
			Size:     len(largeData),
		}
		err = batcher.AddMessage(largeMessage)
		assert.NoError(t, err)

		// Add another large message that should trigger a batch flush
		err = batcher.AddMessage(largeMessage)
		assert.NoError(t, err)

		// Wait for the batch to be sent
		select {
		case batch := <-batchCh:
			assert.Len(t, batch.Messages, 1)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for batch")
		}

		// Stop the batcher
		err = batcher.Stop()
		require.NoError(t, err)
	})

	t.Run("Concurrent Message Adding", func(t *testing.T) {
		// Create a channel to receive batches
		batchCh := make(chan *performance.MessageBatch, 100)

		// Create a send function
		sendFunc := func(batch *performance.MessageBatch) error {
			batchCh <- batch
			return nil
		}

		// Create a message batcher
		config := performance.DefaultMessageBatcherConfig()
		config.MaxBatchSize = 10
		config.BatchTimeout = 100 * time.Millisecond
		batcher := performance.NewMessageBatcher(config, sendFunc, logger)

		// Create a context for the batcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the batcher
		err := batcher.Start(ctx)
		require.NoError(t, err)

		// Add messages concurrently
		var wg sync.WaitGroup
		for p := performance.PriorityLow; p <= performance.PriorityCritical; p++ {
			wg.Add(1)
			go func(priority performance.MessagePriority) {
				defer wg.Done()
				for i := 0; i < 20; i++ {
					message := &performance.BatchableMessage{
						Type:     fmt.Sprintf("priority-%d", priority),
						Data:     json.RawMessage([]byte(`{"id":` + fmt.Sprintf("%d", i) + `}`)),
						Priority: priority,
					}
					err := batcher.AddMessage(message)
					assert.NoError(t, err)
					time.Sleep(10 * time.Millisecond)
				}
			}(p)
		}

		// Wait for all messages to be added
		wg.Wait()

		// Wait a bit for all batches to be processed
		time.Sleep(500 * time.Millisecond)

		// Stop the batcher
		err = batcher.Stop()
		require.NoError(t, err)

		// Count the total messages received
		totalMessages := 0
		for len(batchCh) > 0 {
			batch := <-batchCh
			totalMessages += len(batch.Messages)
		}

		// Verify that all messages were processed
		assert.Equal(t, 80, totalMessages)
	})

	t.Run("Batcher Stats", func(t *testing.T) {
		// Create a channel to receive batches
		batchCh := make(chan *performance.MessageBatch, 10)

		// Create a send function
		sendFunc := func(batch *performance.MessageBatch) error {
			batchCh <- batch
			return nil
		}

		// Create a message batcher
		config := performance.DefaultMessageBatcherConfig()
		config.MaxBatchSize = 5
		config.BatchTimeout = 1 * time.Second
		batcher := performance.NewMessageBatcher(config, sendFunc, logger)

		// Create a context for the batcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the batcher
		err := batcher.Start(ctx)
		require.NoError(t, err)

		// Add messages
		for i := 0; i < 5; i++ {
			message := &performance.BatchableMessage{
				Type:     "test",
				Data:     json.RawMessage([]byte(`{"id":` + fmt.Sprintf("%d", i) + `}`)),
				Priority: performance.PriorityMedium,
			}
			err := batcher.AddMessage(message)
			assert.NoError(t, err)
		}

		// Wait for the batch to be sent
		select {
		case <-batchCh:
			// Batch received
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for batch")
		}

		// Get stats
		stats := batcher.GetStats()

		// Verify stats
		assert.Equal(t, uint64(5), stats["messageCount"])
		assert.Equal(t, uint64(1), stats["batchCount"])
		assert.Greater(t, stats["bytesSent"].(uint64), uint64(0))

		// Stop the batcher
		err = batcher.Stop()
		require.NoError(t, err)
	})

	t.Run("Batcher Configuration", func(t *testing.T) {
		// Create a message batcher
		config := performance.DefaultMessageBatcherConfig()
		batcher := performance.NewMessageBatcher(config, nil, logger)

		// Set configuration
		batcher.SetMaxBatchSize(200)
		batcher.SetMaxBatchBytes(2048)
		batcher.SetBatchTimeout(500 * time.Millisecond)
		batcher.SetPriorityThreshold(performance.PriorityHigh, 5)
		batcher.SetEnableCompression(false)
		batcher.SetCompressionLevel(9)

		// Set send function
		sendCalled := false
		batcher.SetSendFunc(func(batch *performance.MessageBatch) error {
			sendCalled = true
			return nil
		})

		// Create a context for the batcher
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the batcher
		err := batcher.Start(ctx)
		require.NoError(t, err)

		// Add a high priority message
		message := &performance.BatchableMessage{
			Type:     "high",
			Data:     json.RawMessage([]byte(`{"id":1}`)),
			Priority: performance.PriorityHigh,
		}
		err = batcher.AddMessage(message)
		assert.NoError(t, err)

		// Wait a bit for the message to be processed
		time.Sleep(100 * time.Millisecond)

		// Verify that the send function was not called yet (need 5 messages)
		assert.False(t, sendCalled)

		// Add more high priority messages to reach the threshold
		for i := 0; i < 4; i++ {
			message := &performance.BatchableMessage{
				Type:     "high",
				Data:     json.RawMessage([]byte(`{"id":` + fmt.Sprintf("%d", i+2) + `}`)),
				Priority: performance.PriorityHigh,
			}
			err := batcher.AddMessage(message)
			assert.NoError(t, err)
		}

		// Wait a bit for the messages to be processed
		time.Sleep(100 * time.Millisecond)

		// Verify that the send function was called
		assert.True(t, sendCalled)

		// Stop the batcher
		err = batcher.Stop()
		require.NoError(t, err)
	})
}

