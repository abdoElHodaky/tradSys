package order_execution

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/resilience"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrInvalidOrder      = errors.New("invalid order")
	ErrOrderRejected     = errors.New("order rejected")
	ErrServiceUnavailable = errors.New("order execution service unavailable")
)

// OrderExecutionService handles order execution
type OrderExecutionService struct {
	// Engine is the order matching engine
	engine *order_matching.Engine
	
	// CircuitBreaker for resilience
	circuitBreaker *resilience.CircuitBreakerFactory
	
	// Logger
	logger *zap.Logger
	
	// Order callbacks
	orderCallbacks map[string]OrderCallback
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Service state
	running bool
}

// OrderCallback is a callback for order updates
type OrderCallback func(order *orders.OrderResponse)

// OrderRequest represents a request to execute an order
type OrderRequest struct {
	// UserID is the user ID
	UserID string
	
	// AccountID is the account ID
	AccountID string
	
	// Symbol is the trading symbol
	Symbol string
	
	// Side is the side of the order (buy or sell)
	Side orders.OrderSide
	
	// Type is the type of the order
	Type orders.OrderType
	
	// Quantity is the quantity of the order
	Quantity float64
	
	// Price is the price of the order (for limit orders)
	Price float64
	
	// StopPrice is the stop price (for stop orders)
	StopPrice float64
	
	// TimeInForce is the time in force for the order
	TimeInForce orders.TimeInForce
	
	// ClientOrderID is the client order ID
	ClientOrderID string
	
	// Callback is a callback for order updates
	Callback OrderCallback
}

// NewOrderExecutionService creates a new OrderExecutionService
func NewOrderExecutionService(
	engine *order_matching.Engine,
	circuitBreaker *resilience.CircuitBreakerFactory,
	logger *zap.Logger,
) *OrderExecutionService {
	return &OrderExecutionService{
		engine:         engine,
		circuitBreaker: circuitBreaker,
		logger:         logger,
		orderCallbacks: make(map[string]OrderCallback),
		running:        false,
	}
}

// Start starts the order execution service
func (s *OrderExecutionService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.running {
		return nil
	}
	
	s.logger.Info("Starting order execution service")
	s.running = true
	
	// Start listening for trades
	go s.processTrades(ctx)
	
	return nil
}

// Stop stops the order execution service
func (s *OrderExecutionService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return nil
	}
	
	s.logger.Info("Stopping order execution service")
	s.running = false
	
	return nil
}

// IsRunning returns whether the service is running
func (s *OrderExecutionService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.running
}

// ExecuteOrder executes an order
func (s *OrderExecutionService) ExecuteOrder(ctx context.Context, req *OrderRequest) (*orders.OrderResponse, error) {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()
	
	if !running {
		return nil, ErrServiceUnavailable
	}
	
	// Validate the order
	if err := s.validateOrder(req); err != nil {
		return nil, err
	}
	
	// Use circuit breaker for resilience
	result := s.circuitBreaker.ExecuteWithContext(ctx, "order_execution", func(ctx context.Context) (interface{}, error) {
		return s.executeOrderInternal(ctx, req)
	})
	
	if result.Error != nil {
		return nil, result.Error
	}
	
	return result.Value.(*orders.OrderResponse), nil
}

// validateOrder validates an order
func (s *OrderExecutionService) validateOrder(req *OrderRequest) error {
	// Check for required fields
	if req.UserID == "" {
		return errors.New("user ID is required")
	}
	
	if req.AccountID == "" {
		return errors.New("account ID is required")
	}
	
	if req.Symbol == "" {
		return errors.New("symbol is required")
	}
	
	if req.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	
	// Validate based on order type
	switch req.Type {
	case orders.OrderType_LIMIT:
		if req.Price <= 0 {
			return errors.New("price must be positive for limit orders")
		}
	case orders.OrderType_STOP_LIMIT:
		if req.Price <= 0 {
			return errors.New("price must be positive for stop-limit orders")
		}
		if req.StopPrice <= 0 {
			return errors.New("stop price must be positive for stop-limit orders")
		}
	case orders.OrderType_STOP_MARKET:
		if req.StopPrice <= 0 {
			return errors.New("stop price must be positive for stop-market orders")
		}
	}
	
	return nil
}

// executeOrderInternal executes an order internally
func (s *OrderExecutionService) executeOrderInternal(ctx context.Context, req *OrderRequest) (*orders.OrderResponse, error) {
	// Generate a unique order ID if not provided
	orderID := uuid.New().String()
	
	// Create an order
	order := &order_matching.Order{
		ID:            orderID,
		Symbol:        req.Symbol,
		Side:          order_matching.OrderSide(req.Side),
		Type:          order_matching.OrderType(req.Type),
		Price:         req.Price,
		Quantity:      req.Quantity,
		FilledQuantity: 0,
		Status:        order_matching.OrderStatusNew,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ClientOrderID: req.ClientOrderID,
		UserID:        req.UserID,
		StopPrice:     req.StopPrice,
		TimeInForce:   string(req.TimeInForce),
	}
	
	// Register callback if provided
	if req.Callback != nil {
		s.mu.Lock()
		s.orderCallbacks[orderID] = req.Callback
		s.mu.Unlock()
	}
	
	// Place the order
	trades, err := s.engine.PlaceOrder(order)
	if err != nil {
		return nil, err
	}
	
	// Log the trades
	for _, trade := range trades {
		s.logger.Info("Trade executed",
			zap.String("trade_id", trade.ID),
			zap.String("symbol", trade.Symbol),
			zap.Float64("price", trade.Price),
			zap.Float64("quantity", trade.Quantity),
			zap.String("buy_order_id", trade.BuyOrderID),
			zap.String("sell_order_id", trade.SellOrderID))
	}
	
	// Get the updated order
	updatedOrder, err := s.engine.GetOrder(req.Symbol, orderID)
	if err != nil {
		return nil, err
	}
	
	// Convert to OrderResponse
	response := &orders.OrderResponse{
		OrderId:       updatedOrder.ID,
		UserId:        updatedOrder.UserID,
		AccountId:     req.AccountID,
		Symbol:        updatedOrder.Symbol,
		Side:          orders.OrderSide(updatedOrder.Side),
		Type:          orders.OrderType(updatedOrder.Type),
		Quantity:      updatedOrder.Quantity,
		Price:         updatedOrder.Price,
		StopPrice:     updatedOrder.StopPrice,
		TimeInForce:   orders.TimeInForce(updatedOrder.TimeInForce),
		Status:        orders.OrderStatus(updatedOrder.Status),
		FilledQty:     updatedOrder.FilledQuantity,
		AvgPrice:      0, // Calculate average price if needed
		ClientOrderId: updatedOrder.ClientOrderID,
		CreatedAt:     updatedOrder.CreatedAt.UnixNano() / int64(time.Millisecond),
		UpdatedAt:     updatedOrder.UpdatedAt.UnixNano() / int64(time.Millisecond),
	}
	
	return response, nil
}

// CancelOrder cancels an order
func (s *OrderExecutionService) CancelOrder(ctx context.Context, symbol, orderID string) error {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()
	
	if !running {
		return ErrServiceUnavailable
	}
	
	// Use circuit breaker for resilience
	result := s.circuitBreaker.ExecuteWithContext(ctx, "order_cancellation", func(ctx context.Context) (interface{}, error) {
		return nil, s.engine.CancelOrder(symbol, orderID)
	})
	
	return result.Error
}

// GetOrder gets an order
func (s *OrderExecutionService) GetOrder(ctx context.Context, symbol, orderID string) (*orders.OrderResponse, error) {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()
	
	if !running {
		return nil, ErrServiceUnavailable
	}
	
	// Use circuit breaker for resilience
	result := s.circuitBreaker.ExecuteWithContext(ctx, "order_retrieval", func(ctx context.Context) (interface{}, error) {
		// Get the order
		order, err := s.engine.GetOrder(symbol, orderID)
		if err != nil {
			return nil, err
		}
		
		// Convert to OrderResponse
		response := &orders.OrderResponse{
			OrderId:       order.ID,
			UserId:        order.UserID,
			AccountId:     "", // Account ID not stored in the order
			Symbol:        order.Symbol,
			Side:          orders.OrderSide(order.Side),
			Type:          orders.OrderType(order.Type),
			Quantity:      order.Quantity,
			Price:         order.Price,
			StopPrice:     order.StopPrice,
			TimeInForce:   orders.TimeInForce(order.TimeInForce),
			Status:        orders.OrderStatus(order.Status),
			FilledQty:     order.FilledQuantity,
			AvgPrice:      0, // Calculate average price if needed
			ClientOrderId: order.ClientOrderID,
			CreatedAt:     order.CreatedAt.UnixNano() / int64(time.Millisecond),
			UpdatedAt:     order.UpdatedAt.UnixNano() / int64(time.Millisecond),
		}
		
		return response, nil
	})
	
	if result.Error != nil {
		return nil, result.Error
	}
	
	return result.Value.(*orders.OrderResponse), nil
}

// processTrades processes trades from the order matching engine
func (s *OrderExecutionService) processTrades(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case trade := <-s.engine.TradeChannel:
			// Process the trade
			s.processTrade(trade)
		}
	}
}

// processTrade processes a trade
func (s *OrderExecutionService) processTrade(trade *order_matching.Trade) {
	// Get the buy and sell orders
	buyOrder, err := s.engine.GetOrder(trade.Symbol, trade.BuyOrderID)
	if err != nil {
		s.logger.Error("Failed to get buy order",
			zap.String("order_id", trade.BuyOrderID),
			zap.Error(err))
		return
	}
	
	sellOrder, err := s.engine.GetOrder(trade.Symbol, trade.SellOrderID)
	if err != nil {
		s.logger.Error("Failed to get sell order",
			zap.String("order_id", trade.SellOrderID),
			zap.Error(err))
		return
	}
	
	// Notify callbacks for buy order
	s.notifyOrderCallback(buyOrder)
	
	// Notify callbacks for sell order
	s.notifyOrderCallback(sellOrder)
}

// notifyOrderCallback notifies the order callback
func (s *OrderExecutionService) notifyOrderCallback(order *order_matching.Order) {
	s.mu.RLock()
	callback, exists := s.orderCallbacks[order.ID]
	s.mu.RUnlock()
	
	if !exists {
		return
	}
	
	// Convert to OrderResponse
	response := &orders.OrderResponse{
		OrderId:       order.ID,
		UserId:        order.UserID,
		AccountId:     "", // Account ID not stored in the order
		Symbol:        order.Symbol,
		Side:          orders.OrderSide(order.Side),
		Type:          orders.OrderType(order.Type),
		Quantity:      order.Quantity,
		Price:         order.Price,
		StopPrice:     order.StopPrice,
		TimeInForce:   orders.TimeInForce(order.TimeInForce),
		Status:        orders.OrderStatus(order.Status),
		FilledQty:     order.FilledQuantity,
		AvgPrice:      0, // Calculate average price if needed
		ClientOrderId: order.ClientOrderID,
		CreatedAt:     order.CreatedAt.UnixNano() / int64(time.Millisecond),
		UpdatedAt:     order.UpdatedAt.UnixNano() / int64(time.Millisecond),
	}
	
	// Call the callback
	callback(response)
	
	// Remove the callback if the order is filled or cancelled
	if order.Status == order_matching.OrderStatusFilled || 
	   order.Status == order_matching.OrderStatusCancelled {
		s.mu.Lock()
		delete(s.orderCallbacks, order.ID)
		s.mu.Unlock()
	}
}

