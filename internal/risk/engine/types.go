package risk_management

import (
	"time"
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
	Passed     bool      `json:"passed"`
	RiskLevel  RiskLevel `json:"risk_level"`
	Violations []string  `json:"violations"`
	Warnings   []string  `json:"warnings"`
	CheckedAt  time.Time `json:"checked_at"`
}

// RiskLimitType represents the type of risk limit
type RiskLimitType string

const (
	// RiskLimitTypePosition represents a position limit
	RiskLimitTypePosition RiskLimitType = "position"
	// RiskLimitTypeOrderSize represents an order size limit
	RiskLimitTypeOrderSize RiskLimitType = "order_size"
	// RiskLimitTypeExposure represents an exposure limit
	RiskLimitTypeExposure RiskLimitType = "exposure"
	// RiskLimitTypeDrawdown represents a drawdown limit
	RiskLimitTypeDrawdown RiskLimitType = "drawdown"
	// RiskLimitTypeTradeFrequency represents a trade frequency limit
	RiskLimitTypeTradeFrequency RiskLimitType = "trade_frequency"
)

// RiskLimit represents a risk limit
type RiskLimit struct {
	// ID is the unique identifier for the risk limit
	ID string `json:"id"`
	// UserID is the user ID
	UserID string `json:"user_id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Type is the type of risk limit
	Type RiskLimitType `json:"type"`
	// Value is the limit value
	Value float64 `json:"value"`
	// CreatedAt is the time the risk limit was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the time the risk limit was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// Enabled indicates whether the risk limit is enabled
	Enabled bool `json:"enabled"`
}

// Position represents a trading position
type Position struct {
	// ID is the unique identifier for the position
	ID string `json:"id"`
	// UserID is the user ID
	UserID string `json:"user_id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Quantity is the position quantity (positive for long, negative for short)
	Quantity float64 `json:"quantity"`
	// AveragePrice is the average price of the position
	AveragePrice float64 `json:"average_price"`
	// MarketPrice is the current market price
	MarketPrice float64 `json:"market_price"`
	// UnrealizedPnL is the unrealized profit and loss
	UnrealizedPnL float64 `json:"unrealized_pnl"`
	// RealizedPnL is the realized profit and loss
	RealizedPnL float64 `json:"realized_pnl"`
	// Delta is the option delta (for options positions)
	Delta float64 `json:"delta"`
	// Gamma is the option gamma (for options positions)
	Gamma float64 `json:"gamma"`
	// Vega is the option vega (for options positions)
	Vega float64 `json:"vega"`
	// Theta is the option theta (for options positions)
	Theta float64 `json:"theta"`
	// LastUpdated is the time the position was last updated
	LastUpdated time.Time `json:"last_updated"`
	// CreatedAt is the time the position was created
	CreatedAt time.Time `json:"created_at"`
}

// CircuitBreaker represents a circuit breaker for a symbol
type CircuitBreaker struct {
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// PercentageThreshold is the percentage threshold for triggering the circuit breaker
	PercentageThreshold float64 `json:"percentage_threshold"`
	// TimeWindow is the time window for the circuit breaker
	TimeWindow time.Duration `json:"time_window"`
	// CooldownPeriod is the cooldown period after triggering
	CooldownPeriod time.Duration `json:"cooldown_period"`
	// LastPrice is the last known price
	LastPrice float64 `json:"last_price"`
	// LastTriggeredTime is the time the circuit breaker was last triggered
	LastTriggeredTime time.Time `json:"last_triggered"`
	// IsTriggeredFlag indicates whether the circuit breaker is currently triggered
	IsTriggeredFlag bool `json:"is_triggered"`
	// CreatedAt is the time the circuit breaker was created
	CreatedAt time.Time `json:"created_at"`
}

// IsTriggered returns whether the circuit breaker is currently triggered
func (cb *CircuitBreaker) IsTriggered() bool {
	return cb.IsTriggeredFlag
}

// LastTriggered returns the time the circuit breaker was last triggered
func (cb *CircuitBreaker) LastTriggered() time.Time {
	return cb.LastTriggeredTime
}

// RiskOperation represents a batch operation on risk data
type RiskOperation struct {
	// OpType is the operation type
	OpType string
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Data is the operation data
	Data interface{}
	// ResultCh is the result channel
	ResultCh chan RiskOperationResult
}

// RiskOperationResult represents the result of a risk operation
type RiskOperationResult struct {
	// Success indicates whether the operation was successful
	Success bool
	// Error is the error if the operation failed
	Error error
	// Data is the result data
	Data interface{}
}

// MarketDataUpdate represents a market data update
type MarketDataUpdate struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the current price
	Price float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// Operation types for batch processing
const (
	OpTypeUpdatePosition = "update_position"
	OpTypeCheckLimit     = "check_limit"
	OpTypeAddLimit       = "add_limit"
	OpTypeRemoveLimit    = "remove_limit"
	OpTypeGetPosition    = "get_position"
)
