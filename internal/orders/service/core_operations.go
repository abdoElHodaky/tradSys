package service

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// CancelOrder cancels an order using early return pattern and state machine
func (s *Service) CancelOrder(req *OrderCancelRequest) error {
	if req == nil {
		return errors.New("cancel request cannot be nil")
	}
	
	// Validate cancel request
	if err := s.cancelValidator.ValidateCancelRequest(req); err != nil {
		return err
	}
	
	// Get order ID
	orderID, err := s.resolveOrderID(req)
	if err != nil {
		return err
	}
	
	// Get order
	order, err := s.GetOrder(orderID)
	if err != nil {
		return err
	}
	
	// Check ownership
	if order.UserID != req.UserID {
		return errors.New("unauthorized: order belongs to different user")
	}
	
	// Use state machine instead of switch statement
	if err := s.stateMachine.HandleEvent(order, "CANCEL"); err != nil {
		return fmt.Errorf("cancellation failed: %w", err)
	}
	
	s.logger.Info("Order cancelled successfully", 
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID))
	
	return nil
}

// UpdateOrder updates an order using early return pattern
func (s *Service) UpdateOrder(req *OrderUpdateRequest) (*Order, error) {
	if req == nil {
		return nil, errors.New("update request cannot be nil")
	}
	
	// Validate update request
	if err := s.updateValidator.ValidateUpdateRequest(req); err != nil {
		return nil, err
	}
	
	// Get order ID
	orderID, err := s.resolveOrderIDFromUpdate(req)
	if err != nil {
		return nil, err
	}
	
	// Get order
	order, err := s.GetOrder(orderID)
	if err != nil {
		return nil, err
	}
	
	// Check ownership
	if order.UserID != req.UserID {
		return nil, errors.New("unauthorized: order belongs to different user")
	}
	
	// Check if order can be updated
	if !s.canUpdateOrder(order) {
		return nil, fmt.Errorf("order cannot be updated in current state: %s", order.Status)
	}
	
	// Apply updates
	s.applyOrderUpdates(order, req)
	
	s.logger.Info("Order updated successfully", 
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID))
	
	return order, nil
}

// ListOrders lists orders with filtering using early return pattern
func (s *Service) ListOrders(filter *OrderFilter) ([]*Order, error) {
	if filter == nil {
		return nil, errors.New("filter cannot be nil")
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var orders []*Order
	
	// Filter by user if specified
	if filter.UserID != "" {
		userOrderIDs, exists := s.userOrders[filter.UserID]
		if !exists {
			return orders, nil // Return empty slice
		}
		
		for orderID := range userOrderIDs {
			order := s.orders[orderID]
			if s.matchesFilter(order, filter) {
				orders = append(orders, order)
			}
		}
		return orders, nil
	}
	
	// Filter by symbol if specified
	if filter.Symbol != "" {
		symbolOrderIDs, exists := s.symbolOrders[filter.Symbol]
		if !exists {
			return orders, nil // Return empty slice
		}
		
		for orderID := range symbolOrderIDs {
			order := s.orders[orderID]
			if s.matchesFilter(order, filter) {
				orders = append(orders, order)
			}
		}
		return orders, nil
	}
	
	// No specific filter - return all orders (with other filters applied)
	for _, order := range s.orders {
		if s.matchesFilter(order, filter) {
			orders = append(orders, order)
		}
	}
	
	return orders, nil
}

// Helper methods for order operations

// resolveOrderID resolves order ID from cancel request
func (s *Service) resolveOrderID(req *OrderCancelRequest) (string, error) {
	if req.OrderID != "" {
		return req.OrderID, nil
	}
	
	if req.ClientOrderID != "" {
		s.mu.RLock()
		orderID, exists := s.clientOrderIDs[req.ClientOrderID]
		s.mu.RUnlock()
		
		if !exists {
			return "", fmt.Errorf("order not found with client order ID: %s", req.ClientOrderID)
		}
		return orderID, nil
	}
	
	return "", errors.New("either order ID or client order ID must be provided")
}

// resolveOrderIDFromUpdate resolves order ID from update request
func (s *Service) resolveOrderIDFromUpdate(req *OrderUpdateRequest) (string, error) {
	if req.OrderID != "" {
		return req.OrderID, nil
	}
	
	if req.ClientOrderID != "" {
		s.mu.RLock()
		orderID, exists := s.clientOrderIDs[req.ClientOrderID]
		s.mu.RUnlock()
		
		if !exists {
			return "", fmt.Errorf("order not found with client order ID: %s", req.ClientOrderID)
		}
		return orderID, nil
	}
	
	return "", errors.New("either order ID or client order ID must be provided")
}

// canUpdateOrder checks if order can be updated
func (s *Service) canUpdateOrder(order *Order) bool {
	return order.Status == OrderStatusNew || order.Status == OrderStatusPending
}

// applyOrderUpdates applies updates to an order
func (s *Service) applyOrderUpdates(order *Order, req *OrderUpdateRequest) {
	if req.Price > 0 {
		order.Price = req.Price
	}
	if req.StopPrice > 0 {
		order.StopPrice = req.StopPrice
	}
	if req.Quantity > 0 {
		order.Quantity = req.Quantity
	}
	if req.TimeInForce != "" {
		order.TimeInForce = req.TimeInForce
	}
	if !req.ExpiresAt.IsZero() {
		order.ExpiresAt = req.ExpiresAt
	}
	
	order.UpdatedAt = time.Now()
}

// matchesFilter checks if order matches filter criteria
func (s *Service) matchesFilter(order *Order, filter *OrderFilter) bool {
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
