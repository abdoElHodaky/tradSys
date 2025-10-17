package risk

import (
	"context"

	"github.com/abdoElHodaky/tradSys/proto/risk"
	"go.uber.org/zap"
)

// Handler implements the RiskService interface with stub methods
type Handler struct {
	logger *zap.Logger
}

// NewHandler creates a new risk handler
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// ValidateOrder implements the RiskService.ValidateOrder method
func (h *Handler) ValidateOrder(ctx context.Context, req *risk.ValidateOrderRequest) (*risk.ValidateOrderResponse, error) {
	h.logger.Info("ValidateOrder called",
		zap.String("symbol", req.Symbol),
		zap.String("side", req.Side.String()),
		zap.Float64("quantity", req.Quantity),
		zap.String("account_id", req.AccountId))

	// Return a simple valid response
	return &risk.ValidateOrderResponse{
		IsValid:         true,
		RejectionReason: "",
	}, nil
}

// GetAccountRisk implements the RiskService.GetAccountRisk method
func (h *Handler) GetAccountRisk(ctx context.Context, req *risk.AccountRiskRequest) (*risk.AccountRiskResponse, error) {
	h.logger.Info("GetAccountRisk called", zap.String("account_id", req.AccountId))

	return &risk.AccountRiskResponse{
		AccountId:         req.AccountId,
		TotalValue:        100000.0,
		AvailableMargin:   50000.0,
		UsedMargin:        10000.0,
		MarginLevel:       500.0,
		MarginCallLevel:   100.0,
		LiquidationLevel:  50.0,
		DailyPnl:          1000.0,
		TotalPnl:          5000.0,
		RiskLevel:         risk.RiskLevel_LOW,
	}, nil
}

// GetPositionRisk implements the RiskService.GetPositionRisk method
func (h *Handler) GetPositionRisk(ctx context.Context, req *risk.PositionRiskRequest) (*risk.PositionRiskResponse, error) {
	h.logger.Info("GetPositionRisk called",
		zap.String("account_id", req.AccountId),
		zap.String("symbol", req.Symbol))

	return &risk.PositionRiskResponse{
		AccountId:         req.AccountId,
		Symbol:            req.Symbol,
		Size:              1.0,
		EntryPrice:        50000.0,
		CurrentPrice:      51000.0,
		LiquidationPrice:  45000.0,
		UnrealizedPnl:     1000.0,
		RealizedPnl:       0.0,
		InitialMargin:     5000.0,
		MaintenanceMargin: 2500.0,
		RiskLevel:         risk.RiskLevel_LOW,
	}, nil
}

// GetOrderRisk implements the RiskService.GetOrderRisk method
func (h *Handler) GetOrderRisk(ctx context.Context, req *risk.OrderRiskRequest) (*risk.OrderRiskResponse, error) {
	h.logger.Info("GetOrderRisk called",
		zap.String("account_id", req.AccountId),
		zap.String("symbol", req.Symbol))

	return &risk.OrderRiskResponse{
		AccountId:              req.AccountId,
		Symbol:                 req.Symbol,
		Side:                   req.Side,
		Type:                   req.Type,
		Quantity:               req.Quantity,
		Price:                  req.Price,
		RequiredMargin:         1000.0,
		AvailableMarginAfter:   49000.0,
		MarginLevelAfter:       490.0,
		RiskLevel:              risk.RiskLevel_LOW,
		IsAllowed:              true,
		RejectionReason:        "",
	}, nil
}

// UpdateRiskLimits implements the RiskService.UpdateRiskLimits method
func (h *Handler) UpdateRiskLimits(ctx context.Context, req *risk.UpdateRiskLimitsRequest) (*risk.UpdateRiskLimitsResponse, error) {
	h.logger.Info("UpdateRiskLimits called", zap.String("account_id", req.AccountId))

	return &risk.UpdateRiskLimitsResponse{
		AccountId:  req.AccountId,
		RiskLimits: req.RiskLimits,
	}, nil
}
