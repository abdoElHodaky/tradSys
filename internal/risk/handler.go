package risk

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/proto/risk"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// HandlerParams contains the parameters for creating a risk handler
type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Repository *repositories.RiskRepository `optional:"true"`
}

// Handler implements the RiskService handler
type Handler struct {
	risk.UnimplementedRiskServiceServer
	logger     *zap.Logger
	repository *repositories.RiskRepository
}

// NewHandler creates a new risk handler with fx dependency injection
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		logger:     p.Logger,
		repository: p.Repository,
	}
}

// ValidateOrder implements the RiskService.ValidateOrder method
func (h *Handler) ValidateOrder(ctx context.Context, req *risk.ValidateOrderRequest) (*risk.ValidateOrderResponse, error) {
	h.logger.Info("ValidateOrder called",
		zap.String("symbol", req.Symbol),
		zap.String("side", req.Side.String()),
		zap.Float64("quantity", req.Quantity),
		zap.String("account_id", req.AccountId))

	// Implementation would go here
	// For now, just return a placeholder response
	rsp := &risk.ValidateOrderResponse{
		IsValid: true,
	}

	return rsp, nil
}

// GetAccountRisk implements the RiskService.GetAccountRisk method
func (h *Handler) GetAccountRisk(ctx context.Context, req *risk.AccountRiskRequest) (*risk.AccountRiskResponse, error) {
	h.logger.Info("GetAccountRisk called",
		zap.String("account_id", req.AccountId))

	// Implementation would go here
	// For now, just return placeholder risk data
	rsp := &risk.AccountRiskResponse{
		AccountId:       req.AccountId,
		TotalValue:      100000.0,
		AvailableMargin: 50000.0,
		UsedMargin:      25000.0,
		MarginLevel:     200.0,
		MarginCallLevel: 120.0,
		LiquidationLevel: 100.0,
		DailyPnl:        1500.0,
		TotalPnl:        5000.0,
		RiskLevel:       risk.RiskLevel_MEDIUM,
		Positions: []*risk.Position{
			{
				Symbol:        "BTC-USD",
				Size:          1.5,
				EntryPrice:    48000.0,
				CurrentPrice:  50000.0,
				UnrealizedPnl: 3000.0,
				RealizedPnl:   1000.0,
			},
			{
				Symbol:        "ETH-USD",
				Size:          10.0,
				EntryPrice:    3200.0,
				CurrentPrice:  3500.0,
				UnrealizedPnl: 3000.0,
				RealizedPnl:   2000.0,
			},
		},
	}

	return rsp, nil
}

// GetPositionRisk implements the RiskService.GetPositionRisk method
func (h *Handler) GetPositionRisk(ctx context.Context, req *risk.PositionRiskRequest) (*risk.PositionRiskResponse, error) {
	h.logger.Info("GetPositionRisk called",
		zap.String("symbol", req.Symbol),
		zap.String("account_id", req.AccountId))

	// Implementation would go here
	// For now, just return placeholder position risk data
	rsp := &risk.PositionRiskResponse{
		AccountId:         req.AccountId,
		Symbol:           req.Symbol,
		Size:             1.5,
		EntryPrice:       48000.0,
		CurrentPrice:     50000.0,
		LiquidationPrice: 40000.0,
		UnrealizedPnl:    3000.0,
		RealizedPnl:      1000.0,
		InitialMargin:    9600.0,
		MaintenanceMargin: 4800.0,
		RiskLevel:        risk.RiskLevel_MEDIUM,
	}

	return rsp, nil
}

// GetOrderRisk implements the RiskService.GetOrderRisk method
func (h *Handler) GetOrderRisk(ctx context.Context, req *risk.OrderRiskRequest) (*risk.OrderRiskResponse, error) {
	h.logger.Info("GetOrderRisk called",
		zap.String("symbol", req.Symbol),
		zap.String("account_id", req.AccountId))

	// Calculate real risk metrics
	orderValue := req.Quantity * req.Price
	marginRate := h.getMarginRate(req.Symbol) // Get symbol-specific margin rate
	requiredMargin := orderValue * marginRate
	
	// Get current account balance (in production, this would come from account service)
	currentBalance := h.getCurrentAccountBalance(req.AccountId)
	availableMarginAfter := currentBalance - requiredMargin
	
	// Calculate margin level after order
	var marginLevelAfter float64
	if requiredMargin > 0 {
		marginLevelAfter = (availableMarginAfter / requiredMargin) * 100
	} else {
		marginLevelAfter = 100.0
	}
	
	// Determine risk level based on margin level and order size
	riskLevel := h.calculateRiskLevel(marginLevelAfter, orderValue, currentBalance)
	
	// Check if order is allowed based on risk assessment
	isAllowed := h.isOrderAllowed(riskLevel, marginLevelAfter, availableMarginAfter)

	rsp := &risk.OrderRiskResponse{
		AccountId:             req.AccountId,
		Symbol:               req.Symbol,
		Side:                 req.Side,
		Type:                 req.Type,
		Quantity:             req.Quantity,
		Price:                req.Price,
		RequiredMargin:       requiredMargin,
		AvailableMarginAfter: availableMarginAfter,
		MarginLevelAfter:     marginLevelAfter,
		RiskLevel:            riskLevel,
		IsAllowed:            isAllowed,
		RejectionReason:      "",
	}

	return rsp, nil
}

// UpdateRiskLimits implements the RiskService.UpdateRiskLimits method
func (h *Handler) UpdateRiskLimits(ctx context.Context, req *risk.UpdateRiskLimitsRequest) (*risk.UpdateRiskLimitsResponse, error) {
	h.logger.Info("UpdateRiskLimits called",
		zap.String("account_id", req.AccountId))

	// Implementation would go here
	// For now, just return the updated risk limits
	rsp := &risk.UpdateRiskLimitsResponse{
		AccountId:  req.AccountId,
		RiskLimits: req.RiskLimits,
	}

	return rsp, nil
}

// getMarginRate returns the margin rate for a given symbol
func (h *Handler) getMarginRate(symbol string) float64 {
	// In production, this would come from configuration or database
	marginRates := map[string]float64{
		"BTCUSDT": 0.1,  // 10% margin for BTC
		"ETHUSDT": 0.15, // 15% margin for ETH
		"ADAUSDT": 0.2,  // 20% margin for ADA
	}
	
	if rate, exists := marginRates[symbol]; exists {
		return rate
	}
	
	// Default margin rate for unknown symbols
	return 0.25 // 25% margin
}

// getCurrentAccountBalance returns the current account balance
func (h *Handler) getCurrentAccountBalance(accountID string) float64 {
	// In production, this would query the account service
	// For demo purposes, return a mock balance
	return 100000.0 // $100,000 demo balance
}

// calculateRiskLevel determines the risk level based on various factors
func (h *Handler) calculateRiskLevel(marginLevel, orderValue, accountBalance float64) risk.RiskLevel {
	// Calculate order size as percentage of account balance
	orderSizePercent := (orderValue / accountBalance) * 100
	
	// Determine risk level based on margin level and order size
	if marginLevel < 50 || orderSizePercent > 50 {
		return risk.RiskLevel_HIGH
	} else if marginLevel < 100 || orderSizePercent > 25 {
		return risk.RiskLevel_MEDIUM
	} else {
		return risk.RiskLevel_LOW
	}
}

// isOrderAllowed determines if an order should be allowed based on risk assessment
func (h *Handler) isOrderAllowed(riskLevel risk.RiskLevel, marginLevel, availableMargin float64) bool {
	// Reject orders if insufficient margin
	if availableMargin < 0 {
		return false
	}
	
	// Reject high-risk orders with low margin levels
	if riskLevel == risk.RiskLevel_HIGH && marginLevel < 25 {
		return false
	}
	
	// Allow all other orders
	return true
}

// RiskModule provides the risk handler module for fx
var RiskModule = fx.Options(
	fx.Provide(NewHandler),
)
