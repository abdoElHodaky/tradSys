package orders

import (
	"context"

	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// Handler implements the OrderService interface with stub methods
type Handler struct {
	logger *zap.Logger
}

// NewHandler creates a new order handler
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// CreateOrder implements the OrderService.CreateOrder method
func (h *Handler) CreateOrder(ctx context.Context, req *orders.CreateOrderRequest) (*orders.OrderResponse, error) {
	h.logger.Info("CreateOrder called",
		zap.String("symbol", req.Symbol),
		zap.String("side", req.Side.String()),
		zap.Float64("quantity", req.Quantity))

	return &orders.OrderResponse{
		Id:        "order-123",
		UserId:    req.UserId,
		AccountId: req.AccountId,
		Symbol:    req.Symbol,
		Side:      req.Side,
		Type:      req.Type,
		Quantity:  req.Quantity,
		Price:     req.Price,
		Status:    orders.OrderStatus_NEW,
		FilledQty: 0.0,
		AvgPrice:  0.0,
	}, nil
}

// GetOrder implements the OrderService.GetOrder method
func (h *Handler) GetOrder(ctx context.Context, req *orders.GetOrderRequest) (*orders.OrderResponse, error) {
	h.logger.Info("GetOrder called", zap.String("id", req.Id))

	return &orders.OrderResponse{
		Id:        req.Id,
		UserId:    req.UserId,
		Symbol:    "BTC-USD",
		Side:      orders.OrderSide_BUY,
		Type:      orders.OrderType_LIMIT,
		Quantity:  1.0,
		Price:     50000.0,
		Status:    orders.OrderStatus_FILLED,
		FilledQty: 1.0,
		AvgPrice:  50000.0,
	}, nil
}

// CancelOrder implements the OrderService.CancelOrder method
func (h *Handler) CancelOrder(ctx context.Context, req *orders.CancelOrderRequest) (*orders.OrderResponse, error) {
	h.logger.Info("CancelOrder called", zap.String("id", req.Id))

	return &orders.OrderResponse{
		Id:        req.Id,
		UserId:    req.UserId,
		Symbol:    "BTC-USD",
		Side:      orders.OrderSide_BUY,
		Type:      orders.OrderType_LIMIT,
		Quantity:  1.0,
		Price:     50000.0,
		Status:    orders.OrderStatus_CANCELLED,
		FilledQty: 0.0,
		AvgPrice:  0.0,
	}, nil
}

// GetOrders implements the OrderService.GetOrders method
func (h *Handler) GetOrders(ctx context.Context, req *orders.GetOrdersRequest) (*orders.GetOrdersResponse, error) {
	h.logger.Info("GetOrders called", zap.String("user_id", req.UserId))

	return &orders.GetOrdersResponse{
		Orders: []*orders.OrderResponse{
			{
				Id:        "order-1",
				UserId:    req.UserId,
				Symbol:    "BTC-USD",
				Side:      orders.OrderSide_BUY,
				Type:      orders.OrderType_LIMIT,
				Quantity:  1.0,
				Price:     50000.0,
				Status:    orders.OrderStatus_FILLED,
				FilledQty: 1.0,
				AvgPrice:  50000.0,
			},
		},
	}, nil
}

// StreamOrders implements the OrderService.StreamOrders method
func (h *Handler) StreamOrders(req *orders.StreamOrdersRequest, stream orders.OrderService_StreamOrdersServer) error {
	h.logger.Info("StreamOrders called", zap.String("user_id", req.UserId))

	// Send a sample order
	order := &orders.OrderResponse{
		Id:        "order-stream-1",
		UserId:    req.UserId,
		Symbol:    "BTC-USD",
		Side:      orders.OrderSide_BUY,
		Type:      orders.OrderType_LIMIT,
		Quantity:  1.0,
		Price:     50000.0,
		Status:    orders.OrderStatus_NEW,
		FilledQty: 0.0,
		AvgPrice:  0.0,
	}

	return stream.Send(order)
}
