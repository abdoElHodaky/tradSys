// 🎯 **Order Service Core**
// Generated using TradSys Code Splitting Standards
//
// This file contains the main service struct, constructor, and core API methods
// for the Order Management Service component. It follows the established patterns for
// service initialization, lifecycle management, and primary business operations.
//
// Performance Requirements: Standard latency, comprehensive order management
// File size limit: 410 lines

package orders

import (
	"context"
	"time"

	order_matching "github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/google/uuid"
	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Operation types for batch processing
const (
	OpTypeCreate = "create"
	OpTypeUpdate = "update"
	OpTypeCancel = "cancel"
	OpTypeAdd    = "add"

	// Cache keys
	CacheKeyPrefix = "order:"
)

// OrderStats represents order statistics
type OrderStats struct {
	TotalOrders     int64     `json:"total_orders"`
	ActiveOrders    int64     `json:"active_orders"`
	FilledOrders    int64     `json:"filled_orders"`
	CancelledOrders int64     `json:"cancelled_orders"`
	RejectedOrders  int64     `json:"rejected_orders"`
	TotalTrades     int64     `json:"total_trades"`
	TotalVolume     float64   `json:"total_volume"`
	LastUpdateTime  time.Time `json:"last_update_time"`
}

// NewService creates a new order management service
func NewService(engine *order_matching.Engine, logger *zap.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		Engine:         engine,
		Orders:         make(map[string]*Order),
		UserOrders:     make(map[string]map[string]bool),
		SymbolOrders:   make(map[string]map[string]bool),
		ClientOrderIDs: make(map[string]string),
		OrderCache:     cache.New(DefaultCacheExpiration, DefaultCacheCleanupInterval),
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		orderBatchChan: make(chan orderOperation, DefaultBatchChannelSize),
	}

	// Start order expiry checker
	go service.checkOrderExpiry()

	// Start batch processor
	go service.processBatchOperations()

	return service
}

// NewServiceWithConfig creates a new service with custom configuration
func NewServiceWithConfig(engine *order_matching.Engine, config *ServiceConfig, logger *zap.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		Engine:         engine,
		Orders:         make(map[string]*Order),
		UserOrders:     make(map[string]map[string]bool),
		SymbolOrders:   make(map[string]map[string]bool),
		ClientOrderIDs: make(map[string]string),
		OrderCache:     cache.New(config.CacheExpiration, config.CacheCleanupInterval),
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		orderBatchChan: make(chan orderOperation, config.BatchChannelSize),
	}

	// Start background processes if enabled
	if config.EnableBatchProcessing {
		go service.checkOrderExpiry()
		go service.processBatchOperations()
	}

	return service
}

// CreateOrder creates a new order
func (s *Service) CreateOrder(req *OrderRequest) (*Order, error) {
	// Validate request
	if err := s.validateOrderRequest(req); err != nil {
		return nil, err
	}

	// Check user order limits
	if err := s.checkUserOrderLimits(req.UserID); err != nil {
		return nil, err
	}

	// Create order
	order := &Order{
		ID:             uuid.New().String(),
		UserID:         req.UserID,
		ClientOrderID:  req.ClientOrderID,
		Symbol:         req.Symbol,
		Side:           req.Side,
		Type:           req.Type,
		Price:          req.Price,
		StopPrice:      req.StopPrice,
		Quantity:       req.Quantity,
		FilledQuantity: 0,
		Status:         OrderStatusNew,
		TimeInForce:    req.TimeInForce,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		ExpiresAt:      req.ExpiresAt,
		Trades:         make([]*Trade, 0),
		Metadata:       req.Metadata,
	}

	// Add to service
	if err := s.addOrder(order); err != nil {
		return nil, err
	}

	s.logger.Info("Order created",
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID),
		zap.String("symbol", order.Symbol),
		zap.String("side", string(order.Side)),
		zap.String("type", string(order.Type)),
		zap.Float64("price", order.Price),
		zap.Float64("quantity", order.Quantity))

	return order, nil
}

// GetOrder retrieves an order by ID
func (s *Service) GetOrder(orderID string) (*Order, error) {
	// Check cache first
	if cached, found := s.OrderCache.Get(CacheKeyPrefix + orderID); found {
		if order, ok := cached.(*Order); ok {
			return order, nil
		}
	}

	s.mu.RLock()
	order, exists := s.Orders[orderID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrOrderNotFound
	}

	// Cache the order
	s.OrderCache.Set(CacheKeyPrefix+orderID, order, cache.DefaultExpiration)

	return order, nil
}

// GetOrderByClientID retrieves an order by client order ID
func (s *Service) GetOrderByClientID(clientOrderID string) (*Order, error) {
	s.mu.RLock()
	orderID, exists := s.ClientOrderIDs[clientOrderID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrOrderNotFound
	}

	return s.GetOrder(orderID)
}

// GetUserOrders retrieves orders for a user
func (s *Service) GetUserOrders(userID string, filter *OrderFilter) ([]*Order, error) {
	s.mu.RLock()
	userOrderIDs, exists := s.UserOrders[userID]
	s.mu.RUnlock()

	if !exists {
		return []*Order{}, nil
	}

	orders := make([]*Order, 0, len(userOrderIDs))
	for orderID := range userOrderIDs {
		order, err := s.GetOrder(orderID)
		if err != nil {
			continue
		}

		// Apply filter
		if filter != nil && !s.matchesFilter(order, filter) {
			continue
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// GetSymbolOrders retrieves orders for a symbol
func (s *Service) GetSymbolOrders(symbol string, filter *OrderFilter) ([]*Order, error) {
	s.mu.RLock()
	symbolOrderIDs, exists := s.SymbolOrders[symbol]
	s.mu.RUnlock()

	if !exists {
		return []*Order{}, nil
	}

	orders := make([]*Order, 0, len(symbolOrderIDs))
	for orderID := range symbolOrderIDs {
		order, err := s.GetOrder(orderID)
		if err != nil {
			continue
		}

		// Apply filter
		if filter != nil && !s.matchesFilter(order, filter) {
			continue
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// CancelOrder cancels an existing order
func (s *Service) CancelOrder(req *OrderCancelRequest) (*Order, error) {
	var order *Order
	var err error

	// Get order by ID or client order ID
	if req.OrderID != "" {
		order, err = s.GetOrder(req.OrderID)
	} else if req.ClientOrderID != "" {
		order, err = s.GetOrderByClientID(req.ClientOrderID)
	} else {
		return nil, ErrInvalidOrderRequest
	}

	if err != nil {
		return nil, err
	}

	// Validate user ownership
	if order.UserID != req.UserID {
		return nil, ErrOrderNotFound
	}

	// Check if order can be cancelled
	if !s.canCancelOrder(order) {
		return nil, ErrOrderNotCancellable
	}

	// Cancel in matching engine
	if err := s.Engine.CancelOrder(order.ID, order.Symbol); err != nil {
		s.logger.Error("Failed to cancel order in engine",
			zap.String("order_id", order.ID),
			zap.Error(err))
		return nil, err
	}

	// Update order status
	s.mu.Lock()
	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Update cache
	s.OrderCache.Set(CacheKeyPrefix+order.ID, order, cache.DefaultExpiration)

	s.logger.Info("Order cancelled",
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID),
		zap.String("symbol", order.Symbol))

	return order, nil
}

// UpdateOrder updates an existing order
func (s *Service) UpdateOrder(req *OrderUpdateRequest) (*Order, error) {
	var order *Order
	var err error

	// Get order by ID or client order ID
	if req.OrderID != "" {
		order, err = s.GetOrder(req.OrderID)
	} else if req.ClientOrderID != "" {
		order, err = s.GetOrderByClientID(req.ClientOrderID)
	} else {
		return nil, ErrInvalidOrderRequest
	}

	if err != nil {
		return nil, err
	}

	// Validate user ownership
	if order.UserID != req.UserID {
		return nil, ErrOrderNotFound
	}

	// Check if order can be updated
	if !s.canUpdateOrder(order) {
		return nil, ErrOrderNotCancellable
	}

	// Update order fields
	s.mu.Lock()
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
	s.mu.Unlock()

	// Update cache
	s.OrderCache.Set(CacheKeyPrefix+order.ID, order, cache.DefaultExpiration)

	s.logger.Info("Order updated",
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID),
		zap.String("symbol", order.Symbol))

	return order, nil
}

// GetStats returns order statistics
func (s *Service) GetStats() *OrderStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &OrderStats{
		LastUpdateTime: time.Now(),
	}

	var totalVolume float64
	for _, order := range s.Orders {
		stats.TotalOrders++
		totalVolume += order.FilledQuantity * order.Price

		switch order.Status {
		case OrderStatusNew, OrderStatusPending, OrderStatusPartiallyFilled:
			stats.ActiveOrders++
		case OrderStatusFilled:
			stats.FilledOrders++
		case OrderStatusCancelled:
			stats.CancelledOrders++
		case OrderStatusRejected:
			stats.RejectedOrders++
		}

		stats.TotalTrades += int64(len(order.Trades))
	}

	stats.TotalVolume = totalVolume
	return stats
}

// Shutdown gracefully shuts down the service
func (s *Service) Shutdown() error {
	s.logger.Info("Shutting down order service")

	// Cancel context to stop background processes
	s.cancel()

	// Close batch channel
	close(s.orderBatchChan)

	s.logger.Info("Order service shutdown complete")
	return nil
}
