package orders

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	pb "github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the OrderService gRPC interface
type Service struct {
	pb.UnimplementedOrderServiceServer
	logger     *zap.Logger
	orders     map[string]*pb.Order
	mu         sync.RWMutex
	repository *repositories.OrderRepository
	subscribers map[string]map[pb.OrderService_SubscribeOrderUpdatesServer]bool
	subMu       sync.RWMutex
}

// NewService creates a new order service
func NewService(logger *zap.Logger, repository *repositories.OrderRepository) *Service {
	return &Service{
		logger:      logger,
		orders:      make(map[string]*pb.Order),
		repository:  repository,
		subscribers: make(map[string]map[pb.OrderService_SubscribeOrderUpdatesServer]bool),
	}
}

// PlaceOrder handles new order placement
func (s *Service) PlaceOrder(ctx context.Context, req *pb.OrderRequest) (*pb.OrderResponse, error) {
	// Generate order ID
	orderID := uuid.New().String()
	
	// Create order
	order := &pb.Order{
		OrderId:    orderID,
		Symbol:     req.Symbol,
		Type:       req.Type,
		Side:       req.Side,
		Quantity:   req.Quantity,
		Price:      req.Price,
		ClientId:   req.ClientId,
		Timestamp:  time.Now().UnixNano(),
		Status:     pb.OrderStatus_ACCEPTED,
		Exchange:   req.Exchange,
	}
	
	// Store in memory for quick access
	s.mu.Lock()
	s.orders[orderID] = order
	s.mu.Unlock()
	
	// Persist to database
	dbOrder := &models.Order{
//<<<<<<< codegen-bot/pairs-management-implementation
		OrderID:    order.OrderId,
		Symbol:     order.Symbol,
		Type:       models.OrderType(mapOrderTypeToString(order.Type)),
		Side:       models.OrderSide(mapOrderSideToString(order.Side)),
		Quantity:   order.Quantity,
		Price:      order.Price,
		StopLoss:   order.StopLoss,
		TakeProfit: order.TakeProfit,
		ClientID:   order.ClientId,
		Status:     models.OrderStatusAccepted,
		Exchange:   order.Exchange,
		Strategy:   order.Strategy,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Timestamp:  time.Now(),
//=======
		OrderID:   order.OrderId,
		Symbol:    order.Symbol,
		Type:      models.OrderType(mapOrderTypeToString(order.Type)),
		Side:      models.OrderSide(mapOrderSideToString(order.Side)),
		Quantity:  order.Quantity,
		Price:     order.Price,
		ClientID:  order.ClientId,
		Status:    models.OrderStatusAccepted,
		Exchange:  order.Exchange,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
//>>>>>>> main
	}
	
	if err := s.repository.Create(ctx, dbOrder); err != nil {
		s.logger.Error("Failed to persist order", 
			zap.Error(err),
			zap.String("order_id", order.OrderId))
		// Continue processing even if persistence fails
		// In production, you might want to handle this differently
	}
	
	s.logger.Info("Order placed", 
		zap.String("order_id", order.OrderId),
		zap.String("symbol", order.Symbol),
		zap.Int32("side", int32(order.Side)),
		zap.Float64("quantity", order.Quantity),
		zap.Float64("price", order.Price))
	
	// Notify subscribers
	s.notifyOrderUpdate(order)
	
	return &pb.OrderResponse{
		OrderId: order.OrderId,
		Status:  pb.OrderStatus_ACCEPTED,
		Message: "Order successfully placed",
		Order:   order,
	}, nil
}

// CancelOrder handles order cancellation
func (s *Service) CancelOrder(ctx context.Context, req *pb.OrderCancelRequest) (*pb.OrderResponse, error) {
	// Find order
	s.mu.RLock()
	order, exists := s.orders[req.OrderId]
	s.mu.RUnlock()
	
	if !exists {
		// Try to find in database
		dbOrder, err := s.repository.FindByID(ctx, req.OrderId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "database error: %v", err)
		}
		
		if dbOrder == nil {
			return nil, status.Errorf(codes.NotFound, "order not found: %s", req.OrderId)
		}
		
		// Convert to protobuf
		order = &pb.Order{
			OrderId:        dbOrder.OrderID,
			Symbol:         dbOrder.Symbol,
			Type:           mapStringToOrderType(string(dbOrder.Type)),
			Side:           mapStringToOrderSide(string(dbOrder.Side)),
			Quantity:       dbOrder.Quantity,
			Price:          dbOrder.Price,
			ClientId:       dbOrder.ClientID,
			Timestamp:      dbOrder.CreatedAt.UnixNano(),
			Status:         mapStringToOrderStatus(string(dbOrder.Status)),
			FilledQuantity: dbOrder.FilledQty,
			Exchange:       dbOrder.Exchange,
		}
		
		// Cache for future requests
		s.mu.Lock()
		s.orders[req.OrderId] = order
		s.mu.Unlock()
	}
	
	// Check if order can be cancelled
	if order.Status == pb.OrderStatus_FILLED || order.Status == pb.OrderStatus_CANCELLED {
		return nil, status.Errorf(codes.FailedPrecondition, "order cannot be cancelled: %s", order.Status)
	}
	
	// Update order status
	order.Status = pb.OrderStatus_CANCELLED
	
	// Update in memory
	s.mu.Lock()
	s.orders[req.OrderId] = order
	s.mu.Unlock()
	
	// Update in database
	dbOrder, err := s.repository.FindByID(ctx, req.OrderId)
	if err != nil {
		s.logger.Error("Failed to find order for cancellation",
			zap.Error(err),
			zap.String("order_id", req.OrderId))
		// Continue processing even if database operation fails
	} else if dbOrder != nil {
		dbOrder.Status = models.OrderStatusCancelled
		dbOrder.UpdatedAt = time.Now()
		
		if err := s.repository.Update(ctx, dbOrder); err != nil {
			s.logger.Error("Failed to update order in database",
				zap.Error(err),
				zap.String("order_id", req.OrderId))
		}
	}
	
	s.logger.Info("Order cancelled",
		zap.String("order_id", req.OrderId),
		zap.String("client_id", req.ClientId))
	
	// Notify subscribers
	s.notifyOrderUpdate(order)
	
	return &pb.OrderResponse{
		OrderId: order.OrderId,
		Status:  pb.OrderStatus_CANCELLED,
		Message: "Order successfully cancelled",
		Order:   order,
	}, nil
}

// GetOrderStatus retrieves current order status
func (s *Service) GetOrderStatus(ctx context.Context, req *pb.OrderStatusRequest) (*pb.OrderResponse, error) {
	// Find order
	s.mu.RLock()
	order, exists := s.orders[req.OrderId]
	s.mu.RUnlock()
	
	if !exists {
		// Try to find in database
		dbOrder, err := s.repository.FindByID(ctx, req.OrderId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "database error: %v", err)
		}
		
		if dbOrder == nil {
			return nil, status.Errorf(codes.NotFound, "order not found: %s", req.OrderId)
		}
		
		// Convert to protobuf
		order = &pb.Order{
			OrderId:        dbOrder.OrderID,
			Symbol:         dbOrder.Symbol,
			Type:           mapStringToOrderType(string(dbOrder.Type)),
			Side:           mapStringToOrderSide(string(dbOrder.Side)),
			Quantity:       dbOrder.Quantity,
			Price:          dbOrder.Price,
			ClientId:       dbOrder.ClientID,
			Timestamp:      dbOrder.CreatedAt.UnixNano(),
			Status:         mapStringToOrderStatus(string(dbOrder.Status)),
			FilledQuantity: dbOrder.FilledQty,
			Exchange:       dbOrder.Exchange,
		}
		
		// Cache for future requests
		s.mu.Lock()
		s.orders[req.OrderId] = order
		s.mu.Unlock()
	}
	
	return &pb.OrderResponse{
		OrderId: order.OrderId,
		Status:  order.Status,
		Message: "Order status retrieved",
		Order:   order,
	}, nil
}

// GetOrders retrieves orders based on filter criteria
func (s *Service) GetOrders(ctx context.Context, req *pb.OrderListRequest) (*pb.OrderList, error) {
	var orders []*models.Order
	var err error
	
	// Apply filters
	if req.StartTime > 0 && req.EndTime > 0 {
		startTime := time.Unix(0, req.StartTime)
		endTime := time.Unix(0, req.EndTime)
		orders, err = s.repository.FindOrdersByTimeRange(ctx, req.Symbol, startTime, endTime)
	} else if req.Status != pb.OrderStatus_NEW && req.Symbol != "" {
		// Find by symbol and status
		// This is a simplified example - in a real system, you would have more sophisticated filtering
		if req.Status == pb.OrderStatus_FILLED || req.Status == pb.OrderStatus_CANCELLED || req.Status == pb.OrderStatus_REJECTED {
			// Find completed orders
			// Implementation would depend on your repository
			// This is a placeholder
		} else {
			// Find active orders
			orders, err = s.repository.FindActiveOrdersBySymbol(ctx, req.Symbol)
		}
	} else if req.Symbol != "" {
		// Find by symbol
		orders, err = s.repository.FindActiveOrdersBySymbol(ctx, req.Symbol)
	} else {
		// No filters - return recent orders
		// This is a placeholder - in a real system, you would implement this
	}
	
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	
	// Convert to protobuf
	protoOrders := make([]*pb.Order, 0, len(orders))
	for _, o := range orders {
		protoOrders = append(protoOrders, &pb.Order{
			OrderId:        o.OrderID,
			Symbol:         o.Symbol,
			Type:           mapStringToOrderType(string(o.Type)),
			Side:           mapStringToOrderSide(string(o.Side)),
			Quantity:       o.Quantity,
			Price:          o.Price,
			ClientId:       o.ClientID,
			Timestamp:      o.CreatedAt.UnixNano(),
			Status:         mapStringToOrderStatus(string(o.Status)),
			FilledQuantity: o.FilledQty,
			Exchange:       o.Exchange,
		})
	}
	
	return &pb.OrderList{Orders: protoOrders}, nil
}

// SubscribeOrderUpdates streams order updates
func (s *Service) SubscribeOrderUpdates(req *pb.OrderStatusRequest, stream pb.OrderService_SubscribeOrderUpdatesServer) error {
	// Register subscriber
	s.subMu.Lock()
	if _, exists := s.subscribers[req.OrderId]; !exists {
		s.subscribers[req.OrderId] = make(map[pb.OrderService_SubscribeOrderUpdatesServer]bool)
	}
	s.subscribers[req.OrderId][stream] = true
	s.subMu.Unlock()
	
	s.logger.Info("Client subscribed to order updates",
		zap.String("order_id", req.OrderId))
	
	// Send initial order status if available
	s.mu.RLock()
	order, ok := s.orders[req.OrderId]
	s.mu.RUnlock()
	
	if ok {
		response := &pb.OrderResponse{
			OrderId: order.OrderId,
			Status:  order.Status,
			Message: "Initial order status",
			Order:   order,
		}
		
		if err := stream.Send(response); err != nil {
			s.logger.Error("Failed to send initial order status",
				zap.Error(err),
				zap.String("order_id", req.OrderId))
		}
	}
	
	// Keep the stream open until client disconnects
	<-stream.Context().Done()
	
	// Unregister subscriber
	s.subMu.Lock()
	if subs, exists := s.subscribers[req.OrderId]; exists {
		delete(subs, stream)
		if len(subs) == 0 {
			delete(s.subscribers, req.OrderId)
		}
	}
	s.subMu.Unlock()
	
	s.logger.Info("Client unsubscribed from order updates",
		zap.String("order_id", req.OrderId))
	
	return nil
}

// notifyOrderUpdate notifies subscribers of order updates
func (s *Service) notifyOrderUpdate(order *pb.Order) {
	response := &pb.OrderResponse{
		OrderId: order.OrderId,
		Status:  order.Status,
		Message: "Order updated",
		Order:   order,
	}
	
	s.subMu.RLock()
	if subs, exists := s.subscribers[order.OrderId]; exists {
		for stream := range subs {
			if err := stream.Send(response); err != nil {
				s.logger.Error("Failed to send order update",
					zap.Error(err),
					zap.String("order_id", order.OrderId))
				// We'll clean up dead streams in the next subscription request
			}
		}
	}
	s.subMu.RUnlock()
}

// Helper functions to map between protobuf enums and database enums
func mapOrderTypeToString(orderType pb.OrderType) string {
	switch orderType {
	case pb.OrderType_MARKET:
		return string(models.OrderTypeMarket)
	case pb.OrderType_LIMIT:
		return string(models.OrderTypeLimit)
	case pb.OrderType_STOP:
		return string(models.OrderTypeStop)
	case pb.OrderType_STOP_LIMIT:
		return string(models.OrderTypeStopLimit)
	default:
		return string(models.OrderTypeMarket)
	}
}

func mapOrderSideToString(orderSide pb.OrderSide) string {
	switch orderSide {
	case pb.OrderSide_BUY:
		return string(models.OrderSideBuy)
	case pb.OrderSide_SELL:
		return string(models.OrderSideSell)
	default:
		return string(models.OrderSideBuy)
	}
}

func mapStringToOrderType(orderType string) pb.OrderType {
	switch orderType {
	case string(models.OrderTypeMarket):
		return pb.OrderType_MARKET
	case string(models.OrderTypeLimit):
		return pb.OrderType_LIMIT
	case string(models.OrderTypeStop):
		return pb.OrderType_STOP
	case string(models.OrderTypeStopLimit):
		return pb.OrderType_STOP_LIMIT
	default:
		return pb.OrderType_MARKET
	}
}

func mapStringToOrderSide(orderSide string) pb.OrderSide {
	switch orderSide {
	case string(models.OrderSideBuy):
		return pb.OrderSide_BUY
	case string(models.OrderSideSell):
		return pb.OrderSide_SELL
	default:
		return pb.OrderSide_BUY
	}
}

func mapStringToOrderStatus(orderStatus string) pb.OrderStatus {
	switch orderStatus {
	case string(models.OrderStatusNew):
		return pb.OrderStatus_NEW
	case string(models.OrderStatusAccepted):
		return pb.OrderStatus_ACCEPTED
	case string(models.OrderStatusRejected):
		return pb.OrderStatus_REJECTED
	case string(models.OrderStatusFilled):
		return pb.OrderStatus_FILLED
	case string(models.OrderStatusPartial):
		return pb.OrderStatus_PARTIAL
	case string(models.OrderStatusCancelled):
		return pb.OrderStatus_CANCELLED
	default:
		return pb.OrderStatus_NEW
	}
}
//<<<<<<< codegen-bot/pairs-management-implementation
//=======

//>>>>>>> main
