package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OrderServiceImpl implements the OrderService interface
type OrderServiceImpl struct {
	// In a real implementation, these would be proper repositories
	orders map[string]*Order
	riskService RiskService
}

// NewOrderService creates a new order service instance
func NewOrderService(riskService RiskService) OrderService {
	return &OrderServiceImpl{
		orders: make(map[string]*Order),
		riskService: riskService,
	}
}

// CreateOrder creates a new trading order
func (s *OrderServiceImpl) CreateOrder(ctx context.Context, order *Order) (*Order, error) {
	if order == nil {
		return nil, fmt.Errorf("order cannot be nil")
	}

	// Generate ID if not provided
	if order.ID == "" {
		order.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now
	order.Status = "pending"

	// Validate order
	if err := s.validateOrder(order); err != nil {
		return nil, fmt.Errorf("order validation failed: %w", err)
	}

	// Check risk if risk service is available
	if s.riskService != nil {
		riskResult, err := s.riskService.CheckRisk(ctx, order)
		if err != nil {
			return nil, fmt.Errorf("risk check failed: %w", err)
		}
		if !riskResult.Approved {
			return nil, fmt.Errorf("order rejected by risk management: %v", riskResult.Reasons)
		}
	}

	// Store order
	s.orders[order.ID] = order

	return order, nil
}

// UpdateOrder updates an existing order
func (s *OrderServiceImpl) UpdateOrder(ctx context.Context, id string, updates *OrderUpdate) (*Order, error) {
	order, exists := s.orders[id]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", id)
	}

	// Check if order can be updated
	if order.Status == "filled" || order.Status == "cancelled" {
		return nil, fmt.Errorf("cannot update order in status: %s", order.Status)
	}

	// Apply updates
	if updates.Quantity != nil {
		order.Quantity = *updates.Quantity
	}
	if updates.Price != nil {
		order.Price = *updates.Price
	}
	if updates.StopPrice != nil {
		order.StopPrice = *updates.StopPrice
	}

	order.UpdatedAt = time.Now()

	// Validate updated order
	if err := s.validateOrder(order); err != nil {
		return nil, fmt.Errorf("updated order validation failed: %w", err)
	}

	return order, nil
}

// CancelOrder cancels an existing order
func (s *OrderServiceImpl) CancelOrder(ctx context.Context, id string) error {
	order, exists := s.orders[id]
	if !exists {
		return fmt.Errorf("order not found: %s", id)
	}

	// Check if order can be cancelled
	if order.Status == "filled" || order.Status == "cancelled" {
		return fmt.Errorf("cannot cancel order in status: %s", order.Status)
	}

	order.Status = "cancelled"
	order.UpdatedAt = time.Now()

	return nil
}

// GetOrder retrieves an order by ID
func (s *OrderServiceImpl) GetOrder(ctx context.Context, id string) (*Order, error) {
	order, exists := s.orders[id]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", id)
	}

	return order, nil
}

// ListOrders retrieves orders based on filter criteria
func (s *OrderServiceImpl) ListOrders(ctx context.Context, filter *OrderFilter) ([]*Order, error) {
	var result []*Order

	for _, order := range s.orders {
		if s.matchesFilter(order, filter) {
			result = append(result, order)
		}
	}

	// Apply pagination
	if filter != nil {
		start := filter.Offset
		if start > len(result) {
			start = len(result)
		}

		end := start + filter.Limit
		if filter.Limit == 0 || end > len(result) {
			end = len(result)
		}

		if start < end {
			result = result[start:end]
		} else {
			result = []*Order{}
		}
	}

	return result, nil
}

// ExecuteOrder executes an order (simplified implementation)
func (s *OrderServiceImpl) ExecuteOrder(ctx context.Context, id string) (*ExecutionResult, error) {
	order, exists := s.orders[id]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", id)
	}

	if order.Status != "pending" {
		return nil, fmt.Errorf("order cannot be executed, status: %s", order.Status)
	}

	// Simulate execution
	executedPrice := order.Price
	if order.Type == "market" {
		// For market orders, simulate current market price
		executedPrice = order.Price * (1 + 0.001) // Small spread simulation
	}

	now := time.Now()
	order.Status = "filled"
	order.UpdatedAt = now
	order.ExecutedAt = &now

	result := &ExecutionResult{
		OrderID:       order.ID,
		ExecutedPrice: executedPrice,
		ExecutedQty:   order.Quantity,
		Commission:    order.Quantity * executedPrice * 0.001, // 0.1% commission
		ExecutedAt:    now,
	}

	return result, nil
}

// GetOrderStatus retrieves the current status of an order
func (s *OrderServiceImpl) GetOrderStatus(ctx context.Context, id string) (*OrderStatus, error) {
	order, exists := s.orders[id]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", id)
	}

	status := &OrderStatus{
		ID:              order.ID,
		Status:          order.Status,
		LastUpdated:     order.UpdatedAt,
	}

	// Set quantities based on status
	if order.Status == "filled" {
		status.FilledQuantity = order.Quantity
		status.RemainingQuantity = 0
		status.AveragePrice = order.Price
	} else {
		status.FilledQuantity = 0
		status.RemainingQuantity = order.Quantity
		status.AveragePrice = 0
	}

	return status, nil
}

// validateOrder validates order parameters
func (s *OrderServiceImpl) validateOrder(order *Order) error {
	if order.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if order.Side != "buy" && order.Side != "sell" {
		return fmt.Errorf("side must be 'buy' or 'sell'")
	}
	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if order.Type == "limit" && order.Price <= 0 {
		return fmt.Errorf("price must be positive for limit orders")
	}
	if order.Type == "stop" && order.StopPrice <= 0 {
		return fmt.Errorf("stop price must be positive for stop orders")
	}

	return nil
}

// matchesFilter checks if an order matches the given filter
func (s *OrderServiceImpl) matchesFilter(order *Order, filter *OrderFilter) bool {
	if filter == nil {
		return true
	}

	if filter.AccountID != nil && order.AccountID != *filter.AccountID {
		return false
	}
	if filter.Symbol != nil && order.Symbol != *filter.Symbol {
		return false
	}
	if filter.Side != nil && order.Side != *filter.Side {
		return false
	}
	if filter.Status != nil && order.Status != *filter.Status {
		return false
	}
	if filter.From != nil && order.CreatedAt.Before(*filter.From) {
		return false
	}
	if filter.To != nil && order.CreatedAt.After(*filter.To) {
		return false
	}

	return true
}
