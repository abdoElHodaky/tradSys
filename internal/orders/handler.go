package orders

import (
	"context"
	"errors"

	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/abdoElHodaky/tradSys/proto/risk"
	gomicro "go-micro.dev/v4"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// HandlerParams contains the parameters for creating an order handler
type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Repository *repositories.OrderRepository `optional:"true"`
	Service    *gomicro.Service              `optional:"true"`
}

// Handler implements the OrderService handler
type Handler struct {
	orders.UnimplementedOrderServiceServer
	logger      *zap.Logger
	repository  *repositories.OrderRepository
	riskService risk.RiskService
}

// NewHandler creates a new order handler with fx dependency injection
func NewHandler(p HandlerParams) *Handler {
	var riskService risk.RiskService
	
	// Create client for the risk service if service is available
	if p.Service != nil {
		riskService = risk.NewRiskService("risk", (*p.Service).Client())
	}

	return &Handler{
		logger:      p.Logger,
		repository:  p.Repository,
		riskService: riskService,
	}
}

// CreateOrder implements the OrderService.CreateOrder method
func (h *Handler) CreateOrder(ctx context.Context, req *orders.CreateOrderRequest, rsp *orders.OrderResponse) error {
	h.logger.Info("CreateOrder called",
		zap.String("user_id", req.UserId),
		zap.String("symbol", req.Symbol),
		zap.Int32("side", int32(req.Side)),
		zap.Int32("type", int32(req.Type)))

	// Validate order with risk service if available
	if h.riskService != nil {
		validateReq := &risk.ValidateOrderRequest{
			AccountId: req.AccountId,
			Symbol:    req.Symbol,
			Side:      risk.OrderSide(req.Side),
			Type:      risk.OrderType(req.Type),
			Quantity:  req.Quantity,
			Price:     req.Price,
		}

		validateRsp, err := h.riskService.ValidateOrder(ctx, validateReq)
		if err != nil {
			h.logger.Error("Failed to validate order with risk service", zap.Error(err))
			return err
		}

		if !validateRsp.IsValid {
			return errors.New(validateRsp.RejectionReason)
		}
	}

	// Implementation would go here
	// For now, just return a placeholder response
	rsp.Id = "ord-123456"
	rsp.UserId = req.UserId
	rsp.AccountId = req.AccountId
	rsp.Symbol = req.Symbol
	rsp.Side = req.Side
	rsp.Type = req.Type
	rsp.Quantity = req.Quantity
	rsp.Price = req.Price
	rsp.Status = orders.OrderStatus_NEW
	rsp.CreatedAt = 1625097600000

	return nil
}

// GetOrder implements the OrderService.GetOrder method
func (h *Handler) GetOrder(ctx context.Context, req *orders.GetOrderRequest, rsp *orders.OrderResponse) error {
	h.logger.Info("GetOrder called",
		zap.String("id", req.Id),
		zap.String("user_id", req.UserId))

	// Implementation would go here
	// For now, just return a placeholder response
	rsp.Id = req.Id
	rsp.UserId = req.UserId
	rsp.AccountId = "acc-123456"
	rsp.Symbol = "BTC-USD"
	rsp.Side = orders.OrderSide_BUY
	rsp.Type = orders.OrderType_LIMIT
	rsp.Quantity = 1.0
	rsp.Price = 50000.0
	rsp.Status = orders.OrderStatus_FILLED
	rsp.FilledQty = 1.0
	rsp.AvgPrice = 50000.0
	rsp.CreatedAt = 1625097600000
	rsp.UpdatedAt = 1625097660000

	return nil
}

// CancelOrder implements the OrderService.CancelOrder method
func (h *Handler) CancelOrder(ctx context.Context, req *orders.CancelOrderRequest, rsp *orders.OrderResponse) error {
	h.logger.Info("CancelOrder called",
		zap.String("id", req.Id),
		zap.String("user_id", req.UserId))

	// Implementation would go here
	// For now, just return a placeholder response
	rsp.Id = req.Id
	rsp.UserId = req.UserId
	rsp.AccountId = "acc-123456"
	rsp.Symbol = "BTC-USD"
	rsp.Side = orders.OrderSide_BUY
	rsp.Type = orders.OrderType_LIMIT
	rsp.Quantity = 1.0
	rsp.Price = 50000.0
	rsp.Status = orders.OrderStatus_CANCELLED
	rsp.CreatedAt = 1625097600000
	rsp.UpdatedAt = 1625097660000

	return nil
}

// GetOrders implements the OrderService.GetOrders method
func (h *Handler) GetOrders(ctx context.Context, req *orders.GetOrdersRequest, rsp *orders.GetOrdersResponse) error {
	h.logger.Info("GetOrders called",
		zap.String("user_id", req.UserId),
		zap.String("symbol", req.Symbol))

	// Implementation would go here
	// For now, just return placeholder orders
	rsp.Orders = []*orders.OrderResponse{
		{
			Id:        "ord-123456",
			UserId:    req.UserId,
			AccountId: "acc-123456",
			Symbol:    "BTC-USD",
			Side:      orders.OrderSide_BUY,
			Type:      orders.OrderType_LIMIT,
			Quantity:  1.0,
			Price:     50000.0,
			Status:    orders.OrderStatus_FILLED,
			FilledQty: 1.0,
			AvgPrice:  50000.0,
			CreatedAt: 1625097600000,
			UpdatedAt: 1625097660000,
		},
		{
			Id:        "ord-123457",
			UserId:    req.UserId,
			AccountId: "acc-123456",
			Symbol:    "ETH-USD",
			Side:      orders.OrderSide_SELL,
			Type:      orders.OrderType_MARKET,
			Quantity:  5.0,
			Status:    orders.OrderStatus_FILLED,
			FilledQty: 5.0,
			AvgPrice:  3000.0,
			CreatedAt: 1625097700000,
			UpdatedAt: 1625097760000,
		},
	}
	rsp.Total = 2

	return nil
}

// StreamOrders implements the OrderService.StreamOrders method
func (h *Handler) StreamOrders(ctx context.Context, req *orders.StreamOrdersRequest, stream orders.OrderService_StreamOrdersServer) error {
	h.logger.Info("StreamOrders called",
		zap.String("user_id", req.UserId),
		zap.String("symbol", req.Symbol))

	// Implementation would go here
	// For now, just return a placeholder order
	order := &orders.OrderResponse{
		Id:        "ord-123456",
		UserId:    req.UserId,
		AccountId: req.AccountId,
		Symbol:    req.Symbol,
		Side:      orders.OrderSide_BUY,
		Type:      orders.OrderType_LIMIT,
		Quantity:  1.0,
		Price:     50000.0,
		Status:    orders.OrderStatus_NEW,
		CreatedAt: 1625097600000,
	}

	if err := stream.Send(order); err != nil {
		return err
	}

	// In a real implementation, we would continue sending updates
	// until the context is canceled or the stream is closed

	return nil
}

// Module provides the orders module for fx
var Module = fx.Options(
	fx.Provide(NewHandler),
)

