package orders

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceParams contains the parameters for creating an order service
type ServiceParams struct {
	fx.In

	Logger     *zap.Logger
	Repository *repositories.OrderRepository `optional:"true"`
}

// Service provides order management operations with in-memory storage
type Service struct {
	logger     *zap.Logger
	repository *repositories.OrderRepository
	
	// In-memory order storage for high-performance access
	orders     map[string]*orders.OrderResponse
	ordersMux  sync.RWMutex
	
	// Order sequence for unique IDs
	sequence   uint64
	sequenceMux sync.Mutex
}

// NewService creates a new order service with fx dependency injection
func NewService(p ServiceParams) *Service {
	return &Service{
		logger:     p.Logger,
		repository: p.Repository,
		orders:     make(map[string]*orders.OrderResponse),
		sequence:   0,
	}
}

// generateOrderID generates a unique order ID with sequence
func (s *Service) generateOrderID() string {
	s.sequenceMux.Lock()
	defer s.sequenceMux.Unlock()
	s.sequence++
	return fmt.Sprintf("ORD-%d-%s", s.sequence, uuid.New().String()[:8])
}

// CreateOrder creates a new order with validation and storage
func (s *Service) CreateOrder(ctx context.Context, symbol string, orderType orders.OrderType, side orders.OrderSide, quantity, price, stopPrice float64, clientOrderID string) (*orders.OrderResponse, error) {
	s.logger.Info("Creating order",
		zap.String("symbol", symbol),
		zap.String("type", orderType.String()),
		zap.String("side", side.String()),
		zap.Float64("quantity", quantity),
		zap.Float64("price", price))

	// Validate order parameters
	if err := s.validateOrder(symbol, orderType, side, quantity, price, stopPrice); err != nil {
		s.logger.Error("Order validation failed", zap.Error(err))
		return nil, fmt.Errorf("order validation failed: %w", err)
	}

	// Generate unique order ID
	orderID := s.generateOrderID()
	now := time.Now().Unix() * 1000

	// Create order with proper status based on type
	status := orders.OrderStatus_NEW
	if orderType == orders.OrderType_MARKET {
		status = orders.OrderStatus_PENDING_NEW
	}

	order := &orders.OrderResponse{
		Id:            orderID,
		Symbol:        symbol,
		Type:          orderType,
		Side:          side,
		Status:        status,
		Quantity:      quantity,
		FilledQty:     0,
		Price:         price,
		StopPrice:     stopPrice,
		CreatedAt:     now,
		UpdatedAt:     now,
		ClientOrderId: clientOrderID,
	}

	// Store order in memory for fast access
	s.ordersMux.Lock()
	s.orders[orderID] = order
	s.ordersMux.Unlock()

	s.logger.Info("Order created successfully", 
		zap.String("order_id", orderID),
		zap.String("status", status.String()))

	return order, nil
}

// validateOrder validates order parameters
func (s *Service) validateOrder(symbol string, orderType orders.OrderType, side orders.OrderSide, quantity, price, stopPrice float64) error {
	if symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive, got %f", quantity)
	}
	
	if orderType == orders.OrderType_LIMIT && price <= 0 {
		return fmt.Errorf("limit orders must have positive price, got %f", price)
	}
	
	if (orderType == orders.OrderType_STOP || orderType == orders.OrderType_STOP_LIMIT) && stopPrice <= 0 {
		return fmt.Errorf("stop orders must have positive stop price, got %f", stopPrice)
	}
	
	if side != orders.OrderSide_BUY && side != orders.OrderSide_SELL {
		return fmt.Errorf("invalid order side: %v", side)
	}
	
	return nil
}

// GetOrder retrieves an order by ID from memory storage
func (s *Service) GetOrder(ctx context.Context, orderID string) (*orders.OrderResponse, error) {
	s.logger.Info("Getting order", zap.String("order_id", orderID))

	if orderID == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}

	// Retrieve from in-memory storage
	s.ordersMux.RLock()
	order, exists := s.orders[orderID]
	s.ordersMux.RUnlock()

	if !exists {
		s.logger.Warn("Order not found", zap.String("order_id", orderID))
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	s.logger.Info("Order retrieved successfully", 
		zap.String("order_id", orderID),
		zap.String("symbol", order.Symbol),
		zap.String("status", order.Status.String()))

	return order, nil
}

// CancelOrder cancels an existing order
func (s *Service) CancelOrder(ctx context.Context, orderID string) (*orders.OrderResponse, error) {
	s.logger.Info("Canceling order", zap.String("order_id", orderID))

	// Implementation would go here
	// For now, just return a placeholder response
	order := &orders.OrderResponse{
		Id:        orderID,
		Symbol:    "BTC-USD",
		Type:      orders.OrderType_LIMIT,
		Side:      orders.OrderSide_BUY,
		Status:    orders.OrderStatus_CANCELLED,
		Quantity:  1.0,
		FilledQty: 0.5,
		Price:     50000.0,
		CreatedAt: time.Now().Add(-1 * time.Hour).Unix() * 1000,
		UpdatedAt: time.Now().Unix() * 1000,
	}

	return order, nil
}

// GetOrders retrieves a list of orders
func (s *Service) GetOrders(ctx context.Context, symbol string, status orders.OrderStatus, startTime, endTime int64, limit int32) ([]*orders.OrderResponse, error) {
	s.logger.Info("Getting orders",
		zap.String("symbol", symbol),
		zap.String("status", status.String()),
		zap.Int64("start_time", startTime),
		zap.Int64("end_time", endTime),
		zap.Int32("limit", limit))

	// Implementation would go here
	// For now, just return placeholder responses
	orderList := []*orders.OrderResponse{
		{
			Id:        uuid.New().String(),
			Symbol:    symbol,
			Type:      orders.OrderType_LIMIT,
			Side:      orders.OrderSide_BUY,
			Status:    status,
			Quantity:  1.0,
			FilledQty: 0.5,
			Price:     50000.0,
			CreatedAt: time.Now().Add(-1 * time.Hour).Unix() * 1000,
			UpdatedAt: time.Now().Unix() * 1000,
		},
		{
			Id:        uuid.New().String(),
			Symbol:    symbol,
			Type:      orders.OrderType_MARKET,
			Side:      orders.OrderSide_SELL,
			Status:    status,
			Quantity:  0.5,
			FilledQty: 0.5,
			Price:     51000.0,
			CreatedAt: time.Now().Add(-30 * time.Minute).Unix() * 1000,
			UpdatedAt: time.Now().Unix() * 1000,
		},
	}

	return orderList, nil
}

// ServiceModule is defined in module.go to avoid duplication
