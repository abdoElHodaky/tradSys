package risk

import (
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
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
	positions     sync.Map // map[string]*RealtimePosition
	totalPnL      float64
	dailyPnL      float64
	unrealizedPnL float64
	realizedPnL   float64
	mu            sync.RWMutex
}

// RealtimePosition represents a trading position for real-time risk monitoring
type RealtimePosition struct {
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`
	AveragePrice   float64   `json:"average_price"`
	MarketPrice    float64   `json:"market_price"`
	UnrealizedPnL  float64   `json:"unrealized_pnl"`
	RealizedPnL    float64   `json:"realized_pnl"`
	Delta          float64   `json:"delta"`
	Gamma          float64   `json:"gamma"`
	Vega           float64   `json:"vega"`
	Theta          float64   `json:"theta"`
	LastUpdateTime time.Time `json:"last_update_time"`
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
	mu                sync.RWMutex
}

// CircuitBreaker implements circuit breaker pattern for risk management
type CircuitBreaker struct {
	enabled         bool
	failureCount    int64
	successCount    int64
	lastFailureTime time.Time
	threshold       int64
	timeout         time.Duration
	state           CircuitBreakerState
	mu              sync.RWMutex
}

// CircuitBreakerState is defined in circuit_breaker.go to avoid duplication

// RiskCheck represents the result of a risk check
type RiskCheck struct {
	CheckType    string        `json:"check_type"`
	Passed       bool          `json:"passed"`
	CurrentValue float64       `json:"current_value,omitempty"`
	LimitValue   float64       `json:"limit_value,omitempty"`
	Message      string        `json:"message"`
	Latency      time.Duration `json:"latency"`
	Timestamp    time.Time     `json:"timestamp"`
}

// RiskEvent represents a risk-related event
type RiskEvent struct {
	Type      EventType              `json:"type"`
	Symbol    string                 `json:"symbol"`
	Order     *types.Order           `json:"order,omitempty"`
	Trade     *Trade                 `json:"trade,omitempty"`
	Severity  EventSeverity          `json:"severity"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventType represents different types of risk events
type EventType string

const (
	EventLimitBreach    EventType = "limit_breach"
	EventCircuitBreaker EventType = "circuit_breaker"
	EventVaRBreach      EventType = "var_breach"
	EventPositionUpdate EventType = "position_update"
	EventRiskAlert      EventType = "risk_alert"
)

// EventSeverity represents the severity of risk events
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityError    EventSeverity = "error"
	SeverityCritical EventSeverity = "critical"
)

// Trade represents a completed trade for risk calculations
type Trade struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// VaRResult represents Value at Risk calculation result
type VaRResult struct {
	Symbol          string    `json:"symbol"`
	VaR             float64   `json:"var"`
	ConfidenceLevel float64   `json:"confidence_level"`
	TimeHorizon     string    `json:"time_horizon"`
	Timestamp       time.Time `json:"timestamp"`
}

// StressTestScenario represents a stress test scenario
type StressTestScenario struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Shocks      map[string]float64 `json:"shocks"` // symbol -> shock percentage
	Enabled     bool               `json:"enabled"`
}

// StressTestResult represents stress test results
type StressTestResult struct {
	Scenario      string    `json:"scenario"`
	TotalPnL      float64   `json:"total_pnl"`
	WorstPosition string    `json:"worst_position"`
	WorstPnL      float64   `json:"worst_pnl"`
	Timestamp     time.Time `json:"timestamp"`
}

// RiskAlert represents a risk alert
type RiskAlert struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Symbol       string        `json:"symbol"`
	Message      string        `json:"message"`
	Severity     EventSeverity `json:"severity"`
	Threshold    float64       `json:"threshold"`
	CurrentValue float64       `json:"current_value"`
	Timestamp    time.Time     `json:"timestamp"`
	Acknowledged bool          `json:"acknowledged"`
}

// PortfolioRisk represents portfolio-level risk metrics
type PortfolioRisk struct {
	TotalVaR          float64            `json:"total_var"`
	ComponentVaR      map[string]float64 `json:"component_var"`
	MarginalVaR       map[string]float64 `json:"marginal_var"`
	ConcentrationRisk float64            `json:"concentration_risk"`
	LeverageRatio     float64            `json:"leverage_ratio"`
	BetaToMarket      float64            `json:"beta_to_market"`
	Timestamp         time.Time          `json:"timestamp"`
}
