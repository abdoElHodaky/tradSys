// 🎯 **Risk Engine Types**
// Generated using TradSys Code Splitting Standards
// 
// This file contains type definitions, constants, and data structures
// for the RealTime Risk Engine component. All types follow the established
// naming conventions and include comprehensive documentation.
//
// File size limit: 300 lines

package risk_management

import (
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// RiskEventType represents the type of risk event
type RiskEventType string

const (
	RiskEventTypePreTrade     RiskEventType = "pre_trade"
	RiskEventTypePostTrade    RiskEventType = "post_trade"
	RiskEventTypePositionRisk RiskEventType = "position_risk"
	RiskEventTypeVaRUpdate    RiskEventType = "var_update"
	RiskEventTypeCircuitBreak RiskEventType = "circuit_break"
	RiskEventTypeLimitBreach  RiskEventType = "limit_breach"
)

// RiskCheckStatus represents the status of a risk check
type RiskCheckStatus string

const (
	RiskCheckStatusPassed   RiskCheckStatus = "passed"
	RiskCheckStatusRejected RiskCheckStatus = "rejected"
	RiskCheckStatusPending  RiskCheckStatus = "pending"
	RiskCheckStatusError    RiskCheckStatus = "error"
)

// RealTimeRiskEngine provides real-time risk management with HFT performance
type RealTimeRiskEngine struct {
	config          *RiskEngineConfig
	logger          *zap.Logger
	positionManager *PositionManager
	limitManager    *LimitManager
	varCalculator   *VaRCalculator
	circuitBreaker  *CircuitBreaker
	metrics         *RiskMetrics
	eventPool       *pool.ObjectPool
	checkPool       *pool.ObjectPool
	isRunning       int32
	stopChannel     chan struct{}
	eventChannel    chan *RiskEvent
	mu              sync.RWMutex
}

// RiskEngineConfig contains configuration for the risk engine
type RiskEngineConfig struct {
	MaxLatency            time.Duration `json:"max_latency"`             // Target: <10μs
	EnablePreTradeChecks  bool          `json:"enable_pre_trade_checks"`
	EnablePostTradeChecks bool          `json:"enable_post_trade_checks"`
	EnableVaRCalculation  bool          `json:"enable_var_calculation"`
	EnableCircuitBreaker  bool          `json:"enable_circuit_breaker"`
	VaRConfidenceLevel    float64       `json:"var_confidence_level"`    // 95%, 99%
	VaRTimeHorizon        time.Duration `json:"var_time_horizon"`        // 1 day
	MaxPositionSize       float64       `json:"max_position_size"`
	MaxOrderSize          float64       `json:"max_order_size"`
	MaxDailyLoss          float64       `json:"max_daily_loss"`
	StressTestEnabled     bool          `json:"stress_test_enabled"`
}

// RiskMetrics tracks risk engine performance
type RiskMetrics struct {
	ChecksPerSecond     float64       `json:"checks_per_second"`
	AverageLatency      time.Duration `json:"average_latency"`
	MaxLatency          time.Duration `json:"max_latency"`
	TotalChecks         int64         `json:"total_checks"`
	RejectedOrders      int64         `json:"rejected_orders"`
	CircuitBreakerTrips int64         `json:"circuit_breaker_trips"`
	LastUpdateTime      time.Time     `json:"last_update_time"`
}

// PositionManager manages real-time positions
type PositionManager struct {
	positions     sync.Map // map[string]*Position
	totalPnL      float64
	dailyPnL      float64
	unrealizedPnL float64
	realizedPnL   float64
	mu            sync.RWMutex
}

// LimitManager manages trading limits
type LimitManager struct {
	positionLimits   map[string]float64 // symbol -> max position
	orderLimits      map[string]float64 // symbol -> max order size
	dailyLossLimit   float64
	currentDailyLoss float64
	mu               sync.RWMutex
}

// VaRCalculator calculates Value at Risk
type VaRCalculator struct {
	enabled           bool
	confidenceLevel   float64
	timeHorizon       time.Duration
	historicalReturns map[string][]float64 // symbol -> returns
	correlationMatrix map[string]map[string]float64
	currentVaR        float64
	lastCalculation   time.Time
	mu                sync.RWMutex
}

// CircuitBreaker type is defined in types.go to avoid duplication

// RiskEvent represents a risk management event
type RiskEvent struct {
	Type      RiskEventType `json:"type"`
	Symbol    string        `json:"symbol"`
	Order     *types.Order  `json:"order,omitempty"`
	Position  *Position     `json:"position,omitempty"`
	RiskCheck *RiskCheck    `json:"risk_check,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// RiskCheck represents a risk check result
type RiskCheck struct {
	ID           string          `json:"id"`
	Type         RiskEventType   `json:"type"`
	Status       RiskCheckStatus `json:"status"`
	Symbol       string          `json:"symbol"`
	CurrentValue float64         `json:"current_value"`
	LimitValue   float64         `json:"limit_value"`
	Message      string          `json:"message"`
	Latency      time.Duration   `json:"latency"`
	Timestamp    time.Time       `json:"timestamp"`
}

// Position type is defined in types.go to avoid duplication

// RiskCheckRequest represents a risk check request
type RiskCheckRequest struct {
	ID        string        `json:"id"`
	Type      RiskEventType `json:"type"`
	Symbol    string        `json:"symbol"`
	Order     *types.Order  `json:"order,omitempty"`
	Position  *Position     `json:"position,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// RiskCheckResponse represents a risk check response
type RiskCheckResponse struct {
	ID           string          `json:"id"`
	Status       RiskCheckStatus `json:"status"`
	Passed       bool            `json:"passed"`
	CurrentValue float64         `json:"current_value"`
	LimitValue   float64         `json:"limit_value"`
	Message      string          `json:"message"`
	Latency      time.Duration   `json:"latency"`
	Timestamp    time.Time       `json:"timestamp"`
}
