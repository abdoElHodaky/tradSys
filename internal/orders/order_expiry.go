package orders

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// checkOrderExpiry checks for expired orders
func (s *Service) checkOrderExpiry() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()

			// Get expired orders
			s.mu.RLock()
			expiredOrders := make([]*Order, 0)
			for _, order := range s.Orders {
				if !order.ExpiresAt.IsZero() && now.After(order.ExpiresAt) && (order.Status == OrderStatusNew || order.Status == OrderStatusPartiallyFilled) {
					expiredOrders = append(expiredOrders, order)
				}
			}
			s.mu.RUnlock()

			// Process in batches for better performance
			if len(expiredOrders) > 0 {
				batchSize := 100
				for i := 0; i < len(expiredOrders); i += batchSize {
					end := min(i+batchSize, len(expiredOrders))

					batch := expiredOrders[i:end]
					s.processExpiredOrdersBatch(batch, now)
				}
			}
		}
	}
}

// processExpiredOrdersBatch processes a batch of expired orders
func (s *Service) processExpiredOrdersBatch(orders []*Order, now time.Time) {
	// Create a wait group for concurrent processing
	var wg sync.WaitGroup
	wg.Add(len(orders))

	// Process orders concurrently
	for _, order := range orders {
		go func(order *Order) {
			defer wg.Done()

			// Cancel order in matching engine
			err := s.Engine.CancelOrder(order.Symbol, order.ID)
			if err != nil {
				s.logger.Error("Failed to cancel expired order",
					zap.String("order_id", order.ID),
					zap.Error(err))
				return
			}

			// Update order status using batch operation
			resultCh := make(chan orderOperationResult, 1)
			order.Status = OrderStatusExpired
			order.UpdatedAt = now

			s.orderBatchChan <- orderOperation{
				opType:    "update",
				order:     order,
				requestID: order.ID,
				resultCh:  resultCh,
			}

			// Wait for result
			<-resultCh

			s.logger.Info("Order expired",
				zap.String("order_id", order.ID),
				zap.String("symbol", order.Symbol),
				zap.String("user_id", order.UserID))
		}(order)
	}

	// Wait for all orders to be processed
	wg.Wait()
}