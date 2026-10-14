package orders

import (
	"time"

	cache "github.com/patrickmn/go-cache"
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

// processBatchOperations processes batch operations for orders
func (s *Service) processBatchOperations() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]orderOperation, 0, 100)

	for {
		select {
		case <-s.ctx.Done():
			return
		case op := <-s.orderBatchChan:
			batch = append(batch, op)

			// Process batch if it's full
			if len(batch) >= 100 {
				s.processBatch(batch)
				batch = make([]orderOperation, 0, 100)
			}
		case <-ticker.C:
			// Process remaining operations in batch
			if len(batch) > 0 {
				s.processBatch(batch)
				batch = make([]orderOperation, 0, 100)
			}
		}
	}
}

// processBatch processes a batch of order operations
func (s *Service) processBatch(batch []orderOperation) {
	// Group operations by type
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
		}
	}

	// Process add operations
	if len(addOps) > 0 {
		s.processAddBatch(addOps)
	}

	// Process update operations
	if len(updateOps) > 0 {
		s.processUpdateBatch(updateOps)
	}

	// Process cancel operations
	if len(cancelOps) > 0 {
		s.processCancelBatch(cancelOps)
	}
}

// processAddBatch processes a batch of add operations
func (s *Service) processAddBatch(ops []orderOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		order := op.order

		// Add order to maps
		s.Orders[order.ID] = order

		// Add to user orders
		if _, exists := s.UserOrders[order.UserID]; !exists {
			s.UserOrders[order.UserID] = make(map[string]bool)
		}
		s.UserOrders[order.UserID][order.ID] = true

		// Add to symbol orders
		if _, exists := s.SymbolOrders[order.Symbol]; !exists {
			s.SymbolOrders[order.Symbol] = make(map[string]bool)
		}
		s.SymbolOrders[order.Symbol][order.ID] = true

		// Add to client order IDs
		if order.ClientOrderID != "" {
			s.ClientOrderIDs[order.ClientOrderID] = order.ID
		}

		// Add to cache
		s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)
		if order.ClientOrderID != "" {
			s.OrderCache.Set("client:"+order.ClientOrderID, order.ID, cache.DefaultExpiration)
		}

		// Send result
		op.resultCh <- orderOperationResult{order: order, err: nil}
	}
}

// processUpdateBatch processes a batch of update operations
func (s *Service) processUpdateBatch(ops []orderOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		order := op.order

		// Update order in maps
		s.Orders[order.ID] = order

		// Update in cache
		s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)

		// Send result
		op.resultCh <- orderOperationResult{order: order, err: nil}
	}
}

// processCancelBatch processes a batch of cancel operations
func (s *Service) processCancelBatch(ops []orderOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		orderID := op.requestID
		order, exists := s.Orders[orderID]

		if !exists {
			op.resultCh <- orderOperationResult{order: nil, err: ErrOrderNotFound}
			continue
		}

		// Update order status
		order.Status = OrderStatusCancelled
		order.UpdatedAt = time.Now()

		// Update in cache
		s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)

		// Send result
		op.resultCh <- orderOperationResult{order: order, err: nil}
	}
}