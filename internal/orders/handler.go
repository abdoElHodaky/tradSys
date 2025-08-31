package orders

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/abdoElHodaky/tradSys/proto/risk"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// HandlerParams contains the parameters for creating an order handler
type HandlerParams struct {
	fx.In

	Logger          *zap.Logger
	Repository      *repositories.OrderRepository `optional:"true"`
	RiskServiceConn *grpc.ClientConn              `optional:"true" name:"riskService"`
}

// Handler implements the Service handler
type Handler struct {
	orders.UnimplementedOrderServiceServer
	logger     *zap.Logger
	repository *repositories.OrderRepository
	riskClient risk.RiskServiceClient
}

// NewHandler creates a new order handler with fx dependency injection
func NewHandler(p HandlerParams) *Handler {
	var riskClient risk.RiskServiceClient
	if p.RiskServiceConn != nil {
		riskClient = risk.NewRiskServiceClient(p.RiskServiceConn)
	}

	return &Handler{
		logger:     p.Logger,
		repository: p.Repository,
		riskClient: riskClient,
	}
}

// CreateOrder implements the OrderService.CreateOrder method
func (h *Handler) CreateOrder(ctx context.Context, req *orders.CreateOrderRequest, rsp *orders.OrderResponse) error {
	h.logger.Info("CreateOrder called",
		zap.String("symbol", req.Symbol),
		zap.String("side", req.Side.String()),
		zap.Float64("quantity", req.Quantity))

	// Validate order with risk service if available
	if h.riskClient != nil {
		validateReq := &risk.ValidateOrderRequest{
			Symbol:    req.Symbol,
			Side:      risk.OrderSide(req.Side),
			Quantity:  req.Quantity,
			Price:     req.Price,
			AccountId: "default", // In a real implementation, this would come from auth context
		}

		validateRsp, err := h.riskClient.ValidateOrder(ctx, validateReq)
		if err != nil {
			h.logger.Error("Failed to validate order with risk service", zap.Error(err))
			return err
		}

		if !validateRsp.Valid {
			h.logger.Warn("Order validation failed",
				zap.String("reason", validateRsp.Reason),
				zap.Float64("max_allowed_quantity", validateRsp.MaxAllowedQuantity))
			return grpc.Errorf(grpc.Code(400), "Order validation failed: %s", validateRsp.Reason)
		}
	}

	// Implementation would go here
	// For now, just return a placeholder response
	rsp.OrderId = uuid.New().String()
	rsp.Symbol = req.Symbol
	rsp.Type = req.Type
	rsp.Side = req.Side
	rsp.Status = orders.OrderStatus_PENDING
	rsp.Quantity = req.Quantity
	rsp.FilledQuantity = 0
	rsp.Price = req.Price
	rsp.StopPrice = req.StopPrice
	rsp.CreatedAt = 1625097600000
	rsp.UpdatedAt = 1625097600000
	rsp.ClientOrderId = req.ClientOrderId

	return nil
}

// GetOrder implements the OrderService.GetOrder method
func (h *Handler) GetOrder(ctx context.Context, req *orders.GetOrderRequest, rsp *orders.OrderResponse) error {
	h.logger.Info("GetOrder called",
		zap.String("order_id", req.OrderId))

	// Implementation would go here
	// For now, just return a placeholder response
	rsp.OrderId = req.OrderId
	rsp.Symbol = "BTC-USD"
	rsp.Type = orders.OrderType_LIMIT
	rsp.Side = orders.OrderSide_BUY
	rsp.Status = orders.OrderStatus_OPEN
	rsp.Quantity = 1.0
	rsp.FilledQuantity = 0.5
	rsp.Price = 50000.0
	rsp.CreatedAt = 1625097600000
	rsp.UpdatedAt = 1625097660000

	return nil
}

// CancelOrder implements the OrderService.CancelOrder method
func (h *Handler) CancelOrder(ctx context.Context, req *orders.CancelOrderRequest, rsp *orders.OrderResponse) error {
	h.logger.Info("CancelOrder called",
		zap.String("order_id", req.OrderId))

	// Implementation would go here
	// For now, just return a placeholder response
	rsp.OrderId = req.OrderId
	rsp.Symbol = "BTC-USD"
	rsp.Type = orders.OrderType_LIMIT
	rsp.Side = orders.OrderSide_BUY
	rsp.Status = orders.OrderStatus_CANCELED
	rsp.Quantity = 1.0
	rsp.FilledQuantity = 0.5
	rsp.Price = 50000.0
	rsp.CreatedAt = 1625097600000
	rsp.UpdatedAt = 1625097720000

	return nil
}

// GetOrders implements the OrderService.GetOrders method
func (h *Handler) GetOrders(ctx context.Context, req *orders.GetOrdersRequest, rsp *orders.GetOrdersResponse) error {
	h.logger.Info("GetOrders called",
		zap.String("symbol", req.Symbol),
		zap.String("status", req.Status.String()))

	// Implementation would go here
	// For now, just return placeholder responses
	rsp.Orders = []*orders.OrderResponse{
		{
			OrderId:        uuid.New().String(),
			Symbol:         req.Symbol,
			Type:           orders.OrderType_LIMIT,
			Side:           orders.OrderSide_BUY,
			Status:         req.Status,
			Quantity:       1.0,
			FilledQuantity: 0.5,
			Price:          50000.0,
			CreatedAt:      1625097600000,
			UpdatedAt:      1625097660000,
		},
		{
			OrderId:        uuid.New().String(),
			Symbol:         req.Symbol,
			Type:           orders.OrderType_MARKET,
			Side:           orders.OrderSide_SELL,
			Status:         req.Status,
			Quantity:       0.5,
			FilledQuantity: 0.5,
			Price:          51000.0,
			CreatedAt:      1625097700000,
			UpdatedAt:      1625097760000,
		},
	}

	return nil
}

// Module provides the orders module for fx
var Module = fx.Options(
	fx.Provide(NewHandler),
)
