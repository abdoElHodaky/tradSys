package engine

import (
	"time"
)

// RiskEngineConfig contains configuration for the risk engine
type RiskEngineConfig struct {
	MaxLatency            time.Duration `json:"max_latency"` // Target: <10μs
	EnablePreTradeChecks  bool          `json:"enable_pre_trade_checks"`
	EnablePostTradeChecks bool          `json:"enable_post_trade_checks"`
	EnableVaRCalculation  bool          `json:"enable_var_calculation"`
	EnableCircuitBreaker  bool          `json:"enable_circuit_breaker"`
	VaRConfidenceLevel    float64       `json:"var_confidence_level"` // 95%, 99%
	VaRTimeHorizon        time.Duration `json:"var_time_horizon"`     // 1 day
	MaxPositionSize       float64       `json:"max_position_size"`
	MaxOrderSize          float64       `json:"max_order_size"`
	MaxDailyLoss          float64       `json:"max_daily_loss"`
	StressTestEnabled     bool          `json:"stress_test_enabled"`
}

// RiskEventType defines types of risk events
type RiskEventType string

const (
	EventPreTradeCheck  RiskEventType = "pre_trade_check"
	EventPostTradeCheck RiskEventType = "post_trade_check"
	EventLimitBreach    RiskEventType = "limit_breach"
	EventCircuitBreaker RiskEventType = "circuit_breaker"
	EventVaRCalculation RiskEventType = "var_calculation"
	EventPositionUpdate RiskEventType = "position_update"
)

// RiskSeverity defines risk event severity levels
type RiskSeverity string

const (
	SeverityInfo     RiskSeverity = "info"
	SeverityWarning  RiskSeverity = "warning"
	SeverityError    RiskSeverity = "error"
	SeverityCritical RiskSeverity = "critical"
)

// RiskCheck represents a risk check result
type RiskCheck struct {
	CheckType    string        `json:"check_type"`
	Passed       bool          `json:"passed"`
	CurrentValue float64       `json:"current_value"`
	LimitValue   float64       `json:"limit_value"`
	Message      string        `json:"message"`
	Latency      time.Duration `json:"latency"`
	Timestamp    time.Time     `json:"timestamp"`
}

// RiskEvent represents a risk event
type RiskEvent struct {
	ID        string        `json:"id"`
	Type      RiskEventType `json:"type"`
	Severity  RiskSeverity  `json:"severity"`
	Message   string        `json:"message"`
	UserID    string        `json:"user_id,omitempty"`
	Symbol    string        `json:"symbol,omitempty"`
	OrderID   string        `json:"order_id,omitempty"`
	Data      interface{}   `json:"data,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

