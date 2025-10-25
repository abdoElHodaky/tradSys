package orders

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// OrderLifecycle manages the lifecycle of orders
type OrderLifecycle struct {
	orderService *OrderService
	logger       *zap.Logger
	mu           sync.RWMutex
	
	// Lifecycle tracking
	orderStates map[string]*OrderState
	
	// Background processing
	ctx    context.Context
	cancel context.CancelFunc
	
	// Channels for lifecycle events
	stateChangeChan chan *OrderStateChange
	expirationChan  chan *OrderExpiration
}

// OrderState represents the internal state of an order
type OrderState struct {
	OrderID       string
	CurrentStatus OrderStatus
	PreviousStatus OrderStatus
	StateChangedAt time.Time
	ExpiresAt     time.Time
	Metadata      map[string]interface{}
}

// OrderStateChange represents a state change event
type OrderStateChange struct {
	OrderID       string
	FromStatus    OrderStatus
	ToStatus      OrderStatus
	Reason        string
	Timestamp     time.Time
	Metadata      map[string]interface{}
}

// OrderExpiration represents an order expiration event
type OrderExpiration struct {
	OrderID   string
	ExpiresAt time.Time
}

// NewOrderLifecycle creates a new order lifecycle manager
func NewOrderLifecycle(orderService *OrderService, logger *zap.Logger) *OrderLifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &OrderLifecycle{
		orderService:    orderService,
		logger:          logger,
		orderStates:     make(map[string]*OrderState),
		ctx:             ctx,
		cancel:          cancel,
		stateChangeChan: make(chan *OrderStateChange, 1000),
		expirationChan:  make(chan *OrderExpiration, 1000),
	}
}

// InitializeOrder initializes the lifecycle for a new order
func (ol *OrderLifecycle) InitializeOrder(ctx context.Context, order *Order) error {
	ol.mu.Lock()
	defer ol.mu.Unlock()

	// Create order state
	state := &OrderState{
		OrderID:        order.ID,
		CurrentStatus:  order.Status,
		PreviousStatus: "",
		StateChangedAt: time.Now(),
		ExpiresAt:      order.ExpiresAt,
		Metadata:       make(map[string]interface{}),
	}

	ol.orderStates[order.ID] = state

	// Schedule expiration if needed
	if !order.ExpiresAt.IsZero() {
		ol.scheduleExpiration(order.ID, order.ExpiresAt)
	}

	// Emit state change event
	ol.emitStateChange(&OrderStateChange{
		OrderID:    order.ID,
		FromStatus: "",
		ToStatus:   order.Status,
		Reason:     "order_created",
		Timestamp:  time.Now(),
		Metadata:   map[string]interface{}{"user_id": order.UserID, "symbol": order.Symbol},
	})

	ol.logger.Debug("Order lifecycle initialized",
		zap.String("order_id", order.ID),
		zap.String("status", string(order.Status)))

	return nil
}

// UpdateOrder updates the lifecycle state when an order is modified
func (ol *OrderLifecycle) UpdateOrder(ctx context.Context, order *Order) error {
	ol.mu.Lock()
	defer ol.mu.Unlock()

	state, exists := ol.orderStates[order.ID]
	if !exists {
		return ErrOrderStateNotFound
	}

	// Update expiration if changed
	if !order.ExpiresAt.IsZero() && !state.ExpiresAt.Equal(order.ExpiresAt) {
		state.ExpiresAt = order.ExpiresAt
		ol.scheduleExpiration(order.ID, order.ExpiresAt)
	}

	ol.logger.Debug("Order lifecycle updated",
		zap.String("order_id", order.ID))

	return nil
}

// CancelOrder handles order cancellation in the lifecycle
func (ol *OrderLifecycle) CancelOrder(ctx context.Context, order *Order) error {
	return ol.changeOrderStatus(order, OrderStatusCancelled, "user_cancelled")
}

// UpdateOrderAfterExecution updates order status after execution
func (ol *OrderLifecycle) UpdateOrderAfterExecution(ctx context.Context, order *Order) error {
	// Determine new status based on fill
	var newStatus OrderStatus
	var reason string

	if order.FilledQuantity >= order.Quantity {
		newStatus = OrderStatusFilled
		reason = "fully_filled"
	} else if order.FilledQuantity > 0 {
		newStatus = OrderStatusPartiallyFilled
		reason = "partially_filled"
	} else {
		newStatus = OrderStatusPending
		reason = "submitted_to_market"
	}

	return ol.changeOrderStatus(order, newStatus, reason)
}

// ExpireOrder handles order expiration
func (ol *OrderLifecycle) ExpireOrder(ctx context.Context, orderID string) error {
	ol.mu.RLock()
	state, exists := ol.orderStates[orderID]
	ol.mu.RUnlock()

	if !exists {
		return ErrOrderStateNotFound
	}

	// Get the order
	order, err := ol.orderService.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Only expire orders that can be expired
	if ol.canOrderExpire(state.CurrentStatus) {
		return ol.changeOrderStatus(order, OrderStatusExpired, "expired")
	}

	return nil
}

// RejectOrder handles order rejection
func (ol *OrderLifecycle) RejectOrder(ctx context.Context, order *Order, reason string) error {
	return ol.changeOrderStatus(order, OrderStatusRejected, reason)
}

// changeOrderStatus changes the status of an order
func (ol *OrderLifecycle) changeOrderStatus(order *Order, newStatus OrderStatus, reason string) error {
	ol.mu.Lock()
	defer ol.mu.Unlock()

	state, exists := ol.orderStates[order.ID]
	if !exists {
		return ErrOrderStateNotFound
	}

	// Check if status change is valid
	if !ol.isValidStatusTransition(state.CurrentStatus, newStatus) {
		return ErrInvalidStatusTransition
	}

	// Update state
	previousStatus := state.CurrentStatus
	state.PreviousStatus = previousStatus
	state.CurrentStatus = newStatus
	state.StateChangedAt = time.Now()

	// Update order
	order.Status = newStatus
	order.UpdatedAt = time.Now()

	// Emit state change event
	ol.emitStateChange(&OrderStateChange{
		OrderID:    order.ID,
		FromStatus: previousStatus,
		ToStatus:   newStatus,
		Reason:     reason,
		Timestamp:  time.Now(),
		Metadata:   map[string]interface{}{"user_id": order.UserID, "symbol": order.Symbol},
	})

	ol.logger.Info("Order status changed",
		zap.String("order_id", order.ID),
		zap.String("from_status", string(previousStatus)),
		zap.String("to_status", string(newStatus)),
		zap.String("reason", reason))

	return nil
}

// isValidStatusTransition checks if a status transition is valid
func (ol *OrderLifecycle) isValidStatusTransition(from, to OrderStatus) bool {
	validTransitions := map[OrderStatus][]OrderStatus{
		OrderStatusNew: {
			OrderStatusPending,
			OrderStatusPartiallyFilled,
			OrderStatusFilled,
			OrderStatusCancelled,
			OrderStatusRejected,
			OrderStatusExpired,
		},
		OrderStatusPending: {
			OrderStatusPartiallyFilled,
			OrderStatusFilled,
			OrderStatusCancelled,
			OrderStatusExpired,
		},
		OrderStatusPartiallyFilled: {
			OrderStatusFilled,
			OrderStatusCancelled,
			OrderStatusExpired,
		},
		// Terminal states - no transitions allowed
		OrderStatusFilled:    {},
		OrderStatusCancelled: {},
		OrderStatusRejected:  {},
		OrderStatusExpired:   {},
	}

	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedTo := range allowedTransitions {
		if to == allowedTo {
			return true
		}
	}

	return false
}

// canOrderExpire checks if an order can expire
func (ol *OrderLifecycle) canOrderExpire(status OrderStatus) bool {
	return status == OrderStatusNew || 
		   status == OrderStatusPending || 
		   status == OrderStatusPartiallyFilled
}

// scheduleExpiration schedules an order for expiration
func (ol *OrderLifecycle) scheduleExpiration(orderID string, expiresAt time.Time) {
	go func() {
		timer := time.NewTimer(time.Until(expiresAt))
		defer timer.Stop()

		select {
		case <-timer.C:
			ol.expirationChan <- &OrderExpiration{
				OrderID:   orderID,
				ExpiresAt: expiresAt,
			}
		case <-ol.ctx.Done():
			return
		}
	}()
}

// emitStateChange emits a state change event
func (ol *OrderLifecycle) emitStateChange(change *OrderStateChange) {
	select {
	case ol.stateChangeChan <- change:
	default:
		ol.logger.Warn("State change channel full, dropping event",
			zap.String("order_id", change.OrderID))
	}
}

// Start starts the lifecycle manager
func (ol *OrderLifecycle) Start() error {
	ol.logger.Info("Starting order lifecycle manager")

	// Start event processors
	go ol.processStateChanges()
	go ol.processExpirations()

	return nil
}

// Stop stops the lifecycle manager
func (ol *OrderLifecycle) Stop() error {
	ol.logger.Info("Stopping order lifecycle manager")

	ol.cancel()

	// Close channels
	close(ol.stateChangeChan)
	close(ol.expirationChan)

	return nil
}

// processStateChanges processes order state change events
func (ol *OrderLifecycle) processStateChanges() {
	for {
		select {
		case change := <-ol.stateChangeChan:
			if change == nil {
				return
			}

			ol.handleStateChange(change)

		case <-ol.ctx.Done():
			return
		}
	}
}

// processExpirations processes order expiration events
func (ol *OrderLifecycle) processExpirations() {
	for {
		select {
		case expiration := <-ol.expirationChan:
			if expiration == nil {
				return
			}

			ol.handleExpiration(expiration)

		case <-ol.ctx.Done():
			return
		}
	}
}

// handleStateChange handles a state change event
func (ol *OrderLifecycle) handleStateChange(change *OrderStateChange) {
	ol.logger.Debug("Processing state change",
		zap.String("order_id", change.OrderID),
		zap.String("from_status", string(change.FromStatus)),
		zap.String("to_status", string(change.ToStatus)),
		zap.String("reason", change.Reason))

	// Perform any side effects based on state change
	switch change.ToStatus {
	case OrderStatusFilled:
		ol.handleOrderFilled(change)
	case OrderStatusCancelled:
		ol.handleOrderCancelled(change)
	case OrderStatusRejected:
		ol.handleOrderRejected(change)
	case OrderStatusExpired:
		ol.handleOrderExpired(change)
	}
}

// handleExpiration handles an order expiration event
func (ol *OrderLifecycle) handleExpiration(expiration *OrderExpiration) {
	ol.logger.Debug("Processing order expiration",
		zap.String("order_id", expiration.OrderID),
		zap.Time("expires_at", expiration.ExpiresAt))

	// Expire the order
	ctx := context.Background()
	if err := ol.ExpireOrder(ctx, expiration.OrderID); err != nil {
		ol.logger.Error("Failed to expire order",
			zap.String("order_id", expiration.OrderID),
			zap.Error(err))
	}
}

// handleOrderFilled handles when an order is filled
func (ol *OrderLifecycle) handleOrderFilled(change *OrderStateChange) {
	// Perform any cleanup or notifications for filled orders
	ol.logger.Info("Order filled",
		zap.String("order_id", change.OrderID))
}

// handleOrderCancelled handles when an order is cancelled
func (ol *OrderLifecycle) handleOrderCancelled(change *OrderStateChange) {
	// Perform any cleanup for cancelled orders
	ol.logger.Info("Order cancelled",
		zap.String("order_id", change.OrderID))
}

// handleOrderRejected handles when an order is rejected
func (ol *OrderLifecycle) handleOrderRejected(change *OrderStateChange) {
	// Perform any cleanup for rejected orders
	ol.logger.Info("Order rejected",
		zap.String("order_id", change.OrderID),
		zap.String("reason", change.Reason))
}

// handleOrderExpired handles when an order expires
func (ol *OrderLifecycle) handleOrderExpired(change *OrderStateChange) {
	// Perform any cleanup for expired orders
	ol.logger.Info("Order expired",
		zap.String("order_id", change.OrderID))
}

// GetOrderState returns the current state of an order
func (ol *OrderLifecycle) GetOrderState(orderID string) (*OrderState, error) {
	ol.mu.RLock()
	defer ol.mu.RUnlock()

	state, exists := ol.orderStates[orderID]
	if !exists {
		return nil, ErrOrderStateNotFound
	}

	// Return a copy to prevent external modification
	return &OrderState{
		OrderID:        state.OrderID,
		CurrentStatus:  state.CurrentStatus,
		PreviousStatus: state.PreviousStatus,
		StateChangedAt: state.StateChangedAt,
		ExpiresAt:      state.ExpiresAt,
		Metadata:       state.Metadata,
	}, nil
}

// GetOrderHistory returns the state change history for an order
func (ol *OrderLifecycle) GetOrderHistory(orderID string) ([]*OrderStateChange, error) {
	// In a production system, this would query a persistent store
	// For now, return empty history
	return []*OrderStateChange{}, nil
}

// GetStats returns lifecycle statistics
func (ol *OrderLifecycle) GetStats() *LifecycleStats {
	ol.mu.RLock()
	defer ol.mu.RUnlock()

	stats := &LifecycleStats{
		TotalOrders:    len(ol.orderStates),
		LastUpdateTime: time.Now(),
	}

	// Count orders by status
	statusCounts := make(map[OrderStatus]int)
	for _, state := range ol.orderStates {
		statusCounts[state.CurrentStatus]++
	}
	stats.OrdersByStatus = statusCounts

	return stats
}

// LifecycleStats represents lifecycle statistics
type LifecycleStats struct {
	TotalOrders     int                    `json:"total_orders"`
	OrdersByStatus  map[OrderStatus]int    `json:"orders_by_status"`
	LastUpdateTime  time.Time              `json:"last_update_time"`
}

// Error definitions for lifecycle
var (
	ErrOrderStateNotFound      = errors.New("order state not found")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)
