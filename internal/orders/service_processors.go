// 🎯 **Order Service Processors**
// Generated using TradSys Code Splitting Standards
//
// This file contains the batch processing logic, validation methods, and helper functions
// for the Order Management Service. These functions handle order processing workflows,
// expiry checking, and various utility operations.
//
// Performance Requirements: Standard latency, batch processing optimization
// File size limit: 410 lines

package orders

import (
	"time"

	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

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
		case OpTypeAdd:
			addOps = append(addOps, op)
		case OpTypeUpdate:
			updateOps = append(updateOps, op)
		case OpTypeCancel:
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
		s.OrderCache.Set(CacheKeyPrefix+order.ID, order, cache.DefaultExpiration)
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
		s.OrderCache.Set(CacheKeyPrefix+order.ID, order, cache.DefaultExpiration)

		// Send result
		op.resultCh <- orderOperationResult{order: order, err: nil}
	}
}

// processCancelBatch processes a batch of cancel operations
func (s *Service) processCancelBatch(ops []orderOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		order := op.order

		// Update order status
		order.Status = OrderStatusCancelled
		order.UpdatedAt = time.Now()

		// Update in maps
		s.Orders[order.ID] = order

		// Update in cache
		s.OrderCache.Set(CacheKeyPrefix+order.ID, order, cache.DefaultExpiration)

		// Send result
		op.resultCh <- orderOperationResult{order: order, err: nil}
	}
}

// checkOrderExpiry checks for expired orders and cancels them
func (s *Service) checkOrderExpiry() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processExpiredOrders()
		}
	}
}

// processExpiredOrders processes expired orders
func (s *Service) processExpiredOrders() {
	now := time.Now()
	expiredOrders := make([]*Order, 0)

	s.mu.RLock()
	for _, order := range s.Orders {
		if !order.ExpiresAt.IsZero() && order.ExpiresAt.Before(now) {
			if order.Status == OrderStatusNew || order.Status == OrderStatusPending || order.Status == OrderStatusPartiallyFilled {
				expiredOrders = append(expiredOrders, order)
			}
		}
	}
	s.mu.RUnlock()

	// Cancel expired orders
	for _, order := range expiredOrders {
		s.mu.Lock()
		order.Status = OrderStatusExpired
		order.UpdatedAt = now
		s.mu.Unlock()

		// Update cache
		s.OrderCache.Set(CacheKeyPrefix+order.ID, order, cache.DefaultExpiration)

		// Cancel in matching engine
		if err := s.Engine.CancelOrder(order.ID, order.Symbol); err != nil {
			s.logger.Error("Failed to cancel expired order in engine",
				zap.String("order_id", order.ID),
				zap.Error(err))
		}

		s.logger.Info("Order expired and cancelled",
			zap.String("order_id", order.ID),
			zap.String("user_id", order.UserID),
			zap.String("symbol", order.Symbol),
			zap.Time("expires_at", order.ExpiresAt))
	}
}

// addOrder adds an order to the service
func (s *Service) addOrder(order *Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if order already exists
	if _, exists := s.Orders[order.ID]; exists {
		return ErrOrderAlreadyExists
	}

	// Add to orders map
	s.Orders[order.ID] = order

	// Add to user orders
	if s.UserOrders[order.UserID] == nil {
		s.UserOrders[order.UserID] = make(map[string]bool)
	}
	s.UserOrders[order.UserID][order.ID] = true

	// Add to symbol orders
	if s.SymbolOrders[order.Symbol] == nil {
		s.SymbolOrders[order.Symbol] = make(map[string]bool)
	}
	s.SymbolOrders[order.Symbol][order.ID] = true

	// Add client order ID mapping
	if order.ClientOrderID != "" {
		s.ClientOrderIDs[order.ClientOrderID] = order.ID
	}

	// Cache the order
	s.OrderCache.Set(CacheKeyPrefix+order.ID, order, cache.DefaultExpiration)

	return nil
}

// validateOrderRequest validates an order request
func (s *Service) validateOrderRequest(req *OrderRequest) error {
	if req == nil {
		return ErrInvalidOrderRequest
	}

	if req.UserID == "" {
		return ErrInvalidOrderRequest
	}

	if req.Symbol == "" {
		return ErrInvalidSymbol
	}

	if req.Quantity <= 0 {
		return ErrInvalidQuantity
	}

	if req.Type == OrderTypeLimit && req.Price <= 0 {
		return ErrInvalidPrice
	}

	if (req.Type == OrderTypeStopLimit || req.Type == OrderTypeStopMarket) && req.StopPrice <= 0 {
		return ErrInvalidPrice
	}

	return nil
}

// checkUserOrderLimits checks if user has exceeded order limits
func (s *Service) checkUserOrderLimits(userID string) error {
	s.mu.RLock()
	userOrders := s.UserOrders[userID]
	s.mu.RUnlock()

	if len(userOrders) >= DefaultMaxOrdersPerUser {
		return ErrMaxOrdersExceeded
	}

	return nil
}

// canCancelOrder checks if an order can be cancelled
func (s *Service) canCancelOrder(order *Order) bool {
	return order.Status == OrderStatusNew || order.Status == OrderStatusPending || order.Status == OrderStatusPartiallyFilled
}

// canUpdateOrder checks if an order can be updated
func (s *Service) canUpdateOrder(order *Order) bool {
	return order.Status == OrderStatusNew || order.Status == OrderStatusPending
}

// matchesFilter checks if an order matches the given filter
func (s *Service) matchesFilter(order *Order, filter *OrderFilter) bool {
	if filter.UserID != "" && order.UserID != filter.UserID {
		return false
	}

	if filter.Symbol != "" && order.Symbol != filter.Symbol {
		return false
	}

	if filter.Side != "" && order.Side != filter.Side {
		return false
	}

	if filter.Type != "" && order.Type != filter.Type {
		return false
	}

	if filter.Status != "" && order.Status != filter.Status {
		return false
	}

	if !filter.StartTime.IsZero() && order.CreatedAt.Before(filter.StartTime) {
		return false
	}

	if !filter.EndTime.IsZero() && order.CreatedAt.After(filter.EndTime) {
		return false
	}

	return true
}

// updateOrderStatus updates the status of an order
func (s *Service) updateOrderStatus(orderID string, status OrderStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.Orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}

	order.Status = status
	order.UpdatedAt = time.Now()

	// Update cache
	s.OrderCache.Set(CacheKeyPrefix+orderID, order, cache.DefaultExpiration)

	return nil
}

// addTradeToOrder adds a trade to an order
func (s *Service) addTradeToOrder(orderID string, trade *Trade) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.Orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}

	// Add trade to order
	order.Trades = append(order.Trades, trade)
	order.FilledQuantity += trade.Quantity
	order.UpdatedAt = time.Now()

	// Update order status based on filled quantity
	if order.FilledQuantity >= order.Quantity {
		order.Status = OrderStatusFilled
	} else if order.FilledQuantity > 0 {
		order.Status = OrderStatusPartiallyFilled
	}

	// Update cache
	s.OrderCache.Set(CacheKeyPrefix+orderID, order, cache.DefaultExpiration)

	return nil
}
