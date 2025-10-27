package risk

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BatchProcessor handles batch processing of risk operations
type BatchProcessor struct {
	// Batch processing channel for risk operations
	riskBatchChan chan RiskOperation
	// Position manager reference
	positionManager *PositionManager
	// Limit manager reference
	limitManager *LimitManager
	// Logger
	logger *zap.Logger
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
	// Wait group for graceful shutdown
	wg sync.WaitGroup
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(positionManager *PositionManager, limitManager *LimitManager, logger *zap.Logger) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	bp := &BatchProcessor{
		riskBatchChan:   make(chan RiskOperation, 1000),
		positionManager: positionManager,
		limitManager:    limitManager,
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start batch processing goroutine
	bp.wg.Add(1)
	go bp.processBatchOperations()

	return bp
}

// SubmitOperation submits a risk operation for batch processing
func (bp *BatchProcessor) SubmitOperation(op RiskOperation) {
	select {
	case bp.riskBatchChan <- op:
		// Operation submitted successfully
	case <-bp.ctx.Done():
		// Processor is shutting down
		op.ResultCh <- RiskOperationResult{
			Success: false,
			Error:   errors.New("batch processor is shutting down"),
		}
	default:
		// Channel is full, reject operation
		op.ResultCh <- RiskOperationResult{
			Success: false,
			Error:   errors.New("batch processor queue is full"),
		}
	}
}

// processBatchOperations processes risk operations in batches
func (bp *BatchProcessor) processBatchOperations() {
	defer bp.wg.Done()

	ticker := time.NewTicker(10 * time.Millisecond) // Process batches every 10ms
	defer ticker.Stop()

	batch := make([]RiskOperation, 0, 100) // Pre-allocate batch slice

	for {
		select {
		case <-bp.ctx.Done():
			// Process remaining operations before shutdown
			if len(batch) > 0 {
				bp.processBatch(batch)
			}
			return

		case <-ticker.C:
			// Process accumulated batch
			if len(batch) > 0 {
				bp.processBatch(batch)
				batch = batch[:0] // Reset batch slice
			}

		case op := <-bp.riskBatchChan:
			batch = append(batch, op)

			// Process batch if it's full
			if len(batch) >= 100 {
				bp.processBatch(batch)
				batch = batch[:0] // Reset batch slice
			}
		}
	}
}

// processBatch processes a batch of risk operations
func (bp *BatchProcessor) processBatch(batch []RiskOperation) {
	if len(batch) == 0 {
		return
	}

	bp.logger.Debug("Processing risk operation batch", zap.Int("batchSize", len(batch)))

	// Group operations by type for efficient processing
	updatePositionOps := make([]RiskOperation, 0)
	checkLimitOps := make([]RiskOperation, 0)
	addLimitOps := make([]RiskOperation, 0)

	for _, op := range batch {
		switch op.OpType {
		case "updatePosition":
			updatePositionOps = append(updatePositionOps, op)
		case "checkLimit":
			checkLimitOps = append(checkLimitOps, op)
		case "addLimit":
			addLimitOps = append(addLimitOps, op)
		default:
			// Unknown operation type
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   errors.New("unknown operation type: " + op.OpType),
			}
		}
	}

	// Process each operation type in batch
	if len(updatePositionOps) > 0 {
		bp.processUpdatePositionBatch(updatePositionOps)
	}
	if len(checkLimitOps) > 0 {
		bp.processCheckLimitBatch(checkLimitOps)
	}
	if len(addLimitOps) > 0 {
		bp.processAddLimitBatch(addLimitOps)
	}
}

// processUpdatePositionBatch processes a batch of position update operations
func (bp *BatchProcessor) processUpdatePositionBatch(ops []RiskOperation) {
	for _, op := range ops {
		// Extract position update data
		data, ok := op.Data.(map[string]interface{})
		if !ok {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   errors.New("invalid position update data"),
			}
			continue
		}

		quantityDelta, ok := data["quantityDelta"].(float64)
		if !ok {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   errors.New("invalid quantity delta"),
			}
			continue
		}

		price, ok := data["price"].(float64)
		if !ok {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   errors.New("invalid price"),
			}
			continue
		}

		// Update position
		bp.positionManager.UpdatePosition(op.UserID, op.Symbol, quantityDelta, price)

		// Send success result
		op.ResultCh <- RiskOperationResult{
			Success: true,
			Data:    "Position updated successfully",
		}
	}
}

// processCheckLimitBatch processes a batch of limit check operations
func (bp *BatchProcessor) processCheckLimitBatch(ops []RiskOperation) {
	for _, op := range ops {
		// Extract limit check data
		data, ok := op.Data.(map[string]interface{})
		if !ok {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   errors.New("invalid limit check data"),
			}
			continue
		}

		orderSize, ok := data["orderSize"].(float64)
		if !ok {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   errors.New("invalid order size"),
			}
			continue
		}

		currentPrice, ok := data["currentPrice"].(float64)
		if !ok {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   errors.New("invalid current price"),
			}
			continue
		}

		currentPosition, ok := data["currentPosition"].(float64)
		if !ok {
			currentPosition = 0.0 // Default to zero if not provided
		}

		// Check risk limits
		result, err := bp.limitManager.CheckRiskLimits(bp.ctx, op.UserID, op.Symbol, orderSize, currentPrice, currentPosition)
		if err != nil {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   err.Error(),
			}
			continue
		}

		// Send result
		op.ResultCh <- RiskOperationResult{
			Success: true,
			Data:    result,
		}
	}
}

// processAddLimitBatch processes a batch of add limit operations
func (bp *BatchProcessor) processAddLimitBatch(ops []RiskOperation) {
	for _, op := range ops {
		// Extract limit data
		limit, ok := op.Data.(*RiskLimit)
		if !ok {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   errors.New("invalid risk limit data"),
			}
			continue
		}

		// Add risk limit
		addedLimit, err := bp.limitManager.AddRiskLimit(bp.ctx, limit)
		if err != nil {
			op.ResultCh <- RiskOperationResult{
				Success: false,
				Error:   err.Error(),
			}
			continue
		}

		// Send success result
		op.ResultCh <- RiskOperationResult{
			Success: true,
			Data:    addedLimit,
		}
	}
}

// Stop gracefully stops the batch processor
func (bp *BatchProcessor) Stop() {
	bp.logger.Info("Stopping batch processor")
	bp.cancel()
	bp.wg.Wait()
	close(bp.riskBatchChan)
	bp.logger.Info("Batch processor stopped")
}

// GetQueueSize returns the current queue size
func (bp *BatchProcessor) GetQueueSize() int {
	return len(bp.riskBatchChan)
}
