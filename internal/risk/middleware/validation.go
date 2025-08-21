package middleware

import (
	"context"
	"errors"

	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrRiskValidationFailed = errors.New("risk validation failed")
)

// OrderValidationMiddleware validates orders against risk limits
type OrderValidationMiddleware struct {
	// Logger
	logger *zap.Logger

	// Risk validator
	riskValidator *risk.RiskValidator

	// Next handler
	next OrderHandler
}

// OrderHandler handles orders
type OrderHandler interface {
	// HandleOrder handles an order
	HandleOrder(ctx context.Context, order *orders.OrderRequest) (*orders.OrderResponse, error)
}

// NewOrderValidationMiddleware creates a new OrderValidationMiddleware
func NewOrderValidationMiddleware(
	logger *zap.Logger,
	riskValidator *risk.RiskValidator,
	next OrderHandler,
) *OrderValidationMiddleware {
	return &OrderValidationMiddleware{
		logger:        logger,
		riskValidator: riskValidator,
		next:          next,
	}
}

// HandleOrder handles an order
func (m *OrderValidationMiddleware) HandleOrder(
	ctx context.Context,
	order *orders.OrderRequest,
) (*orders.OrderResponse, error) {
	// Validate order
	result, err := m.riskValidator.ValidateOrder(ctx, order)
	if err != nil {
		m.logger.Error("Order validation failed",
			zap.Error(err),
			zap.String("symbol", order.Symbol),
			zap.String("account_id", order.AccountId),
			zap.Float64("quantity", order.Quantity),
			zap.Float64("price", order.Price))
		return nil, err
	}

	// Check result
	if !result.Passed {
		m.logger.Warn("Order rejected by risk validation",
			zap.String("reason", result.Reason),
			zap.Any("details", result.Details),
			zap.String("symbol", order.Symbol),
			zap.String("account_id", order.AccountId),
			zap.Float64("quantity", order.Quantity),
			zap.Float64("price", order.Price))
		return nil, ErrRiskValidationFailed
	}

	// Call next handler
	return m.next.HandleOrder(ctx, order)
}

// ExposureValidationMiddleware validates exposures against risk limits
type ExposureValidationMiddleware struct {
	// Logger
	logger *zap.Logger

	// Risk validator
	riskValidator *risk.RiskValidator

	// Risk manager
	riskManager *risk.RiskManager

	// Next handler
	next OrderHandler
}

// NewExposureValidationMiddleware creates a new ExposureValidationMiddleware
func NewExposureValidationMiddleware(
	logger *zap.Logger,
	riskValidator *risk.RiskValidator,
	riskManager *risk.RiskManager,
	next OrderHandler,
) *ExposureValidationMiddleware {
	return &ExposureValidationMiddleware{
		logger:        logger,
		riskValidator: riskValidator,
		riskManager:   riskManager,
		next:          next,
	}
}

// HandleOrder handles an order
func (m *ExposureValidationMiddleware) HandleOrder(
	ctx context.Context,
	order *orders.OrderRequest,
) (*orders.OrderResponse, error) {
	// Call next handler first
	response, err := m.next.HandleOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	// Validate exposure after order execution
	result, err := m.riskValidator.ValidateExposure(ctx, order.AccountId)
	if err != nil {
		m.logger.Error("Exposure validation failed",
			zap.Error(err),
			zap.String("account_id", order.AccountId))
		// Don't return error, just log it
	} else if !result.Passed {
		m.logger.Warn("Exposure validation failed after order execution",
			zap.String("reason", result.Reason),
			zap.Any("details", result.Details),
			zap.String("account_id", order.AccountId))

		// Generate risk report
		m.riskManager.GenerateRiskReport(order.AccountId)
	}

	return response, nil
}

// CircuitBreakerMiddleware implements circuit breaker pattern for order handling
type CircuitBreakerMiddleware struct {
	// Logger
	logger *zap.Logger

	// Risk manager
	riskManager *risk.RiskManager

	// Next handler
	next OrderHandler

	// Circuit breaker state
	circuitOpen bool

	// Circuit breaker thresholds
	failureThreshold int
	resetTimeout     int
	failureCount     int
	lastFailureTime  int64
}

// NewCircuitBreakerMiddleware creates a new CircuitBreakerMiddleware
func NewCircuitBreakerMiddleware(
	logger *zap.Logger,
	riskManager *risk.RiskManager,
	next OrderHandler,
) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		logger:           logger,
		riskManager:      riskManager,
		next:             next,
		circuitOpen:      false,
		failureThreshold: 5,
		resetTimeout:     60, // seconds
		failureCount:     0,
		lastFailureTime:  0,
	}
}

// HandleOrder handles an order
func (m *CircuitBreakerMiddleware) HandleOrder(
	ctx context.Context,
	order *orders.OrderRequest,
) (*orders.OrderResponse, error) {
	// Check if circuit is open
	if m.circuitOpen {
		// Check if reset timeout has elapsed
		now := time.Now().Unix()
		if now-m.lastFailureTime > int64(m.resetTimeout) {
			// Reset circuit
			m.circuitOpen = false
			m.failureCount = 0
			m.logger.Info("Circuit breaker reset")
		} else {
			// Circuit is still open
			m.logger.Warn("Circuit breaker open, rejecting order",
				zap.String("symbol", order.Symbol),
				zap.String("account_id", order.AccountId))
			return nil, errors.New("circuit breaker open")
		}
	}

	// Call next handler
	response, err := m.next.HandleOrder(ctx, order)
	if err != nil {
		// Increment failure count
		m.failureCount++
		m.lastFailureTime = time.Now().Unix()

		// Check if failure threshold is reached
		if m.failureCount >= m.failureThreshold {
			// Open circuit
			m.circuitOpen = true
			m.logger.Error("Circuit breaker opened due to failures",
				zap.Int("failure_count", m.failureCount),
				zap.Int("failure_threshold", m.failureThreshold))

			// Generate risk report
			m.riskManager.GenerateRiskReport(order.AccountId)
		}

		return nil, err
	}

	// Reset failure count on success
	m.failureCount = 0

	return response, nil
}

