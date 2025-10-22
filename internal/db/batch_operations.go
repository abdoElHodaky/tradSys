package db

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BatchOperations provides utilities for batch database operations
type BatchOperations struct {
	pool        *ConnectionPool
	logger      *zap.Logger
	metrics     *BatchMetrics
	mutex       sync.RWMutex
	batchSize   int
	concurrency int
}

// BatchMetrics tracks batch operation metrics
type BatchMetrics struct {
	BatchesProcessed   int64
	ItemsProcessed     int64
	OperationErrors    int64
	AverageBatchTime   time.Duration
	TotalOperationTime time.Duration
	mutex              sync.RWMutex
}

// BatchMetricsSnapshot represents a snapshot of batch metrics without mutex
type BatchMetricsSnapshot struct {
	BatchesProcessed   int64
	ItemsProcessed     int64
	OperationErrors    int64
	AverageBatchTime   time.Duration
	TotalOperationTime time.Duration
}

// BatchOperationsOptions contains options for batch operations
type BatchOperationsOptions struct {
	BatchSize   int
	Concurrency int
}

// NewBatchOperations creates a new batch operations utility
func NewBatchOperations(pool *ConnectionPool, logger *zap.Logger, options BatchOperationsOptions) *BatchOperations {
	// Set default values if not provided
	if options.BatchSize == 0 {
		options.BatchSize = 100
	}
	if options.Concurrency == 0 {
		options.Concurrency = 4
	}

	metrics := &BatchMetrics{}

	return &BatchOperations{
		pool:        pool,
		logger:      logger,
		metrics:     metrics,
		batchSize:   options.BatchSize,
		concurrency: options.Concurrency,
	}
}

// BatchInsert performs a batch insert operation
func (b *BatchOperations) BatchInsert(ctx context.Context, table string, columns []string, values [][]interface{}) error {
	if len(values) == 0 {
		return nil
	}

	startTime := time.Now()

	// Calculate number of batches
	numBatches := (len(values) + b.batchSize - 1) / b.batchSize

	// Create a wait group for concurrent batches
	var wg sync.WaitGroup
	wg.Add(numBatches)

	// Create a channel for errors
	errChan := make(chan error, numBatches)

	// Create a semaphore for concurrency control
	sem := make(chan struct{}, b.concurrency)

	// Process batches
	for i := 0; i < numBatches; i++ {
		// Get batch values
		start := i * b.batchSize
		end := (i + 1) * b.batchSize
		if end > len(values) {
			end = len(values)
		}
		batchValues := values[start:end]

		// Acquire semaphore
		sem <- struct{}{}

		// Process batch concurrently
		go func(batchIndex int, batchValues [][]interface{}) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Create placeholders for the query
			placeholders := make([]string, len(batchValues))
			flatValues := make([]interface{}, 0, len(batchValues)*len(columns))

			for i, row := range batchValues {
				// Create placeholders for the row
				rowPlaceholders := make([]string, len(columns))
				for j := range columns {
					rowPlaceholders[j] = fmt.Sprintf("$%d", i*len(columns)+j+1)
				}
				placeholders[i] = fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ", "))

				// Add values to flat array
				flatValues = append(flatValues, row...)
			}

			// Build query
			query := fmt.Sprintf(
				"INSERT INTO %s (%s) VALUES %s",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "),
			)

			// Execute query
			batchStartTime := time.Now()
			_, err := b.pool.Exec(ctx, query, flatValues...)
			batchDuration := time.Since(batchStartTime)

			// Track metrics
			b.metrics.mutex.Lock()
			b.metrics.BatchesProcessed++
			b.metrics.ItemsProcessed += int64(len(batchValues))
			b.metrics.AverageBatchTime = (b.metrics.AverageBatchTime*time.Duration(b.metrics.BatchesProcessed-1) + batchDuration) / time.Duration(b.metrics.BatchesProcessed)
			if err != nil {
				b.metrics.OperationErrors++
			}
			b.metrics.mutex.Unlock()

			// Log batch completion
			b.logger.Debug("Batch insert completed",
				zap.Int("batch_index", batchIndex),
				zap.Int("batch_size", len(batchValues)),
				zap.Duration("duration", batchDuration),
				zap.Error(err),
			)

			// Send error to channel if any
			if err != nil {
				errChan <- fmt.Errorf("batch %d failed: %w", batchIndex, err)
			}
		}(i, batchValues)
	}

	// Wait for all batches to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	// Track total operation time
	totalDuration := time.Since(startTime)
	b.metrics.mutex.Lock()
	b.metrics.TotalOperationTime += totalDuration
	b.metrics.mutex.Unlock()

	// Log operation completion
	b.logger.Info("Batch insert operation completed",
		zap.String("table", table),
		zap.Int("total_items", len(values)),
		zap.Int("batch_size", b.batchSize),
		zap.Int("num_batches", numBatches),
		zap.Int("concurrency", b.concurrency),
		zap.Duration("total_duration", totalDuration),
		zap.Int("error_count", len(errs)),
	)

	// Return combined error if any
	if len(errs) > 0 {
		return fmt.Errorf("batch insert operation failed with %d errors: %v", len(errs), errs)
	}

	return nil
}

// BatchUpdate performs a batch update operation
func (b *BatchOperations) BatchUpdate(ctx context.Context, table string, idColumn string, updateColumns []string, idValues []interface{}, updateValues [][]interface{}) error {
	if len(idValues) == 0 || len(updateValues) == 0 {
		return nil
	}

	startTime := time.Now()

	// Calculate number of batches
	numBatches := (len(idValues) + b.batchSize - 1) / b.batchSize

	// Create a wait group for concurrent batches
	var wg sync.WaitGroup
	wg.Add(numBatches)

	// Create a channel for errors
	errChan := make(chan error, numBatches)

	// Create a semaphore for concurrency control
	sem := make(chan struct{}, b.concurrency)

	// Process batches
	for i := 0; i < numBatches; i++ {
		// Get batch values
		start := i * b.batchSize
		end := (i + 1) * b.batchSize
		if end > len(idValues) {
			end = len(idValues)
		}
		batchIdValues := idValues[start:end]
		batchUpdateValues := updateValues[start:end]

		// Acquire semaphore
		sem <- struct{}{}

		// Process batch concurrently
		go func(batchIndex int, batchIdValues []interface{}, batchUpdateValues [][]interface{}) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Create case statements for each column
			caseStatements := make([]string, len(updateColumns))
			flatValues := make([]interface{}, 0, len(batchIdValues)*(len(updateColumns)+1))

			for j, column := range updateColumns {
				// Create when clauses for the column
				whenClauses := make([]string, len(batchIdValues))
				for i, id := range batchIdValues {
					whenClauses[i] = fmt.Sprintf("WHEN $%d THEN $%d", i*len(updateColumns)+j+1, i*len(updateColumns)+j+2)
					flatValues = append(flatValues, id, batchUpdateValues[i][j])
				}

				// Create case statement for the column
				caseStatements[j] = fmt.Sprintf("%s = CASE %s %s END", column, idColumn, strings.Join(whenClauses, " "))
			}

			// Build query
			query := fmt.Sprintf(
				"UPDATE %s SET %s WHERE %s IN (%s)",
				table,
				strings.Join(caseStatements, ", "),
				idColumn,
				strings.Join(strings.Split(strings.Repeat("?", len(batchIdValues)), ""), ", "),
			)

			// Replace ? with $n
			for i := 0; i < len(batchIdValues); i++ {
				query = strings.Replace(query, "?", fmt.Sprintf("$%d", i+1), 1)
			}

			// Execute query
			batchStartTime := time.Now()
			_, err := b.pool.Exec(ctx, query, flatValues...)
			batchDuration := time.Since(batchStartTime)

			// Track metrics
			b.metrics.mutex.Lock()
			b.metrics.BatchesProcessed++
			b.metrics.ItemsProcessed += int64(len(batchIdValues))
			b.metrics.AverageBatchTime = (b.metrics.AverageBatchTime*time.Duration(b.metrics.BatchesProcessed-1) + batchDuration) / time.Duration(b.metrics.BatchesProcessed)
			if err != nil {
				b.metrics.OperationErrors++
			}
			b.metrics.mutex.Unlock()

			// Log batch completion
			b.logger.Debug("Batch update completed",
				zap.Int("batch_index", batchIndex),
				zap.Int("batch_size", len(batchIdValues)),
				zap.Duration("duration", batchDuration),
				zap.Error(err),
			)

			// Send error to channel if any
			if err != nil {
				errChan <- fmt.Errorf("batch %d failed: %w", batchIndex, err)
			}
		}(i, batchIdValues, batchUpdateValues)
	}

	// Wait for all batches to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	// Track total operation time
	totalDuration := time.Since(startTime)
	b.metrics.mutex.Lock()
	b.metrics.TotalOperationTime += totalDuration
	b.metrics.mutex.Unlock()

	// Log operation completion
	b.logger.Info("Batch update operation completed",
		zap.String("table", table),
		zap.Int("total_items", len(idValues)),
		zap.Int("batch_size", b.batchSize),
		zap.Int("num_batches", numBatches),
		zap.Int("concurrency", b.concurrency),
		zap.Duration("total_duration", totalDuration),
		zap.Int("error_count", len(errs)),
	)

	// Return combined error if any
	if len(errs) > 0 {
		return fmt.Errorf("batch update operation failed with %d errors: %v", len(errs), errs)
	}

	return nil
}

// BatchDelete performs a batch delete operation
func (b *BatchOperations) BatchDelete(ctx context.Context, table string, idColumn string, idValues []interface{}) error {
	if len(idValues) == 0 {
		return nil
	}

	startTime := time.Now()

	// Calculate number of batches
	numBatches := (len(idValues) + b.batchSize - 1) / b.batchSize

	// Create a wait group for concurrent batches
	var wg sync.WaitGroup
	wg.Add(numBatches)

	// Create a channel for errors
	errChan := make(chan error, numBatches)

	// Create a semaphore for concurrency control
	sem := make(chan struct{}, b.concurrency)

	// Process batches
	for i := 0; i < numBatches; i++ {
		// Get batch values
		start := i * b.batchSize
		end := (i + 1) * b.batchSize
		if end > len(idValues) {
			end = len(idValues)
		}
		batchIdValues := idValues[start:end]

		// Acquire semaphore
		sem <- struct{}{}

		// Process batch concurrently
		go func(batchIndex int, batchIdValues []interface{}) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Create placeholders for the query
			placeholders := make([]string, len(batchIdValues))
			for i := range batchIdValues {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
			}

			// Build query
			query := fmt.Sprintf(
				"DELETE FROM %s WHERE %s IN (%s)",
				table,
				idColumn,
				strings.Join(placeholders, ", "),
			)

			// Execute query
			batchStartTime := time.Now()
			_, err := b.pool.Exec(ctx, query, batchIdValues...)
			batchDuration := time.Since(batchStartTime)

			// Track metrics
			b.metrics.mutex.Lock()
			b.metrics.BatchesProcessed++
			b.metrics.ItemsProcessed += int64(len(batchIdValues))
			b.metrics.AverageBatchTime = (b.metrics.AverageBatchTime*time.Duration(b.metrics.BatchesProcessed-1) + batchDuration) / time.Duration(b.metrics.BatchesProcessed)
			if err != nil {
				b.metrics.OperationErrors++
			}
			b.metrics.mutex.Unlock()

			// Log batch completion
			b.logger.Debug("Batch delete completed",
				zap.Int("batch_index", batchIndex),
				zap.Int("batch_size", len(batchIdValues)),
				zap.Duration("duration", batchDuration),
				zap.Error(err),
			)

			// Send error to channel if any
			if err != nil {
				errChan <- fmt.Errorf("batch %d failed: %w", batchIndex, err)
			}
		}(i, batchIdValues)
	}

	// Wait for all batches to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	// Track total operation time
	totalDuration := time.Since(startTime)
	b.metrics.mutex.Lock()
	b.metrics.TotalOperationTime += totalDuration
	b.metrics.mutex.Unlock()

	// Log operation completion
	b.logger.Info("Batch delete operation completed",
		zap.String("table", table),
		zap.Int("total_items", len(idValues)),
		zap.Int("batch_size", b.batchSize),
		zap.Int("num_batches", numBatches),
		zap.Int("concurrency", b.concurrency),
		zap.Duration("total_duration", totalDuration),
		zap.Int("error_count", len(errs)),
	)

	// Return combined error if any
	if len(errs) > 0 {
		return fmt.Errorf("batch delete operation failed with %d errors: %v", len(errs), errs)
	}

	return nil
}

// BatchSelect performs a batch select operation
func (b *BatchOperations) BatchSelect(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	startTime := time.Now()

	// Execute query
	err := b.pool.Select(ctx, dest, query, args...)

	// Track metrics
	b.metrics.mutex.Lock()
	b.metrics.BatchesProcessed++
	b.metrics.TotalOperationTime += time.Since(startTime)
	if err != nil {
		b.metrics.OperationErrors++
	}
	b.metrics.mutex.Unlock()

	// Log operation completion
	b.logger.Debug("Batch select operation completed",
		zap.String("query", query),
		zap.Duration("duration", time.Since(startTime)),
		zap.Error(err),
	)

	return err
}

// GetMetrics returns the current batch metrics
func (b *BatchOperations) GetMetrics() BatchMetricsSnapshot {
	b.metrics.mutex.RLock()
	defer b.metrics.mutex.RUnlock()

	return BatchMetricsSnapshot{
		BatchesProcessed:   b.metrics.BatchesProcessed,
		ItemsProcessed:     b.metrics.ItemsProcessed,
		OperationErrors:    b.metrics.OperationErrors,
		AverageBatchTime:   b.metrics.AverageBatchTime,
		TotalOperationTime: b.metrics.TotalOperationTime,
	}
}

// ResetMetrics resets the batch metrics
func (b *BatchOperations) ResetMetrics() {
	b.metrics.mutex.Lock()
	defer b.metrics.mutex.Unlock()

	b.metrics.BatchesProcessed = 0
	b.metrics.ItemsProcessed = 0
	b.metrics.OperationErrors = 0
	b.metrics.AverageBatchTime = 0
	b.metrics.TotalOperationTime = 0

	b.logger.Info("Batch metrics reset")
}
