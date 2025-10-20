package risk

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/proto/risk"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceParams contains the parameters for creating a risk service
type ServiceParams struct {
	fx.In

	Logger     *zap.Logger
	Repository *repositories.RiskRepository `optional:"true"`
}

// Service provides risk management operations
type Service struct {
	logger     *zap.Logger
	repository *repositories.RiskRepository
}

// NewService creates a new risk service with fx dependency injection
func NewService(p ServiceParams) *Service {
	return &Service{
		logger:     p.Logger,
		repository: p.Repository,
	}
}

// ValidateOrder validates an order against risk parameters
func (s *Service) ValidateOrder(ctx context.Context, symbol string, side risk.OrderSide, orderType risk.OrderType, quantity, price float64, accountID string) (*risk.ValidateOrderResponse, error) {
	s.logger.Info("Validating order",
		zap.String("symbol", symbol),
		zap.String("side", side.String()),
		zap.Float64("quantity", quantity),
		zap.Float64("price", price),
		zap.String("account_id", accountID))

	// Implementation would go here
	// For now, just return a placeholder response
	maxAllowedQuantity := 10.0
	maxAllowedNotional := 500000.0
	
	response := &risk.ValidateOrderResponse{
		IsValid: true,
		RiskMetrics: &risk.OrderRiskResponse{
			AccountId:     accountID,
			Symbol:        symbol,
			Side:          side,
			Type:          orderType,
			Quantity:      quantity,
			Price:         price,
			RequiredMargin: quantity * price * 0.2, // 20% margin requirement
			AvailableMarginAfter: 50000.0 - (quantity * price * 0.2),
			MarginLevelAfter: 150.0,
			RiskLevel:     risk.RiskLevel_LOW,
			IsAllowed:     true,
		},
	}

	// Check if the order exceeds position limits
	if quantity > maxAllowedQuantity {
		response.IsValid = false
		response.RejectionReason = "Order quantity exceeds maximum allowed"
<<<<<<< HEAD
=======
		response.RiskMetrics.IsAllowed = false
		response.RiskMetrics.RejectionReason = "Order quantity exceeds maximum allowed"
>>>>>>> codegen-bot/fix-build-errors-1760873074
	}

	// Check if the order exceeds notional value limits
	notionalValue := quantity * price
	if notionalValue > maxAllowedNotional {
		response.IsValid = false
		response.RejectionReason = "Order notional value exceeds maximum allowed"
<<<<<<< HEAD
=======
		response.RiskMetrics.IsAllowed = false
		response.RiskMetrics.RejectionReason = "Order notional value exceeds maximum allowed"
>>>>>>> codegen-bot/fix-build-errors-1760873074
	}

	return response, nil
}

// GetPositions returns current positions
func (s *Service) GetPositions(ctx context.Context, accountID, symbol string) ([]*risk.Position, error) {
	s.logger.Info("Getting positions",
		zap.String("account_id", accountID),
		zap.String("symbol", symbol))

	// Implementation would go here
	// For now, just return placeholder positions
	positions := []*risk.Position{
		{
<<<<<<< HEAD
			Symbol:        "BTC-USD",
			Size:          1.5,
			EntryPrice:    48000.0,
			CurrentPrice:  50000.0,
			UnrealizedPnl: 3000.0,
			RealizedPnl:   1000.0,
=======
			Symbol:          "BTC-USD",
			Size:            1.5,
			EntryPrice:      48000.0,
			CurrentPrice:    50000.0,
			LiquidationPrice: 40000.0,
			UnrealizedPnl:   3000.0,
			RealizedPnl:     1000.0,
>>>>>>> codegen-bot/fix-build-errors-1760873074
		},
	}

	// If symbol is specified, filter the positions
	if symbol != "" {
		var filteredPositions []*risk.Position
		for _, pos := range positions {
			if pos.Symbol == symbol {
				filteredPositions = append(filteredPositions, pos)
			}
		}
		positions = filteredPositions
	}

	return positions, nil
}

// GetRiskLimits returns risk limits for a symbol
func (s *Service) GetRiskLimits(ctx context.Context, symbol, accountID string) (*risk.RiskLimits, error) {
	s.logger.Info("Getting risk limits",
		zap.String("symbol", symbol),
		zap.String("account_id", accountID))

	// Implementation would go here
	// For now, just return placeholder risk limits
	limits := &risk.RiskLimits{
		MaxPositionSize:   10.0,
		MaxOrderSize:      5.0,
		MaxLeverage:       5.0,
		MaxDailyLoss:      1000.0,
<<<<<<< HEAD
		MaxTotalLoss:      10000.0,
		MinMarginLevel:    100.0,
		MarginCallLevel:   120.0,
		LiquidationLevel:  110.0,
=======
		MaxTotalLoss:      5000.0,
		MinMarginLevel:    120.0,
		MarginCallLevel:   120.0,
		LiquidationLevel:  100.0,
>>>>>>> codegen-bot/fix-build-errors-1760873074
	}

	return limits, nil
}

// UpdateRiskLimits updates risk limits for a symbol
func (s *Service) UpdateRiskLimits(ctx context.Context, symbol, accountID string, limits *risk.RiskLimits) (*risk.RiskLimits, error) {
	s.logger.Info("Updating risk limits",
		zap.String("symbol", symbol),
		zap.String("account_id", accountID),
		zap.Float64("max_position_size", limits.MaxPositionSize),
		zap.Float64("max_order_size", limits.MaxOrderSize))

	// Implementation would go here
	// For now, just return the updated limits
	return limits, nil
}

// ServiceModule is defined in module.go to avoid duplication
