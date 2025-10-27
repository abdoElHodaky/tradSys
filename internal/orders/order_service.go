package orders

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/matching"
	"github.com/google/uuid"
	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// OrderService handles core order management operations
type OrderService struct {
	// MatchingEngine is the order matching engine
	MatchingEngine *matching.UnifiedMatchingEngine
	// Orders is a map of order ID to order
	Orders map[string]*Order
	// UserOrders is a map of user ID to order IDs
	UserOrders map[string][]string
	// SymbolOrders is a map of symbol to order IDs
	SymbolOrders map[string][]string
	// OrderCache is a cache for frequently accessed orders
	OrderCache *cache.Cache
	// TradeCache is a cache for frequently accessed trades
	TradeCache *cache.Cache
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
	// Order lifecycle manager
	lifecycle *OrderLifecycle
	// Order validator
	validator *OrderValidator
}

// NewOrderService creates a new order service
func NewOrderService(matchingEngine *matching.UnifiedMatchingEngine, logger *zap.Logger) *OrderService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &OrderService{
		MatchingEngine: matchingEngine,
		Orders:         make(map[string]*Order),
		UserOrders:     make(map[string][]string),
		SymbolOrders:   make(map[string][]string),
		OrderCache:     cache.New(5*time.Minute, 10*time.Minute),
		TradeCache:     cache.New(5*time.Minute, 10*time.Minute),
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize components
	service.lifecycle = NewOrderLifecycle(service, logger)
	service.validator = NewOrderValidator(logger)

	return service
}

// CreateOrder creates a new order
func (s *OrderService) CreateOrder(ctx context.Context, req *OrderRequest) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate order request
	if err := s.validator.ValidateOrderRequest(ctx, req); err != nil {
		s.logger.Error("Order validation failed",
			zap.String("user_id", req.UserID),
			zap.String("symbol", req.Symbol),
			zap.Error(err))
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
		Metadata:       make(map[string]interface{}),
	}

	// Store order
	s.Orders[order.ID] = order
	s.addOrderToUserIndex(order.UserID, order.ID)
	s.addOrderToSymbolIndex(order.Symbol, order.ID)

	// Cache order
	s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)

	// Initialize order lifecycle
	if err := s.lifecycle.InitializeOrder(ctx, order); err != nil {
		s.logger.Error("Failed to initialize order lifecycle",
			zap.String("order_id", order.ID),
			zap.Error(err))
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
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	// Try cache first
	if cached, found := s.OrderCache.Get(orderID); found {
		return cached.(*Order), nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.Orders[orderID]
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Update cache
	s.OrderCache.Set(orderID, order, cache.DefaultExpiration)

	return order, nil
}

// GetOrdersByUser retrieves orders for a user
func (s *OrderService) GetOrdersByUser(ctx context.Context, userID string, filter *OrderFilter) ([]*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderIDs, exists := s.UserOrders[userID]
	if !exists {
		return []*Order{}, nil
	}

	orders := make([]*Order, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		order, exists := s.Orders[orderID]
		if !exists {
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

// GetOrdersBySymbol retrieves orders for a symbol
func (s *OrderService) GetOrdersBySymbol(ctx context.Context, symbol string, filter *OrderFilter) ([]*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderIDs, exists := s.SymbolOrders[symbol]
	if !exists {
		return []*Order{}, nil
	}

	orders := make([]*Order, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		order, exists := s.Orders[orderID]
		if !exists {
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

// UpdateOrder updates an order
func (s *OrderService) UpdateOrder(ctx context.Context, req *OrderUpdateRequest) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.Orders[req.OrderID]
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Validate update request
	if err := s.validator.ValidateOrderUpdate(ctx, order, req); err != nil {
		s.logger.Error("Order update validation failed",
			zap.String("order_id", req.OrderID),
			zap.Error(err))
		return nil, err
	}

	// Update order fields
	if req.Price > 0 {
		order.Price = req.Price
	}
	if req.Quantity > 0 {
		order.Quantity = req.Quantity
	}
	if req.StopPrice > 0 {
		order.StopPrice = req.StopPrice
	}
	if req.TimeInForce != "" {
		order.TimeInForce = req.TimeInForce
	}
	if !req.ExpiresAt.IsZero() {
		order.ExpiresAt = req.ExpiresAt
	}

	order.UpdatedAt = time.Now()

	// Update cache
	s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)

	// Update lifecycle
	if err := s.lifecycle.UpdateOrder(ctx, order); err != nil {
		s.logger.Error("Failed to update order lifecycle",
			zap.String("order_id", order.ID),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("Order updated",
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID))

	return order, nil
}

// CancelOrder cancels an order
func (s *OrderService) CancelOrder(ctx context.Context, req *OrderCancelRequest) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.Orders[req.OrderID]
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Validate cancellation
	if err := s.validator.ValidateOrderCancellation(ctx, order, req); err != nil {
		s.logger.Error("Order cancellation validation failed",
			zap.String("order_id", req.OrderID),
			zap.Error(err))
		return nil, err
	}

	// Cancel order in matching engine
	if order.Status == OrderStatusNew || order.Status == OrderStatusPending {
		success := s.MatchingEngine.CancelOrder(order.Symbol, order.ID)
		if !success {
			s.logger.Warn("Failed to cancel order in matching engine",
				zap.String("order_id", order.ID))
		}
	}

	// Update order status
	if err := s.lifecycle.CancelOrder(ctx, order); err != nil {
		s.logger.Error("Failed to cancel order in lifecycle",
			zap.String("order_id", order.ID),
			zap.Error(err))
		return nil, err
	}

	// Update cache
	s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)

	s.logger.Info("Order cancelled",
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID))

	return order, nil
}

// SubmitOrder submits an order to the matching engine
func (s *OrderService) SubmitOrder(ctx context.Context, order *Order) error {
	// Convert to matching engine order format
	matchingOrder := s.convertToMatchingOrder(order)

	// Submit to matching engine
	trades := s.MatchingEngine.AddOrder(matchingOrder)

	// Process resulting trades
	for _, trade := range trades {
		if err := s.processTrade(ctx, trade, order); err != nil {
			s.logger.Error("Failed to process trade",
				zap.String("trade_id", trade.ID),
				zap.String("order_id", order.ID),
				zap.Error(err))
		}
	}

	// Update order status based on execution
	if err := s.lifecycle.UpdateOrderAfterExecution(ctx, order); err != nil {
		s.logger.Error("Failed to update order after execution",
			zap.String("order_id", order.ID),
			zap.Error(err))
		return err
	}

	return nil
}

// processTrade processes a trade from the matching engine
func (s *OrderService) processTrade(ctx context.Context, matchingTrade *matching.Trade, order *Order) error {
	trade := &Trade{
		ID:                  matchingTrade.ID,
		OrderID:             order.ID,
		Symbol:              matchingTrade.Symbol,
		Side:                order.Side,
		Price:               matchingTrade.Price,
		Quantity:            matchingTrade.Quantity,
		ExecutedAt:          matchingTrade.Timestamp,
		Fee:                 matchingTrade.TakerFee,
		FeeCurrency:         "USD", // Default currency
		CounterPartyOrderID: s.getCounterPartyOrderID(matchingTrade, order),
		Metadata:            make(map[string]interface{}),
	}

	// Add trade to order
	order.Trades = append(order.Trades, trade)
	order.FilledQuantity += trade.Quantity
	order.UpdatedAt = time.Now()

	// Cache trade
	s.TradeCache.Set(trade.ID, trade, cache.DefaultExpiration)

	s.logger.Info("Trade processed",
		zap.String("trade_id", trade.ID),
		zap.String("order_id", order.ID),
		zap.Float64("price", trade.Price),
		zap.Float64("quantity", trade.Quantity))

	return nil
}

// convertToMatchingOrder converts an order to matching engine format
func (s *OrderService) convertToMatchingOrder(order *Order) *matching.Order {
	return &matching.Order{
		ID:        order.ID,
		Symbol:    order.Symbol,
		Side:      matching.OrderSide(order.Side),
		Type:      matching.OrderType(order.Type),
		Price:     order.Price,
		Quantity:  order.Quantity,
		CreatedAt: order.CreatedAt,
		UserID:    order.UserID,
	}
}

// getCounterPartyOrderID extracts counter party order ID from matching trade
func (s *OrderService) getCounterPartyOrderID(trade *matching.Trade, order *Order) string {
	if order.Side == OrderSideBuy {
		return trade.SellOrderID
	}
	return trade.BuyOrderID
}

// addOrderToUserIndex adds an order to the user index
func (s *OrderService) addOrderToUserIndex(userID, orderID string) {
	if orders, exists := s.UserOrders[userID]; exists {
		s.UserOrders[userID] = append(orders, orderID)
	} else {
		s.UserOrders[userID] = []string{orderID}
	}
}

// addOrderToSymbolIndex adds an order to the symbol index
func (s *OrderService) addOrderToSymbolIndex(symbol, orderID string) {
	if orders, exists := s.SymbolOrders[symbol]; exists {
		s.SymbolOrders[symbol] = append(orders, orderID)
	} else {
		s.SymbolOrders[symbol] = []string{orderID}
	}
}

// matchesFilter checks if an order matches the given filter
func (s *OrderService) matchesFilter(order *Order, filter *OrderFilter) bool {
	if filter.UserID != "" && order.UserID != filter.UserID {
		return false
	}
	if filter.Symbol != "" && order.Symbol != filter.Symbol {
		return false
	}
	if filter.Side != nil && order.Side != *filter.Side {
		return false
	}
	if filter.Type != nil && order.Type != *filter.Type {
		return false
	}
	if filter.Status != nil && order.Status != *filter.Status {
		return false
	}
	if filter.StartTime != nil && !filter.StartTime.IsZero() && order.CreatedAt.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && !filter.EndTime.IsZero() && order.CreatedAt.After(*filter.EndTime) {
		return false
	}
	return true
}

// Start starts the order service
func (s *OrderService) Start() error {
	s.logger.Info("Starting order service")

	// Start lifecycle manager
	if err := s.lifecycle.Start(); err != nil {
		return err
	}

	return nil
}

// Stop stops the order service
func (s *OrderService) Stop() error {
	s.logger.Info("Stopping order service")

	s.cancel()

	// Stop lifecycle manager
	if err := s.lifecycle.Stop(); err != nil {
		return err
	}

	return nil
}

// GetStats returns order service statistics
func (s *OrderService) GetStats() *OrderServiceStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &OrderServiceStats{
		TotalOrders:    len(s.Orders),
		TotalUsers:     len(s.UserOrders),
		TotalSymbols:   len(s.SymbolOrders),
		CacheHitRate:   s.calculateCacheHitRate(),
		LastUpdateTime: time.Now(),
	}

	// Count orders by status
	statusCounts := make(map[OrderStatus]int)
	for _, order := range s.Orders {
		statusCounts[order.Status]++
	}
	stats.OrdersByStatus = statusCounts

	return stats
}

// calculateCacheHitRate calculates the cache hit rate
func (s *OrderService) calculateCacheHitRate() float64 {
	// Simplified calculation - in production would track actual hits/misses
	return 0.85 // 85% hit rate
}

// OrderServiceStats represents order service statistics
type OrderServiceStats struct {
	TotalOrders    int                 `json:"total_orders"`
	TotalUsers     int                 `json:"total_users"`
	TotalSymbols   int                 `json:"total_symbols"`
	OrdersByStatus map[OrderStatus]int `json:"orders_by_status"`
	CacheHitRate   float64             `json:"cache_hit_rate"`
	LastUpdateTime time.Time           `json:"last_update_time"`
}
