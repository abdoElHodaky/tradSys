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
func (h *Handler) ValidateOrder(ctx context.Context, req *risk.ValidateOrderRequest, rsp *risk.ValidateOrderResponse) error {
	h.logger.Info("ValidateOrder called",
		zap.String("symbol", req.Symbol),
		zap.String("side", req.Side.String()),
		zap.Float64("quantity", req.Quantity),
		zap.String("account_id", req.AccountId))

	// Implementation would go here
	// For now, just return a placeholder response
	rsp.IsValid = true

	return nil
}

// GetAccountRisk implements the RiskService.GetAccountRisk method
func (h *Handler) GetAccountRisk(ctx context.Context, req *risk.AccountRiskRequest, rsp *risk.AccountRiskResponse) error {
	h.logger.Info("GetAccountRisk called",
		zap.String("account_id", req.AccountId))

	// Implementation would go here
	// For now, just return placeholder risk data
	rsp.AccountId = req.AccountId
	rsp.TotalValue = 100000.0
	rsp.AvailableMargin = 50000.0
	rsp.UsedMargin = 25000.0
	rsp.MarginLevel = 200.0
	rsp.MarginCallLevel = 120.0
	rsp.LiquidationLevel = 100.0
	rsp.DailyPnl = 1500.0
	rsp.TotalPnl = 5000.0
	rsp.RiskLevel = risk.RiskLevel_MEDIUM
	rsp.Positions = []*risk.Position{
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
	}

	return nil
}

// GetPositionRisk implements the RiskService.GetPositionRisk method
func (h *Handler) GetPositionRisk(ctx context.Context, req *risk.PositionRiskRequest, rsp *risk.PositionRiskResponse) error {
	h.logger.Info("GetPositionRisk called",
		zap.String("symbol", req.Symbol),
		zap.String("account_id", req.AccountId))

	// Implementation would go here
	// For now, just return placeholder position risk data
	rsp.AccountId = req.AccountId
	rsp.Symbol = req.Symbol
	rsp.Size = 1.5
	rsp.EntryPrice = 48000.0
	rsp.CurrentPrice = 50000.0
	rsp.LiquidationPrice = 40000.0
	rsp.UnrealizedPnl = 3000.0
	rsp.RealizedPnl = 1000.0
	rsp.InitialMargin = 9600.0
	rsp.MaintenanceMargin = 4800.0
	rsp.RiskLevel = risk.RiskLevel_MEDIUM

	return nil
}

// GetOrderRisk implements the RiskService.GetOrderRisk method
func (h *Handler) GetOrderRisk(ctx context.Context, req *risk.OrderRiskRequest, rsp *risk.OrderRiskResponse) error {
	h.logger.Info("GetOrderRisk called",
		zap.String("symbol", req.Symbol),
		zap.String("account_id", req.AccountId))

	// Calculate real risk metrics
	rsp.AccountId = req.AccountId
	rsp.Symbol = req.Symbol
	rsp.Side = req.Side
	rsp.Type = req.Type
	rsp.Quantity = req.Quantity
	rsp.Price = req.Price
	
	// Calculate required margin based on order value and symbol
	orderValue := req.Quantity * req.Price
	marginRate := h.getMarginRate(req.Symbol) // Get symbol-specific margin rate
	rsp.RequiredMargin = orderValue * marginRate
	
	// Get current account balance (in production, this would come from account service)
	currentBalance := h.getCurrentAccountBalance(req.AccountId)
	rsp.AvailableMarginAfter = currentBalance - rsp.RequiredMargin
	
	// Calculate margin level after order
	if rsp.RequiredMargin > 0 {
		rsp.MarginLevelAfter = (rsp.AvailableMarginAfter / rsp.RequiredMargin) * 100
	} else {
		rsp.MarginLevelAfter = 100.0
	}
	
	// Determine risk level based on margin level and order size
	rsp.RiskLevel = h.calculateRiskLevel(rsp.MarginLevelAfter, orderValue, currentBalance)
	
	// Check if order is allowed based on risk assessment
	rsp.IsAllowed = h.isOrderAllowed(rsp.RiskLevel, rsp.MarginLevelAfter, rsp.AvailableMarginAfter)
	rsp.RejectionReason = ""

	return nil
}

// UpdateRiskLimits implements the RiskService.UpdateRiskLimits method
func (h *Handler) UpdateRiskLimits(ctx context.Context, req *risk.UpdateRiskLimitsRequest, rsp *risk.UpdateRiskLimitsResponse) error {
	h.logger.Info("UpdateRiskLimits called",
		zap.String("account_id", req.AccountId))

	// Implementation would go here
	// For now, just return the updated risk limits
	rsp.AccountId = req.AccountId
	rsp.RiskLimits = req.RiskLimits

	return nil
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
