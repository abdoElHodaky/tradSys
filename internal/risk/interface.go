package risk

import (
	"context"

	"github.com/abdoElHodaky/tradSys/proto/risk"
)

// Service defines the interface for risk management operations
type Service interface {
	// GetPositions gets positions for a user
	GetPositions(ctx context.Context, userID, symbol string) (*risk.GetPositionsResponse, error)
	
	// GetLimits gets risk limits for a user
	GetLimits(ctx context.Context, userID, symbol string, limitType risk.LimitType) (*risk.GetLimitsResponse, error)
	
	// SetLimit sets a risk limit for a user
	SetLimit(ctx context.Context, userID, symbol string, limitType risk.LimitType, value float64, enabled bool) (*risk.RiskLimitResponse, error)
	
	// DeleteLimit deletes a risk limit
	DeleteLimit(ctx context.Context, limitID, userID string) (*risk.DeleteLimitResponse, error)
	
	// ValidateOrder validates an order against risk limits
	ValidateOrder(ctx context.Context, userID, symbol, side, orderType string, quantity, price float64) (*risk.ValidateOrderResponse, error)
}
