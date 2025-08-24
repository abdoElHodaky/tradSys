package risk

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// RiskConfig contains configuration for the risk management system
type RiskConfig struct {
	// MaxPositionSize is the maximum position size allowed
	MaxPositionSize float64
	
	// MaxDrawdown is the maximum drawdown allowed
	MaxDrawdown float64
	
	// MaxLeverage is the maximum leverage allowed
	MaxLeverage float64
	
	// MaxDailyLoss is the maximum daily loss allowed
	MaxDailyLoss float64
	
	// MaxOrderValue is the maximum order value allowed
	MaxOrderValue float64
	
	// MaxOrdersPerSecond is the maximum number of orders allowed per second
	MaxOrdersPerSecond int
	
	// RiskCheckTimeout is the timeout for risk checks
	RiskCheckTimeout time.Duration
}

// DefaultRiskConfig returns the default risk configuration
func DefaultRiskConfig() *RiskConfig {
	return &RiskConfig{
		MaxPositionSize:    1000000,
		MaxDrawdown:        0.1,
		MaxLeverage:        5,
		MaxDailyLoss:       50000,
		MaxOrderValue:      100000,
		MaxOrdersPerSecond: 10,
		RiskCheckTimeout:   100 * time.Millisecond,
	}
}

// RiskManager manages risk for trading operations
type RiskManager struct {
	logger *zap.Logger
	config *RiskConfig
	
	// Position tracking
	positions     map[string]float64
	positionMutex sync.RWMutex
	
	// Daily P&L tracking
	dailyPnL     float64
	dailyPnLMutex sync.RWMutex
	
	// Order rate limiting
	orderTimes     []time.Time
	orderTimesMutex sync.Mutex
}

// NewRiskManager creates a new risk manager
func NewRiskManager(config *RiskConfig, logger *zap.Logger) (*RiskManager, error) {
	if config == nil {
		config = DefaultRiskConfig()
	}
	
	return &RiskManager{
		logger:      logger,
		config:      config,
		positions:   make(map[string]float64),
		orderTimes:  make([]time.Time, 0, 100),
	}, nil
}

// CheckOrderRisk checks if an order passes risk checks
func (m *RiskManager) CheckOrderRisk(ctx context.Context, order *orders.OrderRequest) error {
	// Create a context with timeout for risk checks
	checkCtx, cancel := context.WithTimeout(ctx, m.config.RiskCheckTimeout)
	defer cancel()
	
	// Check order value
	if order.Price*order.Quantity > m.config.MaxOrderValue {
		m.logger.Warn("Order exceeds maximum order value",
			zap.String("order_id", order.ClientOrderID),
			zap.Float64("order_value", order.Price*order.Quantity),
			zap.Float64("max_order_value", m.config.MaxOrderValue))
		return ErrMaxOrderValueExceeded
	}
	
	// Check position size
	if err := m.checkPositionSize(checkCtx, order); err != nil {
		return err
	}
	
	// Check order rate
	if err := m.checkOrderRate(checkCtx); err != nil {
		return err
	}
	
	// Record the order time for rate limiting
	m.recordOrderTime()
	
	return nil
}

// checkPositionSize checks if the position size would exceed limits
func (m *RiskManager) checkPositionSize(ctx context.Context, order *orders.OrderRequest) error {
	m.positionMutex.RLock()
	defer m.positionMutex.RUnlock()
	
	// Get current position
	currentPosition := m.positions[order.Symbol]
	
	// Calculate new position
	var newPosition float64
	if order.Side == "buy" {
		newPosition = currentPosition + order.Quantity
	} else {
		newPosition = currentPosition - order.Quantity
	}
	
	// Check if position exceeds limits
	if newPosition > m.config.MaxPositionSize {
		m.logger.Warn("Position would exceed maximum position size",
			zap.String("symbol", order.Symbol),
			zap.Float64("current_position", currentPosition),
			zap.Float64("new_position", newPosition),
			zap.Float64("max_position_size", m.config.MaxPositionSize))
		return ErrMaxPositionSizeExceeded
	}
	
	return nil
}

// checkOrderRate checks if the order rate would exceed limits
func (m *RiskManager) checkOrderRate(ctx context.Context) error {
	m.orderTimesMutex.Lock()
	defer m.orderTimesMutex.Unlock()
	
	// Remove orders older than 1 second
	now := time.Now()
	cutoff := now.Add(-1 * time.Second)
	
	newOrderTimes := make([]time.Time, 0, len(m.orderTimes))
	for _, t := range m.orderTimes {
		if t.After(cutoff) {
			newOrderTimes = append(newOrderTimes, t)
		}
	}
	
	m.orderTimes = newOrderTimes
	
	// Check if order rate exceeds limits
	if len(m.orderTimes) >= m.config.MaxOrdersPerSecond {
		m.logger.Warn("Order rate would exceed maximum orders per second",
			zap.Int("current_rate", len(m.orderTimes)),
			zap.Int("max_rate", m.config.MaxOrdersPerSecond))
		return ErrMaxOrderRateExceeded
	}
	
	return nil
}

// recordOrderTime records the time of an order for rate limiting
func (m *RiskManager) recordOrderTime() {
	m.orderTimesMutex.Lock()
	defer m.orderTimesMutex.Unlock()
	
	m.orderTimes = append(m.orderTimes, time.Now())
}

// UpdatePosition updates the position for a symbol
func (m *RiskManager) UpdatePosition(symbol string, quantity float64) {
	m.positionMutex.Lock()
	defer m.positionMutex.Unlock()
	
	m.positions[symbol] = quantity
}

// GetPosition gets the position for a symbol
func (m *RiskManager) GetPosition(symbol string) float64 {
	m.positionMutex.RLock()
	defer m.positionMutex.RUnlock()
	
	return m.positions[symbol]
}

// UpdateDailyPnL updates the daily P&L
func (m *RiskManager) UpdateDailyPnL(pnl float64) {
	m.dailyPnLMutex.Lock()
	defer m.dailyPnLMutex.Unlock()
	
	m.dailyPnL = pnl
}

// GetDailyPnL gets the daily P&L
func (m *RiskManager) GetDailyPnL() float64 {
	m.dailyPnLMutex.RLock()
	defer m.dailyPnLMutex.RUnlock()
	
	return m.dailyPnL
}

// ResetDailyPnL resets the daily P&L
func (m *RiskManager) ResetDailyPnL() {
	m.dailyPnLMutex.Lock()
	defer m.dailyPnLMutex.Unlock()
	
	m.dailyPnL = 0
}

// RiskLimitChecker checks risk limits
type RiskLimitChecker struct {
	logger *zap.Logger
	config *RiskConfig
}

// NewRiskLimitChecker creates a new risk limit checker
func NewRiskLimitChecker(config *RiskConfig, logger *zap.Logger) (*RiskLimitChecker, error) {
	if config == nil {
		config = DefaultRiskConfig()
	}
	
	return &RiskLimitChecker{
		logger: logger,
		config: config,
	}, nil
}

// CheckDrawdown checks if drawdown exceeds limits
func (c *RiskLimitChecker) CheckDrawdown(equity, peak float64) error {
	if peak == 0 {
		return nil
	}
	
	drawdown := (peak - equity) / peak
	if drawdown > c.config.MaxDrawdown {
		c.logger.Warn("Drawdown exceeds maximum",
			zap.Float64("drawdown", drawdown),
			zap.Float64("max_drawdown", c.config.MaxDrawdown))
		return ErrMaxDrawdownExceeded
	}
	
	return nil
}

// CheckDailyLoss checks if daily loss exceeds limits
func (c *RiskLimitChecker) CheckDailyLoss(dailyPnL float64) error {
	if dailyPnL < -c.config.MaxDailyLoss {
		c.logger.Warn("Daily loss exceeds maximum",
			zap.Float64("daily_pnl", dailyPnL),
			zap.Float64("max_daily_loss", c.config.MaxDailyLoss))
		return ErrMaxDailyLossExceeded
	}
	
	return nil
}

// CheckLeverage checks if leverage exceeds limits
func (c *RiskLimitChecker) CheckLeverage(exposure, equity float64) error {
	if equity == 0 {
		return ErrInvalidEquity
	}
	
	leverage := exposure / equity
	if leverage > c.config.MaxLeverage {
		c.logger.Warn("Leverage exceeds maximum",
			zap.Float64("leverage", leverage),
			zap.Float64("max_leverage", c.config.MaxLeverage))
		return ErrMaxLeverageExceeded
	}
	
	return nil
}

// RiskReporter generates risk reports
type RiskReporter struct {
	logger *zap.Logger
}

// NewRiskReporter creates a new risk reporter
func NewRiskReporter(logger *zap.Logger) (*RiskReporter, error) {
	return &RiskReporter{
		logger: logger,
	}, nil
}

// GenerateRiskReport generates a risk report
func (r *RiskReporter) GenerateRiskReport(positions map[string]float64, dailyPnL float64) *RiskReport {
	return &RiskReport{
		Timestamp: time.Now(),
		Positions: positions,
		DailyPnL:  dailyPnL,
	}
}

// RiskReport contains risk information
type RiskReport struct {
	Timestamp time.Time
	Positions map[string]float64
	DailyPnL  float64
}

// Errors
var (
	ErrMaxPositionSizeExceeded = NewRiskError("maximum position size exceeded")
	ErrMaxOrderValueExceeded   = NewRiskError("maximum order value exceeded")
	ErrMaxOrderRateExceeded    = NewRiskError("maximum order rate exceeded")
	ErrMaxDrawdownExceeded     = NewRiskError("maximum drawdown exceeded")
	ErrMaxDailyLossExceeded    = NewRiskError("maximum daily loss exceeded")
	ErrMaxLeverageExceeded     = NewRiskError("maximum leverage exceeded")
	ErrInvalidEquity           = NewRiskError("invalid equity value")
)

// RiskError represents a risk error
type RiskError struct {
	message string
}

// NewRiskError creates a new risk error
func NewRiskError(message string) *RiskError {
	return &RiskError{message: message}
}

// Error returns the error message
func (e *RiskError) Error() string {
	return e.message
}

