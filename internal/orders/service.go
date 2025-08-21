package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db"
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
	Repository *repositories.OrderRepository
}

// Service provides order management operations
type Service struct {
	logger     *zap.Logger
	repository *repositories.OrderRepository
}

// NewService creates a new order service with fx dependency injection
func NewService(p ServiceParams) *Service {
	return &Service{
		logger:     p.Logger,
		repository: p.Repository,
	}
}

// mapOrderTypeToString maps an order type enum to a string
func mapOrderTypeToString(orderType orders.OrderType) string {
	switch orderType {
	case orders.OrderType_MARKET:
		return "market"
	case orders.OrderType_LIMIT:
		return "limit"
	case orders.OrderType_STOP:
		return "stop"
	case orders.OrderType_STOP_LIMIT:
		return "stop_limit"
	default:
		return "unknown"
	}
}

// mapOrderSideToString maps an order side enum to a string
func mapOrderSideToString(side orders.OrderSide) string {
	switch side {
	case orders.OrderSide_BUY:
		return "buy"
	case orders.OrderSide_SELL:
		return "sell"
	default:
		return "unknown"
	}
}

// mapOrderStatusToString maps an order status enum to a string
func mapOrderStatusToString(status orders.OrderStatus) string {
	switch status {
	case orders.OrderStatus_NEW:
		return "new"
	case orders.OrderStatus_PARTIAL:
		return "partially_filled"
	case orders.OrderStatus_FILLED:
		return "filled"
	case orders.OrderStatus_CANCELLED:
		return "canceled"
	case orders.OrderStatus_REJECTED:
		return "rejected"
	case orders.OrderStatus_EXPIRED:
		return "expired"
	case orders.OrderStatus_PENDING:
		return "pending"
	case orders.OrderStatus_PROCESSING:
		return "processing"
	default:
		return "unknown"
	}
}

// mapStringToOrderType maps a string to an order type enum
func mapStringToOrderType(orderType string) orders.OrderType {
	switch orderType {
	case "market":
		return orders.OrderType_MARKET
	case "limit":
		return orders.OrderType_LIMIT
	case "stop":
		return orders.OrderType_STOP
	case "stop_limit":
		return orders.OrderType_STOP_LIMIT
	default:
		return orders.OrderType_MARKET
	}
}

// mapStringToOrderSide maps a string to an order side enum
func mapStringToOrderSide(side string) orders.OrderSide {
	switch side {
	case "buy":
		return orders.OrderSide_BUY
	case "sell":
		return orders.OrderSide_SELL
	default:
		return orders.OrderSide_BUY
	}
}

// mapStringToOrderStatus maps a string to an order status enum
func mapStringToOrderStatus(status string) orders.OrderStatus {
	switch status {
	case "new":
		return orders.OrderStatus_NEW
	case "partially_filled":
		return orders.OrderStatus_PARTIAL
	case "filled":
		return orders.OrderStatus_FILLED
	case "canceled":
		return orders.OrderStatus_CANCELLED
	case "rejected":
		return orders.OrderStatus_REJECTED
	case "expired":
		return orders.OrderStatus_EXPIRED
	case "pending":
		return orders.OrderStatus_PENDING
	case "processing":
		return orders.OrderStatus_PROCESSING
	default:
		return orders.OrderStatus_NEW
	}
}

// dbOrderToProto converts a database order to a proto order
func dbOrderToProto(dbOrder *db.Order) *orders.OrderResponse {
	createdAt := dbOrder.CreatedAt.UnixMilli()
	updatedAt := dbOrder.UpdatedAt.UnixMilli()
	
	var expiresAt int64
	if !dbOrder.ExpiresAt.IsZero() {
		expiresAt = dbOrder.ExpiresAt.UnixMilli()
	}
	
	return &orders.OrderResponse{
		OrderId:        dbOrder.ID,
		Symbol:         dbOrder.Symbol,
		Type:           mapStringToOrderType(dbOrder.Type),
		Side:           mapStringToOrderSide(dbOrder.Side),
		Status:         mapStringToOrderStatus(dbOrder.Status),
		Quantity:       dbOrder.Quantity,
		FilledQuantity: dbOrder.FilledQuantity,
		Price:          dbOrder.Price,
		StopPrice:      dbOrder.StopPrice,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		ClientOrderId:  dbOrder.ClientOrderID,
		ExpiresAt:      expiresAt,
	}
}

// CreateOrder creates a new order
func (s *Service) CreateOrder(ctx context.Context, symbol string, orderType orders.OrderType, side orders.OrderSide, quantity, price, stopPrice float64, clientOrderID string) (*orders.OrderResponse, error) {
	s.logger.Info("Creating order",
		zap.String("symbol", symbol),
		zap.String("type", orderType.String()),
		zap.String("side", side.String()),
		zap.Float64("quantity", quantity),
		zap.Float64("price", price))

	// Validate inputs
	if symbol == "" {
		return nil, errors.New("symbol is required")
	}
	
	if quantity <= 0 {
		return nil, errors.New("quantity must be greater than 0")
	}
	
	if orderType == orders.OrderType_LIMIT && price <= 0 {
		return nil, errors.New("price must be greater than 0 for limit orders")
	}
	
	if (orderType == orders.OrderType_STOP || orderType == orders.OrderType_STOP_LIMIT) && stopPrice <= 0 {
		return nil, errors.New("stop price must be greater than 0 for stop orders")
	}
	
	// Extract user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, errors.New("user ID not found in context")
	}
	
	// Check if client order ID already exists for this user
	if clientOrderID != "" {
		existingOrder, err := s.repository.GetByClientOrderID(ctx, userID, clientOrderID)
		if err != nil {
			s.logger.Error("Failed to check client order ID", 
				zap.Error(err), 
				zap.String("client_order_id", clientOrderID))
			return nil, fmt.Errorf("failed to check client order ID: %w", err)
		}
		
		if existingOrder != nil {
			return nil, errors.New("client order ID already exists")
		}
	}
	
	// Generate order ID
	orderID := uuid.New().String()
	
	// Create metadata
	metadata := map[string]interface{}{
		"created_by": "api",
		"ip":         ctx.Value("client_ip"),
	}
	
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		s.logger.Error("Failed to marshal metadata", zap.Error(err))
		metadataJSON = []byte("{}")
	}
	
	// Create order in database
	now := time.Now()
	dbOrder := &db.Order{
		ID:             orderID,
		UserID:         userID,
		ClientOrderID:  clientOrderID,
		Symbol:         symbol,
		Side:           mapOrderSideToString(side),
		Type:           mapOrderTypeToString(orderType),
		Price:          price,
		StopPrice:      stopPrice,
		Quantity:       quantity,
		FilledQuantity: 0,
		Status:         "pending", // Initial status
		Metadata:       string(metadataJSON),
	}
	
	// Set expiry time for GTD orders
	if orderType == orders.OrderType_LIMIT {
		// Default to 30 days for limit orders
		dbOrder.ExpiresAt = now.Add(30 * 24 * time.Hour)
	}
	
	// Save order to database
	if err := s.repository.Create(ctx, dbOrder); err != nil {
		s.logger.Error("Failed to create order", 
			zap.Error(err), 
			zap.String("order_id", orderID))
		return nil, fmt.Errorf("failed to create order: %w", err)
	}
	
	// Convert to proto response
	response := dbOrderToProto(dbOrder)
	
	// In a real system, we would publish the order to a message queue for processing
	// For now, we'll just log it
	s.logger.Info("Order created successfully", 
		zap.String("order_id", orderID),
		zap.String("user_id", userID),
		zap.String("symbol", symbol))
	
	return response, nil
}

// GetOrder retrieves an order by ID
func (s *Service) GetOrder(ctx context.Context, orderID string) (*orders.OrderResponse, error) {
	s.logger.Info("Getting order", zap.String("order_id", orderID))
	
	// Validate inputs
	if orderID == "" {
		return nil, errors.New("order ID is required")
	}
	
	// Extract user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, errors.New("user ID not found in context")
	}
	
	// Get order from database
	dbOrder, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		s.logger.Error("Failed to get order", 
			zap.Error(err), 
			zap.String("order_id", orderID))
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	
	if dbOrder == nil {
		return nil, errors.New("order not found")
	}
	
	// Check if the order belongs to the user
	if dbOrder.UserID != userID {
		// For security, don't reveal that the order exists
		return nil, errors.New("order not found")
	}
	
	// Convert to proto response
	response := dbOrderToProto(dbOrder)
	
	return response, nil
}

// CancelOrder cancels an existing order
func (s *Service) CancelOrder(ctx context.Context, orderID string) (*orders.OrderResponse, error) {
	s.logger.Info("Canceling order", zap.String("order_id", orderID))
	
	// Validate inputs
	if orderID == "" {
		return nil, errors.New("order ID is required")
	}
	
	// Extract user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, errors.New("user ID not found in context")
	}
	
	// Get order from database
	dbOrder, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		s.logger.Error("Failed to get order for cancellation", 
			zap.Error(err), 
			zap.String("order_id", orderID))
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	
	if dbOrder == nil {
		return nil, errors.New("order not found")
	}
	
	// Check if the order belongs to the user
	if dbOrder.UserID != userID {
		// For security, don't reveal that the order exists
		return nil, errors.New("order not found")
	}
	
	// Check if the order can be canceled
	if dbOrder.Status == "filled" || dbOrder.Status == "canceled" || dbOrder.Status == "rejected" || dbOrder.Status == "expired" {
		return nil, fmt.Errorf("order cannot be canceled: status is %s", dbOrder.Status)
	}
	
	// Update order status
	dbOrder.Status = "canceled"
	
	// Save order to database
	if err := s.repository.Update(ctx, dbOrder); err != nil {
		s.logger.Error("Failed to update order status", 
			zap.Error(err), 
			zap.String("order_id", orderID))
		return nil, fmt.Errorf("failed to update order: %w", err)
	}
	
	// Convert to proto response
	response := dbOrderToProto(dbOrder)
	
	// In a real system, we would publish the cancellation to a message queue
	// For now, we'll just log it
	s.logger.Info("Order canceled successfully", 
		zap.String("order_id", orderID),
		zap.String("user_id", userID))
	
	return response, nil
}

// GetOrders retrieves a list of orders
func (s *Service) GetOrders(ctx context.Context, symbol string, status orders.OrderStatus, startTime, endTime int64, limit int32) ([]*orders.OrderResponse, error) {
	s.logger.Info("Getting orders",
		zap.String("symbol", symbol),
		zap.String("status", status.String()),
		zap.Int64("start_time", startTime),
		zap.Int64("end_time", endTime),
		zap.Int32("limit", limit))
	
	// Extract user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, errors.New("user ID not found in context")
	}
	
	// Set default limit if not provided
	if limit <= 0 {
		limit = 100
	}
	
	// Get orders from database
	// In a real implementation, we would use the filters provided
	// For now, we'll just get all orders for the user
	dbOrders, err := s.repository.GetOrdersByUserID(ctx, userID, int(limit), 0)
	if err != nil {
		s.logger.Error("Failed to get orders", 
			zap.Error(err), 
			zap.String("user_id", userID))
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	
	// Filter orders by symbol and status if provided
	var filteredOrders []*db.Order
	for _, order := range dbOrders {
		if symbol != "" && order.Symbol != symbol {
			continue
		}
		
		if status != orders.OrderStatus_UNKNOWN {
			statusStr := mapOrderStatusToString(status)
			if order.Status != statusStr {
				continue
			}
		}
		
		// Filter by time range if provided
		if startTime > 0 {
			if order.CreatedAt.UnixMilli() < startTime {
				continue
			}
		}
		
		if endTime > 0 {
			if order.CreatedAt.UnixMilli() > endTime {
				continue
			}
		}
		
		filteredOrders = append(filteredOrders, order)
	}
	
	// Convert to proto responses
	var responses []*orders.OrderResponse
	for _, order := range filteredOrders {
		responses = append(responses, dbOrderToProto(order))
	}
	
	return responses, nil
}

// ServiceModule provides the order service module for fx
var ServiceModule = fx.Options(
	fx.Provide(NewService),
)
