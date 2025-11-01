package risk

import (
	"errors"
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
	// RiskLimitTypeConcentration represents a concentration limit
	RiskLimitTypeConcentration RiskLimitType = "concentration"
	// RiskLimitTypeVaR represents a Value at Risk limit
	RiskLimitTypeVaR RiskLimitType = "var"
)

// Position represents a trading position
type Position struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`
	AveragePrice   float64   `json:"average_price"`
	MarketValue    float64   `json:"market_value"`
	UnrealizedPnL  float64   `json:"unrealized_pnl"`
	RealizedPnL    float64   `json:"realized_pnl"`
	InstrumentType string    `json:"instrument_type"` // "stock", "option", "future", etc.
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RemainingQuantity returns the remaining quantity of the position
func (p *Position) RemainingQuantity() float64 {
	return p.Quantity
}

// RiskCheckResult represents the result of a risk check
type RiskCheckResult struct {
	Passed     bool      `json:"passed"`
	RiskLevel  RiskLevel `json:"risk_level"`
	Violations []string  `json:"violations"`
	Warnings   []string  `json:"warnings"`
	CheckedAt  time.Time `json:"checked_at"`
	Details    string    `json:"details,omitempty"`
}

// PositionRiskMetrics represents risk metrics for a position
type PositionRiskMetrics struct {
	Symbol               string    `json:"symbol"`
	UserID               string    `json:"user_id"`
	Quantity             float64   `json:"quantity"`
	AveragePrice         float64   `json:"average_price"`
	CurrentPrice         float64   `json:"current_price"`
	MarketValue          float64   `json:"market_value"`
	UnrealizedPnL        float64   `json:"unrealized_pnl"`
	UnrealizedPnLPercent float64   `json:"unrealized_pnl_percent"`
	VaR95                float64   `json:"var_95"`
	VaR99                float64   `json:"var_99"`
	ExpectedShortfall    float64   `json:"expected_shortfall"`
	Delta                float64   `json:"delta,omitempty"`
	Gamma                float64   `json:"gamma,omitempty"`
	Theta                float64   `json:"theta,omitempty"`
	Vega                 float64   `json:"vega,omitempty"`
	RiskLevel            RiskLevel `json:"risk_level"`
	CalculatedAt         time.Time `json:"calculated_at"`
}

// AccountRiskMetrics represents risk metrics for an entire account
type AccountRiskMetrics struct {
	UserID                    string                 `json:"user_id"`
	TotalUnrealizedPnL        float64                `json:"total_unrealized_pnl"`
	TotalUnrealizedPnLPercent float64                `json:"total_unrealized_pnl_percent"`
	TotalMarketValue          float64                `json:"total_market_value"`
	PortfolioVaR95            float64                `json:"portfolio_var_95"`
	PortfolioVaR99            float64                `json:"portfolio_var_99"`
	ConcentrationRisk         float64                `json:"concentration_risk"`
	CorrelationRisk           float64                `json:"correlation_risk"`
	RiskLevel                 RiskLevel              `json:"risk_level"`
	Positions                 []*PositionRiskMetrics `json:"positions"`
	CalculatedAt              time.Time              `json:"calculated_at"`
}

// OrderRiskMetrics represents risk metrics for an order
type OrderRiskMetrics struct {
	OrderID               string    `json:"order_id"`
	Symbol                string    `json:"symbol"`
	UserID                string    `json:"user_id"`
	Side                  string    `json:"side"`
	Quantity              float64   `json:"quantity"`
	Price                 float64   `json:"price"`
	CurrentPrice          float64   `json:"current_price"`
	OrderValue            float64   `json:"order_value"`
	CurrentPosition       float64   `json:"current_position"`
	NewPosition           float64   `json:"new_position"`
	PositionChange        float64   `json:"position_change"`
	PositionChangePercent float64   `json:"position_change_percent"`
	LeverageImpact        float64   `json:"leverage_impact"`
	MarginRequirement     float64   `json:"margin_requirement"`
	MaxLossPotential      float64   `json:"max_loss_potential"`
	RiskLevel             RiskLevel `json:"risk_level"`
	CalculatedAt          time.Time `json:"calculated_at"`
}

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

// Error definitions
var (
	ErrInvalidPosition       = errors.New("invalid position")
	ErrInvalidOrder          = errors.New("invalid order")
	ErrRiskLimitExceeded     = errors.New("risk limit exceeded")
	ErrCircuitBreakerTripped = errors.New("circuit breaker tripped")
	ErrInsufficientMargin    = errors.New("insufficient margin")
	ErrPositionNotFound      = errors.New("position not found")
	ErrRiskLimitNotFound     = errors.New("risk limit not found")
	ErrInvalidRiskLevel      = errors.New("invalid risk level")
	ErrRiskCheckFailed       = errors.New("risk check failed")
)
