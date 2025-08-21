package risk

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/resilience"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrRiskCheckFailed = errors.New("risk check failed")
	ErrRiskLimitExceeded = errors.New("risk limit exceeded")
	ErrInvalidOrder = errors.New("invalid order")
)

// RiskCheckResult represents the result of a risk check
type RiskCheckResult struct {
	// Passed indicates if the check passed
	Passed bool

	// Reason is the reason for failure
	Reason string

	// Details contains additional details
	Details map[string]interface{}
}

// RiskValidator validates orders against risk limits
type RiskValidator struct {
	// Logger
	logger *zap.Logger

	// Position limit manager
	positionLimitManager *PositionLimitManager

	// Exposure tracker
	exposureTracker *ExposureTracker

	// Circuit breaker factory
	circuitBreakerFactory *resilience.CircuitBreakerFactory

	// Risk limits
	riskLimits map[string]*RiskLimit

	// Mutex for thread safety
	mu sync.RWMutex
}

// RiskLimit represents a risk limit
type RiskLimit struct {
	// AccountID is the account ID
	AccountID string

	// MaxNotionalExposure is the maximum notional exposure
	MaxNotionalExposure float64

	// MaxBetaExposure is the maximum beta-adjusted exposure
	MaxBetaExposure float64

	// MaxSectorExposure is the maximum sector exposure
	MaxSectorExposure map[string]float64

	// MaxCurrencyExposure is the maximum currency exposure
	MaxCurrencyExposure map[string]float64

	// MaxDrawdown is the maximum drawdown
	MaxDrawdown float64

	// MaxDailyLoss is the maximum daily loss
	MaxDailyLoss float64

	// MaxOrderSize is the maximum order size
	MaxOrderSize map[string]float64

	// MaxOrderValue is the maximum order value
	MaxOrderValue float64
}

// NewRiskValidator creates a new RiskValidator
func NewRiskValidator(
	logger *zap.Logger,
	positionLimitManager *PositionLimitManager,
	exposureTracker *ExposureTracker,
	circuitBreakerFactory *resilience.CircuitBreakerFactory,
) *RiskValidator {
	return &RiskValidator{
		logger:                logger,
		positionLimitManager:  positionLimitManager,
		exposureTracker:       exposureTracker,
		circuitBreakerFactory: circuitBreakerFactory,
		riskLimits:            make(map[string]*RiskLimit),
	}
}

// SetRiskLimit sets a risk limit
func (v *RiskValidator) SetRiskLimit(limit *RiskLimit) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.riskLimits[limit.AccountID] = limit

	v.logger.Info("Set risk limit",
		zap.String("account_id", limit.AccountID),
		zap.Float64("max_notional_exposure", limit.MaxNotionalExposure),
		zap.Float64("max_beta_exposure", limit.MaxBetaExposure),
		zap.Float64("max_drawdown", limit.MaxDrawdown),
		zap.Float64("max_daily_loss", limit.MaxDailyLoss),
		zap.Float64("max_order_value", limit.MaxOrderValue))
}

// GetRiskLimit gets a risk limit
func (v *RiskValidator) GetRiskLimit(accountID string) *RiskLimit {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.riskLimits[accountID]
}

// RemoveRiskLimit removes a risk limit
func (v *RiskValidator) RemoveRiskLimit(accountID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.riskLimits, accountID)

	v.logger.Info("Removed risk limit",
		zap.String("account_id", accountID))
}

// ValidateOrder validates an order against risk limits
func (v *RiskValidator) ValidateOrder(
	ctx context.Context,
	order *orders.OrderRequest,
) (*RiskCheckResult, error) {
	// Use circuit breaker for resilience
	result := v.circuitBreakerFactory.ExecuteWithContext(ctx, "risk_validation", func(ctx context.Context) (interface{}, error) {
		return v.validateOrderInternal(order)
	})

	if result.Error != nil {
		return nil, result.Error
	}

	return result.Value.(*RiskCheckResult), nil
}

// validateOrderInternal validates an order against risk limits
func (v *RiskValidator) validateOrderInternal(order *orders.OrderRequest) (*RiskCheckResult, error) {
	// Basic validation
	if order.Symbol == "" || order.AccountId == "" {
		return &RiskCheckResult{
			Passed: false,
			Reason: "invalid order: missing symbol or account ID",
		}, ErrInvalidOrder
	}

	if order.Quantity <= 0 {
		return &RiskCheckResult{
			Passed: false,
			Reason: "invalid order: quantity must be positive",
		}, ErrInvalidOrder
	}

	// Get current position
	position := v.exposureTracker.GetPosition(order.Symbol, order.AccountId)
	currentLong := 0.0
	currentShort := 0.0
	if position != nil {
		currentLong = position.Long
		currentShort = position.Short
	}

	// Calculate delta positions
	deltaLong := 0.0
	deltaShort := 0.0
	if order.Side == orders.OrderSide_BUY {
		deltaLong = order.Quantity
	} else {
		deltaShort = order.Quantity
	}

	// Check position limits
	err := v.positionLimitManager.CheckLimit(
		order.Symbol,
		order.AccountId,
		currentLong,
		currentShort,
		deltaLong,
		deltaShort,
	)
	if err != nil {
		return &RiskCheckResult{
			Passed: false,
			Reason: "position limit exceeded: " + err.Error(),
		}, err
	}

	// Get risk limit
	riskLimit := v.GetRiskLimit(order.AccountId)
	if riskLimit == nil {
		// No risk limit, allow the order
		return &RiskCheckResult{
			Passed: true,
		}, nil
	}

	// Check order size limit
	if maxSize, exists := riskLimit.MaxOrderSize[order.Symbol]; exists && order.Quantity > maxSize {
		return &RiskCheckResult{
			Passed: false,
			Reason: "order size limit exceeded",
			Details: map[string]interface{}{
				"max_size":  maxSize,
				"order_size": order.Quantity,
			},
		}, ErrRiskLimitExceeded
	}

	// Check order value limit
	orderValue := order.Quantity * order.Price
	if riskLimit.MaxOrderValue > 0 && orderValue > riskLimit.MaxOrderValue {
		return &RiskCheckResult{
			Passed: false,
			Reason: "order value limit exceeded",
			Details: map[string]interface{}{
				"max_value":  riskLimit.MaxOrderValue,
				"order_value": orderValue,
			},
		}, ErrRiskLimitExceeded
	}

	// Get current exposure
	exposure := v.exposureTracker.GetExposure(order.AccountId)
	if exposure == nil {
		// No exposure, allow the order
		return &RiskCheckResult{
			Passed: true,
		}, nil
	}

	// Calculate new exposure
	newNotional := exposure.Notional
	if order.Side == orders.OrderSide_BUY {
		newNotional += orderValue
	} else {
		newNotional -= orderValue
	}

	// Check notional exposure limit
	if riskLimit.MaxNotionalExposure > 0 && abs(newNotional) > riskLimit.MaxNotionalExposure {
		return &RiskCheckResult{
			Passed: false,
			Reason: "notional exposure limit exceeded",
			Details: map[string]interface{}{
				"max_exposure":     riskLimit.MaxNotionalExposure,
				"current_exposure": exposure.Notional,
				"new_exposure":     newNotional,
			},
		}, ErrRiskLimitExceeded
	}

	// More checks could be added here for beta exposure, sector exposure, etc.

	return &RiskCheckResult{
		Passed: true,
	}, nil
}

// ValidateExposure validates an account's exposure against risk limits
func (v *RiskValidator) ValidateExposure(
	ctx context.Context,
	accountID string,
) (*RiskCheckResult, error) {
	// Use circuit breaker for resilience
	result := v.circuitBreakerFactory.ExecuteWithContext(ctx, "exposure_validation", func(ctx context.Context) (interface{}, error) {
		return v.validateExposureInternal(accountID)
	})

	if result.Error != nil {
		return nil, result.Error
	}

	return result.Value.(*RiskCheckResult), nil
}

// validateExposureInternal validates an account's exposure against risk limits
func (v *RiskValidator) validateExposureInternal(accountID string) (*RiskCheckResult, error) {
	// Get risk limit
	riskLimit := v.GetRiskLimit(accountID)
	if riskLimit == nil {
		// No risk limit, allow the exposure
		return &RiskCheckResult{
			Passed: true,
		}, nil
	}

	// Get current exposure
	exposure := v.exposureTracker.GetExposure(accountID)
	if exposure == nil {
		// No exposure, allow it
		return &RiskCheckResult{
			Passed: true,
		}, nil
	}

	// Check notional exposure limit
	if riskLimit.MaxNotionalExposure > 0 && abs(exposure.Notional) > riskLimit.MaxNotionalExposure {
		return &RiskCheckResult{
			Passed: false,
			Reason: "notional exposure limit exceeded",
			Details: map[string]interface{}{
				"max_exposure":     riskLimit.MaxNotionalExposure,
				"current_exposure": exposure.Notional,
			},
		}, ErrRiskLimitExceeded
	}

	// Check beta exposure limit
	if riskLimit.MaxBetaExposure > 0 && abs(exposure.Beta) > riskLimit.MaxBetaExposure {
		return &RiskCheckResult{
			Passed: false,
			Reason: "beta exposure limit exceeded",
			Details: map[string]interface{}{
				"max_exposure":     riskLimit.MaxBetaExposure,
				"current_exposure": exposure.Beta,
			},
		}, ErrRiskLimitExceeded
	}

	// Check sector exposure limits
	for sector, sectorExposure := range exposure.Sector {
		if maxExposure, exists := riskLimit.MaxSectorExposure[sector]; exists && abs(sectorExposure) > maxExposure {
			return &RiskCheckResult{
				Passed: false,
				Reason: "sector exposure limit exceeded",
				Details: map[string]interface{}{
					"sector":           sector,
					"max_exposure":     maxExposure,
					"current_exposure": sectorExposure,
				},
			}, ErrRiskLimitExceeded
		}
	}

	// Check currency exposure limits
	for currency, currencyExposure := range exposure.Currency {
		if maxExposure, exists := riskLimit.MaxCurrencyExposure[currency]; exists && abs(currencyExposure) > maxExposure {
			return &RiskCheckResult{
				Passed: false,
				Reason: "currency exposure limit exceeded",
				Details: map[string]interface{}{
					"currency":         currency,
					"max_exposure":     maxExposure,
					"current_exposure": currencyExposure,
				},
			}, ErrRiskLimitExceeded
		}
	}

	return &RiskCheckResult{
		Passed: true,
	}, nil
}

