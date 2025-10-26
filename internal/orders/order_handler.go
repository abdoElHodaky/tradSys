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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HandlerParams contains the parameters for creating an order handler
type HandlerParams struct {
	fx.In

	Logger          *zap.Logger
	Repository      *repositories.OrderRepository `optional:"true"`
	RiskServiceConn *grpc.ClientConn              `optional:"true" name:"riskService"`
}

// Handler implements the OrderService handler
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
func (h *Handler) CreateOrder(ctx context.Context, req *orders.CreateOrderRequest) (*orders.OrderResponse, error) {
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
			return nil, err
		}

		if !validateRsp.IsValid {
			h.logger.Warn("Order validation failed",
				zap.String("reason", validateRsp.RejectionReason))
			return nil, status.Errorf(codes.InvalidArgument, "Order validation failed: %s", validateRsp.RejectionReason)
		}
	}

	// Implementation would go here
	// For now, just return a placeholder response
	rsp := &orders.OrderResponse{
		Id:            uuid.New().String(),
		Symbol:        req.Symbol,
		Type:          req.Type,
		Side:          req.Side,
		Status:        orders.OrderStatus_PENDING,
		Quantity:      req.Quantity,
		FilledQty:     0,
		Price:         req.Price,
		StopPrice:     req.StopPrice,
		CreatedAt:     1625097600000,
		UpdatedAt:     1625097600000,
		ClientOrderId: req.ClientOrderId,
	}

	return rsp, nil
}

// GetOrder implements the OrderService.GetOrder method
func (h *Handler) GetOrder(ctx context.Context, req *orders.GetOrderRequest) (*orders.OrderResponse, error) {
	h.logger.Info("GetOrder called",
		zap.String("order_id", req.Id))

	// Implementation would go here
	// For now, just return a placeholder response
	rsp := &orders.OrderResponse{
		Id:        req.Id,
		Symbol:    "BTC-USD",
		Type:      orders.OrderType_LIMIT,
		Side:      orders.OrderSide_BUY,
		Status:    orders.OrderStatus_NEW,
		Quantity:  1.0,
		FilledQty: 0.5,
		Price:     50000.0,
		CreatedAt: 1625097600000,
		UpdatedAt: 1625097660000,
	}

	return rsp, nil
}

// CancelOrder implements the OrderService.CancelOrder method
func (h *Handler) CancelOrder(ctx context.Context, req *orders.CancelOrderRequest) (*orders.OrderResponse, error) {
	h.logger.Info("CancelOrder called",
		zap.String("order_id", req.Id))

	// Implementation would go here
	// For now, just return a placeholder response
	rsp := &orders.OrderResponse{
		Id:        req.Id,
		Symbol:    "BTC-USD",
		Type:      orders.OrderType_LIMIT,
		Side:      orders.OrderSide_BUY,
		Status:    orders.OrderStatus_CANCELLED,
		Quantity:  1.0,
		FilledQty: 0.5,
		Price:     50000.0,
		CreatedAt: 1625097600000,
		UpdatedAt: 1625097720000,
	}

	return rsp, nil
}

// GetOrders implements the OrderService.GetOrders method
func (h *Handler) GetOrders(ctx context.Context, req *orders.GetOrdersRequest) (*orders.GetOrdersResponse, error) {
	h.logger.Info("GetOrders called",
		zap.String("symbol", req.Symbol),
		zap.String("status", req.Status.String()))

	// Implementation would go here
	// For now, just return placeholder responses
	rsp := &orders.GetOrdersResponse{
		Orders: []*orders.OrderResponse{
			{
				Id:        uuid.New().String(),
				Symbol:    req.Symbol,
				Type:      orders.OrderType_LIMIT,
				Side:      orders.OrderSide_BUY,
				Status:    req.Status,
				Quantity:  1.0,
				FilledQty: 0.5,
				Price:     50000.0,
				CreatedAt: 1625097600000,
				UpdatedAt: 1625097660000,
			},
			{
				Id:        uuid.New().String(),
				Symbol:    req.Symbol,
				Type:      orders.OrderType_MARKET,
				Side:      orders.OrderSide_SELL,
				Status:    req.Status,
				Quantity:  0.5,
				FilledQty: 0.5,
				Price:     51000.0,
				CreatedAt: 1625097700000,
				UpdatedAt: 1625097760000,
			},
		},
	}

	return rsp, nil
}

// OrdersModule provides the orders handler module for fx
var OrdersModule = fx.Options(
	fx.Provide(NewHandler),
)
