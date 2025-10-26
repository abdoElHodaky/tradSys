package orders

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// orderOperation represents a batch operation on orders
type orderOperation struct {
	opType    string
	order     *Order
	requestID string
	resultCh  chan orderOperationResult
}

// orderOperationResult represents the result of a batch operation
type orderOperationResult struct {
	order *Order
	err   error
}

// BatchProcessor handles batch processing of order operations
type BatchProcessor struct {
	service        *Service
	logger         *zap.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	orderBatchChan chan orderOperation
	wg             sync.WaitGroup
	
	// Configuration
	batchSize     int
	batchTimeout  time.Duration
	workerCount   int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(service *Service, logger *zap.Logger) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	bp := &BatchProcessor{
		service:        service,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		orderBatchChan: make(chan orderOperation, 1000),
		batchSize:      100,
		batchTimeout:   100 * time.Millisecond,
		workerCount:    4,
	}
	
	return bp
}

// Start starts the batch processor
func (bp *BatchProcessor) Start() {
	bp.logger.Info("Starting batch processor", 
		zap.Int("batch_size", bp.batchSize),
		zap.Duration("batch_timeout", bp.batchTimeout),
		zap.Int("worker_count", bp.workerCount))
	
	// Start batch processing workers
	for i := 0; i < bp.workerCount; i++ {
		bp.wg.Add(1)
		go bp.processBatchOperations(i)
	}
}

// Stop stops the batch processor
func (bp *BatchProcessor) Stop() {
	bp.logger.Info("Stopping batch processor")
	bp.cancel()
	close(bp.orderBatchChan)
	bp.wg.Wait()
	bp.logger.Info("Batch processor stopped")
}

// SubmitOperation submits an operation for batch processing
func (bp *BatchProcessor) SubmitOperation(op orderOperation) {
	select {
	case bp.orderBatchChan <- op:
		// Operation submitted successfully
	case <-bp.ctx.Done():
		// Service is shutting down
		if op.resultCh != nil {
			op.resultCh <- orderOperationResult{
				order: nil,
				err:   context.Canceled,
			}
		}
	default:
		// Channel is full, handle overflow
		bp.logger.Warn("Batch channel full, dropping operation",
			zap.String("op_type", op.opType),
			zap.String("request_id", op.requestID))
		if op.resultCh != nil {
			op.resultCh <- orderOperationResult{
				order: nil,
				err:   ErrBatchChannelFull,
			}
		}
	}
}

// processBatchOperations processes batch operations for orders
func (bp *BatchProcessor) processBatchOperations(workerID int) {
	defer bp.wg.Done()
	
	bp.logger.Info("Starting batch processor worker", zap.Int("worker_id", workerID))
	
	ticker := time.NewTicker(bp.batchTimeout)
	defer ticker.Stop()

	batch := make([]orderOperation, 0, bp.batchSize)

	for {
		select {
		case <-bp.ctx.Done():
			// Process remaining batch before shutdown
			if len(batch) > 0 {
				bp.processBatch(batch, workerID)
			}
			bp.logger.Info("Batch processor worker stopped", zap.Int("worker_id", workerID))
			return

		case op, ok := <-bp.orderBatchChan:
			if !ok {
				// Channel closed, process remaining batch
				if len(batch) > 0 {
					bp.processBatch(batch, workerID)
				}
				return
			}

			batch = append(batch, op)
			if len(batch) >= bp.batchSize {
				bp.processBatch(batch, workerID)
				batch = batch[:0] // Reset batch
			}

		case <-ticker.C:
			if len(batch) > 0 {
				bp.processBatch(batch, workerID)
				batch = batch[:0] // Reset batch
			}
		}
	}
}

// processBatch processes a batch of operations
func (bp *BatchProcessor) processBatch(batch []orderOperation, workerID int) {
	if len(batch) == 0 {
		return
	}

	start := time.Now()
	bp.logger.Debug("Processing batch",
		zap.Int("worker_id", workerID),
		zap.Int("batch_size", len(batch)))

	// Group operations by type for efficient processing
	addOps := make([]orderOperation, 0)
	updateOps := make([]orderOperation, 0)
	cancelOps := make([]orderOperation, 0)

	for _, op := range batch {
		switch op.opType {
		case "add":
			addOps = append(addOps, op)
		case "update":
			updateOps = append(updateOps, op)
		case "cancel":
			cancelOps = append(cancelOps, op)
		default:
			bp.logger.Warn("Unknown operation type", 
				zap.String("op_type", op.opType),
				zap.Int("worker_id", workerID))
			if op.resultCh != nil {
				op.resultCh <- orderOperationResult{
					order: nil,
					err:   ErrInvalidOperationType,
				}
			}
		}
	}

	// Process each type of operation
	if len(addOps) > 0 {
		bp.processAddBatch(addOps, workerID)
	}
	if len(updateOps) > 0 {
		bp.processUpdateBatch(updateOps, workerID)
	}
	if len(cancelOps) > 0 {
		bp.processCancelBatch(cancelOps, workerID)
	}

	duration := time.Since(start)
	bp.logger.Debug("Batch processed",
		zap.Int("worker_id", workerID),
		zap.Int("total_ops", len(batch)),
		zap.Int("add_ops", len(addOps)),
		zap.Int("update_ops", len(updateOps)),
		zap.Int("cancel_ops", len(cancelOps)),
		zap.Duration("duration", duration))
}

// processAddBatch processes a batch of add operations
func (bp *BatchProcessor) processAddBatch(ops []orderOperation, workerID int) {
	bp.service.mu.Lock()
	defer bp.service.mu.Unlock()

	for _, op := range ops {
		order := op.order
		if order == nil {
			if op.resultCh != nil {
				op.resultCh <- orderOperationResult{
					order: nil,
					err:   ErrInvalidOrder,
				}
			}
			continue
		}

		// Add order to service maps
		bp.service.Orders[order.ID] = order

		// Update user orders index
		if bp.service.UserOrders[order.UserID] == nil {
			bp.service.UserOrders[order.UserID] = make(map[string]bool)
		}
		bp.service.UserOrders[order.UserID][order.ID] = true

		// Update symbol orders index
		if bp.service.SymbolOrders[order.Symbol] == nil {
			bp.service.SymbolOrders[order.Symbol] = make(map[string]bool)
		}
		bp.service.SymbolOrders[order.Symbol][order.ID] = true

		// Update client order ID mapping
		if order.ClientOrderID != "" {
			bp.service.ClientOrderIDs[order.ClientOrderID] = order.ID
		}

		// Cache the order
		bp.service.OrderCache.Set(order.ID, order, cache.DefaultExpiration)

		if op.resultCh != nil {
			op.resultCh <- orderOperationResult{
				order: order,
				err:   nil,
			}
		}
	}

	bp.logger.Debug("Processed add batch",
		zap.Int("worker_id", workerID),
		zap.Int("count", len(ops)))
}

// processUpdateBatch processes a batch of update operations
func (bp *BatchProcessor) processUpdateBatch(ops []orderOperation, workerID int) {
	bp.service.mu.Lock()
	defer bp.service.mu.Unlock()

	for _, op := range ops {
		order := op.order
		if order == nil {
			if op.resultCh != nil {
				op.resultCh <- orderOperationResult{
					order: nil,
					err:   ErrInvalidOrder,
				}
			}
			continue
		}

		// Update order in service
		existingOrder, exists := bp.service.Orders[order.ID]
		if !exists {
			if op.resultCh != nil {
				op.resultCh <- orderOperationResult{
					order: nil,
					err:   ErrOrderNotFound,
				}
			}
			continue
		}

		// Update the order
		*existingOrder = *order
		existingOrder.UpdatedAt = time.Now()

		// Update cache
		bp.service.OrderCache.Set(order.ID, existingOrder, cache.DefaultExpiration)

		if op.resultCh != nil {
			op.resultCh <- orderOperationResult{
				order: existingOrder,
				err:   nil,
			}
		}
	}

	bp.logger.Debug("Processed update batch",
		zap.Int("worker_id", workerID),
		zap.Int("count", len(ops)))
}

// processCancelBatch processes a batch of cancel operations
func (bp *BatchProcessor) processCancelBatch(ops []orderOperation, workerID int) {
	bp.service.mu.Lock()
	defer bp.service.mu.Unlock()

	for _, op := range ops {
		order := op.order
		if order == nil {
			if op.resultCh != nil {
				op.resultCh <- orderOperationResult{
					order: nil,
					err:   ErrInvalidOrder,
				}
			}
			continue
		}

		// Find and cancel order
		existingOrder, exists := bp.service.Orders[order.ID]
		if !exists {
			if op.resultCh != nil {
				op.resultCh <- orderOperationResult{
					order: nil,
					err:   ErrOrderNotFound,
				}
			}
			continue
		}

		// Update order status
		existingOrder.UpdateStatus(OrderStatusCancelled)

		// Update cache
		bp.service.OrderCache.Set(order.ID, existingOrder, cache.DefaultExpiration)

		if op.resultCh != nil {
			op.resultCh <- orderOperationResult{
				order: existingOrder,
				err:   nil,
			}
		}
	}

	bp.logger.Debug("Processed cancel batch",
		zap.Int("worker_id", workerID),
		zap.Int("count", len(ops)))
}

// GetStats returns batch processor statistics
func (bp *BatchProcessor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"batch_size":     bp.batchSize,
		"batch_timeout":  bp.batchTimeout.String(),
		"worker_count":   bp.workerCount,
		"channel_length": len(bp.orderBatchChan),
		"channel_cap":    cap(bp.orderBatchChan),
	}
}
