package engine

import (
	"time"

	"github.com/google/uuid"
)

// RiskLevel represents the risk level of an operation
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// RiskCheckResult represents the result of a risk check
type RiskCheckResult struct {
	Passed       bool                   `json:"passed"`
	Approved     bool                   `json:"approved"`
	Reason       string                 `json:"reason"`
	RiskLevel    RiskLevel              `json:"risk_level"`
	RiskScore    float64                `json:"risk_score"`
	Violations   []string               `json:"violations"`
	Warnings     []string               `json:"warnings"`
	MaxOrderSize float64                `json:"max_order_size"`
	Latency      time.Duration          `json:"latency"`
	CheckedAt    time.Time              `json:"checked_at"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// RiskLimitType represents the type of risk limit
type RiskLimitType string

const (
	// RiskLimitTypePosition represents a position limit
	RiskLimitTypePosition RiskLimitType = "position"
	// RiskLimitTypePositionSize represents a position size limit
	RiskLimitTypePositionSize RiskLimitType = "position_size"
	// RiskLimitTypeOrderSize represents an order size limit
	RiskLimitTypeOrderSize RiskLimitType = "order_size"
	// RiskLimitTypeExposure represents an exposure limit
	RiskLimitTypeExposure RiskLimitType = "exposure"
	// RiskLimitTypeDrawdown represents a drawdown limit
	RiskLimitTypeDrawdown RiskLimitType = "drawdown"
	// RiskLimitTypeMaxDrawdown represents a maximum drawdown limit
	RiskLimitTypeMaxDrawdown RiskLimitType = "max_drawdown"
	// RiskLimitTypeDailyLoss represents a daily loss limit
	RiskLimitTypeDailyLoss RiskLimitType = "daily_loss"
	// RiskLimitTypeTradeFrequency represents a trade frequency limit
	RiskLimitTypeTradeFrequency RiskLimitType = "trade_frequency"
	// RiskLimitTypeVaR represents a Value at Risk limit
	RiskLimitTypeVaR RiskLimitType = "var"
	// RiskLimitTypeConcentration represents a concentration limit
	RiskLimitTypeConcentration RiskLimitType = "concentration"
	// RiskLimitTypeLeverage represents a leverage limit
	RiskLimitTypeLeverage RiskLimitType = "leverage"
)

// RiskLimit represents a risk limit configuration
type RiskLimit struct {
	ID          string        `json:"id"`
	Type        RiskLimitType `json:"type"`
	Symbol      string        `json:"symbol,omitempty"`
	UserID      string        `json:"user_id,omitempty"`
	AccountID   string        `json:"account_id,omitempty"`
	Limit       float64       `json:"limit"`
	Value       float64       `json:"value"`       // Current value for comparison
	Warning     float64       `json:"warning,omitempty"`
	Enabled     bool          `json:"enabled"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Description string        `json:"description,omitempty"`
	
	// Time-based limits
	TimeWindow *time.Duration `json:"time_window,omitempty"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RiskOperation represents a risk operation for batch processing
type RiskOperation struct {
	OpType    string      `json:"op_type"`
	UserID    string      `json:"user_id"`
	AccountID string      `json:"account_id"`
	Symbol    string      `json:"symbol"`
	Data      interface{} `json:"data"`
	ResultCh  chan RiskOperationResult `json:"-"`
}

// RiskOperationResult represents the result of a risk operation
type RiskOperationResult struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result"`
	Error   error       `json:"error"`
}

// Position represents a trading position
type Position struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	AccountID     string    `json:"account_id"`
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	AveragePrice  float64   `json:"average_price"`
	AvgPrice      float64   `json:"avg_price"`      // Alias for AveragePrice for compatibility
	MarketPrice   float64   `json:"market_price"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	RealizedPnL   float64   `json:"realized_pnl"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	
	// Risk metrics
	VaR           float64 `json:"var"`
	Beta          float64 `json:"beta"`
	Volatility    float64 `json:"volatility"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MarketDataUpdate represents a market data update
type MarketDataUpdate struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

// RiskMetrics represents comprehensive risk metrics
type RiskMetrics struct {
	UserID        string    `json:"user_id"`
	AccountID     string    `json:"account_id"`
	TotalExposure float64   `json:"total_exposure"`
	NetExposure   float64   `json:"net_exposure"`
	GrossExposure float64   `json:"gross_exposure"`
	VaR           float64   `json:"var"`
	MaxDrawdown   float64   `json:"max_drawdown"`
	Leverage      float64   `json:"leverage"`
	Beta          float64   `json:"beta"`
	Volatility    float64   `json:"volatility"`
	Sharpe        float64   `json:"sharpe"`
	UpdatedAt     time.Time `json:"updated_at"`
	
	// Position breakdown
	Positions map[string]*Position `json:"positions"`
	
	// Risk limits status
	LimitUtilization map[string]float64 `json:"limit_utilization"`
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerStateClosed CircuitBreakerState = "closed"
	CircuitBreakerStateOpen   CircuitBreakerState = "open"
	CircuitBreakerStateHalf   CircuitBreakerState = "half_open"
)

// CircuitBreaker represents a circuit breaker for risk management
type CircuitBreaker struct {
	ID               string              `json:"id"`
	Symbol           string              `json:"symbol"`
	State            CircuitBreakerState `json:"state"`
	Threshold        float64             `json:"threshold"`
	TimeWindow       time.Duration       `json:"time_window"`
	CooldownPeriod   time.Duration       `json:"cooldown_period"`
	FailureCount     int                 `json:"failure_count"`
	LastFailureTime  *time.Time          `json:"last_failure_time"`
	LastSuccessTime  *time.Time          `json:"last_success_time"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	
	// Configuration
	MaxFailures      int           `json:"max_failures"`
	HalfOpenRequests int           `json:"half_open_requests"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RiskAlert represents a risk alert
type RiskAlert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Level       RiskLevel `json:"level"`
	UserID      string    `json:"user_id"`
	AccountID   string    `json:"account_id"`
	Symbol      string    `json:"symbol,omitempty"`
	Message     string    `json:"message"`
	Details     map[string]interface{} `json:"details"`
	CreatedAt   time.Time `json:"created_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

// NewRiskLimit creates a new risk limit
func NewRiskLimit(limitType RiskLimitType, limit float64) *RiskLimit {
	return &RiskLimit{
		ID:        uuid.New().String(),
		Type:      limitType,
		Limit:     limit,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewPosition creates a new position
func NewPosition(userID, accountID, symbol string) *Position {
	return &Position{
		ID:        uuid.New().String(),
		UserID:    userID,
		AccountID: accountID,
		Symbol:    symbol,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(symbol string, threshold float64) *CircuitBreaker {
	return &CircuitBreaker{
		ID:               uuid.New().String(),
		Symbol:           symbol,
		State:            CircuitBreakerStateClosed,
		Threshold:        threshold,
		TimeWindow:       5 * time.Minute,
		CooldownPeriod:   1 * time.Minute,
		MaxFailures:      5,
		HalfOpenRequests: 3,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         make(map[string]interface{}),
	}
}

// NewRiskAlert creates a new risk alert
func NewRiskAlert(alertType string, level RiskLevel, userID, message string) *RiskAlert {
	return &RiskAlert{
		ID:        uuid.New().String(),
		Type:      alertType,
		Level:     level,
		UserID:    userID,
		Message:   message,
		Details:   make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
}

// IsActive returns true if the position has a non-zero quantity
func (p *Position) IsActive() bool {
	return p.Quantity != 0
}

// IsLong returns true if the position is long
func (p *Position) IsLong() bool {
	return p.Quantity > 0
}

// IsShort returns true if the position is short
func (p *Position) IsShort() bool {
	return p.Quantity < 0
}

// GetMarketValue returns the market value of the position
func (p *Position) GetMarketValue() float64 {
	return p.Quantity * p.MarketPrice
}

// GetNotionalValue returns the notional value of the position
func (p *Position) GetNotionalValue() float64 {
	return p.Quantity * p.AveragePrice
}

// UpdatePrice updates the position with a new market price
func (p *Position) UpdatePrice(price float64) {
	p.MarketPrice = price
	p.UnrealizedPnL = (price - p.AveragePrice) * p.Quantity
	p.UpdatedAt = time.Now()
}

// AddTrade adds a trade to the position
func (p *Position) AddTrade(quantity, price float64) {
	if p.Quantity == 0 {
		// New position
		p.Quantity = quantity
		p.AveragePrice = price
	} else if (p.Quantity > 0 && quantity > 0) || (p.Quantity < 0 && quantity < 0) {
		// Adding to existing position
		totalNotional := (p.Quantity * p.AveragePrice) + (quantity * price)
		p.Quantity += quantity
		p.AveragePrice = totalNotional / p.Quantity
	} else {
		// Reducing or reversing position
		if abs(quantity) >= abs(p.Quantity) {
			// Position reversal or closure
			realizedPnL := (price - p.AveragePrice) * p.Quantity
			p.RealizedPnL += realizedPnL
			
			remainingQuantity := quantity + p.Quantity
			if remainingQuantity != 0 {
				p.Quantity = remainingQuantity
				p.AveragePrice = price
			} else {
				p.Quantity = 0
				p.AveragePrice = 0
			}
		} else {
			// Partial reduction
			realizedPnL := (price - p.AveragePrice) * (-quantity)
			p.RealizedPnL += realizedPnL
			p.Quantity += quantity
		}
	}
	
	p.UpdatedAt = time.Now()
}

// IsBreached returns true if the limit is breached
func (rl *RiskLimit) IsBreached(value float64) bool {
	if !rl.Enabled {
		return false
	}
	return value > rl.Limit
}

// IsWarning returns true if the warning threshold is breached
func (rl *RiskLimit) IsWarning(value float64) bool {
	if !rl.Enabled || rl.Warning == 0 {
		return false
	}
	return value > rl.Warning
}

// GetUtilization returns the utilization percentage of the limit
func (rl *RiskLimit) GetUtilization(value float64) float64 {
	if rl.Limit == 0 {
		return 0
	}
	return (value / rl.Limit) * 100
}

// CanExecute returns true if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	return cb.State == CircuitBreakerStateClosed || cb.State == CircuitBreakerStateHalf
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	now := time.Now()
	cb.LastSuccessTime = &now
	cb.FailureCount = 0
	
	if cb.State == CircuitBreakerStateHalf {
		cb.State = CircuitBreakerStateClosed
	}
	
	cb.UpdatedAt = now
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	now := time.Now()
	cb.LastFailureTime = &now
	cb.FailureCount++
	
	if cb.FailureCount >= cb.MaxFailures {
		cb.State = CircuitBreakerStateOpen
	}
	
	cb.UpdatedAt = now
}

// ShouldAttemptReset returns true if the circuit breaker should attempt to reset
func (cb *CircuitBreaker) ShouldAttemptReset() bool {
	if cb.State != CircuitBreakerStateOpen {
		return false
	}
	
	if cb.LastFailureTime == nil {
		return false
	}
	
	return time.Since(*cb.LastFailureTime) >= cb.CooldownPeriod
}

// AttemptReset attempts to reset the circuit breaker to half-open state
func (cb *CircuitBreaker) AttemptReset() {
	if cb.ShouldAttemptReset() {
		cb.State = CircuitBreakerStateHalf
		cb.UpdatedAt = time.Now()
	}
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
